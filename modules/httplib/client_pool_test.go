// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"forgejo.org/modules/setting"
)

func TestClientPool(t *testing.T) {
	// Initialize settings for testing
	setting.HTTPClient.MaxIdleConns = 50
	setting.HTTPClient.MaxIdleConnsPerHost = 5
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	// Test getting a client
	client1 := pool.GetClient("test")
	if client1 == nil {
		t.Fatal("Expected non-nil client")
	}

	// Test getting the same client again (should be cached)
	client2 := pool.GetClient("test")
	if client1 != client2 {
		t.Fatal("Expected same client instance for same key")
	}

	// Test getting a different client
	client3 := pool.GetClient("test2")
	if client3 == client1 {
		t.Fatal("Expected different client instance for different key")
	}

	// Test transport configuration
	transport := client1.Transport.(*http.Transport)
	if transport.MaxIdleConns != setting.HTTPClient.MaxIdleConns {
		t.Errorf("Expected MaxIdleConns %d, got %d", setting.HTTPClient.MaxIdleConns, transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != setting.HTTPClient.MaxIdleConnsPerHost {
		t.Errorf("Expected MaxIdleConnsPerHost %d, got %d", setting.HTTPClient.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != setting.HTTPClient.IdleConnTimeout {
		t.Errorf("Expected IdleConnTimeout %v, got %v", setting.HTTPClient.IdleConnTimeout, transport.IdleConnTimeout)
	}

	if client1.Timeout != setting.HTTPClient.DefaultTimeout {
		t.Errorf("Expected Timeout %v, got %v", setting.HTTPClient.DefaultTimeout, client1.Timeout)
	}
}

func TestClientPoolConcurrentAccess(t *testing.T) {
	pool := GetGlobalClientPool()
	var wg sync.WaitGroup
	clients := make([]*http.Client, 100)

	// Test concurrent access to the same client key
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			clients[index] = pool.GetClient("concurrent_test")
		}(i)
	}

	wg.Wait()

	// All clients should be the same instance
	firstClient := clients[0]
	for i := 1; i < 100; i++ {
		if clients[i] != firstClient {
			t.Errorf("Expected all clients to be the same instance, but client[%d] is different", i)
		}
	}
}

func TestClientPoolDifferentKeys(t *testing.T) {
	pool := GetGlobalClientPool()

	// Test that different keys return different clients
	client1 := pool.GetClient("key1")
	client2 := pool.GetClient("key2")
	client3 := pool.GetClient("key3")

	if client1 == client2 || client1 == client3 || client2 == client3 {
		t.Fatal("Expected different keys to return different client instances")
	}
}

func TestGetClientWithTimeout(t *testing.T) {
	pool := GetGlobalClientPool()

	customTimeout := 15 * time.Second
	client := pool.GetClientWithTimeout("timeout_test", customTimeout)

	if client.Timeout != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, client.Timeout)
	}

	// Test that the base client is not affected
	baseClient := pool.GetClient("timeout_test")
	if baseClient.Timeout == customTimeout {
		t.Error("Expected base client timeout to be unchanged")
	}
}

func TestGetClientWithTLS(t *testing.T) {
	pool := GetGlobalClientPool()

	baseClient := pool.GetClient("tls_test")
	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	client := pool.GetClientWithTLS("tls_test", tlsConfig)

	if client == baseClient {
		t.Fatal("Expected different client instance with TLS config")
	}

	// Verify TLS config is applied
	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig != tlsConfig {
		t.Error("Expected TLS config to be applied to transport")
	}
}

func TestGetDefaultClient(t *testing.T) {
	client := GetDefaultClient()
	if client == nil {
		t.Fatal("Expected non-nil default client")
	}

	// Test that it's the same as getting from pool with "default" key
	poolClient := GetGlobalClientPool().GetClient("default")
	if client != poolClient {
		t.Error("Expected GetDefaultClient to return the same client as pool.GetClient(\"default\")")
	}
}

func TestGetWebhookClient(t *testing.T) {
	// Set webhook timeout for testing
	setting.Webhook.DeliverTimeout = 10

	client := GetWebhookClient()
	if client == nil {
		t.Fatal("Expected non-nil webhook client")
	}

	expectedTimeout := time.Duration(setting.Webhook.DeliverTimeout) * time.Second
	if client.Timeout != expectedTimeout {
		t.Errorf("Expected webhook timeout %v, got %v", expectedTimeout, client.Timeout)
	}
}

