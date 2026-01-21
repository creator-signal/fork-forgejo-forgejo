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
	ErrNoRemoteRegistry = util.NewNotExistErrorf("no remote container registry found for given hostname")
	ErrNotAuthenticated = util.NewPermissionDeniedErrorf("authentication to given remote container registry failed")
)

func PingRemoteRegistry(ctx context.Context, rr *rr_model.RemoteRegistry) (*http.Response, error) {
	// Parse the URL
	registryURL, err := url.Parse(rr.RemoteURL)
	if err != nil {
		return &http.Response{}, fmt.Errorf("invalid registry URL: %w", err)
	}

	// Create HTTP client with reasonable timeout
	client := &http.Client{Timeout: 12 * time.Second}

	// Try to access the registry's base or /v2/ endpoint
	url := registryURL.ResolveReference(&url.URL{Path: "/v2/"})
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("User-Agent", "Forgejo/1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Warn("There was an error pinging %q: %v", rr.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to host: %w", err)
	}
	defer resp.Body.Close()

	return resp, nil
}

// RemoteRegistryAvailable tests if the remote registry exists
func RemoteRegistryAvailable(resp *http.Response, rr *rr_model.RemoteRegistry) (bool, error) {
	log.Trace("Checking response from %q at %s", rr.Name, rr.RemoteURL)

	if resp.StatusCode == http.StatusOK {
		log.Trace("Remote registry %q exists", rr.Name)
		return true, nil
	}

	if resp.StatusCode == http.StatusUnauthorized && validateRemoteRegistryHeader(*resp) {
		log.Trace("Remote registry %q exists but request was unauthenticated", rr.Name)
		return true, nil
	}

	return false, ErrNoRemoteRegistry
}

func AuthenticateRemoteRegistry(ctx context.Context, resp *http.Response, rr *rr_model.RemoteRegistry) (*http.Response, error) {
	hasUserAndPW := rr.RemoteUser != "" && rr.RemotePassword != ""
	hasToken := rr.RemoteToken != ""

	authHeader := resp.Header.Get("WWW-Authenticate")
	authURL, err := extractAuthURL(authHeader)
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to extract auth URL: %w", err)
	}

	serviceURL, err := extractServiceURL(authHeader) // TODO Forgejo itself only returns container_registry
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to extract service URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", authURL, nil)
	query := req.URL.Query()
	query.Add("service", serviceURL)
	query.Add("scope", "\"*\"")
	req.URL.RawQuery = query.Encode()
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("User-Agent", "Forgejo/1.0")

	if hasToken {
		req.Header.Set("Authorization", "token "+rr.RemoteToken)
	} else if hasUserAndPW {
		req.SetBasicAuth(rr.RemoteUser, rr.RemotePassword)
	} else {
		return &http.Response{}, fmt.Errorf("no authentication info given for %q:", rr.Name)
	}

	// Create HTTP client with reasonable timeout
	client := &http.Client{Timeout: 12 * time.Second}

	authResp, err := client.Do(req)
	if err != nil {
		log.Warn("Remote registry authentication failed for %q: %v", rr.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to auth endpoint: %w", err)
	}
	defer authResp.Body.Close()

	return authResp, nil
}

// RemoteRegistryAuthenticated does authentication tests against an available remote registry
func RemoteRegistryAuthenticated(resp *http.Response, rr *rr_model.RemoteRegistry) (bool, error) {
	log.Trace("Checking authentication against %q at %s", rr.Name, rr.RemoteURL)

	if resp.StatusCode == http.StatusOK {
		log.Trace("Connected to remote registry %q", rr.Name)
		return true, nil
	}

	return false, ErrNotAuthenticated
}

func RemoteRegistryConnected(ctx context.Context, rr *rr_model.RemoteRegistry) (bool, error) {

	resp, err := PingRemoteRegistry(ctx, rr)
	if err != nil {
		return false, err
	}

	isRegistry, err := RemoteRegistryAvailable(resp, rr)
	if err != nil {
		return false, err
	}

	authResp, err := AuthenticateRemoteRegistry(ctx, resp, rr)
	if err != nil {
		return false, err
	}

	isAuthenticated := false
	if isRegistry {
		isAuthenticated, err = RemoteRegistryAuthenticated(authResp, rr)
	}

	if isRegistry && isAuthenticated {
		return true, nil
	}

	return false, err
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

func extractServiceURL(bearer string) (string, error) {
	re := regexp.MustCompile(`service="([^"]+)"`)
	matches := re.FindStringSubmatch(bearer)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract realm from bearer string")
	}

	return matches[1], nil
}
