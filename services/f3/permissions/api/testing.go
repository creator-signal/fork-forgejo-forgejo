// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package api

var permissionsCheckCalled [][]any

func ResetPermissionsCheckCalled() {
	permissionsCheckCalled = nil
}

func GetPermissionsCheckCalled() [][]any {
	return permissionsCheckCalled
}

func AddPermissionsCheckCall(signature ...any) {
	permissionsCheckCalled = append(permissionsCheckCalled, signature)
}
