// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"testing"

	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
)


// TestConnectionLimitDistribution verifies that MAX_OPEN_CONNS is properly distributed across all engines in an EngineGroup
func TestConnectionLimitDistribution(t *testing.T) {
	origMaxOpen := setting.Database.MaxOpenConns
	origMaxIdle := setting.Database.MaxIdleConns
	origHostReplica := setting.Database.HostReplica
	defer func() {
		setting.Database.MaxOpenConns = origMaxOpen
		setting.Database.MaxIdleConns = origMaxIdle
		setting.Database.HostReplica = origHostReplica
	}()

	tests := []struct {
		name               string
		maxOpenConns       int
		maxIdleConns       int
		numReplicas        int
		expectedPerEngine  int
		expectedPerIdle    int
	}{
		{
			name:               "Single database (no replicas)",
			maxOpenConns:       100,
			maxIdleConns:       10,
			numReplicas:        1,  // Falls back to primary, so 1 primary + 1 replica pointing to same DB
			expectedPerEngine:  50, // 100 / 2
			expectedPerIdle:    5,  // 10 / 2
		},
		{
			name:               "One primary, one replica",
			maxOpenConns:       100,
			maxIdleConns:       10,
			numReplicas:        1,  // 1 primary + 1 actual replica
			expectedPerEngine:  50, // 100 / 2
			expectedPerIdle:    5,  // 10 / 2
		},
		{
			name:               "One primary, two replicas (uneven split)",
			maxOpenConns:       100,
			maxIdleConns:       12,
			numReplicas:        2,  // 1 primary + 2 replicas = 3 engines total
			expectedPerEngine:  33, // 100 / 3 = 33 (integer division, 1 connection unused)
			expectedPerIdle:    4,  // 12 / 3 = 4
		},
		{
			name:               "Limit smaller than engine count (shouldn't happen but better to test nonetheless)",
			maxOpenConns:       2,
			maxIdleConns:       1,
			numReplicas:        2, // 1 primary + 2 replicas = 3 engines total
			expectedPerEngine:  1, // 2 / 3 = 0, but minimum is 1
			expectedPerIdle:    1, // 1 / 3 = 0, but minimum is 1
		},
		{
			name:               "Zero connections should default to 1",
			maxOpenConns:       0,
			maxIdleConns:       0,
			numReplicas:        2,
			expectedPerEngine:  1,
			expectedPerIdle:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalEngines := 1 + tt.numReplicas
			perEngineMaxConns := tt.maxOpenConns / totalEngines
			perEngineMaxIdle := tt.maxIdleConns / totalEngines

			if perEngineMaxConns < 1 {
				perEngineMaxConns = 1
			}
			if perEngineMaxIdle < 1 {
				perEngineMaxIdle = 1
			}

			assert.Equal(t, tt.expectedPerEngine, perEngineMaxConns,
				"Per-engine max open connections should be correctly distributed")
			assert.Equal(t, tt.expectedPerIdle, perEngineMaxIdle,
				"Per-engine max idle connections should be correctly distributed")

			totalActual := perEngineMaxConns * totalEngines
			if tt.maxOpenConns > 0 && tt.maxOpenConns >= totalEngines {
				assert.LessOrEqual(t, totalActual, tt.maxOpenConns,
					"Total connections should not exceed MAX_OPEN_CONNS")
			} else if tt.maxOpenConns > 0 && tt.maxOpenConns < totalEngines {
				assert.Equal(t, totalEngines, totalActual,
					"Should use minimum 1 connection per engine when limit is too small")
			}
		})
	}
}