func TestGetLFSClient(t *testing.T) {
	client := GetLFSClient()
	if client == nil {
		t.Fatal("Expected non-nil LFS client")
	}
}

func TestGetMigrationClient(t *testing.T) {
	client := GetMigrationClient()
	if client == nil {
		t.Fatal("Expected non-nil migration client")
	}
}

func TestClientPoolClose(t *testing.T) {
	pool := GetGlobalClientPool()

	// Get some clients
	client1 := pool.GetClient("close_test1")
	client2 := pool.GetClient("close_test2")

	if client1 == nil || client2 == nil {
		t.Fatal("Expected non-nil clients")
	}

	// Close the pool
	pool.Close()

	// Verify clients are still accessible (they should be recreated)
	client3 := pool.GetClient("close_test1")
	client4 := pool.GetClient("close_test2")

	if client3 == nil || client4 == nil {
		t.Fatal("Expected clients to be recreated after pool close")
	}
}

func TestClientPoolTransportConfiguration(t *testing.T) {
	pool := GetGlobalClientPool()
	client := pool.GetClient("transport_test")

	transport := client.Transport.(*http.Transport)

	// Test that all expected transport settings are configured
	if transport.MaxIdleConns == 0 {
		t.Error("Expected MaxIdleConns to be configured")
	}

	if transport.MaxIdleConnsPerHost == 0 {
		t.Error("Expected MaxIdleConnsPerHost to be configured")
	}

	if transport.IdleConnTimeout == 0 {
		t.Error("Expected IdleConnTimeout to be configured")
	}

	if transport.TLSHandshakeTimeout == 0 {
		t.Error("Expected TLSHandshakeTimeout to be configured")
	}

	if transport.ExpectContinueTimeout == 0 {
		t.Error("Expected ExpectContinueTimeout to be configured")
	}

	if transport.DialContext == nil {
		t.Error("Expected DialContext to be configured")
	}

	if transport.Proxy == nil {
		t.Error("Expected Proxy to be configured")
	}
}

func TestClientPoolHTTP2Support(t *testing.T) {
	pool := GetGlobalClientPool()
	client := pool.GetClient("http2_test")

	transport := client.Transport.(*http.Transport)

	// Test HTTP/2 support
	if !transport.ForceAttemptHTTP2 {
		t.Error("Expected ForceAttemptHTTP2 to be enabled")
	}
}

func TestClientPoolKeepAliveSettings(t *testing.T) {
	pool := GetGlobalClientPool()
	client := pool.GetClient("keepalive_test")

	transport := client.Transport.(*http.Transport)

	// Test that keep-alive is enabled
	if transport.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be false")
	}
}

func TestClientPoolWithRealHTTPRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Get a client from the pool
	client := GetDefaultClient()

	// Make a real HTTP request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClientPoolMultipleRequests(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Get a client from the pool
	client := GetDefaultClient()

	// Make multiple requests to test connection reuse
	for i := 0; i < 10; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Failed to make HTTP request %d: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}
}

func TestClientPoolTimeoutBehavior(t *testing.T) {
	// Create a slow test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Get a client with short timeout
	client := GetGlobalClientPool().GetClientWithTimeout("timeout_test", 1*time.Second)

	// Make a request that should timeout
	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("Expected request to timeout")
	}
}

func TestClientPoolTLSConfiguration(t *testing.T) {
	// Create a test server with TLS
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Get a client with TLS config that skips verification
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := GetGlobalClientPool().GetClientWithTLS("tls_test", tlsConfig)

	// Make a request to the TLS server
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make TLS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClientPoolSingletonBehavior(t *testing.T) {
	// Test that GetGlobalClientPool always returns the same instance
	pool1 := GetGlobalClientPool()
	pool2 := GetGlobalClientPool()

	if pool1 != pool2 {
		t.Fatal("Expected GetGlobalClientPool to return the same instance")
	}
}

func TestClientPoolEmptyKey(t *testing.T) {
	pool := GetGlobalClientPool()

	// Test with empty key
	client := pool.GetClient("")
	if client == nil {
		t.Fatal("Expected non-nil client for empty key")
	}
}

func TestClientPoolSpecialCharacters(t *testing.T) {
	pool := GetGlobalClientPool()

	// Test with special characters in key
	specialKeys := []string{
		"key with spaces",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key:with:colons",
		"key/with/slashes",
		"key\\with\\backslashes",
	}

	for _, key := range specialKeys {
		client := pool.GetClient(key)
		if client == nil {
			t.Errorf("Expected non-nil client for key: %s", key)
		}
	}
}
