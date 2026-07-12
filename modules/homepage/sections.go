// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package homepage

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
)

const (
	repoCount = 6
	orgCount  = 6
)

// ResolvedSection is a section filled with actual data.
type ResolvedSection struct {
	Type  string
	Title string
	Repos []*repo_model.Repository
	Orgs  []*user_model.User
	Links []*Link
	Stats []Stat
}

// Stat is one counter in a statistics section. Label is a locale key.
type Stat struct {
	Icon  string
	Value int64
	Label string
}

type sectionsSnapshot struct {
	cfg      *Config
	sections []*ResolvedSection
	fetched  time.Time
}

var (
	sectionsCache    atomic.Pointer[sectionsSnapshot]
	sectionsMu       sync.Mutex
	sectionsCacheTTL = time.Minute
)

func (s *sectionsSnapshot) fresh(cfg *Config) bool {
	return s != nil && s.cfg == cfg && time.Since(s.fetched) < sectionsCacheTTL
}

// Sections loads and caches home sections for template rendering.
func Sections(ctx context.Context) []*ResolvedSection {
	cfg := Get()
	if setting.IsInTesting {
		return resolve(ctx, cfg.Sections)
	}
	if s := sectionsCache.Load(); s.fresh(cfg) {
		return s.sections
	}

	sectionsMu.Lock()
	defer sectionsMu.Unlock()
	if s := sectionsCache.Load(); s.fresh(cfg) { // another goroutine may have refreshed while we waited
		return s.sections
	}
	sections := resolve(ctx, cfg.Sections)
	sectionsCache.Store(&sectionsSnapshot{cfg, sections, time.Now()})
	return sections
}

var repoSortMap = map[string]string{
	"stars":  "moststars",
	"forks":  "mostforks",
	"recent": "recentupdate",
	"newest": "newest",
	"oldest": "oldest",
}

func resolve(ctx context.Context, sections []*Section) []*ResolvedSection {
	resolved := make([]*ResolvedSection, 0, len(sections))
	for _, s := range sections {
		rs := &ResolvedSection{Type: s.Type, Title: s.Title}
		switch s.Type {
		case SectionRepos:
			sortKey, ok := repoSortMap[s.Sort]
			if !ok {
				sortKey = "moststars"
			}
			orderBy, ok := repo_model.OrderByFlatMap[sortKey]
			if !ok {
				orderBy = db.SearchOrderByStars
			}
			repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
				ListOptions:        db.ListOptions{Page: 1, PageSize: repoCount},
				OrderBy:            orderBy,
				AllPublic:          true,
				IncludeDescription: setting.UI.SearchRepoDescription,
			})
			if err != nil {
				log.Error("homepage repositories: %v", err)
				continue
			}
			rs.Repos = repos
		case SectionStats:
			rs.Stats = resolveStats(ctx, s.Items)
		case SectionOrgs:
			orgs, _, err := user_model.SearchUsers(ctx, &user_model.SearchUserOptions{
				Type:        user_model.UserTypeOrganization,
				ListOptions: db.ListOptions{Page: 1, PageSize: orgCount},
				Visible:     []structs.VisibleType{structs.VisibleTypePublic},
			})
			if err != nil {
				log.Error("homepage organizations: %v", err)
				continue
			}
			rs.Orgs = orgs
		case SectionLinks:
			rs.Links = s.Links
		default:
			log.Warn("homepage: unknown section type %q", s.Type)
			continue
		}
		resolved = append(resolved, rs)
	}
	return resolved
}

func resolveStats(ctx context.Context, items []string) []Stat {
	if len(items) == 0 {
		items = []string{"repositories", "users", "organizations"}
	}
	stats := make([]Stat, 0, len(items))
	for _, item := range items {
		switch item {
		case "repositories":
			stats = append(stats, Stat{"octicon-repo", countPublicRepos(ctx), "landing.stats.repositories"})
		case "users":
			stats = append(stats, Stat{"octicon-person", countPublicUsers(ctx, user_model.UserTypeIndividual), "landing.stats.users"})
		case "organizations":
			stats = append(stats, Stat{"octicon-organization", countPublicUsers(ctx, user_model.UserTypeOrganization), "landing.stats.organizations"})
		default:
			log.Warn("homepage stats: unknown item %q", item)
		}
	}
	return stats
}

func countPublicRepos(ctx context.Context) int64 {
	_, n, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{PageSize: 1},
		AllPublic:   true,
	})
	if err != nil {
		log.Error("homepage stats repositories: %v", err)
	}
	return n
}

func countPublicUsers(ctx context.Context, t user_model.UserType) int64 {
	_, n, err := user_model.SearchUsers(ctx, &user_model.SearchUserOptions{
		Type:        t,
		ListOptions: db.ListOptions{PageSize: 1},
		Visible:     []structs.VisibleType{structs.VisibleTypePublic},
	})
	if err != nil {
		log.Error("homepage stats users: %v", err)
	}
	return n
}
