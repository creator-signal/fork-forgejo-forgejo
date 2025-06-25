// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"forgejo.org/modules/proxy"
	"forgejo.org/modules/setting"
)

func BenchmarkClientPoolGetClient(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := pool.GetClient("benchmark_test")
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkClientPoolConcurrentAccess(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("benchmark_key_%d", i%10)
			client := pool.GetClient(key)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
			i++
		}
	})
}

func BenchmarkClientPoolWithHTTPRequests(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()
	client := pool.GetClient("http_benchmark")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatalf("Failed to make HTTP request: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkClientPoolWithTimeout(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := pool.GetClientWithTimeout("timeout_benchmark", 5*time.Second)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkClientPoolWithTLS(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client := pool.GetClientWithTLS("tls_benchmark", tlsConfig)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkTraditionalHTTPClient(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := &http.Client{
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     60 * time.Second,
				},
				Timeout: 30 * time.Second,
			}
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkClientPoolMemoryUsage(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	// Pre-create clients to test memory usage
	clients := make([]*http.Client, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			clients[j] = pool.GetClient(fmt.Sprintf("memory_test_%d", j))
		}
	}
}

func BenchmarkClientPoolConnectionReuse(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()
	client := pool.GetClient("connection_reuse_benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Failed to make HTTP request: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
	}
}

func BenchmarkClientPoolMixedOperations(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				client := pool.GetClient(fmt.Sprintf("mixed_test_%d", i))
				if client == nil {
					b.Fatal("Expected non-nil client")
				}
			case 1:
				client := pool.GetClientWithTimeout(fmt.Sprintf("mixed_timeout_%d", i), 5*time.Second)
				if client == nil {
					b.Fatal("Expected non-nil client")
				}
			case 2:
				tlsConfig := &tls.Config{InsecureSkipVerify: true}
				client := pool.GetClientWithTLS(fmt.Sprintf("mixed_tls_%d", i), tlsConfig)
				if client == nil {
					b.Fatal("Expected non-nil client")
				}
			case 3:
				client := GetDefaultClient()
				if client == nil {
					b.Fatal("Expected non-nil client")
				}
			}
			i++
		}
	})
}

func BenchmarkClientPoolStressTest(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	// Create multiple goroutines to stress test the pool
	var wg sync.WaitGroup
	numGoroutines := 100
	clientsPerGoroutine := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(numGoroutines)
		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < clientsPerGoroutine; j++ {
					key := fmt.Sprintf("stress_test_%d_%d", goroutineID, j)
					client := pool.GetClient(key)
					if client == nil {
						b.Errorf("Expected non-nil client for key: %s", key)
					}
				}
			}(g)
		}
		wg.Wait()
	}
}

func BenchmarkClientPoolCloseAndRecreate(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get some clients
		client1 := pool.GetClient("close_test1")
		client2 := pool.GetClient("close_test2")

		if client1 == nil || client2 == nil {
			b.Fatal("Expected non-nil clients")
		}

		// Close the pool
		pool.Close()

		// Get clients again (should be recreated)
		client3 := pool.GetClient("close_test1")
		client4 := pool.GetClient("close_test2")

		if client3 == nil || client4 == nil {
			b.Fatal("Expected non-nil clients after close")
		}
	}
}

// Direct comparison benchmarks - Old vs New behavior

func BenchmarkOldWayCreateClient(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Old way: Create new client every time
			client := &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
				},
				Timeout: 30 * time.Second,
			}
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkNewWayGetClient(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// New way: Get client from pool
			client := pool.GetClient("benchmark_comparison")
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkOldWayMultipleRequests(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Old way: Create new client for each request
			client := &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
				},
				Timeout: 30 * time.Second,
			}

			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatalf("Failed to make HTTP request: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkNewWayMultipleRequests(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()
	client := pool.GetClient("multiple_requests_comparison")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// New way: Reuse client from pool
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatalf("Failed to make HTTP request: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkOldWayConcurrentClients(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Old way: Create different clients for different services
			key := fmt.Sprintf("service_%d", i%10)
			client := &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
				},
				Timeout: 30 * time.Second,
			}
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
			_ = key // Use key to simulate different service types
			i++
		}
	})
}

func BenchmarkNewWayConcurrentClients(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// New way: Get different clients from pool
			key := fmt.Sprintf("service_%d", i%10)
			client := pool.GetClient(key)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
			i++
		}
	})
}

func BenchmarkOldWayWithTimeout(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Old way: Create client with custom timeout
			client := &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
				},
				Timeout: 5 * time.Second, // Custom timeout
			}
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkNewWayWithTimeout(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// New way: Get client with custom timeout
			client := pool.GetClientWithTimeout("timeout_comparison", 5*time.Second)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkOldWayWithTLS(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Old way: Create client with TLS config
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client := &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
					TLSClientConfig:       tlsConfig,
				},
				Timeout: 30 * time.Second,
			}
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkNewWayWithTLS(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// New way: Get client with TLS config
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client := pool.GetClientWithTLS("tls_comparison", tlsConfig)
			if client == nil {
				b.Fatal("Expected non-nil client")
			}
		}
	})
}

func BenchmarkOldWayMemoryUsage(b *testing.B) {
	// Pre-create clients to test memory usage
	clients := make([]*http.Client, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			// Old way: Create new client for each key
			clients[j] = &http.Client{
				Transport: &http.Transport{
					Proxy:                 proxy.Proxy(),
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   10,
					IdleConnTimeout:       60 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
				},
				Timeout: 30 * time.Second,
			}
		}
	}
}

func BenchmarkNewWayMemoryUsage(b *testing.B) {
	// Initialize settings for benchmarking
	setting.HTTPClient.MaxIdleConns = 100
	setting.HTTPClient.MaxIdleConnsPerHost = 10
	setting.HTTPClient.IdleConnTimeout = 60 * time.Second
	setting.HTTPClient.DefaultTimeout = 30 * time.Second

	pool := GetGlobalClientPool()

	// Pre-create clients to test memory usage
	clients := make([]*http.Client, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			// New way: Get client from pool (shared instances)
			clients[j] = pool.GetClient(fmt.Sprintf("memory_test_%d", j))
		}
	}
}
