// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	webhook_module "forgejo.org/modules/webhook"
)

type (
	linkFormatter = func(string, string) string
	nameFormatter = func(string) string
)

// noneLinkFormatter does not create a link but just returns the text
func noneLinkFormatter(url, text string) string {
	return text
}

// htmlLinkFormatter creates a HTML link
func htmlLinkFormatter(url, text string) string {
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(url), html.EscapeString(text))
}

// noneNameFormatter just returns the name
func noneNameFormatter(name string) string {
	return name
}

// getPullRequestInfo gets the information for a pull request
func getPullRequestInfo(p *api.PullRequestPayload, nameFormatter nameFormatter) (title, link, by, operator, operateResult, assignees string) {
	title = fmt.Sprintf("[PullRequest-%s #%d]: %s\n%s", p.Repository.FullName, p.PullRequest.Index, p.Action, p.PullRequest.Title)
	assignList := p.PullRequest.Assignees
	assignStringList := make([]string, len(assignList))

	for i, user := range assignList {
		assignStringList[i] = nameFormatter(user.UserName)
	}
	switch p.Action {
	case api.HookIssueAssigned:
		operateResult = fmt.Sprintf("%s assign this to %s", nameFormatter(p.Sender.UserName), nameFormatter(assignList[len(assignList)-1].UserName))
	case api.HookIssueUnassigned:
		operateResult = fmt.Sprintf("%s unassigned this for someone", nameFormatter(p.Sender.UserName))
	case api.HookIssueMilestoned:
		operateResult = fmt.Sprintf("%s/milestone/%d", p.Repository.HTMLURL, p.PullRequest.Milestone.ID)
	}
	link = p.PullRequest.HTMLURL
	by = fmt.Sprintf("PullRequest by %s", nameFormatter(p.PullRequest.Poster.UserName))
	if len(assignStringList) > 0 {
		assignees = fmt.Sprintf("Assignees: %s", strings.Join(assignStringList, ", "))
	}
	operator = fmt.Sprintf("Operator: %s", nameFormatter(p.Sender.UserName))
	return title, link, by, operator, operateResult, assignees
}

// getIssuesInfo gets the information for an issue
func getIssuesInfo(p *api.IssuePayload, nameFormatter nameFormatter) (issueTitle, link, by, operator, operateResult, assignees string) {
	issueTitle = fmt.Sprintf("[Issue-%s #%d]: %s\n%s", p.Repository.FullName, p.Issue.Index, p.Action, p.Issue.Title)
	assignList := p.Issue.Assignees
	assignStringList := make([]string, len(assignList))

	for i, user := range assignList {
		assignStringList[i] = nameFormatter(user.UserName)
	}
	switch p.Action {
	case api.HookIssueAssigned:
		operateResult = fmt.Sprintf("%s assign this to %s", nameFormatter(p.Sender.UserName), nameFormatter(assignList[len(assignList)-1].UserName))
	case api.HookIssueUnassigned:
		operateResult = fmt.Sprintf("%s unassigned this for someone", nameFormatter(p.Sender.UserName))
	case api.HookIssueMilestoned:
		operateResult = fmt.Sprintf("%s/milestone/%d", p.Repository.HTMLURL, p.Issue.Milestone.ID)
	}
	link = p.Issue.HTMLURL
	by = fmt.Sprintf("Issue by %s", nameFormatter(p.Issue.Poster.UserName))
	if len(assignStringList) > 0 {
		assignees = fmt.Sprintf("Assignees: %s", strings.Join(assignStringList, ", "))
	}
	operator = fmt.Sprintf("Operator: %s", nameFormatter(p.Sender.UserName))
	return issueTitle, link, by, operator, operateResult, assignees
}

// getIssuesCommentInfo gets the information for a comment
func getIssuesCommentInfo(p *api.IssueCommentPayload, nameFormatter nameFormatter) (title, link, by, operator string) {
	title = fmt.Sprintf("[Comment-%s #%d]: %s\n%s", p.Repository.FullName, p.Issue.Index, p.Action, p.Issue.Title)
	link = p.Issue.HTMLURL
	if p.IsPull {
		by = fmt.Sprintf("PullRequest by %s", nameFormatter(p.Issue.Poster.UserName))
	} else {
		by = fmt.Sprintf("Issue by %s", nameFormatter(p.Issue.Poster.UserName))
	}
	operator = fmt.Sprintf("Operator: %s", nameFormatter(p.Sender.UserName))
	return title, link, by, operator
}

