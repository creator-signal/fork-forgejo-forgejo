// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"context"
	"strings"

	"forgejo.org/modules/setting/config"
)

const UserHeatmapWeekStartDynKey = "service.user_heatmap_week_start"

var userHeatmapWeekStartValues = map[string]struct{}{
	"monday":    {},
	"tuesday":   {},
	"wednesday": {},
	"thursday":  {},
	"friday":    {},
	"saturday":  {},
	"sunday":    {},
}

func NormalizeUserHeatmapWeekStart(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := userHeatmapWeekStartValues[normalized]; ok {
		return normalized, true
	}
	return "", false
}

func UserHeatmapWeekStartValue(ctx context.Context) string {
	if dg := config.GetDynGetter(); dg != nil {
		if v, has := dg.GetValue(ctx, UserHeatmapWeekStartDynKey); has {
			if normalized, ok := NormalizeUserHeatmapWeekStart(v); ok {
				return normalized
			}
		}
	}
	return Service.UserHeatmapWeekStart
}
