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

type ContainerRegistryClient struct {
	httpClient     *http.Client
	remoteRegistry *rr_model.RemoteRegistry
}

func NewContainerRegistryClient(rr *rr_model.RemoteRegistry) ContainerRegistryClient {
	client := &http.Client{Timeout: 12 * time.Second}

	crc := ContainerRegistryClient{
		httpClient:     client,
		remoteRegistry: rr,
	}

	return crc
}

func (crc *ContainerRegistryClient) PingRemoteRegistry(ctx context.Context) (*http.Response, error) {
	// Parse the URL
	registryURL, err := url.Parse(crc.remoteRegistry.RemoteURL)
	if err != nil {
		return &http.Response{}, fmt.Errorf("invalid registry URL: %w", err)
	}

	// Try to access the registry's base or /v2/ endpoint
	url := registryURL.ResolveReference(&url.URL{Path: "/v2/"})
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("User-Agent", "Forgejo/1.0")

	resp, err := crc.httpClient.Do(req)
	if err != nil {
		log.Warn("There was an error pinging %q: %v", crc.remoteRegistry.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to host: %w", err)
	}
	defer resp.Body.Close()

	return resp, nil
}

// RemoteRegistryAvailable tests if the remote registry exists
func (crc *ContainerRegistryClient) RemoteRegistryAvailable(resp *http.Response) (bool, error) {
	log.Trace("Checking response from %q at %s", crc.remoteRegistry.Name, crc.remoteRegistry.RemoteURL)

	if resp.StatusCode == http.StatusOK {
		log.Trace("Remote registry %q exists", crc.remoteRegistry.Name)
		return true, nil
	}

	if resp.StatusCode == http.StatusUnauthorized && validateRemoteRegistryHeader(*resp) {
		log.Trace("Remote registry %q exists but request was unauthenticated", crc.remoteRegistry.Name)
		return true, nil
	}

	return false, ErrNoRemoteRegistry
}

func (crc *ContainerRegistryClient) AuthenticateRemoteRegistry(ctx context.Context, resp *http.Response) (*http.Response, error) {
	hasUserAndPW := crc.remoteRegistry.RemoteUser != "" && crc.remoteRegistry.RemotePassword != ""
	hasToken := crc.remoteRegistry.RemoteToken != ""

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
		req.Header.Set("Authorization", "token "+crc.remoteRegistry.RemoteToken)
	} else if hasUserAndPW {
		req.SetBasicAuth(crc.remoteRegistry.RemoteUser, crc.remoteRegistry.RemotePassword)
	} else {
		return &http.Response{}, fmt.Errorf("no authentication info given for %q:", crc.remoteRegistry.Name)
	}

	authResp, err := crc.httpClient.Do(req)
	if err != nil {
		log.Warn("Remote registry authentication failed for %q: %v", crc.remoteRegistry.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to auth endpoint: %w", err)
	}
	defer authResp.Body.Close()

	return authResp, nil
}

// RemoteRegistryAuthenticated does authentication tests against an available remote registry
func (crc *ContainerRegistryClient) RemoteRegistryAuthenticated(resp *http.Response) (bool, error) {
	log.Trace("Checking authentication against %q at %s", crc.remoteRegistry.Name, crc.remoteRegistry.RemoteURL)

	if resp.StatusCode == http.StatusOK {
		log.Trace("Connected to remote registry %q", crc.remoteRegistry.Name)
		return true, nil
	}

	return false, ErrNotAuthenticated
}

func (crc *ContainerRegistryClient) RemoteRegistryConnected(ctx context.Context) (bool, error) {

	resp, err := crc.PingRemoteRegistry(ctx)
	if err != nil {
		return false, err
	}

	isRegistry, err := crc.RemoteRegistryAvailable(resp)
	if err != nil {
		return false, err
	}

	authResp, err := crc.AuthenticateRemoteRegistry(ctx, resp)
	if err != nil {
		return false, err
	}

	isAuthenticated := false
	if isRegistry {
		isAuthenticated, err = crc.RemoteRegistryAuthenticated(authResp)
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