func getIssuesPayloadInfo(p *api.IssuePayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool, withRepoName bool) (text string, issueTitle string, attachmentText string, color int) {
	issueTitle = fmt.Sprintf("#%d %s", p.Index, p.Issue.Title)
	titleLink := linkFormatter(fmt.Sprintf("%s/issues/%d", p.Repository.HTMLURL, p.Index), issueTitle)
	color = yellowColor

	repoPrefix := ""
	if withRepoName {
		repoPrefix = fmt.Sprintf("[%s] ", p.Repository.FullName)
	}

	switch p.Action {
	case api.HookIssueOpened:
		text = fmt.Sprintf("%sIssue opened: %s", repoPrefix, titleLink)
		color = orangeColor
	case api.HookIssueClosed:
		text = fmt.Sprintf("%sIssue closed: %s", repoPrefix, titleLink)
		color = redColor
	case api.HookIssueReOpened:
		text = fmt.Sprintf("%sIssue re-opened: %s", repoPrefix, titleLink)
	case api.HookIssueEdited:
		text = fmt.Sprintf("%sIssue edited: %s", repoPrefix, titleLink)
	case api.HookIssueAssigned:
		list := make([]string, len(p.Issue.Assignees))
		for i, user := range p.Issue.Assignees {
			list[i] = linkFormatter(setting.AppURL+url.PathEscape(user.UserName), user.UserName)
		}
		text = fmt.Sprintf("%sIssue assigned to %s: %s", repoPrefix, strings.Join(list, ", "), titleLink)
		color = greenColor
	case api.HookIssueUnassigned:
		text = fmt.Sprintf("%sIssue unassigned: %s", repoPrefix, titleLink)
	case api.HookIssueLabelUpdated:
		text = fmt.Sprintf("%sIssue labels updated: %s", repoPrefix, titleLink)
	case api.HookIssueLabelCleared:
		text = fmt.Sprintf("%sIssue labels cleared: %s", repoPrefix, titleLink)
	case api.HookIssueSynchronized:
		text = fmt.Sprintf("%sIssue synchronized: %s", repoPrefix, titleLink)
	case api.HookIssueMilestoned:
		text = fmt.Sprintf("%sIssue milestoned to %s: %s", repoPrefix, p.Issue.Milestone.Title, titleLink)
	case api.HookIssueDemilestoned:
		text = fmt.Sprintf("%sIssue milestone cleared: %s", repoPrefix, titleLink)
	}
	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	if p.Action == api.HookIssueOpened || p.Action == api.HookIssueEdited {
		attachmentText = p.Issue.Body
	}

	return text, issueTitle, attachmentText, color
}

