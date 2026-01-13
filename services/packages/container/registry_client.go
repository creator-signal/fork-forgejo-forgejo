package container

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
)

var (
	ErrRemoteRegistryConnectionFailed = util.NewInvalidArgumentErrorf("could not connect to remote registry")
)

// RemoteRegistryConnected tests connectivity to a remote registry
func RemoteRegistryConnected(ctx context.Context, rr *rr_model.RemoteRegistry) (bool, error) {
	log.Trace("Testing connection to remote registry %q at %s", rr.Name, rr.RemoteURL)

	hasUserAndPW := rr.RemoteUser != "" && rr.RemotePassword != ""
	hasToken := rr.RemoteToken != ""

	// Parse the URL
	registryURL, err := url.Parse(rr.RemoteURL)
	if err != nil {
		return false, fmt.Errorf("invalid registry URL: %w", err)
	}

	// Create HTTP client with reasonable timeout
	client := &http.Client{Timeout: 12 * time.Second}

	// Try to access the registry's base or /v2/ endpoint
	url := registryURL.ResolveReference(&url.URL{Path: "/v2/"})
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("User-Agent", "Forgejo/1.0")

	if hasToken {
		req.Header.Set("Authorization", "Bearer "+rr.RemoteToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Warn("Not connected to %q: %v", rr.Name, err)
		return false, ErrRemoteRegistryConnectionFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Trace("Connected to remote registry %q", rr.Name)
		return true, nil
	}

	if resp.StatusCode == http.StatusUnauthorized && hasToken {
		return false, fmt.Errorf("Bearer token invalid for remote registry %q at %q", rr.Name, rr.RemoteURL)
	}

	if resp.StatusCode == http.StatusUnauthorized && validateRemoteRegistryHeader(*resp) {
		if !hasUserAndPW {
			log.Trace("Connected to remote registry %q", rr.Name)
			return true, nil
		}

		authHeader := resp.Header.Get("WWW-Authenticate")
		authURL, err := extractAuthURL(authHeader)
		if err != nil {
			return false, fmt.Errorf("failed to extract auth URL: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", authURL, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create auth request: %w", err)
		}

		req.Header.Set("User-Agent", "Forgejo/1.0")
		req.SetBasicAuth(rr.RemoteUser, rr.RemotePassword)

		resp, err := client.Do(req)
		if err != nil {
			log.Warn("Remote registry connection test failed for %q: %v", rr.Name, err)
			return false, ErrRemoteRegistryConnectionFailed
		}
		defer resp.Body.Close()

		log.Error("%+v", resp)
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}

		return false, fmt.Errorf("authentication failed for remote registry %q", rr.Name)
	}

	return false, fmt.Errorf("remote registry %q returned unexpected status %d", rr.Name, resp.StatusCode)
}

func validateRemoteRegistryHeader(resp http.Response) bool {
	return resp.Header.Get("Docker-Distribution-API-Version") != "" && resp.Header.Get("WWW-Authenticate") != ""
}

func extractAuthURL(bearer string) (string, error) {
	re := regexp.MustCompile(`realm="([^"]+)"`)
	matches := re.FindStringSubmatch(bearer)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract realm from bearer string")
	}

	return matches[1], nil
}
