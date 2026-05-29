// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	actions_model "forgejo.org/models/actions"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/util"
)

// Sentinel errors returned by OpenJobLogReader. The HTTP handler maps each
// of these (and util.ErrNotExist) to 404; anything else is a 500.
var (
	ErrJobNotExecuted = errors.New("job has not been executed yet")
	ErrLogsExpired    = errors.New("logs have expired")
	ErrStepOutOfRange = errors.New("step index out of range")
)

// boundedReadSeekCloser exposes a [start, start+size) window of the underlying
// reader as if it started at offset 0. It exists so the step-filtered path
// can return an io.ReadSeekCloser (required by http.ServeContent for Range)
// without buffering the entire step slice into memory.
type boundedReadSeekCloser struct {
	inner io.ReadSeekCloser
	start int64
	size  int64
	pos   int64
}

func newBoundedReadSeekCloser(inner io.ReadSeekCloser, start, size int64) (*boundedReadSeekCloser, error) {
	if _, err := inner.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	return &boundedReadSeekCloser{inner: inner, start: start, size: size}, nil
}

func (b *boundedReadSeekCloser) Read(p []byte) (int, error) {
	if b.pos >= b.size {
		return 0, io.EOF
	}
	if remaining := b.size - b.pos; int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := b.inner.Read(p)
	b.pos += int64(n)
	return n, err
}

func (b *boundedReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = b.pos + offset
	case io.SeekEnd:
		abs = b.size + offset
	default:
		return 0, errors.New("boundedReadSeekCloser: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("boundedReadSeekCloser: negative position")
	}
	if _, err := b.inner.Seek(b.start+abs, io.SeekStart); err != nil {
		return 0, err
	}
	b.pos = abs
	return abs, nil
}

func (b *boundedReadSeekCloser) Close() error { return b.inner.Close() }

// OpenJobLogReader returns a reader for an action job's log along with the
// filename and modtime to expose via Content-Disposition / Last-Modified.
//
// attempt, when set, selects a specific historical attempt (uses
// GetTaskByJobAttempt). When unset, the latest attempt is used via the
// job.TaskID pointer maintained by the runner.
//
// stepFilter, when set, narrows the returned bytes to the slice covered by
// that step in the FullSteps numbering: 0 is the "Set up job" head, the
// last index is the "Complete job" tail, real steps are in between. Range
// requests still work via http.ServeContent over the bounded window.
//
// The caller is responsible for closing the returned reader.
func OpenJobLogReader(
	ctx context.Context,
	repo *repo_model.Repository,
	jobID int64,
	attempt optional.Option[int64],
	stepFilter optional.Option[int],
) (io.ReadSeekCloser, string, time.Time, error) {
	job, err := actions_model.GetRunJobByID(ctx, jobID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	// Run-jobs live in their own table; enforce repo ownership here so the
	// API layer can stay thin.
	if job.RepoID != repo.ID {
		return nil, "", time.Time{}, util.ErrNotExist
	}

	hasAttempt, attemptVal := attempt.Get()
	hasStep, stepIdx := stepFilter.Get()

	var task *actions_model.ActionTask
	switch {
	case hasAttempt:
		task, err = actions_model.GetTaskByJobAttempt(ctx, job.ID, attemptVal)
		if err != nil {
			return nil, "", time.Time{}, err
		}
	case job.TaskID == 0:
		// Job exists, but no runner has picked it up yet (or a re-run has
		// zeroed TaskID and the next runner hasn't claimed it).
		return nil, "", time.Time{}, ErrJobNotExecuted
	default:
		task, err = actions_model.GetTaskByID(ctx, job.TaskID)
		if err != nil {
			return nil, "", time.Time{}, err
		}
	}

	if task.LogExpired {
		return nil, "", time.Time{}, ErrLogsExpired
	}

	// Resolve the step's byte window (if any) BEFORE opening the log reader
	// so a request that's going to 404 doesn't pay for an OpenLogs call.
	var startByte, length int64
	if hasStep {
		// Targeted step load — task.LoadAttributes would also pull job + run
		// which we don't need (and which makes unit-test setup heavier).
		steps, err := actions_model.GetTaskStepsByTaskID(ctx, task.ID)
		if err != nil {
			return nil, "", time.Time{}, fmt.Errorf("could not load steps for task %d: %w", task.ID, err)
		}
		task.Steps = steps
		full := actions.FullSteps(task)
		if stepIdx < 0 || stepIdx >= len(full) {
			return nil, "", time.Time{}, ErrStepOutOfRange
		}
		step := full[stepIdx]
		// FullSteps populates LogIndex/LogLength uniformly for setup/real/complete.
		// LogLength==0 (e.g. a tail on a still-running task) maps to an empty window.
		if step.LogLength > 0 && step.LogIndex < int64(len(task.LogIndexes)) {
			startByte = task.LogIndexes[step.LogIndex]
			endIdx := step.LogIndex + step.LogLength
			var endByte int64
			if endIdx < int64(len(task.LogIndexes)) {
				endByte = task.LogIndexes[endIdx]
			} else {
				// Step's log range extends beyond recorded indexes (step is
				// still running or task hasn't fully landed yet).
				endByte = task.LogSize
			}
			length = endByte - startByte
		}
	}

	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("open logs for task %d: %w", task.ID, err)
	}

	modtime := task.Stopped.AsTime()
	if task.Stopped == 0 {
		modtime = task.Updated.AsTime() // Best-guess modtime while still running.
	}

	if !hasStep {
		filename := fmt.Sprintf("job-%d-attempt-%d.log", job.ID, task.Attempt)
		return reader, filename, modtime, nil
	}

	bounded, err := newBoundedReadSeekCloser(reader, startByte, length)
	if err != nil {
		reader.Close()
		return nil, "", time.Time{}, fmt.Errorf("could not position log reader at step %d start (offset %d): %w", stepIdx, startByte, err)
	}

	stepFilename := fmt.Sprintf("job-%d-attempt-%d-step-%d.log", job.ID, task.Attempt, stepIdx)
	return bounded, stepFilename, modtime, nil
}

