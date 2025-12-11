// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package source

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"forgejo.org/models"
	"forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
)

type syncType int

const (
	syncAdd syncType = iota
	syncRemove
)

// SyncGroupsToTeams maps authentication source groups to organization and team memberships
func SyncGroupsToTeams(ctx context.Context, user *user_model.User, sourceUserGroups container.Set[string], sourceGroupTeamMapping map[string]map[string][]string, performRemoval bool) error {
	orgCache := make(map[string]*organization.Organization)
	teamCache := make(map[string]*organization.Team)
	return SyncGroupsToTeamsCached(ctx, user, sourceUserGroups, sourceGroupTeamMapping, performRemoval, orgCache, teamCache)
}

// SyncGroupsToTeamsCached maps authentication source groups to organization and team memberships
func SyncGroupsToTeamsCached(ctx context.Context, user *user_model.User, sourceUserGroups container.Set[string], sourceGroupTeamMapping map[string]map[string][]string, performRemoval bool, orgCache map[string]*organization.Organization, teamCache map[string]*organization.Team) error {
	membershipsToAdd, membershipsToRemove := resolveMappedMemberships(ctx, user, sourceUserGroups, sourceGroupTeamMapping, performRemoval)

	if performRemoval {
		if err := syncGroupsToTeamsCached(ctx, user, membershipsToRemove, syncRemove, orgCache, teamCache); err != nil {
			return fmt.Errorf("could not sync[remove] user groups: %w", err)
		}
	}

	if err := syncGroupsToTeamsCached(ctx, user, membershipsToAdd, syncAdd, orgCache, teamCache); err != nil {
		return fmt.Errorf("could not sync[add] user groups: %w", err)
	}

	return nil
}

// getMembershipsToAdd returns the memberships to add based on the source
// groups (retrieved from the auth source) and the group team mappings
func getMembershipsToAdd(
	srcGroups container.Set[string],
	mappings map[string]map[string][]string,
) map[string][]string {

	membershipsToAdd := map[string][]string{}

	for group, memberships := range mappings {
		// replace placeholders with regex
		group = strings.Replace(group, "{org}", `(?<org>[\w-]+)`, 1)
		group = strings.Replace(group, "{team}", `(?<team>[\w-]+)`, 1)
		group = fmt.Sprintf("^%s$", group)

		// create regex
		r, err := regexp.Compile(group)
		if err != nil {
			log.Error("group sync: could not compile regex: %v", err)
			continue
		}

		// check if src group matches regex
		for sg := range srcGroups {
			match := r.FindStringSubmatch(sg)
			if match == nil {
				continue
			}

			// match, try to get org and team
			orgPH := ""
			teamPH := ""
			for i, name := range r.SubexpNames() {
				switch name {
				case "org":
					orgPH = match[i]
				case "team":
					teamPH = match[i]
				}
			}

			// get org and team memberships
			for org, teams := range memberships {
				org = strings.Replace(org, "{org}", orgPH, 1)
				org = strings.Replace(org, "{team}", teamPH, 1)

				for _, team := range teams {
					team = strings.Replace(team, "{org}", orgPH, 1)
					team = strings.Replace(team, "{team}", teamPH, 1)

					membershipsToAdd[org] = append(membershipsToAdd[org], team)
				}
			}
		}
	}

	return membershipsToAdd
}

// getMembershipsToRemoveNotAdded returns the memberships to remove. It returns
// all current memberships of the user that are not added based on group team
// mappings (in membershipsToAdd) as memberships to remove.
func getMembershipsToRemoveNotAdded(
	ctx context.Context,
	user *user_model.User,
	membershipsToAdd map[string][]string,
) map[string][]string {

	membershipsToRemove := map[string][]string{}

	// get user's organizations
	orgs, err := organization.GetUserOrgsList(ctx, user)
	if err != nil {
		log.Warn("group sync: could not get organizations: %v", err)
	}
	for _, org := range orgs {
		log.Debug("group sync: found user org: %#v", org)

		// get user's teams in organization
		teams, err := organization.GetUserOrgTeams(ctx, org.ID, user.ID)
		if err != nil {
			log.Warn("group sync: could not get organization teams: %v", err)
		}
		for _, team := range teams {
			log.Debug("group sync: found user org team: %#v", team)

			// remove membership if it's not added via group team mapping
			if !slices.Contains(membershipsToAdd[org.Name], team.LowerName) {
				membershipsToRemove[org.Name] = append(membershipsToRemove[org.Name], team.LowerName)
			}
		}
	}

	return membershipsToRemove
}

func resolveMappedMemberships(
	ctx context.Context,
	user *user_model.User,
	sourceUserGroups container.Set[string],
	sourceGroupTeamMapping map[string]map[string][]string,
	performRemoval bool,
) (map[string][]string, map[string][]string) {

	membershipsToAdd := getMembershipsToAdd(sourceUserGroups, sourceGroupTeamMapping)
	membershipsToRemove := map[string][]string{}
	if performRemoval {
		membershipsToRemove = getMembershipsToRemoveNotAdded(ctx, user, membershipsToAdd)
	}
	log.Debug("group sync: memberships to add: %v", membershipsToAdd)
	log.Debug("group sync: memberships to remove: %v", membershipsToRemove)
	return membershipsToAdd, membershipsToRemove
}

func syncGroupsToTeamsCached(ctx context.Context, user *user_model.User, orgTeamMap map[string][]string, action syncType, orgCache map[string]*organization.Organization, teamCache map[string]*organization.Team) error {
	for orgName, teamNames := range orgTeamMap {
		var err error
		org, ok := orgCache[orgName]
		if !ok {
			org, err = organization.GetOrgByName(ctx, orgName)
			if err != nil {
				if organization.IsErrOrgNotExist(err) {
					// organization must be created before group sync
					log.Warn("group sync: Could not find organisation %s: %v", orgName, err)
					continue
				}
				return err
			}
			orgCache[orgName] = org
		}
		for _, teamName := range teamNames {
			team, ok := teamCache[orgName+teamName]
			if !ok {
				team, err = org.GetTeam(ctx, teamName)
				if err != nil {
					if organization.IsErrTeamNotExist(err) {
						// team must be created before group sync
						log.Warn("group sync: Could not find team %s: %v", teamName, err)
						continue
					}
					return err
				}
				teamCache[orgName+teamName] = team
			}

			isMember, err := organization.IsTeamMember(ctx, org.ID, team.ID, user.ID)
			if err != nil {
				return err
			}

			if action == syncAdd && !isMember {
				if err := models.AddTeamMember(ctx, team, user.ID); err != nil {
					log.Error("group sync: Could not add user to team: %v", err)
					return err
				}
			} else if action == syncRemove && isMember {
				if err := models.RemoveTeamMember(ctx, team, user.ID); err != nil {
					log.Error("group sync: Could not remove user from team: %v", err)
					return err
				}
			}
		}
	}
	return nil
}
