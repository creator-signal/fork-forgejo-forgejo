// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package options

import (
	"context"
	"net/http"
	"strings"

	"forgejo.org/modules/setting"

	"code.forgejo.org/f3/gof3/v3/options"
	options_http "code.forgejo.org/f3/gof3/v3/options/http"
	"code.forgejo.org/f3/gof3/v3/options/logger"
	"github.com/urfave/cli/v3"
)

type NewMigrationHTTPClientFun func() *http.Client

type Options struct {
	options.Options
	logger.OptionsLogger
	options_http.Implementation

	token string
}

func (o *Options) SetURL(string) {}

func (o *Options) GetURL() string {
	return strings.TrimSuffix(setting.AppURL, "/")
}

func (o *Options) SetBaseURL(string) {}

func (o *Options) GetBaseURL() string {
	return strings.TrimSuffix(setting.AppURL, "/")
}

func ForgeTokenOption(prefix string) string {
	return prefix + "-token"
}

func (o *Options) FromFlags(ctx context.Context, c *cli.Command, prefix string) {
	o.SetToken(c.String(ForgeTokenOption(prefix)))
}

func (o *Options) GetFlags(prefix, category string) []cli.Flag {
	flags := make([]cli.Flag, 0, 10)

	flags = append(flags, &cli.StringFlag{
		Name:     ForgeTokenOption(prefix),
		Usage:    "`TOKEN` of the user",
		Category: prefix,
	})

	return flags
}

func (o *Options) SetToken(token string) {
	o.token = token
}

func (o *Options) GetToken() string {
	return o.token
}
