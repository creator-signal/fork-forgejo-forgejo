package container

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/blob"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

var (
	ErrNoRemoteRegistry = util.NewNotExistErrorf("no remote container registry found for given hostname")
	ErrNotAuthenticated = util.NewPermissionDeniedErrorf("authentication to given remote container registry failed")
	ErrNoAuthInfo       = util.NewNotExistErrorf("no authentication info for given remote")
)

type RegistryClient struct {
	httpClient     *http.Client
	RegClient      *regclient.RegClient
	RemoteRegistry *rr_model.RemoteRegistry
	Reference      *ref.Ref
}

func NewContainerRegistryClient(rr *rr_model.RemoteRegistry, names ...string) (RegistryClient, error) {
	client := &http.Client{Timeout: 12 * time.Second}

	rrURL, err := url.Parse(rr.RemoteURL)
	if err != nil {
		return RegistryClient{}, fmt.Errorf("Unable to create registry client, url was invalid: %s", err)
	}

	tls := config.TLSDisabled
	if strings.Contains(rr.RemoteURL, "https") {
		tls = config.TLSEnabled
	}

	pass := ""
	if len(rr.RemotePassword) > 0 {
		pass, err = rr.GetRemotePassword()
		if err != nil {
			return RegistryClient{}, fmt.Errorf("Unable to create registry client: %s", err)
		}
	}

	token := ""
	if len(rr.RemoteToken) > 0 {
		token, err = rr.GetRemoteToken()
		if err != nil {
			return RegistryClient{}, fmt.Errorf("Unable to create registry client: %s", err)
		}
	}

	remoteRegistryConfig := config.Host{
		Name:  rrURL.Host,
		TLS:   tls,
		User:  rr.RemoteUser,
		Pass:  pass,
		Token: token,
	}

	regclient := regclient.New(
		regclient.WithConfigHost(remoteRegistryConfig),
	)

	// Ref host, repo if exists, image
	crc := RegistryClient{
		httpClient:     client,
		RegClient:      regclient,
		RemoteRegistry: rr,
	}

	if len(names) > 0 {
		host := crc.RemoteRegistry.RemoteHost
		refStr := host + "/" + names[0]
		r, err := ref.New(refStr)
		if err != nil {
			return RegistryClient{}, fmt.Errorf("Unable to create registry client, failed to create reference: %s", err)
		}
		crc.Reference = &r
	}

	return crc, nil
}

func (crc *RegistryClient) PingRemoteRegistry(ctx context.Context) (*http.Response, error) {
	// Parse the URL
	registryURL, err := url.Parse(crc.RemoteRegistry.RemoteURL)
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
		log.Warn("There was an error pinging %q: %v", crc.RemoteRegistry.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to host: %w", err)
	}
	defer resp.Body.Close()

	return resp, nil
}

// RemoteRegistryAvailable tests if the remote registry exists
func (crc *RegistryClient) RemoteRegistryAvailable(resp *http.Response) error {
	log.Trace("Checking response from %q at %s", crc.RemoteRegistry.Name, crc.RemoteRegistry.RemoteURL)

	if resp.StatusCode == http.StatusOK {
		log.Trace("Remote registry %q exists", crc.RemoteRegistry.Name)
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized && validateRemoteRegistryHeader(*resp) {
		log.Trace("Remote registry %q exists but request was unauthenticated", crc.RemoteRegistry.Name)
		return nil
	}

	return ErrNoRemoteRegistry
}

func (crc *RegistryClient) AuthenticateRemoteRegistry(ctx context.Context, resp *http.Response) (*http.Response, error) {
	authResp := &http.Response{}
	var credential string
	hasUserAndPW := crc.RemoteRegistry.RemoteUser != "" && len(crc.RemoteRegistry.RemotePassword) > 0
	hasToken := len(crc.RemoteRegistry.RemoteToken) > 0

	authHeader := resp.Header.Get("WWW-Authenticate")
	authURL, err := extractAuthURL(authHeader)
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to extract auth URL: %w", err)
	}

	serviceURL, err := extractServiceURL(authHeader) // TODO Forgejo itself only returns container_registry
	if err != nil {
		return &http.Response{}, fmt.Errorf("failed to extract service URL: %w", err)
	}

	req := &http.Request{}
	req, err = http.NewRequestWithContext(ctx, "GET", authURL, nil)
	if err != nil {
		log.Warn("Remote registry authentication failed for %q: %v", crc.RemoteRegistry.Name, err)
		return &http.Response{}, err
	}
	query := req.URL.Query()

	if hasToken {
		credential, err = crc.RemoteRegistry.GetRemoteToken()
		if err != nil {
			return &http.Response{}, fmt.Errorf("failed to create auth request: %w", err)
		}
		req.SetBasicAuth(crc.RemoteRegistry.RemoteUser, credential)
	} else if hasUserAndPW {
		credential, err = crc.RemoteRegistry.GetRemotePassword()
		if err != nil {
			return &http.Response{}, fmt.Errorf("failed to create auth request: %w", err)
		}
		req.SetBasicAuth(crc.RemoteRegistry.RemoteUser, credential)
	} else {
		return &http.Response{}, ErrNoAuthInfo
	}
	query.Add("service", serviceURL)
	query.Add("scope", "pull")
	query.Add("client_id", "docker") // TODO this may need to be configurable

	// https://distribution.github.io/distribution/spec/auth/oauth/
	if strings.Contains(crc.RemoteRegistry.RemoteHost, "docker") {
		req.Method = http.MethodPost
		query.Add("grant_type", "password")
		query.Add("access_type", "offline")
		query.Add("username", crc.RemoteRegistry.RemoteUser)
		query.Add("password", credential)
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	} else {
		query = req.URL.Query()
		query.Add("account", crc.RemoteRegistry.RemoteUser)
		if hasToken || hasUserAndPW {
			query.Add("offline_token", "true")
		}
		req.Header.Set("User-Agent", "Forgejo/1.0")
	}

	req.URL.RawQuery = query.Encode()
	authResp, err = crc.httpClient.Do(req)
	if err != nil {
		log.Warn("Remote registry authentication failed for %q: %v", crc.RemoteRegistry.Name, err)
		return &http.Response{}, err
	}

	if authResp.StatusCode != 200 {
		log.Warn("Remote registry authentication failed for %q: %v", crc.RemoteRegistry.Name, err)
		return &http.Response{}, fmt.Errorf("failed to connect to auth endpoint with Status: %s", authResp.Status)
	}
	defer authResp.Body.Close()

	return authResp, nil
}

