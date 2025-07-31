package mux

import (
	"context"
	"testing"
	"time"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/protocols"
	"github.com/stretchr/testify/require"
)

// TestServer wraps a server instance with test utilities
type TestServer struct {
	server     *Server
	addr       string
	cancel     context.CancelFunc
	serverDone chan error
}

// NewTestServer creates a new test server with common configuration
func NewTestServer(t *testing.T, cfg *config.Config) *TestServer {
	t.Helper()

	// Use default config if none provided
	if cfg == nil {
		cfg = &config.Config{
			Iface: "127.0.0.1",
			Port:  "0", // Use random available port
		}
	}

	// Create handlers
	httpHandler := protocols.NewHTTPHandler(cfg.ServeDir, cfg.Address())
	shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

	// Create server
	server, err := NewServer(cfg, httpHandler, nil, shellHandler)
	require.NoError(t, err)

	// Start server with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	serverDone := make(chan error, 1)

	go func() {
		serverDone <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return &TestServer{
		server:     server,
		addr:       server.listener.Addr().String(),
		cancel:     cancel,
		serverDone: serverDone,
	}
}

// Addr returns the server's listening address
func (ts *TestServer) Addr() string {
	return ts.addr
}

// Close stops the test server and cleans up resources
func (ts *TestServer) Close(t *testing.T) {
	t.Helper()

	// Stop server
	ts.server.Stop()
	ts.cancel()

	// Wait for server to finish with timeout
	select {
	case <-ts.serverDone:
		// Server stopped cleanly
	case <-time.After(3 * time.Second):
		t.Log("Server did not shut down within timeout (expected in some cases)")
	}
}

// NewTestConfig creates a test configuration with common defaults
func NewTestConfig() *config.Config {
	return &config.Config{
		Iface: "127.0.0.1",
		Port:  "0", // Use random available port
	}
}

// NewTestConfigWithHTTP creates a test configuration with HTTP serving enabled
func NewTestConfigWithHTTP(serveDir string) *config.Config {
	cfg := NewTestConfig()
	cfg.ServeDir = serveDir
	return cfg
}
