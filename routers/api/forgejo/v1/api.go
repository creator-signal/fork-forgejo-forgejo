// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/shared"
)

func Routes() *web.Route {
	m := web.NewRoute()

	m.Use(shared.Middlewares()...)

	forgejo := NewForgejo()
	m.Get("", Root)
	m.Get("/version", forgejo.GetVersion)

	// Mount the generated OpenAPI routes (project endpoints).
	// HandlerFromMux registers all spec-defined routes on the chi.Router
	// underlying our web.Route. Shared middlewares (auth, CORS, etc.)
	// are already applied above via m.Use().
	HandlerFromMux(forgejo, m.R)

	return m
}
