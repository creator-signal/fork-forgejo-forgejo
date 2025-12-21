// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitgraph

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"forgejo.org/modules/git"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkGetCommitGraph(b *testing.B) {
	currentRepo, err := git.OpenRepository(git.DefaultContext, ".")
	if err != nil || currentRepo == nil {
		b.Error("Could not open repository")
	}
	defer currentRepo.Close()

	for i := 0; i < b.N; i++ {
		graph, err := GetCommitGraph(currentRepo, 1, 0, false, nil, nil)
		if err != nil {
			b.Error("Could get commit graph")
		}

		if len(graph.Commits) < 100 {
			b.Error("Should get 100 log lines.")
		}
	}
}

func BenchmarkParseCommitString(b *testing.B) {
	testString := "* DATA:abc123||4e61bacab44e9b4730e44a6615d04098dd3a8eaf|Tue, 20 Dec 2016 21:10:41 +0100|4e61bac|Add route for graph"

	parser := &Parser{}
	parser.Reset()
	for i := 0; i < b.N; i++ {
		parser.Reset()
		graph := NewGraph()
		if err := parser.AddLineToGraph(graph, 0, []byte(testString)); err != nil {
			b.Error("could not parse teststring")
		}
		if graph.Flows[1].Commits[0].Rev != "4e61bacab44e9b4730e44a6615d04098dd3a8eaf" {
			b.Error("Did not get expected data")
		}
	}
}

func BenchmarkParseGlyphs(b *testing.B) {
	parser := &Parser{}
	parser.Reset()
	tgBytes := []byte(testglyphs)
	var tg []byte
	for i := 0; i < b.N; i++ {
		parser.Reset()
		tg = tgBytes
		idx := bytes.Index(tg, []byte("\n"))
		for idx > 0 {
			parser.ParseGlyphs(tg[:idx])
			tg = tg[idx+1:]
			idx = bytes.Index(tg, []byte("\n"))
		}
	}
}

