// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"bytes"

	"forgejo.org/modules/json"
)

type JsonUtils struct{} //nolint:revive

var jsonUtils = JsonUtils{}

func NewJsonUtils() *JsonUtils { //nolint:revive
	return &jsonUtils
}

func (su *JsonUtils) EncodeToString(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(out)
}

func (su *JsonUtils) PrettyIndent(s string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(s), "", "  ")
	if err != nil {
		return ""
	}
	return out.String()
}

func (su *JsonUtils) Comma(index, length int) string {
	if index+1 == length {
		return ""
	}
	return ","
}

func (su *JsonUtils) Count(data map[string]any, key string, increment int) string {
	var accs map[string]int
	accumulators, ok := data["accumulators"]
	if ok {
		accs = accumulators.(map[string]int)
	} else {
		accs = make(map[string]int)
		data["accumulators"] = accs
	}
	n, ok := accs[key]
	if ok {
		n += increment
	} else {
		n = increment
	}
	accs[key] = n
	return ""
}

func (su *JsonUtils) Counted(data map[string]any, key string) int {
	var accs map[string]int
	accumulators, ok := data["accumulators"]
	if ok {
		accs = accumulators.(map[string]int)
	} else {
		accs = make(map[string]int)
		data["accumulators"] = accs
	}
	n, ok := accs[key]
	if ok {
		return n
	}
	return 0
}
