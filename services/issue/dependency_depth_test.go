// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeDependencyDepthLinearChain(t *testing.T) {
	deps := map[int64][]int64{1: {2}, 2: {3}}
	ids := []int64{1, 2, 3}

	result := ComputeDependencyDepth(ids, deps)
	assert.Empty(t, result.Cycles)
	assert.Equal(t, 0, result.Depths[3])
	assert.Equal(t, 1, result.Depths[2])
	assert.Equal(t, 2, result.Depths[1])
	assert.Equal(t, 2, result.MaxDepth)
}

func TestComputeDependencyDepthCycle(t *testing.T) {
	deps := map[int64][]int64{1: {2}, 2: {1}}
	ids := []int64{1, 2}

	result := ComputeDependencyDepth(ids, deps)
	assert.NotEmpty(t, result.Cycles)
	assert.Contains(t, result.Depths, int64(1))
	assert.Contains(t, result.Depths, int64(2))
}

func TestComputeDependencyDepthNoDeps(t *testing.T) {
	deps := map[int64][]int64{}
	ids := []int64{4, 5}

	result := ComputeDependencyDepth(ids, deps)
	assert.Empty(t, result.Cycles)
	assert.Equal(t, 0, result.Depths[4])
	assert.Equal(t, 0, result.Depths[5])
	assert.Equal(t, 0, result.MaxDepth)
}

func TestComputeDependencyDepthDiamond(t *testing.T) {
	deps := map[int64][]int64{1: {2, 3}, 2: {4}, 3: {4}}
	ids := []int64{1, 2, 3, 4}

	result := ComputeDependencyDepth(ids, deps)
	assert.Empty(t, result.Cycles)
	assert.Equal(t, 0, result.Depths[4])
	assert.Equal(t, 1, result.Depths[2])
	assert.Equal(t, 1, result.Depths[3])
	assert.Equal(t, 2, result.Depths[1])
}

func TestComputeDependencyDepthThreeWayCycle(t *testing.T) {
	deps := map[int64][]int64{1: {2}, 2: {3}, 3: {1}}
	ids := []int64{1, 2, 3}

	result := ComputeDependencyDepth(ids, deps)
	assert.NotEmpty(t, result.Cycles)
	assert.Equal(t, result.Depths[1], result.Depths[2])
	assert.Equal(t, result.Depths[2], result.Depths[3])

	cycleNodes := make(map[int64]bool)
	for _, cycle := range result.Cycles {
		for _, id := range cycle {
			cycleNodes[id] = true
		}
	}
	assert.True(t, cycleNodes[1], "node 1 should be in a cycle")
	assert.True(t, cycleNodes[2], "node 2 should be in a cycle")
	assert.True(t, cycleNodes[3], "node 3 should be in a cycle")
}

func TestComputeDependencyDepthMultipleCycles(t *testing.T) {
	deps := map[int64][]int64{
		1: {2},
		2: {1},
		3: {4},
		4: {3},
	}
	ids := []int64{1, 2, 3, 4}

	result := ComputeDependencyDepth(ids, deps)
	assert.Len(t, result.Cycles, 2)

	allCycleNodes := make(map[int64]bool)
	for _, cycle := range result.Cycles {
		assert.GreaterOrEqual(t, len(cycle), 2)
		for _, id := range cycle {
			allCycleNodes[id] = true
		}
	}
	assert.True(t, allCycleNodes[1])
	assert.True(t, allCycleNodes[2])
	assert.True(t, allCycleNodes[3])
	assert.True(t, allCycleNodes[4])

	assert.Equal(t, result.Depths[1], result.Depths[2])
	assert.Equal(t, result.Depths[3], result.Depths[4])
}

func TestComputeDependencyDepthMixed(t *testing.T) {
	deps := map[int64][]int64{
		1: {2, 3},
		2: {4},
		3: {4},
		6: {5},
	}
	ids := []int64{1, 2, 3, 4, 5, 6}

	result := ComputeDependencyDepth(ids, deps)
	assert.Empty(t, result.Cycles)
	assert.Equal(t, 0, result.Depths[4])
	assert.Equal(t, 1, result.Depths[5])
	assert.Equal(t, 1, result.Depths[2])
	assert.Equal(t, 1, result.Depths[3])
	assert.Equal(t, 2, result.Depths[1])
	assert.Equal(t, 2, result.Depths[6])
}
