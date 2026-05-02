package edu

import (
	"context"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/services/notify"
)

type EduNotifier struct {
	notify.NullNotifier
	repo        Repository
	gradeParser func(ctx context.Context, run *actions_model.ActionRun) (int, bool)
}

var _ notify.Notifier = &EduNotifier{}

func RegisterNotifier(repo Repository) {
	n := &EduNotifier{repo: repo}
	n.gradeParser = n.parseGradeFromRun
	notify.RegisterNotifier(n)
}

// TODO
func (n *EduNotifier) ActionRunNowDone(
	ctx context.Context,
	run *actions_model.ActionRun,
	priorStatus actions_model.Status,
	lastRun *actions_model.ActionRun,
) {
}

// TODO
func (n *EduNotifier) parseGradeFromRun(ctx context.Context, run *actions_model.ActionRun) (int, bool) {
	return 0, false
}
