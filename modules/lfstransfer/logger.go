// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package lfstransfer

import (
	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

var _ transfer.Logger = (*ForgejoLogger)(nil)

// noop logger for passing into transfer
type ForgejoLogger struct{}

func newLogger() transfer.Logger {
	return &ForgejoLogger{}
}

// Log implements transfer.Logger
func (g *ForgejoLogger) Log(msg string, items ...any) {
}