func TestReleaseUnusedColors(t *testing.T) {
	testcases := []struct {
		availableColors []int
		oldColors       []int
		firstInUse      int // these values have to be either be correct or suggest less is
		firstAvailable  int // available than possibly is - i.e. you cannot say 10 is available when it
	}{
		{
			availableColors: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			oldColors:       []int{1, 1, 1, 1, 1},
			firstAvailable:  -1,
			firstInUse:      1,
		},
		{
			availableColors: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			oldColors:       []int{1, 2, 3, 4},
			firstAvailable:  6,
			firstInUse:      0,
		},
		{
			availableColors: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			oldColors:       []int{6, 0, 3, 5, 3, 4, 0, 0},
			firstAvailable:  6,
			firstInUse:      0,
		},
		{
			availableColors: []int{1, 2, 3, 4, 5, 6, 7},
			oldColors:       []int{6, 1, 3, 5, 3, 4, 2, 7},
			firstAvailable:  -1,
			firstInUse:      0,
		},
		{
			availableColors: []int{1, 2, 3, 4, 5, 6, 7},
			oldColors:       []int{6, 0, 3, 5, 3, 4, 2, 7},
			firstAvailable:  -1,
			firstInUse:      0,
		},
	}
	for _, testcase := range testcases {
		parser := &Parser{}
		parser.Reset()
		parser.availableColors = append([]int{}, testcase.availableColors...)
		parser.oldColors = append(parser.oldColors, testcase.oldColors...)
		parser.firstAvailable = testcase.firstAvailable
		parser.firstInUse = testcase.firstInUse
		parser.releaseUnusedColors()

		if parser.firstAvailable == -1 {
			// All in use
			for _, color := range parser.availableColors {
				found := false
				for _, oldColor := range parser.oldColors {
					if oldColor == color {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("In testcase:\n%d\t%d\t%d %d =>\n%d\t%d\t%d %d: %d should be available but is not",
						testcase.availableColors,
						testcase.oldColors,
						testcase.firstAvailable,
						testcase.firstInUse,
						parser.availableColors,
						parser.oldColors,
						parser.firstAvailable,
						parser.firstInUse,
						color)
				}
			}
		} else if parser.firstInUse != -1 {
			// Some in use
			for i := parser.firstInUse; i != parser.firstAvailable; i = (i + 1) % len(parser.availableColors) {
				color := parser.availableColors[i]
				found := false
				for _, oldColor := range parser.oldColors {
					if oldColor == color {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("In testcase:\n%d\t%d\t%d %d =>\n%d\t%d\t%d %d: %d should be available but is not",
						testcase.availableColors,
						testcase.oldColors,
						testcase.firstAvailable,
						testcase.firstInUse,
						parser.availableColors,
						parser.oldColors,
						parser.firstAvailable,
						parser.firstInUse,
						color)
				}
			}
			for i := parser.firstAvailable; i != parser.firstInUse; i = (i + 1) % len(parser.availableColors) {
				color := parser.availableColors[i]
				found := false
				for _, oldColor := range parser.oldColors {
					if oldColor == color {
						found = true
						break
					}
				}
				if found {
					t.Errorf("In testcase:\n%d\t%d\t%d %d =>\n%d\t%d\t%d %d: %d should not be available but is",
						testcase.availableColors,
						testcase.oldColors,
						testcase.firstAvailable,
						testcase.firstInUse,
						parser.availableColors,
						parser.oldColors,
						parser.firstAvailable,
						parser.firstInUse,
						color)
				}
			}
		} else {
			// None in use
			for _, color := range parser.oldColors {
				if color != 0 {
					t.Errorf("In testcase:\n%d\t%d\t%d %d =>\n%d\t%d\t%d %d: %d should not be available but is",
						testcase.availableColors,
						testcase.oldColors,
						testcase.firstAvailable,
						testcase.firstInUse,
						parser.availableColors,
						parser.oldColors,
						parser.firstAvailable,
						parser.firstInUse,
						color)
				}
			}
		}
	}
}

func TestParseGlyphs(t *testing.T) {
	parser := &Parser{}
	parser.Reset()
	tgBytes := []byte(testglyphs)
	tg := tgBytes
	idx := bytes.Index(tg, []byte("\n"))
	row := 0
	for idx > 0 {
		parser.ParseGlyphs(tg[:idx])
		tg = tg[idx+1:]
		idx = bytes.Index(tg, []byte("\n"))
		if parser.flows[0] != 1 {
			t.Errorf("First column flow should be 1 but was %d", parser.flows[0])
		}
		colorToFlow := map[int]int64{}
		flowToColor := map[int64]int{}

		for i, flow := range parser.flows {
			if flow == 0 {
				continue
			}
			color := parser.colors[i]

			if fColor, in := flowToColor[flow]; in && fColor != color {
				t.Errorf("Row %d column %d flow %d has color %d but should be %d", row, i, flow, color, fColor)
			}
			flowToColor[flow] = color
			if cFlow, in := colorToFlow[color]; in && cFlow != flow {
				t.Errorf("Row %d column %d flow %d has color %d but conflicts with flow %d", row, i, flow, color, cFlow)
			}
			colorToFlow[color] = flow
		}
		row++
	}
	if len(parser.availableColors) != 9 {
		t.Errorf("Expected 9 colors but have %d", len(parser.availableColors))
	}
}

func TestCommitStringParsing(t *testing.T) {
	dataFirstPart := "* DATA:abc123||4e61bacab44e9b4730e44a6615d04098dd3a8eaf|Tue, 20 Dec 2016 21:10:41 +0100|4e61bac|"
	tests := []struct {
		shouldPass    bool
		testName      string
		commitMessage string
	}{
		{true, "normal", "not a fancy message"},
		{true, "extra pipe", "An extra pipe: |"},
		{true, "extra 'Data:'", "DATA: might be trouble"},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			testString := fmt.Sprintf("%s%s", dataFirstPart, test.commitMessage)
			idx := strings.Index(testString, "DATA:")
			commit, err := NewCommit(0, 0, []byte(testString[idx+5:]))
			if err != nil && test.shouldPass {
				t.Errorf("Could not parse %s", testString)
				return
			}

			if test.commitMessage != commit.Subject {
				t.Errorf("%s does not match %s", test.commitMessage, commit.Subject)
			}
		})
	}
}

