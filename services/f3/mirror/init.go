// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"runtime"
	"strings"

	"code.forgejo.org/f3/gof3/v3/util"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	projectPackagePrefix := strings.TrimSuffix(filename, "mirror/init.go")
	if projectPackagePrefix == filename {
		// in case the source code file is moved, we can not trim the suffix, the code above should also be updated.
		panic("unable to detect correct package prefix, please update file: " + filename)
	}
	util.AppendPanicPackagePrefix("forgejo", projectPackagePrefix)
}
