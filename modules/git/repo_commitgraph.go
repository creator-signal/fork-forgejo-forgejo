// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/modules/log"
)

// CommitGraph maps each commit IDs to the list of its parents
type CommitGraph map[ObjectIDKey][]ObjectID

// Load the entire commit graph into memory to speed up otherwise expensive operations
func (repo *Repository) LoadCommitGraph(ctx context.Context, force ...bool) error {
	if repo.commitGraph != nil && (len(force) == 0 || !force[0]) {
		return nil
	}

	commitGraph := make(CommitGraph)

	stdout, _, err := NewCommand(ctx, "rev-list", "--all", "--parents", "--topo-order").RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return fmt.Errorf("unable to load commit-graph for repo %s: %w", repo.Path, err)
	}

	for line := range strings.SplitSeq(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		commitID, err := NewIDFromString(parts[0])
		if err != nil {
			return fmt.Errorf("unable to parse commit ID %q in commit-graph: %w", parts[0], err)
		}

		graphEntry := make([]ObjectID, 0, len(parts)-1)

		for _, parentIDStr := range parts[1:] {
			parentID, err := NewIDFromString(parentIDStr)
			if err != nil {
				return fmt.Errorf("unable to parse parent ID %q in commit-graph: %w", parentIDStr, err)
			}
			graphEntry = append(graphEntry, parentID)
		}

		commitGraph[commitID.Key()] = graphEntry
	}

	log.Debug("Loaded commit graph for repo %s with %d commits", repo.Path, len(commitGraph))

	repo.commitGraph = commitGraph

	return nil
}

// WriteCommitGraph write commit graph to speed up repo access
// this requires git v2.18 to be installed
func WriteCommitGraph(ctx context.Context, repoPath string) error {
	if _, _, err := NewCommand(ctx, "commit-graph", "write").RunStdString(&RunOpts{Dir: repoPath}); err != nil {
		return fmt.Errorf("unable to write commit-graph for '%s' : %w", repoPath, err)
	}
	return nil
}
