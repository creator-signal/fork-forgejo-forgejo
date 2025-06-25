// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"forgejo.org/modules/proxy"
	"forgejo.org/modules/setting"
)

// ClientPool manages HTTP clients with connection pooling
type ClientPool struct {
	clients map[string]*http.Client
	mutex   sync.RWMutex
}

var (
	globalClientPool *ClientPool
	once             sync.Once
)

// GetGlobalClientPool returns the global HTTP client pool
func GetGlobalClientPool() *ClientPool {
	once.Do(func() {
		globalClientPool = &ClientPool{
			clients: make(map[string]*http.Client),
		}
	})
	return globalClientPool
}

// GetClient returns an HTTP client for the given configuration key
func (cp *ClientPool) GetClient(key string) *http.Client {
	cp.mutex.RLock()
	if client, exists := cp.clients[key]; exists {
		cp.mutex.RUnlock()
		return client
	}
	cp.mutex.RUnlock()

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cp.clients[key]; exists {
		return client
	}

	client := cp.createClient(key)
	cp.clients[key] = client
	return client
}

// createClient creates a new HTTP client with optimized connection pooling
func (cp *ClientPool) createClient(key string) *http.Client {
	transport := &http.Transport{
		Proxy: proxy.Proxy(),
		DialContext: (&net.Dialer{
			Timeout:   setting.HTTPClient.DialTimeout,
			KeepAlive: setting.HTTPClient.KeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     setting.HTTPClient.ForceHTTP2,
		MaxIdleConns:          setting.HTTPClient.MaxIdleConns,
		MaxIdleConnsPerHost:   setting.HTTPClient.MaxIdleConnsPerHost,
		IdleConnTimeout:       setting.HTTPClient.IdleConnTimeout,
		TLSHandshakeTimeout:   setting.HTTPClient.TLSHandshakeTimeout,
		ExpectContinueTimeout: setting.HTTPClient.ExpectContinueTimeout,
		// Enable connection pooling
		DisableKeepAlives: false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   setting.HTTPClient.DefaultTimeout,
	}
}

// GetClientWithTimeout returns an HTTP client with custom timeout
func (cp *ClientPool) GetClientWithTimeout(key string, timeout time.Duration) *http.Client {
	client := cp.GetClient(key)
	// Create a copy with custom timeout
	return &http.Client{
		Transport: client.Transport,
		Timeout:   timeout,
	}
}

// GetClientWithTLS returns an HTTP client with custom TLS configuration
func (cp *ClientPool) GetClientWithTLS(key string, tlsConfig *tls.Config) *http.Client {
	baseClient := cp.GetClient(key)
	baseTransport := baseClient.Transport.(*http.Transport)

	// Create a new transport with custom TLS config
	transport := &http.Transport{
		Proxy:                 baseTransport.Proxy,
		DialContext:           baseTransport.DialContext,
		ForceAttemptHTTP2:     baseTransport.ForceAttemptHTTP2,
		MaxIdleConns:          baseTransport.MaxIdleConns,
		MaxIdleConnsPerHost:   baseTransport.MaxIdleConnsPerHost,
		IdleConnTimeout:       baseTransport.IdleConnTimeout,
		TLSHandshakeTimeout:   baseTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: baseTransport.ExpectContinueTimeout,
		DisableKeepAlives:     baseTransport.DisableKeepAlives,
		TLSClientConfig:       tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   baseClient.Timeout,
	}
}

// Close closes all clients in the pool
func (cp *ClientPool) Close() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	for _, client := range cp.clients {
		client.CloseIdleConnections()
	}
	cp.clients = make(map[string]*http.Client)
}

// GetDefaultClient returns the default HTTP client
func GetDefaultClient() *http.Client {
	return GetGlobalClientPool().GetClient("default")
}

// GetWebhookClient returns an HTTP client optimized for webhook delivery
func GetWebhookClient() *http.Client {
	pool := GetGlobalClientPool()
	timeout := time.Duration(setting.Webhook.DeliverTimeout) * time.Second
	return pool.GetClientWithTimeout("webhook", timeout)
}

// GetLFSClient returns an HTTP client optimized for LFS operations
func GetLFSClient() *http.Client {
	return GetGlobalClientPool().GetClient("lfs")
}

// GetMigrationClient returns an HTTP client for repository migrations
func GetMigrationClient() *http.Client {
	return GetGlobalClientPool().GetClient("migration")
}