func TestNewCommitParentHashes(t *testing.T) {
	tests := []struct {
		name            string
		data            string
		expectedParents []string
	}{
		{
			name:            "no parents (orphan)",
			data:            "||4e61bacab44e9b4730e44a6615d04098dd3a8eaf|Tue, 20 Dec 2016 21:10:41 +0100|4e61bac|subject",
			expectedParents: nil,
		},
		{
			name:            "single parent",
			data:            "abc123||4e61bacab44e9b4730e44a6615d04098dd3a8eaf|Tue, 20 Dec 2016 21:10:41 +0100|4e61bac|subject",
			expectedParents: []string{"abc123"},
		},
		{
			name:            "multiple parents (merge)",
			data:            "abc123 def456||4e61bacab44e9b4730e44a6615d04098dd3a8eaf|Tue, 20 Dec 2016 21:10:41 +0100|4e61bac|subject",
			expectedParents: []string{"abc123", "def456"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commit, err := NewCommit(0, 0, []byte(test.data))
			require.NoError(t, err)
			assert.Equal(t, test.expectedParents, commit.ParentHashes)
		})
	}
}

func TestComputeGlyphConnectivity(t *testing.T) {
	// helper to create a test commit
	makeCommit := func(row, col int, hash string, parents []string) *Commit {
		return &Commit{Row: row, Column: col, Rev: hash, ParentHashes: parents, Flow: int64(col + 1)}
	}

	// helper to set up graph with commits
	setupGraph := func(commits []*Commit) *Graph {
		graph := NewGraph()
		for _, c := range commits {
			graph.AddGlyph(c.Row, c.Column, c.Flow, 1, '*')
			graph.Commits = append(graph.Commits, c)
			graph.Flows[c.Flow].Commits = append(graph.Flows[c.Flow].Commits, c)
		}
		graph.ComputeGlyphConnectivity()
		return graph
	}

	// helper to find glyph connectivity for a commit
	getConnectivity := func(graph *Graph, row, col int) (up, down bool) {
		for _, flow := range graph.Flows {
			for _, g := range flow.Glyphs {
				if g.Row == row && g.Column == col {
					return g.ConnectsUp, g.ConnectsDown
				}
			}
		}
		return false, false
	}

	t.Run("orphan has no connections", func(t *testing.T) {
		graph := setupGraph([]*Commit{makeCommit(0, 0, "aaa", nil)})
		up, down := getConnectivity(graph, 0, 0)
		assert.False(t, up, "orphan should not connect up")
		assert.False(t, down, "orphan should not connect down")
	})

	t.Run("linear history", func(t *testing.T) {
		graph := setupGraph([]*Commit{
			makeCommit(0, 0, "child", []string{"parent"}),
			makeCommit(1, 0, "parent", nil),
		})

		childUp, childDown := getConnectivity(graph, 0, 0)
		assert.False(t, childUp, "child should not connect up")
		assert.True(t, childDown, "child should connect down")

		parentUp, parentDown := getConnectivity(graph, 1, 0)
		assert.True(t, parentUp, "parent should connect up")
		assert.False(t, parentDown, "parent should not connect down")
	})

	t.Run("orphan followed by unrelated commit", func(t *testing.T) {
		graph := setupGraph([]*Commit{
			makeCommit(0, 0, "orphan", nil),
			makeCommit(1, 0, "unrelated", nil),
		})
		orphanUp, orphanDown := getConnectivity(graph, 0, 0)
		assert.False(t, orphanUp || orphanDown, "orphan should have no connections")

		unrelatedUp, unrelatedDown := getConnectivity(graph, 1, 0)
		assert.False(t, unrelatedUp || unrelatedDown, "unrelated should have no connections")
	})

	t.Run("merge commit with two parents", func(t *testing.T) {
		graph := setupGraph([]*Commit{
			makeCommit(0, 0, "merge", []string{"parent1", "parent2"}),
			makeCommit(1, 0, "parent1", nil),
			makeCommit(2, 1, "parent2", nil),
		})

		mergeUp, mergeDown := getConnectivity(graph, 0, 0)
		assert.False(t, mergeUp, "merge should not connect up")
		assert.True(t, mergeDown, "merge should connect down")

		parent1Up, parent1Down := getConnectivity(graph, 1, 0)
		assert.True(t, parent1Up, "parent1 should connect up")
		assert.False(t, parent1Down, "parent1 should not connect down")

		parent2Up, parent2Down := getConnectivity(graph, 2, 1)
		assert.True(t, parent2Up, "parent2 should connect up")
		assert.False(t, parent2Down, "parent2 should not connect down")
	})

	t.Run("non-commit glyphs always connect", func(t *testing.T) {
		graph := NewGraph()
		for _, glyph := range []byte{'|', '/', '\\'} {
			graph.AddGlyph(0, 0, 1, 1, glyph)
		}
		graph.ComputeGlyphConnectivity()

		for _, g := range graph.Flows[1].Glyphs {
			assert.True(t, g.ConnectsUp, "glyph %q should connect up", g.Glyph)
			assert.True(t, g.ConnectsDown, "glyph %q should connect down", g.Glyph)
		}
	})
}

