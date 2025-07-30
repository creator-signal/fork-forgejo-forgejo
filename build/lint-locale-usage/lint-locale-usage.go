// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"fmt"
	"go/ast"
	goParser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	tmplParser "text/template/parse"

	"forgejo.org/modules/container"
	fjTemplates "forgejo.org/modules/templates"
	"forgejo.org/modules/translation/localeiter"
	"forgejo.org/modules/util"
)

// this works by first gathering all valid source string IDs from `en-US` reference files
// and then checking if all used source strings are actually defined

type LocatedError struct {
	Location string
	Kind     string
	Err      error
}

func (e LocatedError) Error() string {
	var sb strings.Builder

	sb.WriteString(e.Location)
	sb.WriteString(":\t")
	if e.Kind != "" {
		sb.WriteString(e.Kind)
		sb.WriteString(": ")
	}
	sb.WriteString("ERROR: ")
	sb.WriteString(e.Err.Error())

	return sb.String()
}

func InitLocaleTrFunctions() map[string][]uint {
	ret := make(map[string][]uint)

	f0 := []uint{0}
	ret["Tr"] = f0
	ret["TrString"] = f0
	ret["TrHTML"] = f0

	ret["TrPluralString"] = []uint{1}
	ret["TrN"] = []uint{1, 2}

	return ret
}

type Handler struct {
	OnMsgid            func(fset *token.FileSet, pos token.Pos, msgid string)
	OnMsgidPrefix      func(fset *token.FileSet, pos token.Pos, msgidPrefix string)
	OnUnexpectedInvoke func(fset *token.FileSet, pos token.Pos, funcname string, argc int)
	OnWarning          func(fset *token.FileSet, pos token.Pos, msg string)
	LocaleTrFunctions  map[string][]uint
}

type StringTrie interface {
	Matches(key []string) bool
}

type StringTrieMap map[string]StringTrie

func (m *StringTrieMap) Matches(key []string) bool {
	if len(key) == 0 || m == nil {
		return true
	}
	value, ok := (*m)[key[0]]
	if !ok {
		return false
	}
	if value == nil {
		return true
	}
	return value.Matches(key[1:])
}

func (m *StringTrieMap) Insert(key []string) {
	if m == nil {
		return
	}

	switch len(key) {
	case 0:
		return

	case 1:
		(*m)[key[0]] = nil

	default:
		if value, ok := (*m)[key[0]]; ok {
			if value == nil {
				return
			}
		} else {
			tmp := make(StringTrieMap)
			(*m)[key[0]] = &tmp
		}
		(*m)[key[0]].(*StringTrieMap).Insert(key[1:])
	}
}

func DecodeKeyForStm(key string) []string {
	ret := strings.Split(key, ".")
	i := len(ret)
	for i > 0 && ret[i-1] == "" {
		i--
	}
	return ret[:i]
}

func ParseAllowedMaskedUsages(fname string, usedMsgids *container.Set[string], allowedMaskedPrefixes *StringTrieMap) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ".") {
			allowedMaskedPrefixes.Insert(DecodeKeyForStm(line))
		} else {
			(*usedMsgids)[line] = struct{}{}
		}
	}
	return scanner.Err()
}

func (handler Handler) HandleGoTrBasicLit(fset *token.FileSet, argLit *ast.BasicLit) {
	if argLit.Kind == token.STRING {
		// extract string content
		arg, err := strconv.Unquote(argLit.Value)
		if err == nil {
			// found interesting strings
			if strings.HasSuffix(arg, ".") || strings.HasSuffix(arg, "_") {
				handler.OnMsgidPrefix(fset, argLit.ValuePos, arg)
			} else {
				handler.OnMsgid(fset, argLit.ValuePos, arg)
			}
		}
	}
}

