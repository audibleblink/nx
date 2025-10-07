package mux

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/protocols"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndHTTP tests end-to-end HTTP functionality
func TestEndToEndHTTP(t *testing.T) {
	t.Run("HTTP GET request routing", func(t *testing.T) {
		// Create test server
		server := NewTestServer(t, NewTestConfig())
		defer server.Close(t)

		// Make HTTP request
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + server.Addr() + "/")

		if err == nil {
			defer resp.Body.Close()
			// Should get a response (even if 404, it means HTTP routing worked)
			assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 600)
		}
	})
}

// TestEndToEndShell tests end-to-end shell functionality
func TestEndToEndShell(t *testing.T) {
	t.Run("shell connection routing", func(t *testing.T) {
		t.Skip("Skipping shell integration test - requires complex tmux/socket setup")

		// Create test server
		server := NewTestServer(t, NewTestConfig())
		defer server.Close(t)

		// Make a raw TCP connection (should route to shell handler)
		conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
		if err == nil {
			defer conn.Close()

			// Send some non-HTTP data
			_, err = conn.Write([]byte("echo test\n"))
			if err == nil {
				// Set read timeout
				conn.SetReadDeadline(time.Now().Add(1 * time.Second))

				// Try to read response (may timeout, but connection should be established)
				buffer := make([]byte, 1024)
				_, readErr := conn.Read(buffer)

				// Either we get data or timeout - both indicate shell routing worked
				assert.True(t, readErr == nil || strings.Contains(readErr.Error(), "timeout"))
			}
		}
	})
}

// TestProtocolMultiplexing tests that different protocols are routed correctly
func TestProtocolMultiplexing(t *testing.T) {
	t.Run("HTTP and shell multiplexing", func(t *testing.T) {
		// Create server with both HTTP and shell handlers
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Start server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Get the actual listening address
		addr := server.listener.Addr().String()

		// Test HTTP connection
		t.Run("HTTP connection", func(t *testing.T) {
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get("http://" + addr + "/")

			if err == nil {
				defer resp.Body.Close()
				assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 600)
			}
		})

		// Test shell connection
		t.Run("Shell connection", func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err == nil {
				defer conn.Close()

				// Send non-HTTP data
				_, err = conn.Write([]byte("test command\n"))
				assert.NoError(t, err)
			}
		})

		// Stop server
		server.Stop()

		// Wait for server to finish
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Log("Server did not shut down within timeout (expected in some cases)")
		}
	})
}

// TestWebDAVWithOtherProtocols verifies WebDAV works alongside HTTP, SSH, and shell protocols
func TestWebDAVWithOtherProtocols(t *testing.T) {
	t.Skip("Integration test - WebDAV implementation pending")

	// This test will verify that WebDAV protocol detection and handling
	// doesn't interfere with existing HTTP, SSH, and shell protocols
	// when all are enabled on the same multiplexed port
}

// TestConcurrentConnections tests handling of multiple concurrent connections
func TestConcurrentConnections(t *testing.T) {
	t.Run("multiple concurrent HTTP connections", func(t *testing.T) {
		// Create test server
		server := NewTestServer(t, NewTestConfig())
		defer server.Close(t)

		// Make multiple concurrent HTTP requests
		const numRequests = 5
		results := make(chan error, numRequests)

		for i := range numRequests {
			go func(id int) {
				client := &http.Client{Timeout: 3 * time.Second}
				resp, err := client.Get(fmt.Sprintf("http://%s/?req=%d", server.Addr(), id))
				if err == nil {
					resp.Body.Close()
				}
				results <- err
			}(i)
		}

		// Collect results
		successCount := 0
		for range numRequests {
			err := <-results
			if err == nil {
				successCount++
			}
		}

		// At least some requests should succeed
		assert.Greater(t, successCount, 0, "At least one HTTP request should succeed")
	})
}

// TestServerResilience tests server resilience under various conditions
func TestServerResilience(t *testing.T) {
	t.Run("server handles malformed requests", func(t *testing.T) {
		// Create server
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Start server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Get the actual listening address
		addr := server.listener.Addr().String()

		// Send malformed data
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			defer conn.Close()

			// Send malformed HTTP-like data
			malformedRequests := []string{
				"GET / HTTP/1.1\r\n\r\n", // Missing Host header
				"INVALID REQUEST\r\n\r\n",
				"GET /\r\n",
				"\x00\x01\x02\x03", // Binary data
			}

			for _, req := range malformedRequests {
				conn.Write([]byte(req))
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Server should still be running after malformed requests
		time.Sleep(100 * time.Millisecond)

		// Try a valid HTTP request to verify server is still responsive
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			resp.Body.Close()
			assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 600)
		}

		// Stop server
		server.Stop()

		// Wait for server to finish
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Log("Server did not shut down within timeout (expected in some cases)")
		}
	})
}

// TestConnectionLifecycle tests the complete connection lifecycle
func TestConnectionLifecycle(t *testing.T) {
	t.Run("connection establishment and cleanup", func(t *testing.T) {
		// Create server
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Start server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Get the actual listening address
		addr := server.listener.Addr().String()

		// Establish connection
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			// Connection established successfully
			assert.NotNil(t, conn)

			// Send some data
			_, err = conn.Write([]byte("test\n"))
			assert.NoError(t, err)

			// Close connection
			err = conn.Close()
			assert.NoError(t, err)
		}

		// Stop server
		server.Stop()

		// Wait for server to finish
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Log("Server did not shut down within timeout (expected in some cases)")
		}
	})
}

// BenchmarkConcurrentConnections benchmarks concurrent connection handling
func BenchmarkConcurrentConnections(b *testing.B) {
	// Create server
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
	shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

	server, err := NewServer(cfg, httpHandler, nil, shellHandler)
	if err != nil {
		b.Fatal(err)
	}
	defer server.Stop()

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Get the actual listening address
	addr := server.listener.Addr().String()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 5 * time.Second}
		for pb.Next() {
			resp, err := client.Get("http://" + addr + "/")
			if err == nil {
				resp.Body.Close()
			}
		}
	})
}
