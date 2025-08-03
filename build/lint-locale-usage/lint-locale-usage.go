// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forgejo.org/modules/container"
	"forgejo.org/modules/translation/localeiter"
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

func ParseAllowedMaskedUsages(fname string, usedMsgids *container.Set[string], allowedMaskedPrefixes *StringTrieMap, chkMsgid func(msgid string) bool) error {
	file, err := os.Open(fname)
	if err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Open",
			Err:      err,
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lno := 0
	for scanner.Scan() {
		lno++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ".") {
			allowedMaskedPrefixes.Insert(DecodeKeyForStm(line))
		} else {
			if !chkMsgid(line) {
				return LocatedError{
					Location: fmt.Sprintf("%s: line %d", fname, lno),
					Kind:     "undefined msgid",
					Err:      errors.New(line),
				}
			}
			(*usedMsgids)[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Scanner",
			Err:      err,
		}
	}
	return nil
}

// Truncating a message id prefix to the last dot
func PrepareMsgidPrefix(s string) (string, bool) {
	index := strings.LastIndexByte(s, 0x2e)
	if index == -1 {
		return "", true
	}
	return s[:index], index != len(s)-1
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

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--allow-missing-msgids":
			allowMissingMsgids = true

		case "--allow-unused-msgids":
			allowUnusedMsgids = true

		default:
			if argval, found := strings.CutPrefix(arg, "--allow-masked-usages-from="); found {
				if err := ParseAllowedMaskedUsages(argval, &usedMsgids, &allowedMaskedPrefixes, func(msgid string) bool {
					return msgids.Contains(msgid)
				}); err != nil {
					fmt.Printf("%s\n", err.Error())
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

	handler := Handler{
		OnMsgidPrefix: func(fset *token.FileSet, pos token.Pos, msgidPrefix string) {
			if !allowedMaskedPrefixes.Matches(DecodeKeyForStm(msgidPrefix)) {
				gotAnyMsgidError = true
				fmt.Printf("%s:\tmissing msgid prefix: %s\n", fset.Position(pos).String(), msgidPrefix)
			}
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
