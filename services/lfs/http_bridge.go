// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package lfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	lfs_module "forgejo.org/modules/lfs"
	"forgejo.org/modules/proxy"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
)

type HTTPBridge struct {
	client     *http.Client
	bearer     string
	pathPrefix string
	repository *repo_model.Repository
	user       *user_model.User
}

func newHTTPBridge(token string, user *user_model.User, repository *repo_model.Repository) *HTTPBridge {
	httpTransport := &http.Transport{
		Proxy:           proxy.Proxy(),
		MaxConnsPerHost: 1,
		MaxIdleConns:    1,
		IdleConnTimeout: time.Second,
		WriteBufferSize: MaxPacketLength,
	}

	hc := &http.Client{Transport: httpTransport}
	client := &HTTPBridge{
		client:     hc,
		bearer:     fmt.Sprintf("Bearer %s", token),
		user:       user,
		repository: repository,
		pathPrefix: setting.LocalURL + repository.FullName(),
	}

	return client
}

func (b *HTTPBridge) createRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating bridge request: %v", err)
	}
	req.Header.Set("Accept", lfs_module.AcceptHeader)
	req.Header.Set("Content-Type", lfs_module.AcceptHeader)
	req.Header.Set("User-Agent", lfs_module.UserAgentHeader)
	req.Header.Set("username", b.user.Name)
	req.Header.Set("reponame", b.repository.Name)
	req.Header.Set("Authorization", b.bearer)
	return req, nil
}

func (b *HTTPBridge) performRequest(ctx context.Context, request *http.Request) (*http.Response, error) {
	resp, err := b.client.Do(request)
	if err != nil {
		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		default:
		}
		return resp, err
	}
	return resp, nil
}

func (b *HTTPBridge) performOKRequest(ctx context.Context, request *http.Request) (*http.Response, error) {
	resp, err := b.performRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("bridge failed with %d status and body cannot by read: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("bridge failed with %d status: %s", resp.StatusCode, body)
	}

	return resp, nil
}

func (b *HTTPBridge) Batch(ctx context.Context, operation string, objects []lfs_module.Pointer) error {
	url := fmt.Sprintf("%s.git/info/lfs/objects/batch", b.pathPrefix)

	request := &lfs_module.BatchRequest{Operation: operation, Transfers: nil, Ref: nil, Objects: objects}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(request)
	if err != nil {
		return fmt.Errorf("Error encoding batch request: %v", err)
	}

	req, err := b.createRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}
	resp, err := b.performOKRequest(ctx, req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (b *HTTPBridge) Upload(ctx context.Context, pointer *lfs_module.Pointer, pktAdapter *PktAdapter) error {
	url := fmt.Sprintf("%s.git/info/lfs/objects/%s/%d", b.pathPrefix, pointer.Oid, pointer.Size)

	binaryReader := pktAdapter.GetBinaryReader(int(pointer.Size))
	var err error
	defer func() {
		_, _ = io.Copy(io.Discard, binaryReader) // Make sure to consume all data
	}()

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := b.createRequest(requestCtx, http.MethodPut, url, binaryReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Transfer-Encoding", "chunked")
	req.ContentLength = pointer.Size
	resp, err := b.performOKRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !binaryReader.IsDone() { // Needed if server didn't read all the request data
		cancel()
		binaryReader.Close()
	}
	return nil
}

func (b *HTTPBridge) Verify(ctx context.Context, pointer *lfs_module.Pointer) error {
	url := fmt.Sprintf("%s.git/info/lfs/verify", b.pathPrefix)

	body, err := json.Marshal(pointer)
	if err != nil {
		return fmt.Errorf("Error encoding verify request: %v", err)
	}

	req, err := b.createRequest(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := b.performOKRequest(ctx, req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (b *HTTPBridge) Download(ctx context.Context, pointer *lfs_module.Pointer, pktAdapter *PktAdapter) error {
	url := fmt.Sprintf("%s.git/info/lfs/objects/%s/%d", b.pathPrefix, pointer.Oid, pointer.Size)

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := b.createRequest(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := b.performOKRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 0)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP Content-Length from header: %v", resp.Header)
	}
	return pktAdapter.WriteBinaryData(size, resp.Body)
}

func (b *HTTPBridge) ListLock(ctx context.Context, args map[string]string) (*api.LFSLockList, error) {
	query := make(url.Values)
	for _, argName := range []string{"path", "id", "limit", "cursor", "refspec"} {
		if argValue, ok := args[argName]; ok {
			query[argName] = []string{argValue}
		}
	}
	url := fmt.Sprintf("%s.git/info/lfs/locks?%s", b.pathPrefix, query.Encode())

	req, err := b.createRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.performOKRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response api.LFSLockList
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("Error decoding json: %v", err)
	}
	return &response, nil
}

func (b *HTTPBridge) Lock(ctx context.Context, path string) (*api.LFSLockResponse, *api.LFSLockError, int, bool, error) {
	url := fmt.Sprintf("%s.git/info/lfs/locks", b.pathPrefix)

	request := &api.LFSLockRequest{Path: path}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(request)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, fmt.Errorf("Error encoding lock request: %v", err)
	}

	req, err := b.createRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, err
	}
	resp, err := b.performRequest(ctx, req)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, err
	}
	defer resp.Body.Close()

	var response api.LFSLockResponse
	var errorResponse api.LFSLockError
	ok := false
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, ok, fmt.Errorf("Error decoding lock error response json: %v", err)
		}
	} else {
		ok = true
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, ok, fmt.Errorf("Error decoding lock response json: %v", err)
		}
	}
	return &response, &errorResponse, resp.StatusCode, ok, nil
}

func (b *HTTPBridge) Unlock(ctx context.Context, lid string, force bool,
) (*api.LFSLockResponse, *api.LFSLockError, int, bool, error) {
	url := fmt.Sprintf("%s.git/info/lfs/locks/%s/unlock", b.pathPrefix, lid)

	request := &api.LFSLockDeleteRequest{Force: force}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(request)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, fmt.Errorf("Error encoding unlock request: %v", err)
	}

	req, err := b.createRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, err
	}
	resp, err := b.performRequest(ctx, req)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, false, fmt.Errorf("Unlocking error with body \"%s\": %v", body, err)
	}
	defer resp.Body.Close()

	var response api.LFSLockResponse
	var errorResponse api.LFSLockError
	ok := false
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, false, fmt.Errorf("Error decoding unlock error response json: %v", err)
		}
	} else {
		ok = true
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, false, fmt.Errorf("Error decoding unlock response json: %v", err)
		}
	}
	return &response, &errorResponse, resp.StatusCode, ok, nil
}