func (handler Handler) HandleGoTrArgument(fset *token.FileSet, n ast.Expr) {
	if argLit, ok := n.(*ast.BasicLit); ok {
		handler.HandleGoTrBasicLit(fset, argLit)
	} else if argBinExpr, ok := n.(*ast.BinaryExpr); ok {
		if argBinExpr.Op != token.ADD {
			// pass
		} else if argLit, ok := argBinExpr.X.(*ast.BasicLit); ok && argLit.Kind == token.STRING {
			// extract string content
			arg, err := strconv.Unquote(argLit.Value)
			if err == nil {
				// found interesting strings
				handler.OnMsgidPrefix(fset, argLit.ValuePos, arg)
			}
		}
	}
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
					handler.HandleGoTrArgument(fset, call.Args[int(argNum)])
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
			matches := false
			if function.Doc != nil {
				for _, comment := range function.Doc.List {
					if strings.TrimSpace(comment.Text) == "//llu:returnsTrKey" {
						matches = true
						break
					}
				}
			}
			if !matches {
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
								handler.HandleGoTrArgument(fset, expr)
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
			matches := false
			if decl.Doc != nil {
				for _, comment := range decl.Doc.List {
					if strings.TrimSpace(comment.Text) == "// llu:TrKeys" {
						matches = true
						break
					}
				}
			}
			if !matches {
				return true
			}
			for _, spec := range decl.Specs {
				// interpret all contained strings as message IDs
				ast.Inspect(spec, func(n ast.Node) bool {
					if argLit, ok := n.(*ast.BasicLit); ok {
						handler.HandleGoTrBasicLit(fset, argLit)
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

// derived from source: modules/templates/scopedtmpl/scopedtmpl.go, L169-L213
func (handler Handler) handleTemplateNode(fset *token.FileSet, node tmplParser.Node) {
	switch node.Type() {
	case tmplParser.NodeAction:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.ActionNode).Pipe)
	case tmplParser.NodeList:
		nodeList := node.(*tmplParser.ListNode)
		handler.handleTemplateFileNodes(fset, nodeList.Nodes)
	case tmplParser.NodePipe:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.PipeNode))
	case tmplParser.NodeTemplate:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.TemplateNode).Pipe)
	case tmplParser.NodeIf:
		nodeIf := node.(*tmplParser.IfNode)
		handler.handleTemplateBranchNode(fset, nodeIf.BranchNode)
	case tmplParser.NodeRange:
		nodeRange := node.(*tmplParser.RangeNode)
		handler.handleTemplateBranchNode(fset, nodeRange.BranchNode)
	case tmplParser.NodeWith:
		nodeWith := node.(*tmplParser.WithNode)
		handler.handleTemplateBranchNode(fset, nodeWith.BranchNode)

	case tmplParser.NodeCommand:
		nodeCommand := node.(*tmplParser.CommandNode)

		handler.handleTemplateFileNodes(fset, nodeCommand.Args)

		if len(nodeCommand.Args) < 2 {
			return
		}

		funcname := ""
		if nodeChain, ok := nodeCommand.Args[0].(*tmplParser.ChainNode); ok {
			if nodeIdent, ok := nodeChain.Node.(*tmplParser.IdentifierNode); ok {
				if nodeIdent.Ident != "ctx" || len(nodeChain.Field) != 2 || nodeChain.Field[0] != "Locale" {
					return
				}
				funcname = nodeChain.Field[1]
			}
		} else if nodeField, ok := nodeCommand.Args[0].(*tmplParser.FieldNode); ok {
			if len(nodeField.Ident) != 2 || !(nodeField.Ident[0] == "locale" || nodeField.Ident[0] == "Locale") {
				return
			}
			funcname = nodeField.Ident[1]
		}

		var gotUnexpectedInvoke *int
		ltf, ok := handler.LocaleTrFunctions[funcname]
		if !ok {
			return
		}

		for _, argNum := range ltf {
			if len(nodeCommand.Args) >= int(argNum+2) {
				nodeString, ok := nodeCommand.Args[int(argNum+1)].(*tmplParser.StringNode)
				if ok {
					// found interesting strings
					// the column numbers are a bit "off", but much better than nothing
					handler.OnMsgid(fset, token.Pos(nodeString.Pos), nodeString.Text)
				}
			} else {
				argc := len(nodeCommand.Args) - 1
				gotUnexpectedInvoke = &argc
			}
		}

		if gotUnexpectedInvoke != nil {
			handler.OnUnexpectedInvoke(fset, token.Pos(nodeCommand.Pos), funcname, *gotUnexpectedInvoke)
		}

	default:
	}
}

