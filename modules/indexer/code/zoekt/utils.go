// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package zoekt

import "regexp"

func QuoteMeta(s string) string {
	return regexp.QuoteMeta(s)
}
