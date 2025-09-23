// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	activities_model "forgejo.org/models/activities"
	api "forgejo.org/modules/structs"
)

// User
// swagger:response User
type swaggerResponseUser struct {
	// in:body
	Body api.User `json:"body"`
}

// UserList
// swagger:response UserList
type swaggerResponseUserList struct {
	// in:body
	Body []api.User `json:"body"`

	// The total number of users
	// in:header
	TotalCount int64 `json:"X-Total-Count"`
}

// EmailList
// swagger:response EmailList
type swaggerResponseEmailList struct {
	// in:body
	Body []api.Email `json:"body"`

	// The total number of emails
	// in:header
	TotalCount int64 `json:"X-Total-Count"`
}

// swagger:model EditUserOption
type swaggerModelEditUserOption struct {
	// in:body
	Options api.EditUserOption
}

// UserHeatmapData
// swagger:response UserHeatmapData
type swaggerResponseUserHeatmapData struct {
	// in:body
	Body []activities_model.UserHeatmapData `json:"body"`
}

// UserSettings
// swagger:response UserSettings
type swaggerResponseUserSettings struct {
	// in:body
	Body api.UserSettings `json:"body"`
}

// StopWatchesList
// swagger:response StopWatchesList
type swaggerResponseStopWatchesList struct {
	// in:body
	Body []api.StopWatch `json:"body"`

	// The total number of stopwatches
	// in:header
	TotalCount int64 `json:"X-Total-Count"`
}

// ActivityList
// swagger:response ActivityList
type swaggerResponseActivityList struct {
	// in:body
	Body []api.Activity `json:"body"`

	// The total number of activities
	// in:header
	TotalCount int64 `json:"X-Total-Count"`
}
