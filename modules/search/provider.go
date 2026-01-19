// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package search

import "context"

type Provider interface {
	Search(context.Context, string) (any, error)
}
