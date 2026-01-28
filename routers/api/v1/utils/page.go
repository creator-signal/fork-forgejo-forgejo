// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"forgejo.org/models/db"
	"forgejo.org/services/context"
)

// GetListOptions returns list options using the page and limit parameters
func GetListOptions(ctx *context.APIContext) db.ListOptions {
	opts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: ctx.FormInt("limit"),
	}

	opts.SetDefaultValues()

	return opts
}
