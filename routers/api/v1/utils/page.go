// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"forgejo.org/models/db"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// GetListOptions returns list options using the page and limit parameters
func GetListOptions(ctx *context.APIContext) db.ListOptions {
	return db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
	}
}

// Sets the `Link` and `X-Total-Count` headers together
func SetPaginationHeaders(ctx *context.APIContext, total int64) {
	ctx.SetLinkHeader(int(total), GetListOptions(ctx).PageSize)
	ctx.SetTotalCountHeader(total)
}
