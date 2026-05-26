// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	ErrStepNoLogRange = errors.New("step has no recorded log range")
)

// nopReadSeekCloser wraps a bytes.Reader so the step-filtered path can return
// the same io.ReadSeekCloser shape as actions.OpenLogs.
type nopReadSeekCloser struct {
	*bytes.Reader
}

func (n nopReadSeekCloser) Close() error { return nil }

// OpenJobLogReader returns a reader for an action job's log along with the
// filename and modtime to expose via Content-Disposition / Last-Modified.
//
//   - attempt, when set, selects a specific historical attempt (uses
//     GetTaskByJobAttempt). When unset, the latest attempt is used via the
//     job.TaskID pointer maintained by the runner.
//   - stepFilter, when set, narrows the returned bytes to that 0-indexed
//     step's portion of the log. The reader is then a bytes.Reader over the
//     pre-sliced bytes; Range requests still work via http.ServeContent.
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

	// Validate the step filter (if any) BEFORE opening the log reader so a
	// request that's going to 404 doesn't pay for an OpenLogs call.
	var startByte, length int64
	if hasStep {
		step, err := actions_model.GetTaskStepByTaskIDAndIndex(ctx, task.ID, int64(stepIdx))
		if err != nil {
			if errors.Is(err, util.ErrNotExist) {
				return nil, "", time.Time{}, ErrStepOutOfRange
			}
			return nil, "", time.Time{}, fmt.Errorf("look up step %d on task %d: %w", stepIdx, task.ID, err)
		}

		// LogIndexes maps line number -> byte offset; guard against a task
		// whose log indexes haven't been populated (or whose step references
		// a line beyond what's been recorded).
		if step.LogIndex < 0 || step.LogIndex >= int64(len(task.LogIndexes)) {
			return nil, "", time.Time{}, ErrStepNoLogRange
		}

		startByte = task.LogIndexes[step.LogIndex]
		endIdx := step.LogIndex + step.LogLength
		var endByte int64
		if endIdx < int64(len(task.LogIndexes)) {
			endByte = task.LogIndexes[endIdx]
		} else {
			// Step's log range extends beyond recorded indexes (e.g. the
			// step is still running). Treat the slice as running to the end
			// of the log rather than refusing to serve.
			endByte = task.LogSize
		}
		length = endByte - startByte
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
		filename := fmt.Sprintf("job-%d.log", job.ID)
		if hasAttempt {
			filename = fmt.Sprintf("job-%d-attempt-%d.log", job.ID, attemptVal)
		}
		return reader, filename, modtime, nil
	}

	if _, err := reader.Seek(startByte, io.SeekStart); err != nil {
		reader.Close()
		return nil, "", time.Time{}, fmt.Errorf("seek log to step start: %w", err)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		reader.Close()
		return nil, "", time.Time{}, fmt.Errorf("read step log slice: %w", err)
	}
	reader.Close()

	// Step-filtered filename keeps step and attempt distinct so multiple
	// step-filtered entries can coexist inside a ZIP of run logs.
	stepFilename := fmt.Sprintf("job-%d-step-%d.log", job.ID, stepIdx)
	if hasAttempt {
		stepFilename = fmt.Sprintf("job-%d-attempt-%d-step-%d.log", job.ID, attemptVal, stepIdx)
	}

	return nopReadSeekCloser{bytes.NewReader(buf)}, stepFilename, modtime, nil
}
