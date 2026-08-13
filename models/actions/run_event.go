// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

type ActionRunEvent interface {
	GetRun() *ActionRun
}

var _ ActionRunEvent = &NewWorkflowRunAttempt{}

type NewWorkflowRunAttempt struct {
	run *ActionRun
}

func (e *NewWorkflowRunAttempt) GetRun() *ActionRun {
	return e.run
}

var _ ActionRunEvent = &WorkflowRunStatusChanged{}

type WorkflowRunStatusChanged struct {
	run         *ActionRun
	priorStatus Status
}

func (e *WorkflowRunStatusChanged) GetRun() *ActionRun {
	return e.run
}

func (e *WorkflowRunStatusChanged) GetPriorStatus() Status {
	return e.priorStatus
}

var _ ActionRunEvent = &WorkflowRunCompleted{}

type WorkflowRunCompleted struct {
	run         *ActionRun
	priorStatus Status
}

func (e *WorkflowRunCompleted) GetRun() *ActionRun {
	return e.run
}

func (e *WorkflowRunCompleted) GetPriorStatus() Status {
	return e.priorStatus
}
