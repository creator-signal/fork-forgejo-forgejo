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

func (su *JsonUtils) Comma(index int, length int) string {
	if index+1 == length {
		return ""
	} else {
		return ","
	}
}

var accumulators map[string]int = make(map[string]int)

func (su *JsonUtils) Count(key string, increment int) string {
	n, ok := accumulators[key]
	if ok {
		n += increment
	} else {
		n = increment
	}
	accumulators[key] = n
	return ""
}

func (su *JsonUtils) Counted(key string) int {
	n, ok := accumulators[key]
	if ok {
		return n
	} else {
		return 0
	}
}
