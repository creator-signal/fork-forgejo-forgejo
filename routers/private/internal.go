// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package private contains all internal routes. The package name "internal" isn't usable because Golang reserves it for disabling cross-package usage.
package private

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/private"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/routers/common"
	"forgejo.org/services/context"

	"code.forgejo.org/go-chi/binding"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
)

// checkInternalToken validates internal API authentication.
// It accepts the InternalToken via Authorization header (existing behavior) or
// via X-Forgejo-Internal-Auth header (used by LFS SSH transfers to allow Authorization
// to carry an LFS JWT token for the LFS handler's access control).
func checkInternalToken(req *http.Request) bool {
	if setting.InternalToken == "" {
		log.Warn(`The INTERNAL_TOKEN setting is missing from the configuration file: %q, internal API can't work.`, setting.CustomConf)
		return false
	}
	// Check Authorization header (standard internal API requests)
	if tokens := req.Header.Get("Authorization"); tokens != "" {
		fields := strings.SplitN(tokens, " ", 2)
		if len(fields) == 2 && fields[0] == "Bearer" && subtle.ConstantTimeCompare([]byte(fields[1]), []byte(setting.InternalToken)) == 1 {
			return true
		}
	}
	// Check X-Forgejo-Internal-Auth header (LFS SSH transfer requests, where Authorization carries the LFS JWT)
	if tokens := req.Header.Get("X-Forgejo-Internal-Auth"); tokens != "" {
		after, found := strings.CutPrefix(tokens, "Bearer ")
		if found && subtle.ConstantTimeCompare([]byte(after), []byte(setting.InternalToken)) == 1 {
			return true
		}
	}
	return false
}

// CheckInternalToken check internal token is set
func CheckInternalToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !checkInternalToken(req) {
			log.Debug("Forbidden attempt to access internal url: %s", req.RequestURI)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

// bind binding an obj to a handler
func bind[T any](_ T) any {
	return func(ctx *context.PrivateContext) {
		theObj := new(T) // create a new form obj for every request but not use obj directly
		binding.Bind(ctx.Req, theObj)
		web.SetForm(ctx, theObj)
	}
}

// Routes registers all internal APIs routes to web application.
// These APIs will be invoked by internal commands for example `gitea serv` and etc.
func Routes() *web.Route {
	r := web.NewRoute()
	r.Use(context.PrivateContexter())
	r.Use(CheckInternalToken)
	// Log the real ip address of the request from SSH is really helpful for diagnosing sometimes.
	// Since internal API will be sent only from Gitea sub commands and it's under control (checked by InternalToken), we can trust the headers.
	r.Use(chi_middleware.RealIP)

	r.Post("/ssh/authorized_keys", AuthorizedPublicKeyByContent)
	r.Post("/ssh/{id}/update/{repoid}", UpdatePublicKeyInRepo)
	r.Post("/ssh/log", bind(private.SSHLogOption{}), SSHLog)
	r.Post("/hook/pre-receive/{owner}/{repo}", RepoAssignment, bind(private.HookOptions{}), HookPreReceive)
	r.Post("/hook/post-receive/{owner}/{repo}", context.OverrideContext, bind(private.HookOptions{}), HookPostReceive)
	r.Post("/hook/proc-receive/{owner}/{repo}", context.OverrideContext, RepoAssignment, bind(private.HookOptions{}), HookProcReceive)
	r.Post("/hook/set-default-branch/{owner}/{repo}/{branch}", RepoAssignment, SetDefaultBranch)
	r.Get("/serv/none/{keyid}", ServNoCommand)
	r.Get("/serv/command/{keyid}/{owner}/{repo}", ServCommand)
	r.Post("/manager/shutdown", Shutdown)
	r.Post("/manager/restart", Restart)
	r.Post("/manager/reload-templates", ReloadTemplates)
	r.Post("/manager/flush-queues", bind(private.FlushOptions{}), FlushQueues)
	r.Post("/manager/pause-logging", PauseLogging)
	r.Post("/manager/resume-logging", ResumeLogging)
	r.Post("/manager/release-and-reopen-logging", ReleaseReopenLogging)
	r.Post("/manager/set-log-sql", SetLogSQL)
	r.Post("/manager/add-logger", bind(private.LoggerOptions{}), AddLogger)
	r.Post("/manager/remove-logger/{logger}/{writer}", RemoveLogger)
	r.Get("/manager/processes", Processes)
	r.Post("/mail/send", SendEmail)
	r.Post("/restore_repo", RestoreRepo)
	r.Post("/actions/generate_actions_runner_token", GenerateActionsRunnerToken)

	r.Group("/repo", func() {
		// FIXME: it is not right to use context.Contexter here because all routes here should use PrivateContext.
		// Fortunately, the LFS handlers are able to handle requests without a complete web context.
		common.AddOwnerRepoGitLFSRoutes(r, func(ctx *context.PrivateContext) {
			webContext := &context.Context{Base: ctx.Base}
			ctx.AppendContextValue(context.WebContextKey, webContext)
		})
	})

	return r
}
