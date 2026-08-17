// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	forgejo_log "forgejo.org/modules/log"
	driver_options "forgejo.org/services/f3/driver/options"
	f3_permissions_helpers "forgejo.org/services/f3/permissions/helpers"
	"forgejo.org/services/f3/util"

	"code.forgejo.org/f3/gof3/v3/options"
	"github.com/stretchr/testify/require"
)

func newTestOptions(t *testing.T) options.Interface {
	o := options.GetFactory(driver_options.Name)().(*driver_options.Options)
	l := forgejo_log.GetLogger(forgejo_log.DEFAULT)
	o.SetLogger(util.NewF3Logger(nil, l))

	token, _, err := f3_permissions_helpers.CreateAdminToken(t.Context(), auth_model.Write)
	require.NoError(t, err)
	o.SetToken(token)
	return o
}
