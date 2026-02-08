// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/services/context"

	"github.com/stretchr/testify/assert"
)

func TestGetListOptions(t *testing.T) {
	cases := []struct {
		path     string
		pageSize int
		page     int
	}{
		// lower bound for page
		{path: "http://localhost?limit=2", pageSize: 2, page: 1},
		{path: "http://localhost?page=-1", pageSize: setting.API.DefaultPagingNum, page: 1},
		{path: "http://localhost?page=0", pageSize: setting.API.DefaultPagingNum, page: 1},
		// bounds for pageSize
		{path: "http://localhost?limit=-1", pageSize: setting.API.DefaultPagingNum, page: 1},
		{path: "http://localhost", pageSize: setting.API.DefaultPagingNum, page: 1},
		{path: "http://localhost?limit=1000", pageSize: setting.API.MaxResponseItems, page: 1},
		// both together
		{path: "http://localhost?page=4&limit=25", pageSize: 25, page: 4},
	}

	for n, c := range cases {
		req, _ := http.NewRequest("GET", c.path, nil)
		ctx := &context.APIContext{Base: &context.Base{Req: req}}
		opts := GetListOptions(ctx)

		assert.Equal(t, c.page, opts.Page, "case %d: error in page", n)
		assert.Equal(t, c.pageSize, opts.PageSize, "case %d: error in pageSize", n)
	}
}
