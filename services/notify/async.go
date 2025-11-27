// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package notify

import (
	"sync"
)

var testNotificationWaitGroup *sync.WaitGroup

func RunAsync(fn func()) {
	if testNotificationWaitGroup != nil {
		testNotificationWaitGroup.Add(1)
	}
	go func() {
		if testNotificationWaitGroup != nil {
			defer testNotificationWaitGroup.Done()
		}
		fn()
	}()
}

func SetTestNotificationWaitGroup(wg *sync.WaitGroup) {
	testNotificationWaitGroup = wg
}