func getPullRequestPayloadInfo(p *api.PullRequestPayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool, withRepoName bool) (text string, issueTitle string, attachmentText string, color int) {
	issueTitle = fmt.Sprintf("#%d %s", p.Index, p.PullRequest.Title)
	titleLink := linkFormatter(p.PullRequest.URL, issueTitle)
	color = yellowColor

	repoPrefix := ""
	if withRepoName {
		repoPrefix = fmt.Sprintf("[%s] ", p.Repository.FullName)
	}

	switch p.Action {
	case api.HookIssueOpened:
		text = fmt.Sprintf("%sPull request opened: %s", repoPrefix, titleLink)
		attachmentText = p.PullRequest.Body
		color = greenColor
	case api.HookIssueClosed:
		if p.PullRequest.HasMerged {
			text = fmt.Sprintf("%sPull request merged: %s", repoPrefix, titleLink)
			color = purpleColor
		} else {
			text = fmt.Sprintf("%sPull request closed: %s", repoPrefix, titleLink)
			color = redColor
		}
	case api.HookIssueReOpened:
		text = fmt.Sprintf("%sPull request re-opened: %s", repoPrefix, titleLink)
	case api.HookIssueEdited:
		text = fmt.Sprintf("%sPull request edited: %s", repoPrefix, titleLink)
		attachmentText = p.PullRequest.Body
	case api.HookIssueAssigned:
		list := make([]string, len(p.PullRequest.Assignees))
		for i, user := range p.PullRequest.Assignees {
			list[i] = linkFormatter(setting.AppURL+user.UserName, user.UserName)
		}
		text = fmt.Sprintf("%sPull request assigned to %s: %s", repoPrefix,
			strings.Join(list, ", "), titleLink)
		color = greenColor
	case api.HookIssueUnassigned:
		text = fmt.Sprintf("%sPull request unassigned: %s", repoPrefix, titleLink)
	case api.HookIssueLabelUpdated:
		text = fmt.Sprintf("%sPull request labels updated: %s", repoPrefix, titleLink)
	case api.HookIssueLabelCleared:
		text = fmt.Sprintf("%sPull request labels cleared: %s", repoPrefix, titleLink)
	case api.HookIssueSynchronized:
		text = fmt.Sprintf("%sPull request synchronized: %s", repoPrefix, titleLink)
	case api.HookIssueMilestoned:
		text = fmt.Sprintf("%sPull request milestoned to %s: %s", repoPrefix, p.PullRequest.Milestone.Title, titleLink)
	case api.HookIssueDemilestoned:
		text = fmt.Sprintf("%sPull request milestone cleared: %s", repoPrefix, titleLink)
	case api.HookIssueReviewed:
		text = fmt.Sprintf("%sPull request reviewed: %s", repoPrefix, titleLink)
		attachmentText = p.Review.Content
	case api.HookIssueReviewRequested:
		text = fmt.Sprintf("%sPull request review requested: %s", repoPrefix, titleLink)
	case api.HookIssueReviewRequestRemoved:
		text = fmt.Sprintf("%sPull request review request removed: %s", repoPrefix, titleLink)
	}
	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	return text, issueTitle, attachmentText, color
}

func getReleasePayloadInfo(p *api.ReleasePayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool, withRepoName bool) (text string, color int) {
	refLink := linkFormatter(p.Repository.HTMLURL+"/releases/tag/"+util.PathEscapeSegments(p.Release.TagName), p.Release.TagName)

	repoPrefix := ""
	if withRepoName {
		repoPrefix = fmt.Sprintf("[%s] ", p.Repository.FullName)
	}

	switch p.Action {
	case api.HookReleasePublished:
		text = fmt.Sprintf("%sRelease created: %s", repoPrefix, refLink)
		color = greenColor
	case api.HookReleaseUpdated:
		text = fmt.Sprintf("%sRelease updated: %s", repoPrefix, refLink)
		color = yellowColor
	case api.HookReleaseDeleted:
		text = fmt.Sprintf("%sRelease deleted: %s", repoPrefix, refLink)
		color = redColor
	}
	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	return text, color
}

func getWikiPayloadInfo(p *api.WikiPayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool, withRepoName bool, withCommitMessage bool) (text string, color int, pageLink string) {
	pageLink = linkFormatter(p.Repository.HTMLURL+"/wiki/"+url.PathEscape(p.Page), p.Page)

	color = greenColor

	repoPrefix := ""
	if withRepoName {
		repoPrefix = fmt.Sprintf("[%s] ", p.Repository.FullName)
	}

	switch p.Action {
	case api.HookWikiCreated:
		text = fmt.Sprintf("%sNew wiki page \"%s\"", repoPrefix, pageLink)
	case api.HookWikiEdited:
		text = fmt.Sprintf("%sWiki page \"%s\" edited", repoPrefix, pageLink)
		color = yellowColor
	case api.HookWikiDeleted:
		text = fmt.Sprintf("%sWiki page \"%s\" deleted", repoPrefix, pageLink)
		color = redColor
	}

	if p.Action != api.HookWikiDeleted && p.Comment != "" && withCommitMessage {
		text += fmt.Sprintf(" (%s)", p.Comment)
	}

	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	return text, color, pageLink
}

