// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

type DepthResult struct {
	Depths     map[int64]int
	InDegree   map[int64]int
	Successors map[int64][]int64
	MaxDepth   int
	Cycles     [][]int64
}

func ComputeDependencyDepth(allIssueIDs []int64, dependencies map[int64][]int64) *DepthResult {
	inBoard := make(map[int64]bool, len(allIssueIDs))
	for _, id := range allIssueIDs {
		inBoard[id] = true
	}

	successors := make(map[int64][]int64)
	for id, deps := range dependencies {
		for _, dep := range deps {
			if inBoard[dep] {
				successors[dep] = append(successors[dep], id)
			}
		}
	}

	pending := make(map[int64]int, len(allIssueIDs))
	for _, id := range allIssueIDs {
		pending[id] = len(successors[id])
	}

	depths := make(map[int64]int, len(allIssueIDs))
	queue := make([]int64, 0)
	for _, id := range allIssueIDs {
		depths[id] = 0
		if pending[id] == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		processed++

		for _, dep := range dependencies[curr] {
			if !inBoard[dep] {
				continue
			}
			newDepth := depths[curr] + 1
			if newDepth > depths[dep] {
				depths[dep] = newDepth
			}
			pending[dep]--
			if pending[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	maxDepth := 0
	for _, d := range depths {
		if d > maxDepth {
			maxDepth = d
		}
	}

	for id := range depths {
		depths[id] = maxDepth - depths[id]
	}

	var cycles [][]int64
	if processed < len(allIssueIDs) {
		cycles = detectCycles(allIssueIDs, pending, dependencies)
		for id := range depths {
			if pending[id] > 0 {
				depths[id] = maxDepth + 1
			}
		}
		maxDepth++
	}

	inDegree := make(map[int64]int, len(allIssueIDs))
	for _, id := range allIssueIDs {
		inDegree[id] = len(successors[id])
	}

	return &DepthResult{
		Depths:     depths,
		InDegree:   inDegree,
		Successors: successors,
		MaxDepth:   maxDepth,
		Cycles:     cycles,
	}
}

type tarjanState struct {
	sccIndex     int
	stack        []int64
	onStack      map[int64]bool
	nodeIndex    map[int64]int
	lowlink      map[int64]int
	dependencies map[int64][]int64
	cycleSet     map[int64]bool
	cycles       [][]int64
}

func (t *tarjanState) strongConnect(v int64) {
	t.nodeIndex[v] = t.sccIndex
	t.lowlink[v] = t.sccIndex
	t.sccIndex++
	t.stack = append(t.stack, v)
	t.onStack[v] = true

	for _, w := range t.dependencies[v] {
		if !t.cycleSet[w] {
			continue
		}
		if _, visited := t.nodeIndex[w]; !visited {
			t.strongConnect(w)
			if t.lowlink[w] < t.lowlink[v] {
				t.lowlink[v] = t.lowlink[w]
			}
		} else if t.onStack[w] {
			if t.nodeIndex[w] < t.lowlink[v] {
				t.lowlink[v] = t.nodeIndex[w]
			}
		}
	}

	if t.lowlink[v] == t.nodeIndex[v] {
		var scc []int64
		for {
			w := t.stack[len(t.stack)-1]
			t.stack = t.stack[:len(t.stack)-1]
			t.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		if len(scc) > 1 {
			t.cycles = append(t.cycles, scc)
		}
	}
}

func detectCycles(allIssueIDs []int64, pending map[int64]int, dependencies map[int64][]int64) [][]int64 {
	cycleSet := make(map[int64]bool)
	for _, id := range allIssueIDs {
		if pending[id] > 0 {
			cycleSet[id] = true
		}
	}

	t := &tarjanState{
		onStack:      make(map[int64]bool),
		nodeIndex:    make(map[int64]int),
		lowlink:      make(map[int64]int),
		dependencies: dependencies,
		cycleSet:     cycleSet,
	}

	for _, id := range allIssueIDs {
		if cycleSet[id] {
			if _, visited := t.nodeIndex[id]; !visited {
				t.strongConnect(id)
			}
		}
	}

	return t.cycles
}