var testglyphs = `* 
* 
* 
* 
* 
* 
* 
*   
|\  
* | 
* | 
* | 
* | 
* | 
| * 
* | 
| *   
| |\  
* | | 
| | *   
| | |\  
* | | \   
|\ \ \ \  
| * | | | 
| |\| | | 
* | | | | 
|/ / / /  
| | | * 
| * | | 
| * | | 
| * | | 
* | | | 
* | | | 
* | | | 
* | | | 
* | | |   
|\ \ \ \  
| | * | | 
| | |\| | 
| | | * | 
| | | | * 
* | | | | 
* | | | | 
* | | | | 
* | | | | 
* | | | |   
|\ \ \ \ \  
| * | | | | 
|/| | | | | 
| | |/ / /  
| |/| | |   
| | | | * 
| * | | | 
|/| | | | 
| * | | | 
|/| | | | 
| | |/ /  
| |/| |   
| * | | 
| * | |   
| |\ \ \  
| | * | | 
| |/| | | 
| | | |/  
| | |/|   
| * | | 
| * | | 
| * | | 
| | * |   
| | |\ \  
| | | * | 
| | |/| | 
| | | * |   
| | | |\ \  
| | | | * | 
| | | |/| | 
| | * | | | 
| | * | | |   
| | |\ \ \ \  
| | | * | | | 
| | |/| | | | 
| | | | | * | 
| | | | |/ /  
* | | | / / 
|/ / / / /  
* | | | |   
|\ \ \ \ \  
| * | | | | 
|/| | | | | 
| * | | | | 
| * | | | |   
| |\ \ \ \ \  
| | | * \ \ \   
| | | |\ \ \ \  
| | | | * | | | 
| | | |/| | | | 
| | | | | |/ /  
| | | | |/| |   
* | | | | | | 
* | | | | | | 
* | | | | | | 
| | | | * | | 
* | | | | | | 
| | * | | | | 
| |/| | | | | 
* | | | | | | 
| |/ / / / /  
|/| | | | |   
| | | | * | 
| | | |/ /  
| | |/| |   
| * | | | 
| | | | * 
| | * | |   
| | |\ \ \  
| | | * | | 
| | |/| | | 
| | | |/ /  
| | | * | 
| | * | |   
| | |\ \ \  
| | | * | | 
| | |/| | | 
| | | |/ /  
| | | * | 
* | | | |   
|\ \ \ \ \  
| * \ \ \ \   
| |\ \ \ \ \  
| | | |/ / /  
| | |/| | |   
| | | | * | 
| | | | * | 
* | | | | | 
* | | | | | 
|/ / / / /  
| | | * | 
* | | | | 
* | | | | 
* | | | | 
* | | | |   
|\ \ \ \ \  
| * | | | | 
|/| | | | | 
| | * | | |   
| | |\ \ \ \  
| | | * | | | 
| | |/| | | | 
| |/| | |/ /  
| | | |/| |   
| | | | | * 
| |_|_|_|/  
|/| | | |   
| | * | | 
| |/ / /  
* | | | 
* | | | 
| | * | 
* | | | 
* | | | 
| * | | 
| | * | 
| * | | 
* | | |   
|\ \ \ \  
| * | | | 
|/| | | | 
| |/ / /  
| * | |   
| |\ \ \  
| | * | | 
| |/| | | 
| | |/ /  
| | * |   
| | |\ \  
| | | * | 
| | |/| | 
* | | | | 
* | | | |   
|\ \ \ \ \  
| * | | | | 
|/| | | | | 
| | * | | | 
| | * | | | 
| | * | | | 
| |/ / / /  
| * | | |   
| |\ \ \ \  
| | * | | | 
| |/| | | | 
* | | | | | 
* | | | | | 
* | | | | | 
* | | | | | 
* | | | | | 
| | | | * | 
* | | | | |   
|\ \ \ \ \ \  
| * | | | | | 
|/| | | | | | 
| | | | | * | 
| | | | |/ /  
* | | | | |   
|\ \ \ \ \ \  
* | | | | | | 
* | | | | | | 
| | | | * | | 
* | | | | | | 
* | | | | | |   
|\ \ \ \ \ \ \  
| | |_|_|/ / /  
| |/| | | | |   
| | | | * | | 
| | | | * | | 
| | | | * | | 
| | | | * | | 
| | | | * | | 
| | | | * | | 
| | | |/ / /  
| | | * | | 
| | | * | | 
| | | * | | 
| | |/| | | 
| | | * | | 
| | |/| | | 
| | | |/ /  
| | * | | 
| |/| | | 
| | | * | 
| | |/ /  
| | * | 
| * | |   
| |\ \ \  
| * | | | 
| | * | | 
| |/| | | 
| | |/ /  
| | * |   
| | |\ \  
| | * | | 
* | | | | 
|\| | | | 
| * | | | 
| * | | | 
| * | | | 
| | * | | 
| * | | | 
| |\| | | 
| * | | | 
| | * | | 
| | * | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| | * | | 
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| | * | | 
* | | | | 
|\| | | | 
| | * | | 
| * | | | 
| |\| | | 
| | * | | 
| | * | | 
| | * | | 
| | | * | 
* | | | | 
|\| | | | 
| | * | | 
| | |/ /  
| * | | 
| * | | 
| |\| | 
* | | | 
|\| | | 
| | * | 
| | * | 
| | * | 
| * | | 
| | * | 
| * | | 
| | * | 
| | * | 
| | * | 
| * | | 
| * | | 
| * | | 
| * | | 
| * | | 
| * | | 
| * | | 
* | | | 
|\| | | 
| * | | 
| |\| | 
| | * |   
| | |\ \  
* | | | | 
|\| | | | 
| * | | | 
| |\| | | 
| | * | | 
| | | * | 
| | |/ /  
* | | | 
* | | | 
|\| | | 
| * | | 
| |\| | 
| | * | 
| | * | 
| | * | 
| | | * 
* | | | 
|\| | | 
| * | | 
| * | | 
| | | *   
| | | |\  
* | | | | 
| |_|_|/  
|/| | |   
| * | | 
| |\| | 
| | * | 
| | * | 
| | * | 
| | * | 
| | * | 
| * | | 
* | | | 
|\| | | 
| * | | 
|/| | | 
| |/ /  
| * |   
| |\ \  
| * | | 
| * | | 
* | | | 
|\| | | 
| | * | 
| * | | 
| * | | 
| * | | 
* | | | 
|\| | | 
| * | | 
| * | | 
| | * |   
| | |\ \  
| | |/ /  
| |/| |   
| * | | 
* | | | 
|\| | | 
| * | | 
* | | | 
|\| | | 
| * | |   
| |\ \ \  
| * | | | 
| * | | | 
| | | * | 
| * | | | 
| * | | | 
| | |/ /  
| |/| |   
| | * | 
* | | | 
|\| | | 
| * | | 
| * | | 
| * | | 
| * | | 
| * | |   
| |\ \ \  
* | | | | 
|\| | | | 
| * | | | 
| * | | | 
* | | | | 
* | | | | 
|\| | | | 
| | | | *   
| | | | |\  
| |_|_|_|/  
|/| | | |   
| * | | | 
* | | | | 
* | | | | 
|\| | | | 
| * | | |   
| |\ \ \ \  
| | | |/ /  
| | |/| |   
| * | | | 
| * | | | 
| * | | | 
| * | | | 
| | * | | 
| | | * | 
| | |/ /  
| |/| |   
* | | | 
|\| | | 
| * | | 
| * | | 
| * | | 
| * | | 
| * | | 
* | | | 
|\| | | 
| * | | 
| * | | 
* | | | 
| * | | 
| * | | 
| * | | 
* | | | 
* | | | 
* | | | 
|\| | | 
| * | | 
* | | | 
* | | | 
* | | | 
* | | | 
| | | * 
* | | | 
|\| | | 
| * | | 
| * | | 
| * | | 
`