func (handler Handler) handleTemplatePipeNode(fset *token.FileSet, pipeNode *tmplParser.PipeNode) {
	if pipeNode == nil {
		return
	}

	// NOTE: we can't pass `pipeNode.Cmds` to handleTemplateFileNodes due to incompatible argument types
	for _, node := range pipeNode.Cmds {
		handler.handleTemplateNode(fset, node)
	}
}

func (handler Handler) handleTemplateBranchNode(fset *token.FileSet, branchNode tmplParser.BranchNode) {
	handler.handleTemplatePipeNode(fset, branchNode.Pipe)
	handler.handleTemplateFileNodes(fset, branchNode.List.Nodes)
	if branchNode.ElseList != nil {
		handler.handleTemplateFileNodes(fset, branchNode.ElseList.Nodes)
	}
}

func (handler Handler) handleTemplateFileNodes(fset *token.FileSet, nodes []tmplParser.Node) {
	for _, node := range nodes {
		handler.handleTemplateNode(fset, node)
	}
}

func (handler Handler) HandleTemplateFile(fname string, src any) error {
	var tmplContent []byte
	switch src2 := src.(type) {
	case nil:
		var err error
		tmplContent, err = os.ReadFile(fname)
		if err != nil {
			return LocatedError{
				Location: fname,
				Kind:     "ReadFile",
				Err:      err,
			}
		}
	case []byte:
		tmplContent = src2
	case string:
		// SAFETY: we do not modify tmplContent below
		tmplContent = util.UnsafeStringToBytes(src2)
	default:
		panic("invalid type for 'src'")
	}

	fset := token.NewFileSet()
	fset.AddFile(fname, 1, len(tmplContent)).SetLinesForContent(tmplContent)
	// SAFETY: we do not modify tmplContent2 below
	tmplContent2 := util.UnsafeBytesToString(tmplContent)

	tmpl := template.New(fname)
	tmpl.Funcs(fjTemplates.NewFuncMap())
	tmplParsed, err := tmpl.Parse(tmplContent2)
	if err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Template parser",
			Err:      err,
		}
	}
	handler.handleTemplateFileNodes(fset, tmplParsed.Root.Nodes)
	return nil
}

