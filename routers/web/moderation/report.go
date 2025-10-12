// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/moderation"
	"forgejo.org/models/repo"
	"forgejo.org/models/system"
	"forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/federation"
	"forgejo.org/services/forms"
	moderation_service "forgejo.org/services/moderation"
)

const (
	tplSubmitAbuseReport base.TplName = "moderation/new_abuse_report"
)

// NewReport renders the page for new abuse reports.
func NewReport(ctx *context.Context) {
	contentID := ctx.FormInt64("id")
	if contentID <= 0 {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The content ID is expected to be an integer greater that 0; the provided value is %s.", ctx.FormString("id"))
		return
	}

	contentTypeString := ctx.FormString("type")
	var contentType moderation.ReportedContentType
	switch contentTypeString {
	case "user", "org":
		contentType = moderation.ReportedContentTypeUser
	case "repo":
		contentType = moderation.ReportedContentTypeRepository
	case "issue", "pull":
		contentType = moderation.ReportedContentTypeIssue
	case "comment":
		contentType = moderation.ReportedContentTypeComment
	default:
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The provided content type `%s` is not among the expected values.", contentTypeString)
		return
	}

	if moderation.AlreadyReportedByAndOpen(ctx, ctx.Doer.ID, contentType, contentID) {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.already_reported"), tplSubmitAbuseReport, nil)
		return
	}

	setContextDataAndRender(ctx, contentType, contentID)
}

// setMinimalContextData adds minimal values (Title and CancelLink) into context data.
func setMinimalContextData(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("moderation.report_abuse")
	ctx.Data["CancelLink"] = ctx.Doer.DashboardLink()
}

func fillApContextData(ctx *context.Context, contentType moderation.ReportedContentType, contentID int64) error {
	ctx.Data["FederatedContent"] = false

	federationHostID := int64(-1)
	activityPubID := ""

	switch contentType {
	case moderation.ReportedContentTypeRepository:
		federatedRepo, err := repo.GetFollowingRepoByID(ctx, contentID)
		if err != nil {
			if repo.IsErrRepoNotExist(err) {
				log.Error("Missing following repo %d", contentID)
				return nil
			}

			return err
		}

		federationHostID = federatedRepo.FederationHostID
		activityPubID = federatedRepo.URI

	case moderation.ReportedContentTypeUser:
		_, federatedUser, err := user.GetFederatedUserByUserID(ctx, contentID)
		if err != nil {
			if user.IsErrUserNotExist(err) {
				return nil
			}

			return err
		}

		federationHostID = federatedUser.FederationHostID
		activityPubID = federatedUser.NormalizedOriginalURL

	case moderation.ReportedContentTypeComment, moderation.ReportedContentTypeIssue:
		// TODO: these are not federated yet. When federation for issues pull requests is eventually implemented, handle these cases
		return nil
	}

	federationHost, err := forgefed.GetFederationHost(ctx, federationHostID)
	if err != nil {
		return err
	}

	log.Debug("ActivityPub ID of remote content: %s", activityPubID)

	ctx.Data["FederatedContent"] = true
	ctx.Data["FederationHost"] = federationHost.HostFqdn
	ctx.Data["FederationHostID"] = federationHostID
	ctx.Data["ActivityPubID"] = activityPubID

	return nil
}

// setContextDataAndRender adds some values into context data and renders the new abuse report page.
func setContextDataAndRender(ctx *context.Context, contentType moderation.ReportedContentType, contentID int64) {
	setMinimalContextData(ctx)
	ctx.Data["ContentID"] = contentID
	ctx.Data["ContentType"] = contentType
	ctx.Data["AbuseCategories"] = moderation.GetAbuseCategoriesList()

	if setting.Federation.Enabled {
		err := fillApContextData(ctx, contentType, contentID)
		if err != nil {
			log.Error("moderation.fillApContextData: %s", err)
			ctx.Error(http.StatusInternalServerError, "moderation.fillApContextData", err.Error())
			return
		}
	}

	ctx.HTML(http.StatusOK, tplSubmitAbuseReport)
}

// CreatePost handles the POST for creating a new abuse report.
func CreatePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.ReportAbuseForm)

	if form.ContentID <= 0 || !form.ContentType.IsValid() {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		return
	}

	if ctx.HasError() {
		setContextDataAndRender(ctx, form.ContentType, form.ContentID)
		return
	}

	can, err := moderation_service.CanReport(*ctx, ctx.Doer, form.ContentType, form.ContentID)
	if err != nil {
		if errors.Is(err, moderation_service.ErrContentDoesNotExist) || errors.Is(err, moderation_service.ErrDoerNotAllowed) {
			ctx.Flash.Error(ctx.Tr("moderation.report_abuse_form.invalid"))
			ctx.Redirect(ctx.Doer.DashboardLink())
		} else {
			ctx.ServerError("Failed to check if user can report content", err)
		}
		return
	} else if !can {
		ctx.Flash.Error(ctx.Tr("moderation.report_abuse_form.invalid"))
		ctx.Redirect(ctx.Doer.DashboardLink())
		return
	}

	report := moderation.AbuseReport{
		ReporterID:  ctx.Doer.ID,
		ContentType: form.ContentType,
		ContentID:   form.ContentID,
		Category:    form.AbuseCategory,
		Remarks:     form.Remarks,
	}

	if setting.Federation.Enabled {
		if form.FederationHostID > 0 && form.ForwardRemote && form.ActivityPubID != "" {
			reportUUID, err := federation.ReportContent(ctx, report, form.ActivityPubID)
			if err != nil {
				_ = system.CreateNotice(ctx, system.NoticeTask, fmt.Sprintf("Failed to forward moderation report: %s", err))
			} else {
				report.FederationUUID = sql.NullString{
					String: reportUUID.String(),
					Valid:  true,
				}
			}
		}
	}

	if err := moderation.ReportAbuse(ctx, &report); err != nil {
		if errors.Is(err, moderation.ErrSelfReporting) {
			ctx.Flash.Error(ctx.Tr("moderation.reporting_failed", err))
			ctx.Redirect(ctx.Doer.DashboardLink())
		} else {
			ctx.ServerError("Failed to save new abuse report", err)
		}
		return
	}

	ctx.Flash.Success(ctx.Tr("moderation.reported_thank_you"))
	ctx.Redirect(ctx.Doer.DashboardLink())
}
