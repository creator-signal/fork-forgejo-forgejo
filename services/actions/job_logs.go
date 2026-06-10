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

var (
	ErrJobNotExecuted = errors.New("job has not been executed yet")
	ErrLogsExpired    = errors.New("logs have expired")
)

func OpenJobLogReader(
	ctx context.Context,
	repo *repo_model.Repository,
	jobID int64,
	attempt optional.Option[int64],
) (io.ReadSeekCloser, string, time.Time, error) {
	job, err := actions_model.GetRunJobByID(ctx, jobID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if job.RepoID != repo.ID {
		return nil, "", time.Time{}, util.ErrNotExist
	}

	hasAttempt, attemptVal := attempt.Get()

	var task *actions_model.ActionTask
	switch {
	case hasAttempt:
		task, err = actions_model.GetTaskByJobAttempt(ctx, job.ID, attemptVal)
		if err != nil {
			return nil, "", time.Time{}, err
		}
	case job.TaskID == 0:
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
		return nil, "", time.Time{}, fmt.Errorf("open logs for task %d: %w", task.ID, err)
	}

	modtime := task.Stopped.AsTime()
	if task.Stopped == 0 {
		modtime = task.Updated.AsTime()
	}

	filename := fmt.Sprintf("job-%d-attempt-%d.log", job.ID, task.Attempt)
	return reader, filename, modtime, nil
}

func WriteRunLogsZip(ctx context.Context, w io.Writer, run *actions_model.ActionRun) error {
	jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("get jobs for run %d: %w", run.ID, err)
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

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