// WriteRunLogsZip writes a ZIP of the latest per-job logs for the run to w.
// Each entry is named {job-name}-{job-id}-attempt-{N}.log, where N is that
// job's current attempt — the run itself has no attempt number, so jobs that
// were re-run independently show different attempts here. Jobs that haven't
// run, can't be looked up, or have expired logs get a .MISSING marker; a
// mid-stream read failure gets a sibling .PARTIAL marker. Any ZIP-level
// write failure (e.g. the HTTP client disconnects mid-stream) is propagated
// so the caller can abort instead of churning through the remaining jobs.
// Caller sets Content-Type / Content-Disposition before calling.
func WriteRunLogsZip(ctx context.Context, w io.Writer, run *actions_model.ActionRun) error {
	jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("get jobs for run %d: %w", run.ID, err)
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	// strip control bytes and path separators; UTF-8 passes through.
	sanitize := func(name string) string {
		cleaned := strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f || r == '/' || r == '\\' {
				return -1
			}
			return r
		}, name)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			cleaned = "job"
		}
		return cleaned
	}

	entryName := func(job *actions_model.ActionRunJob, suffix string) string {
		return fmt.Sprintf("%s-%d-attempt-%d.%s", sanitize(job.Name), job.ID, job.Attempt, suffix)
	}

	writeMarker := func(job *actions_model.ActionRunJob, suffix, msg string) error {
		entry, werr := zw.Create(entryName(job, suffix))
		if werr != nil {
			return werr
		}
		_, werr = entry.Write([]byte(msg))
		return werr
	}

	// Inner closure so reader.Close runs per iteration via defer.
	processJob := func(job *actions_model.ActionRunJob) error {
		if job.TaskID == 0 {
			return writeMarker(job, "MISSING", "job has not been executed yet\n")
		}
		task, err := actions_model.GetTaskByID(ctx, job.TaskID)
		if err != nil {
			return writeMarker(job, "MISSING", fmt.Sprintf("task lookup failed: %v\n", err))
		}
		if task.LogExpired {
			return writeMarker(job, "MISSING", "logs have been cleaned up\n")
		}

		reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
		if err != nil {
			return writeMarker(job, "MISSING", fmt.Sprintf("log open failed: %v\n", err))
		}
		defer reader.Close()

		entry, err := zw.Create(entryName(job, "log"))
		if err != nil {
			return writeMarker(job, "MISSING", fmt.Sprintf("zip entry create failed: %v\n", err))
		}

		if _, copyErr := io.Copy(entry, reader); copyErr != nil {
			return writeMarker(job, "PARTIAL", fmt.Sprintf("log read failed mid-stream: %v\n", copyErr))
		}
		return nil
	}

	for _, job := range jobs {
		if err := processJob(job); err != nil {
			return err
		}
	}
	return nil
}
