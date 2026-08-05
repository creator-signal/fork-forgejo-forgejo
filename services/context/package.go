// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	stdctx "context"
	"errors"
	"fmt"
	"net/http"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/perm"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/templates"
	packages_service "forgejo.org/services/packages"
)

// StatusClientClosedRequest (499, non-standard) signals that the client closed
// the connection before the server finished. It is returned instead of a 500
// when a package request fails because its context was canceled (e.g. docker
// push aborting some of its many concurrent blob HEAD requests), so a normal
// client cancellation is not misreported as a server error.
// See https://codeberg.org/forgejo/forgejo/issues/13782
const StatusClientClosedRequest = 499

// Package contains owner, access mode and optional the package descriptor
type Package struct {
	Owner      *user_model.User
	AccessMode perm.AccessMode
	Descriptor *packages_model.PackageDescriptor
}

type packageAssignmentCtx struct {
	*Base
	Doer        *user_model.User
	ContextUser *user_model.User
}

// PackageAssignment returns a middleware to handle Context.Package assignment
func PackageAssignment() func(ctx *Context) {
	return func(ctx *Context) {
		errorFn := func(status int, title string, obj any) {
			err, ok := obj.(error)
			if !ok {
				err = fmt.Errorf("%s", obj)
			}
			switch {
			case errors.Is(err, stdctx.Canceled), errors.Is(err, stdctx.DeadlineExceeded):
				// The client canceled the request (e.g. docker push aborting
				// some of its many concurrent blob HEAD requests). Not a server
				// error; do not emit a 500.
				ctx.Error(StatusClientClosedRequest)
			case status == http.StatusNotFound:
				ctx.NotFound(title, err)
			default:
				ctx.ServerError(title, err)
			}
		}
		paCtx := &packageAssignmentCtx{Base: ctx.Base, Doer: ctx.Doer, ContextUser: ctx.ContextUser}
		ctx.Package = packageAssignment(paCtx, errorFn)
	}
}

// PackageAssignmentAPI returns a middleware to handle Context.Package assignment
func PackageAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		paCtx := &packageAssignmentCtx{Base: ctx.Base, Doer: ctx.Doer(), ContextUser: ctx.User()}
		ctx.pkg = packageAssignment(paCtx, ctx.Error)
	}
}

func packageAssignment(ctx *packageAssignmentCtx, errCb func(int, string, any)) *Package {
	pkg := &Package{
		Owner: ctx.ContextUser,
	}
	var err error
	pkg.AccessMode, err = packages_service.DeterminePackageAccessMode(ctx.Base, pkg.Owner, ctx.Doer)
	if err != nil {
		errCb(http.StatusInternalServerError, "determineAccessMode", err)
		return pkg
	}

	packageType := ctx.Params("type")
	name := ctx.Params("name")
	version := ctx.Params("version")
	if packageType != "" && name != "" && version != "" {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx, pkg.Owner.ID, packages_model.Type(packageType), name, version)
		if err != nil {
			if err == packages_model.ErrPackageNotExist {
				errCb(http.StatusNotFound, "GetVersionByNameAndVersion", err)
			} else {
				errCb(http.StatusInternalServerError, "GetVersionByNameAndVersion", err)
			}
			return pkg
		}

		pkg.Descriptor, err = packages_model.GetPackageDescriptor(ctx, pv)
		if err != nil {
			errCb(http.StatusInternalServerError, "GetPackageDescriptor", err)
			return pkg
		}
	}

	return pkg
}

// PackageContexter initializes a package context for a request.
func PackageContexter() func(next http.Handler) http.Handler {
	renderer := templates.HTMLRenderer()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			base, baseCleanUp := NewBaseContext(resp, req)
			defer baseCleanUp()

			// it is still needed when rendering 500 page in a package handler
			ctx := NewWebContext(base, renderer, nil)
			ctx.AppendContextValue(WebContextKey, ctx)
			next.ServeHTTP(ctx.Resp, ctx.Req)
		})
	}
}
