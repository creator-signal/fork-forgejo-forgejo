// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	org_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

// forge-feed.org link extension types, see https://forge-feed.org for more details.
const (
	FORGE_FEED_REL_AVATAR         = "http://forge-feed.org/rel/avatar"
	FORGE_FEED_REL_TICKETING      = "http://forge-feed.org/rel/ticketing-system"
	FORGE_FEED_REL_REPOSITORY     = "http://forge-feed.org/rel/repository"
	FORGE_FEED_REL_REPOSITORY_URI = "http://forge-feed.org/rel/repository-uri"
	FORGE_FEED_REL_PROJECT        = "http://forge-feed.org/rel/project"
	FORGE_FEED_REL_HOMEPAGE       = "http://forge-feed.org/rel/homepage"
	FORGE_FEED_REL_DESCRIPTION    = "http://forge-feed.org/rel/description"
	FORGE_FEED_REL_LICENSE        = "http://forge-feed.org/rel/license"
	FORGE_FEED_REL_VCS_CLONE_LINK = "http://forge-feed.org/rel/clone"
	FORGE_FEED_REL_LABEL          = "http://forge-feed.org/rel/label"
	FORGE_FEED_NS_LABEL           = "http://forge-feed.org/ns/label"
	FORGE_FEED_NS_VCS_TYPE        = "http://forge-feed.org/ns/vcs-type"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

type webfingerJRD struct {
	Subject    string           `json:"subject,omitempty"`
	Aliases    []string         `json:"aliases,omitempty"`
	Properties map[string]any   `json:"properties,omitempty"`
	Links      []*webfingerLink `json:"links,omitempty"`
}

type webfingerLink struct {
	Rel        string            `json:"rel,omitempty"`
	Type       string            `json:"type,omitempty"`
	Href       string            `json:"href,omitempty"`
	Titles     map[string]string `json:"titles,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
}

// WebfingerQuery returns information about a resource
// https://datatracker.ietf.org/doc/html/rfc7565
func WebfingerQuery(ctx *context.Context) {
	appURL, _ := url.Parse(setting.AppURL)

	resource, err := url.Parse(ctx.FormTrim("resource"))
	if err != nil {
		ctx.Error(http.StatusBadRequest)
		return
	}

	var u *user_model.User

	switch resource.Scheme {
	case "acct":
		// allow only the current host
		parts := strings.SplitN(resource.Opaque, "@", 2)
		if len(parts) != 2 {
			ctx.Error(http.StatusBadRequest)
			return
		}
		if parts[1] != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		// Instance actor
		if parts[0] == "ghost" {
			aliases := []string{
				appURL.String() + "api/v1/activitypub/actor",
			}

			links := []*webfingerLink{
				{
					Rel:  "self",
					Type: "application/activity+json",
					Href: appURL.String() + "api/v1/activitypub/actor",
				},
			}

			ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
			ctx.JSON(http.StatusOK, &webfingerJRD{
				Subject: fmt.Sprintf("acct:%s@%s", "ghost", appURL.Host),
				Aliases: aliases,
				Links:   links,
			})
			ctx.Resp.Header().Set("Content-Type", "application/jrd+json")

			return
		}

		u, err = user_model.GetUserByName(ctx, parts[0])

	case "mailto":
		u, err = user_model.GetUserByEmail(ctx, resource.Opaque)
		if u != nil && u.KeepEmailPrivate {
			err = user_model.ErrUserNotExist{}
		}
	case "https", "http":
		if resource.Host != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		p := strings.Trim(resource.Path, "/")
		if len(p) == 0 {
			ctx.Error(http.StatusNotFound)
			return
		}

		parts := strings.Split(p, "/")

		switch len(parts) {
		case 1: // user
			u, err = user_model.GetUserByName(ctx, parts[0])
		case 2: // repository
			ctx.Error(http.StatusNotFound)
			return

		case 3:
			switch parts[2] {
			case "issues":
				ctx.Error(http.StatusNotFound)
				return

			case "pulls":
				ctx.Error(http.StatusNotFound)
				return

			case "projects":
				ctx.Error(http.StatusNotFound)
				return

			default:
				ctx.Error(http.StatusNotFound)
				return
			}

		default:
			ctx.Error(http.StatusNotFound)
			return
		}

	case "project", "repository":
		parts := strings.Split(resource.Opaque, "/")
		var jrdSubject string
		links := []*webfingerLink{}
		if resource.Scheme == "project" && len(parts) == 1 {
			jrdSubject = fmt.Sprintf("project:%s", parts[0])
			org_name := parts[0]
			org, err := org_model.GetOrgByName(ctx, org_name)
			if err != nil {
				if org_model.IsErrOrgNotExist(err) {
					ctx.Error(http.StatusNotFound)
				} else {
					log.Warn("Failed to look up organization: %s", err)
					ctx.Error(http.StatusInternalServerError)
				}
				return
			}

			// TODO: Need to verify this is correct
			if !org.Visibility.IsPublic() {
				ctx.Error(http.StatusNotFound)
				return
			}

			if org.Description != "" {
				links = append(links, &webfingerLink{
					Rel: FORGE_FEED_REL_DESCRIPTION,
					Titles: map[string]string{
						"en-us": org.Description, // FIXME: Localization ?
					},
				})
			}

			if org.Website != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_HOMEPAGE,
					Href: org.Website,
				})
			}

			if avatar_link := org.AvatarLink(ctx); avatar_link != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_AVATAR,
					Href: avatar_link,
				})
			}

			repos, err := org_model.GetOrgRepositories(ctx, org.ID)
			if err != nil {
				log.Warn("Failed to lookup org/user repositories: %s", err)
				ctx.Error(http.StatusInternalServerError)
				return
			}

			for _, repo := range repos {
				titles := map[string]string{}
				if repo.Description != "" {
					titles["en-us"] = repo.Description
				}
				links = append(links, &webfingerLink{
					Rel:    FORGE_FEED_REL_REPOSITORY,
					Href:   repo.HTMLURL(),
					Titles: titles,
					Properties: map[string]any{
						FORGE_FEED_REL_REPOSITORY_URI: fmt.Sprintf("repository:%s/%s", parts[0], repo.Name),
					},
				})
			}

		} else if resource.Scheme == "repository" && len(parts) == 2 {
			jrdSubject = fmt.Sprintf("repository:%s/%s", parts[0], parts[1])
			repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, parts[0], parts[1])
			if err != nil {
				if repo_model.IsErrRepoNotExist(err) {
					ctx.Error(http.StatusNotFound)
				} else {
					log.Warn("Failed to lookup repository %s", err)
					ctx.Error(http.StatusInternalServerError)
				}
				return
			}

			if repo.IsPrivate {
				// TODO: You can determine presence of a repository by response time
				// here but I think all of Forgejo's repositories are subject to this?
				ctx.Error(http.StatusNotFound)
				return
			}

			links = append(links, &webfingerLink{
				Rel:  FORGE_FEED_REL_TICKETING,
				Href: fmt.Sprintf("%s", appURL.JoinPath(parts[0], parts[1], "issues")),
			})
			if repo.Description != "" {
				links = append(links, &webfingerLink{
					Rel: FORGE_FEED_REL_DESCRIPTION,
					Titles: map[string]string{
						"en-us": repo.Description, // FIXME: Localization ?
					},
				})
			}

			if repo.Website != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_HOMEPAGE,
					Href: repo.Website,
				})
			}

			if avatar_link := repo.AvatarLink(ctx); avatar_link != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_AVATAR,
					Href: avatar_link,
				})
			}

			clone_link := repo.CloneLink()
			if https_link := clone_link.HTTPS; https_link != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_VCS_CLONE_LINK,
					Href: https_link,
					Properties: map[string]any{
						FORGE_FEED_NS_VCS_TYPE: "https",
					},
				})
			}

			if git_link := clone_link.HTTPS; git_link != "" {
				links = append(links, &webfingerLink{
					Rel:  FORGE_FEED_REL_VCS_CLONE_LINK,
					Href: git_link,
					Properties: map[string]any{
						FORGE_FEED_NS_VCS_TYPE: "git",
					},
				})
			}

			for _, topic := range repo.Topics {
				links = append(links, &webfingerLink{
					Rel: FORGE_FEED_REL_LABEL,
					Properties: map[string]any{
						FORGE_FEED_NS_LABEL: topic,
					},
				})
			}

		} else {
			log.Warn("Webfinger resource query invalid %s", resource.Opaque)
			ctx.Error(http.StatusBadRequest)
			return
		}

		ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
		ctx.JSON(http.StatusOK, &webfingerJRD{
			Subject: jrdSubject,
			Links:   links,
		})
		ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
		return

	default:
		ctx.Error(http.StatusBadRequest)
		return
	}
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusNotFound)
		} else {
			log.Error("Error getting user: %s Error: %v", resource.Opaque, err)
			ctx.Error(http.StatusInternalServerError)
		}

		return
	}

	if !user_model.IsUserVisibleToViewer(ctx, u, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	aliases := []string{
		u.HTMLURL(),
		appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
	}
	if !u.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", u.Email))
	}

	links := []*webfingerLink{
		{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: u.HTMLURL(),
		},
		{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: u.AvatarLink(ctx),
		},
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: strings.TrimSuffix(appURL.String(), "/"),
		},
	}

	ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
	ctx.JSON(http.StatusOK, &webfingerJRD{
		Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(u.Name), appURL.Host),
		Aliases: aliases,
		Links:   links,
	})
	ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
}
