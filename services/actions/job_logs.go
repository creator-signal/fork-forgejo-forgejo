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
	"forgejo.org/modules/util"
)

// Sentinel errors returned by OpenJobLogReader. The HTTP handler maps each
// of these (and util.ErrNotExist to 404; anything else is a 500.)
var (
	ErrJobNotExecuted = errors.New("job has not been executed yet")
	ErrLogsExpired    = errors.New("logs have expired")
	ErrStepOutOfRange = errors.New("step index out of range")
	ErrStepNoLogRange = errors.New("step has no recorded log range")
)

// nopReadSeekCloser wraps a bytes.Reader so the step-filtered path can return
// the same io.ReadSeekCloser shape as actions.OpenLogs
type nopReadSeekCloser struct {
	*bytes.Reader
}

func (n nopReadSeekCloser) Close() error { return nil }

// OpenJobLogReader returns a reader for an action job's log along with the
// filename and modtime to expose via Content-Disposition / Last-Modified.
//
//   - attempt > 0 selects a specific historical attempt (uses
//     GetTaskByJobAttempt). attempt == nil means "latest",
//     which uses the job.TaskID pointer maintained by the runner.
//   - stepFilter, when non-nil, narrows the returned bytes to that 0-indexed
//     step's portion of the log. The reader is then a bytes.Reader over the
//     pre-sliced bytes; Range requests still work via http.ServeContent.
//
// The caller is responsible for closing the returned reader.
func OpenJobLogReader(
	ctx context.Context,
	repo *repo_model.Repository,
	jobID int64,
	attempt *int64,
	stepFilter *int,
) (io.ReadSeekCloser, string, time.Time, error) {
	job, err := actions_model.GetRunJobByID(ctx, jobID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	// Run-jobs live in their own table; enforce repo ownership here so the
	// API layer can stay thin
	if job.RepoID != repo.ID {
		return nil, "", time.Time{}, util.ErrNotExist
	}

	var task *actions_model.ActionTask
	switch {
	case attempt != nil:
		task, err = actions_model.GetTaskByJobAttempt(ctx, job.ID, *attempt)
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

	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	modtime := task.Stopped.AsTime()
	if task.Stopped == 0 {
		modtime = task.Updated.AsTime() // Grab a best guess for modtime if still running
	}

	filename := fmt.Sprintf("job-%d.log", job.ID)
	if attempt != nil {
		filename = fmt.Sprintf("job-%d-attempt-%d.log", job.ID, *attempt)
	}

	if stepFilter == nil {
		return reader, filename, modtime, nil
	}

	// Step filter path: slice the log to the requested step's byte range
	step, err := actions_model.GetTaskStepByTaskIDAndIndex(ctx, task.ID, int64(*stepFilter))
	if err != nil {
		reader.Close()
		if errors.Is(err, util.ErrNotExist) {
			return nil, "", time.Time{}, ErrStepOutOfRange
		}
		return nil, "", time.Time{}, err
	}

	// LogIndexes maps line number --> byte offset; guard against a task whose
	// log indexes haven't been populated (or whose step references a line
	// beyond what's been recorded).
	if step.LogIndex < 0 || step.LogIndex >= int64(len(task.LogIndexes)) {
		reader.Close()
		return nil, "", time.Time{}, ErrStepNoLogRange
	}

	startByte := task.LogIndexes[step.LogIndex]
	endIdx := step.LogIndex + step.LogLength
	var endByte int64
	if endIdx < int64(len(task.LogIndexes)) {
		endByte = task.LogIndexes[endIdx]
	} else {
		// If the step's log range extends beyond the recorded indexes,
		// assume it goes to the end of the log. This can happen if the step is
		// still running and producing logs, so we don't want to be too strict
		// about the presence of a "next" index.
		endByte = task.LogSize
	}
	length := endByte - startByte

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

	// Format the filename to reflect the job/step/attempt filters applied,
	// to ensure unique-ness amongst multiple step-filtered entries in a ZIP of logs for a workflow run
	stepFilename := fmt.Sprintf("job-%d-step-%d.log", job.ID, *stepFilter)
	if attempt != nil {
		stepFilename = fmt.Sprintf("job-%d-attempt-%d-step-%d.log", job.ID, *attempt, *stepFilter)
	}

	return nopReadSeekCloser{bytes.NewReader(buf)}, stepFilename, modtime, nil
}
