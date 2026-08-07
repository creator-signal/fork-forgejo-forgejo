// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

// StatusClientClosedRequest is the status nginx introduced for a client that closed the connection before the server
// answered. It is not part of the HTTP standard, but proxies such as nginx and Traefik report it in this situation, so
// it keeps requests that were abandoned by the client out of the 5xx range.
const StatusClientClosedRequest = 499
