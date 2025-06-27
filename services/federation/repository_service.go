// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

func ProcessRepositoryInbox(ctx *context_service.APIContext, form any) (int, string, error) {
	activity := form.(*ap.Activity)
	repository := ctx.Repo.Repository

	switch activity.Type {
	case ap.LikeType:
		return ProcessLikeActivity(ctx, activity, repository.ID)
	default:
		return http.StatusNotAcceptable, "Invalid activity", fmt.Errorf("invalid activity: %v", activity.Type)
	}
}
