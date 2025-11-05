// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

// HTTP headers sent in the response to a request to a paginated endpoint
type Paginated struct {
	// Includes links to the first, previous, next, and last pages
	Link string `json:"Link"`
	// The size of the complete list of items
	TotalCount int64 `json:"X-Total-Count"`
}
