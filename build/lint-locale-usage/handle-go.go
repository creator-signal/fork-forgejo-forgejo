// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"go/ast"
	goParser "go/parser"
	"go/token"
	"strconv"
	"strings"
)

func (handler Handler) handleGoTrBasicLit(fset *token.FileSet, argLit *ast.BasicLit, prefix string) {
	if argLit.Kind == token.STRING {
		// extract string content
		arg, err := strconv.Unquote(argLit.Value)
		if err != nil {
			return
		}
		// found interesting strings
		arg = prefix + arg
		if strings.HasSuffix(arg, ".") || strings.HasSuffix(arg, "_") {
			prep, trunc := PrepareMsgidPrefix(arg)
			if trunc {
				handler.OnWarning(fset, argLit.ValuePos, fmt.Sprintf("needed to truncate message id prefix: %s", arg))
			}
			handler.OnMsgidPrefix(fset, argLit.ValuePos, prep, trunc)
		} else {
			handler.OnMsgid(fset, argLit.ValuePos, arg)
		}
	}
}

func (handler Handler) handleGoTrArgument(fset *token.FileSet, n ast.Expr, prefix string) {
	if argLit, ok := n.(*ast.BasicLit); ok {
		handler.handleGoTrBasicLit(fset, argLit, prefix)
	} else if argBinExpr, ok := n.(*ast.BinaryExpr); ok {
		if argBinExpr.Op != token.ADD {
			// pass
		} else if argLit, ok := argBinExpr.X.(*ast.BasicLit); ok && argLit.Kind == token.STRING {
			// extract string content
			arg, err := strconv.Unquote(argLit.Value)
			if err != nil {
				return
			}
			// found interesting strings
			arg = prefix + arg
			prep, trunc := PrepareMsgidPrefix(arg)
			if trunc {
				handler.OnWarning(fset, argLit.ValuePos, fmt.Sprintf("needed to truncate message id prefix: %s", arg))
			}
			handler.OnMsgidPrefix(fset, argLit.ValuePos, prep, trunc)
		}
	}
}

func (handler Handler) handleGoCommentGroup(fset *token.FileSet, cg *ast.CommentGroup, commentPrefix string) *string {
	matches := false
	matchInsPrefix := ""
	var multimatches *token.Pos
	if cg == nil {
		return nil
	}
	commentPrefix = "//" + commentPrefix
	for _, comment := range cg.List {
		ctxt := strings.TrimSpace(comment.Text)
		if ctxt == commentPrefix {
			if matches {
				multimatches = &comment.Slash
			}
			matches = true
		} else if after, found := strings.CutPrefix(ctxt, commentPrefix+"Suffix "); found {
			if matches {
				multimatches = &comment.Slash
			}
			matches = true
			matchInsPrefix = strings.TrimSpace(after)
		}
	}
	if !matches {
		return nil
	}
	if multimatches != nil {
		handler.OnWarning(
			fset,
			*multimatches,
			fmt.Sprintf("encountered multiple %s... directives", strings.TrimSpace(commentPrefix)),
		)
	}
	return &matchInsPrefix
}

// the `Handle*File` functions follow the following calling convention:
// * `fname` is the name of the input file
// * `src` is either `nil` (then the function invokes `ReadFile` to read the file)
//   or the contents of the file as {`[]byte`, or a `string`}

func (handler Handler) HandleGoFile(fname string, src any) error {
	fset := token.NewFileSet()
	node, err := goParser.ParseFile(fset, fname, src, goParser.SkipObjectResolution|goParser.ParseComments)
	if err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Go parser",
			Err:      err,
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// search for function calls of the form `anything.Tr(any-string-lit, ...)`

		if call, ok := n.(*ast.CallExpr); ok && len(call.Args) >= 1 {
			funSel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ltf, ok := handler.LocaleTrFunctions[funSel.Sel.Name]
			if !ok {
				return true
			}

			var gotUnexpectedInvoke *int

			for _, argNum := range ltf {
				if len(call.Args) < int(argNum+1) {
					argc := len(call.Args)
					gotUnexpectedInvoke = &argc
				} else {
					handler.handleGoTrArgument(fset, call.Args[int(argNum)], "")
				}
			}

			if gotUnexpectedInvoke != nil {
				handler.OnUnexpectedInvoke(fset, funSel.Sel.NamePos, funSel.Sel.Name, *gotUnexpectedInvoke)
			}
		} else if composite, ok := n.(*ast.CompositeLit); ok {
			ident, ok := composite.Type.(*ast.Ident)
			if !ok {
				return true
			}

			// special case: models/unit/unit.go
			if strings.HasSuffix(fname, "unit.go") && ident.Name == "Unit" && len(composite.Elts) == 6 {
				// NameKey has index 2
				//   invoked like '{{ctx.Locale.Tr $unit.NameKey}}'
				nameKey, ok := composite.Elts[2].(*ast.BasicLit)
				if !ok || nameKey.Kind != token.STRING {
					handler.OnWarning(fset, composite.Elts[2].Pos(), "unexpected initialization of 'Unit'")
					return true
				}

				// extract string content
				arg, err := strconv.Unquote(nameKey.Value)
				if err == nil {
					// found interesting strings
					handler.OnMsgid(fset, nameKey.ValuePos, arg)
				}
			}
		} else if function, ok := n.(*ast.FuncDecl); ok {
			matchInsPrefix := handler.handleGoCommentGroup(fset, function.Doc, "llu:returnsTrKey")
			if matchInsPrefix == nil {
				return true
			}
			results := function.Type.Results.List
			if len(results) != 1 {
				handler.OnWarning(fset, function.Type.Func, fmt.Sprintf("function %s has unexpected return type; expected single return value", function.Name.Name))
				return true
			}

			ast.Inspect(function.Body, func(n ast.Node) bool {
				// search for return stmts
				// TODO: what about nested functions?
				if ret, ok := n.(*ast.ReturnStmt); ok {
					for _, res := range ret.Results {
						ast.Inspect(res, func(n ast.Node) bool {
							if expr, ok := n.(ast.Expr); ok {
								handler.handleGoTrArgument(fset, expr, *matchInsPrefix)
							}
							return true
						})
					}
					return false
				}
				return true
			})
			return true
		} else if decl, ok := n.(*ast.GenDecl); ok && (decl.Tok == token.CONST || decl.Tok == token.VAR) {
			matchInsPrefix := handler.handleGoCommentGroup(fset, decl.Doc, " llu:TrKeys")
			if matchInsPrefix == nil {
				return true
			}
			for _, spec := range decl.Specs {
				// interpret all contained strings as message IDs
				ast.Inspect(spec, func(n ast.Node) bool {
					if argLit, ok := n.(*ast.BasicLit); ok {
						handler.handleGoTrBasicLit(fset, argLit, *matchInsPrefix)
						return false
					}
					return true
				})
			}
		}

		return true
	})

	return nil
}