// RemoteRegistryAuthenticated does authentication tests against an available remote registry
func (crc *RegistryClient) RemoteRegistryAuthenticated(resp *http.Response) error {
	log.Trace("Checking authentication against %q at %s", crc.RemoteRegistry.Name, crc.RemoteRegistry.RemoteURL)

	if resp.StatusCode != http.StatusOK {
		log.Trace("Failed to connect to remote registry %q with code: %v", crc.RemoteRegistry.Name, resp.StatusCode)
		return ErrNotAuthenticated
	}

	return nil
}

func (crc *RegistryClient) RemoteRegistryConnected(ctx context.Context) (bool, error) {
	resp, err := crc.PingRemoteRegistry(ctx)
	if err != nil {
		return false, err
	}

	err = crc.RemoteRegistryAvailable(resp)
	if err != nil {
		return false, err
	}

	authResp, err := crc.AuthenticateRemoteRegistry(ctx, resp)
	if err == ErrNoAuthInfo {
		return true, nil
	} else if err != nil {
		return false, err
	}

	err = crc.RemoteRegistryAuthenticated(authResp)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (crc *RegistryClient) NewImager(man manifest.Manifest) manifest.Imager {
	img := man.(manifest.Imager)
	return img
}

// Close cleans up resources used by the client
func (crc *RegistryClient) Close(ctx context.Context) error {
	// Close idle connections
	if transport, ok := crc.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	err := crc.RegClient.Close(ctx, *crc.Reference)
	return err
}

func (crc *RegistryClient) HeadBlob(ctx context.Context, d descriptor.Descriptor) (blob.Reader, error) {
	reader, err := crc.RegClient.BlobHead(ctx, *crc.Reference, d)
	if err != nil {
		return nil, fmt.Errorf("failed to head blob %s: %w", crc.Reference, err)
	}
	return reader, nil
}

func (crc *RegistryClient) GetBlob(ctx context.Context, d descriptor.Descriptor) (blob.Reader, error) {
	reader, err := crc.RegClient.BlobGet(ctx, *crc.Reference, d)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob %s: %w", crc.Reference, err)
	}
	return reader, nil
}

func (crc *RegistryClient) HeadManifest(ctx context.Context) (manifest.Manifest, error) {
	m, err := crc.RegClient.ManifestHead(ctx, *crc.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to head manifest %s: %w", crc.Reference, err)
	}
	return m, nil
}

func (crc *RegistryClient) GetManifest(ctx context.Context) (manifest.Manifest, error) {
	m, err := crc.RegClient.ManifestGet(ctx, *crc.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest %s: %w", crc.Reference, err)
	}
	return m, nil
}

func (crc *RegistryClient) ListTags(ctx context.Context, ownerLower, image string) (*TagList, error) {
	tag, err := crc.RegClient.TagList(ctx, *crc.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest %s: %w", crc.Reference, err)
	}

	return &TagList{
		Name: strings.ToLower(ownerLower + "/" + image),
		Tags: tag.Tags,
	}, nil
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
		return "", fmt.Errorf("could not extract service from bearer string")
	}

	return matches[1], nil
}
