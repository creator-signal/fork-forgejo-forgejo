// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package lintLocaleUsage

import (
	"go/ast"
	"go/token"
	"strconv"

	llu "forgejo.org/build/lint-locale-usage"
)

func HandleCompositeStat(handler llu.Handler, fset *token.FileSet, n *ast.CompositeLit) {
	ident, ok := n.Type.(*ast.Ident)
	if !ok || ident.Name != "Stat" {
		return
	}

	if len(n.Elts) != 3 {
		handler.OnWarning(fset, n.Pos(), "unexpected initialization of 'Stat' (unexpected number of arguments)")
		return
	}
	// Label has index 2
	//   invoked like '{{ctx.Locale.Tr $stat.Label}}'
	label, ok := n.Elts[2].(*ast.BasicLit)
	if !ok || label.Kind != token.STRING {
		handler.OnWarning(fset, n.Elts[2].Pos(), "unexpected initialization of 'Stat' (expected string literal as Label)")
		return
	}

	// extract string content
	arg, err := strconv.Unquote(label.Value)
	if err == nil {
		// found interesting strings
		handler.OnMsgid(fset, label.ValuePos, arg, false)
	}
}
