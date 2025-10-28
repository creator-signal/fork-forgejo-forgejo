// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/moderation"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
	"github.com/google/uuid"
)

func Reports(ctx *context.APIContext) {
	reportUUIDString := ctx.Req.PathValue("report-uuid")
	reportUUID, err := uuid.Parse(reportUUIDString)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "Invalid UUID", err)
		return
	}

	log.Debug("Report UUID: %s", reportUUID)
	report, forwardedReport, err := moderation.GetReportByUUID(ctx, reportUUID)
	if err != nil {
		if moderation.IsErrReportNotExists(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}

		return
	}

	if setting.Federation.SignatureEnforced {
		federationHost, err := forgefed.GetFederationHost(ctx, forwardedReport.FederationHostID)
		if err != nil {
			ctx.InternalServerError(err)
			return
		}

		verifier, err := httpsig.NewVerifier(ctx.Req)
		if err != nil {
			ctx.InternalServerError(err)
			return
		}

		keyID, err := url.Parse(verifier.KeyId())
		if err != nil {
			ctx.InternalServerError(err)
			return
		}

		keyIDPort := keyID.Port()
		if keyIDPort == "" {
			keyIDPort = "443"
		}

		federationHostPort := strconv.Itoa(int(federationHost.HostPort))
		if keyID.Hostname() != federationHost.HostFqdn || keyIDPort != federationHostPort {
			ctx.Error(http.StatusUnauthorized, "KeyID hostname does not match FederationHost fqdn", "Invalid request origin")
			return
		}
	}

	reportedObjects := []string{}
	if forwardedReport.ActivityPubIDs.Valid {
		parts := strings.Split(forwardedReport.ActivityPubIDs.String, ";")
		reportedObjects = append(reportedObjects, parts...)
	}

	flagJSON, _, err := federation.BuildReport(reportUUID, forwardedReport.ActorID, reportedObjects, report)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	_, err = ctx.Resp.Write(flagJSON)
	if err != nil {
		log.Error("Error writing response: %s", err)
	}
}