// This command assumes that we get started from the project root directory
//
// Possible command line flags:
//
//	--allow-missing-msgids        don't return an error code if missing message IDs are found
//	--allow-unused-msgids         don't return an error code if unused message IDs are found
//
// EXIT CODES:
//
//	0  success, no issues found
//	1  unable to walk directory tree
//	2  unable to parse locale ini/json files
//	3  unable to parse go or text/template files
//	4  found missing message IDs
//	5  found unused message IDs
//	6  invalid command line argument
//	7  unable to parse allowed masked usages file
//
// SPECIAL GO DOC COMMENTS:
//
//	//llu:returnsTrKey
//					can be used in front of functions to indicate
//					that the function returns message IDs
//
//	// llu:TrKeys
//					can be used in front of 'const' and 'var' blocks
//					in order to mark all contained strings as message IDs
//
//nolint:forbidigo
func main() {
	allowMissingMsgids := false
	allowUnusedMsgids := false
	usedMsgids := make(container.Set[string])
	allowedMaskedPrefixes := make(StringTrieMap)

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--allow-missing-msgids":
			allowMissingMsgids = true

		case "--allow-unused-msgids":
			allowUnusedMsgids = true

		default:
			if argval, found := strings.CutPrefix(arg, "--allow-masked-usages-from="); found {
				if err := ParseAllowedMaskedUsages(argval, &usedMsgids, &allowedMaskedPrefixes); err != nil {
					fmt.Printf("%s:\tERROR: unable to parse masked usages: %s\n", argval, err.Error())
					os.Exit(7)
				}
			} else {
				fmt.Printf("<command line>:\tERROR: unknown argument: %s\n", arg)
				os.Exit(6)
			}
		}
	}

	onError := func(err error) {
		if err == nil {
			return
		}
		fmt.Println(err.Error())
		os.Exit(3)
	}

	msgids := make(container.Set[string])

	localeFile := filepath.Join(filepath.Join("options", "locale"), "locale_en-US.ini")
	localeContent, err := os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err = localeiter.IterateMessagesContent(localeContent, func(trKey, trValue string) error {
		msgids[trKey] = struct{}{}
		return nil
	}); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	localeFile = filepath.Join(filepath.Join("options", "locale_next"), "locale_en-US.json")
	localeContent, err = os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err := localeiter.IterateMessagesNextContent(localeContent, func(trKey, pluralForm, trValue string) error {
		// ignore plural form
		msgids[trKey] = struct{}{}
		return nil
	}); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	gotAnyMsgidError := false

	handler := Handler{
		OnMsgidPrefix: func(fset *token.FileSet, pos token.Pos, msgidPrefix string) {
			// TODO: perhaps we should check if we have any strings with such a prefix, but that's slow...
			allowedMaskedPrefixes.Insert(DecodeKeyForStm(msgidPrefix))
		},
		OnMsgid: func(fset *token.FileSet, pos token.Pos, msgid string) {
			if !msgids.Contains(msgid) {
				gotAnyMsgidError = true
				fmt.Printf("%s:\tmissing msgid: %s\n", fset.Position(pos).String(), msgid)
			} else {
				usedMsgids[msgid] = struct{}{}
			}
		},
		OnUnexpectedInvoke: func(fset *token.FileSet, pos token.Pos, funcname string, argc int) {
			gotAnyMsgidError = true
			fmt.Printf("%s:\tunexpected invocation of %s with %d arguments\n", fset.Position(pos).String(), funcname, argc)
		},
		OnWarning: func(fset *token.FileSet, pos token.Pos, msg string) {
			fmt.Printf("%s:\tWARNING: %s\n", fset.Position(pos).String(), msg)
		},
		LocaleTrFunctions: InitLocaleTrFunctions(),
	}

	if err := filepath.WalkDir(".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == "docker" || name == ".git" || name == "node_modules" {
				return fs.SkipDir
			}
		} else if name == "bindata.go" || fpath == "modules/translation/i18n/i18n_test.go" {
			// skip false positives
		} else if strings.HasSuffix(name, ".go") {
			onError(handler.HandleGoFile(fpath, nil))
		} else if strings.HasSuffix(name, ".tmpl") {
			if strings.HasPrefix(fpath, "tests") && strings.HasSuffix(name, ".ini.tmpl") {
				// skip false positives
			} else {
				onError(handler.HandleTemplateFile(fpath, nil))
			}
		}
		return nil
	}); err != nil {
		fmt.Printf("walkdir ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	unusedMsgids := []string{}

	for msgid := range msgids {
		if !usedMsgids.Contains(msgid) && !allowedMaskedPrefixes.Matches(DecodeKeyForStm(msgid)) {
			unusedMsgids = append(unusedMsgids, msgid)
		}
	}

	sort.Strings(unusedMsgids)

	if len(unusedMsgids) != 0 {
		fmt.Printf("=== unused msgids (%d): ===\n", len(unusedMsgids))
		for _, msgid := range unusedMsgids {
			fmt.Printf("- %s\n", msgid)
		}
	}

	if !allowMissingMsgids && gotAnyMsgidError {
		os.Exit(4)
	}
	if !allowUnusedMsgids && len(unusedMsgids) != 0 {
		os.Exit(5)
	}
}
