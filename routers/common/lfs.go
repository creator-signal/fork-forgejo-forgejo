// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"net/http"

	"forgejo.org/modules/web"
	"forgejo.org/services/lfs"
)

// AddOwnerRepoGitLFSRoutes adds LFS API routes shared between the web and internal (SSH LFS transfer) routers.
func AddOwnerRepoGitLFSRoutes(m *web.Route, middlewares ...any) {
	m.Group("/{username}/{reponame}/info/lfs", func() {
		m.Post("/objects/batch", lfs.CheckAcceptMediaType, lfs.BatchHandler)
		m.Put("/objects/{oid}/{size}", lfs.UploadHandler)
		m.Get("/objects/{oid}/{filename}", lfs.DownloadHandler)
		m.Get("/objects/{oid}", lfs.DownloadHandler)
		m.Post("/verify", lfs.CheckAcceptMediaType, lfs.VerifyHandler)
		m.Group("/locks", func() {
			m.Get("/", lfs.GetListLockHandler)
			m.Post("/", lfs.PostLockHandler)
			m.Post("/verify", lfs.VerifyLockHandler)
			m.Post("/{lid}/unlock", lfs.UnLockHandler)
		}, lfs.CheckAcceptMediaType)
		m.Any("/*", http.NotFound)
	}, middlewares...)
}
