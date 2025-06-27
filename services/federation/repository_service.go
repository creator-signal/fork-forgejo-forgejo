// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

func ProcessRepositoryInbox(ctx *context_service.APIContext, form any, repositoryID int64) (int, string, error) {
	activity := form.(*ap.Activity)

	switch activity.Type {
	case ap.LikeType:
		return ProcessLikeActivity(ctx, activity, repositoryID)
	default:
		return http.StatusNotAcceptable, "Invalid activity", fmt.Errorf("invalid activity: %v", activity.Type)
	}
}