func getIssueCommentPayloadInfo(p *api.IssueCommentPayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool, withRepoName bool) (text string, issueTitle string, color int) {
	issueTitle = fmt.Sprintf("#%d %s", p.Issue.Index, p.Issue.Title)

	var typ, titleLink string
	color = yellowColor

	repoPrefix := ""
	if withRepoName {
		repoPrefix = fmt.Sprintf("[%s] ", p.Repository.FullName)
	}

	if p.IsPull {
		typ = "pull request"
		titleLink = linkFormatter(p.Comment.PRURL, issueTitle)
	} else {
		typ = "issue"
		titleLink = linkFormatter(p.Comment.IssueURL, issueTitle)
	}

	switch p.Action {
	case api.HookIssueCommentCreated:
		text = fmt.Sprintf("%sNew comment on %s %s", repoPrefix, typ, titleLink)
		if p.IsPull {
			color = greenColorLight
		} else {
			color = orangeColorLight
		}
	case api.HookIssueCommentEdited:
		text = fmt.Sprintf("%sComment edited on %s %s", repoPrefix, typ, titleLink)
	case api.HookIssueCommentDeleted:
		text = fmt.Sprintf("%sComment deleted on %s %s", repoPrefix, typ, titleLink)
		color = redColor
	}
	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	return text, issueTitle, color
}

func getPackagePayloadInfo(p *api.PackagePayload, linkFormatter linkFormatter, nameFormatter nameFormatter, withSender bool) (text string, color int) {
	refLink := linkFormatter(p.Package.HTMLURL, p.Package.Name+":"+p.Package.Version)

	switch p.Action {
	case api.HookPackageCreated:
		text = fmt.Sprintf("Package created: %s", refLink)
		color = greenColor
	case api.HookPackageDeleted:
		text = fmt.Sprintf("Package deleted: %s", refLink)
		color = redColor
	}
	if withSender {
		text += fmt.Sprintf(" by %s", nameFormatter(p.Sender.UserName))
	}

	return text, color
}

func getActionPayloadInfo(p *api.ActionPayload, linkFormatter linkFormatter) (text string, color int) {
	runLink := linkFormatter(p.Run.HTMLURL, p.Run.Title)
	repoLink := linkFormatter(p.Run.Repo.HTMLURL, p.Run.Repo.FullName)

	switch p.Action {
	case api.HookActionFailure:
		text = fmt.Sprintf("%s Action Failed in %s %s", runLink, repoLink, p.Run.PrettyRef)
		color = redColor
	case api.HookActionRecover:
		text = fmt.Sprintf("%s Action Recovered in %s %s", runLink, repoLink, p.Run.PrettyRef)
		color = greenColor
	case api.HookActionSuccess:
		text = fmt.Sprintf("%s Action Succeeded in %s %s", runLink, repoLink, p.Run.PrettyRef)
		color = greenColor
	}

	return text, color
}

// ToHook convert models.Webhook to api.Hook
// This function is not part of the convert package to prevent an import cycle
func ToHook(repoLink string, w *webhook_model.Webhook) (*api.Hook, error) {
	// config is deprecated, but kept for compatibility
	config := map[string]string{
		"url":          w.URL,
		"content_type": w.ContentType.Name(),
	}
	if w.Type == webhook_module.SLACK {
		if s, ok := (slackHandler{}.Metadata(w)).(*SlackMeta); ok {
			config["channel"] = s.Channel
			config["username"] = s.Username
			config["icon_url"] = s.IconURL
			config["color"] = s.Color
		}
	}

	authorizationHeader, err := w.HeaderAuthorization()
	if err != nil {
		return nil, err
	}
	var metadata any
	if handler := GetWebhookHandler(w.Type); handler != nil {
		metadata = handler.Metadata(w)
	}

	return &api.Hook{
		ID:                  w.ID,
		Type:                w.Type,
		BranchFilter:        w.BranchFilter,
		URL:                 w.URL,
		Config:              config,
		Events:              w.EventsArray(),
		AuthorizationHeader: authorizationHeader,
		ContentType:         w.ContentType.Name(),
		Metadata:            metadata,
		Active:              w.IsActive,
		Updated:             w.UpdatedUnix.AsTime(),
		Created:             w.CreatedUnix.AsTime(),
	}, nil
}
