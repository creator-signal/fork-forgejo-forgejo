// Copyright 2025 Guiorgy Potskhishvili. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"text/tabwriter"
	"encoding/json"
	"net/http"
	"context"
	"strings"
	"errors"
	"time"
	"fmt"
	"os"

	"forgejo.org/modules/setting"
	"forgejo.org/routers/web/healthcheck"

	"github.com/urfave/cli/v3"
)

// CmdHealth represents the available health sub-command.
func cmdHealth() *cli.Command {
	return &cli.Command{
		Name:  "health",
		Usage: "Check the health of the Forgejo web server",
		Description: "Check the health of the Forgejo web server",
		Action: check,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Do not write anything to standard output. Exit with a status code 0 if the check passes, otherwise, exit with status code 1",
			},
		},
	}
}

func check(ctx context.Context, cmd *cli.Command) error {
	switch setting.Protocol {
		case setting.HTTPUnix:
			// TODO
			return errors.New("HTTPUnix not supported")
		case setting.FCGI:
			// TODO
			return errors.New("FCGI not supported")
		case setting.FCGIUnix:
			// TODO
			return errors.New("FCGIUnix not supported")
		case setting.HTTP:
		case setting.HTTPS:
		default:
			return fmt.Errorf("Invalid protocol: %s", setting.Protocol)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	protocol := strings.ToUpper(string(setting.Protocol))
	url := fmt.Sprintf("%sapi/healthz", setting.LocalURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", setting.Protocol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFailedDependency {
		return fmt.Errorf("%s request failed with status code: %d %s", protocol, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var status healthcheck.Response
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return fmt.Errorf("Failed to decode JSON response: %w", err)
	}

	if status.Status != healthcheck.Pass {
		if !cmd.IsSet("quiet") {
			w := tabwriter.NewWriter(os.Stderr, 0, 8, 1, '\t', 0)
			_, _ = w.Write([]byte("Component\tStatus\tDescription\n"))

			for component, statuses := range status.Checks {
				for _, compStatus := range statuses {
					_, _ = w.Write([]byte(component))
					_, _ = w.Write([]byte{'\t'})
					_, _ = w.Write([]byte(compStatus.Status))
					_, _ = w.Write([]byte{'\t'})
					_, _ = w.Write([]byte(compStatus.Output))
					_, _ = w.Write([]byte{'\n'})
				}
			}

			err = w.Flush()
			if err != nil {
				return fmt.Errorf("Failed to flush tabwriter: %w", err)
			}
		}

		return cli.Exit(fmt.Sprintf("%s server is unhealthy\n", protocol), 1)
	}

	if !cmd.IsSet("quiet") {
		fmt.Printf("%s server is healthy\n", protocol)
	}

	return nil;
}
