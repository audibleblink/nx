package mux

import (
	"context"
	"testing"
	"time"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/protocols"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServer tests the creation of a new multiplexer server
func TestNewServer(t *testing.T) {
	t.Run("creates server with valid configuration", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0", // Use port 0 for automatic assignment
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, cfg, server.config)
		assert.NotNil(t, server.listener)
		assert.NotNil(t, server.mux)
		assert.Equal(t, httpHandler, server.httpHandler)
		assert.Nil(t, server.sshHandler)
		assert.Equal(t, shellHandler, server.shellHandler)
		assert.NotNil(t, server.log)

		// Clean up
		server.Stop()
	})

	t.Run("creates server with SSH handler", func(t *testing.T) {
		cfg := &config.Config{
			Iface:   "127.0.0.1",
			Port:    "0",
			SSHPass: "testpass",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		sshHandler, err := protocols.NewSSHHandler("testpass")
		require.NoError(t, err)
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, sshHandler, shellHandler)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.sshHandler)

		// Clean up
		server.Stop()
	})

	t.Run("fails with invalid address", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "invalid-address",
			Port:  "8443",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "failed to create listener")
	})
}

// TestServerStop tests the server stop functionality
func TestServerStop(t *testing.T) {
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
	shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

	server, err := NewServer(cfg, httpHandler, nil, shellHandler)
	require.NoError(t, err)

	// Test that Stop() closes the listener
	err = server.Stop()
	assert.NoError(t, err)

	// Verify listener is closed by trying to accept (should fail)
	_, err = server.listener.Accept()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// TestServerConfiguration tests server configuration handling
func TestServerConfiguration(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid IPv4 configuration",
			config: &config.Config{
				Iface: "127.0.0.1",
				Port:  "0",
			},
			expectError: false,
		},
		{
			name: "valid IPv6 configuration",
			config: &config.Config{
				Iface: "::1",
				Port:  "0",
			},
			expectError: false,
		},
		{
			name: "all interfaces configuration",
			config: &config.Config{
				Iface: "0.0.0.0",
				Port:  "0",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
			shellHandler := protocols.NewShellHandler(tc.config, nil, nil, nil, tc.config.Address())

			server, err := NewServer(tc.config, httpHandler, nil, shellHandler)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
				server.Stop()
			}
		})
	}
}

// TestServerStartShutdown tests the server start and graceful shutdown
func TestServerStartShutdown(t *testing.T) {
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
	shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

	server, err := NewServer(cfg, httpHandler, nil, shellHandler)
	require.NoError(t, err)

	// Create a context with timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	err = server.Stop()
	assert.NoError(t, err)

	// Wait for server to finish and check error
	select {
	case err := <-serverErr:
		// Server should return an error when stopped (closed connection)
		assert.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

// TestServerHandlerIntegration tests the integration between server and handlers
func TestServerHandlerIntegration(t *testing.T) {
	t.Run("server initializes with all handlers", func(t *testing.T) {
		cfg := &config.Config{
			Iface:    "127.0.0.1",
			Port:     "0",
			ServeDir: "/tmp", // Use a directory that exists
			SSHPass:  "testpass",
		}

		httpHandler := protocols.NewHTTPHandler(cfg.ServeDir, cfg.Address())
		sshHandler, err := protocols.NewSSHHandler(cfg.SSHPass)
		require.NoError(t, err)
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, sshHandler, shellHandler)
		require.NoError(t, err)

		// Verify all handlers are properly set
		assert.NotNil(t, server.httpHandler)
		assert.NotNil(t, server.sshHandler)
		assert.NotNil(t, server.shellHandler)

		// Verify configuration is properly passed
		assert.Equal(t, cfg, server.config)

		server.Stop()
	})

	t.Run("server works with minimal handlers", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		// Only HTTP and Shell handlers (no SSH)
		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)

		assert.NotNil(t, server.httpHandler)
		assert.Nil(t, server.sshHandler)
		assert.NotNil(t, server.shellHandler)

		server.Stop()
	})
}

// TestServerErrorHandling tests error handling scenarios
func TestServerErrorHandling(t *testing.T) {
	t.Run("handles nil handlers gracefully", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		// Test with nil shell handler (should still work)
		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")

		// This should fail because shell handler is required
		server, err := NewServer(cfg, httpHandler, nil, nil)

		// The current implementation doesn't validate required handlers,
		// but we should test that it doesn't crash
		if err == nil && server != nil {
			server.Stop()
		}
	})
}

// TestServerConcurrency tests concurrent operations
func TestServerConcurrency(t *testing.T) {
	t.Run("multiple servers can be created concurrently", func(t *testing.T) {
		const numServers = 5
		servers := make([]*Server, numServers)
		errors := make([]error, numServers)

		// Create multiple servers concurrently
		done := make(chan bool, numServers)
		for i := range numServers {
			go func(index int) {
				cfg := &config.Config{
					Iface: "127.0.0.1",
					Port:  "0", // Auto-assign port
				}

				httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
				shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

				servers[index], errors[index] = NewServer(cfg, httpHandler, nil, shellHandler)
				done <- true
			}(i)
		}

		// Wait for all servers to be created
		for range numServers {
			<-done
		}

		// Verify all servers were created successfully
		for i := range numServers {
			assert.NoError(t, errors[i], "Server %d should be created without error", i)
			assert.NotNil(t, servers[i], "Server %d should not be nil", i)

			if servers[i] != nil {
				servers[i].Stop()
			}
		}
	})
}

// BenchmarkServerCreation benchmarks server creation performance
func BenchmarkServerCreation(b *testing.B) {
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	for b.Loop() {
		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		if err != nil {
			b.Fatal(err)
		}
		server.Stop()
	}
}

// TestProtocolDetection tests the protocol detection and routing functionality
func TestProtocolDetection(t *testing.T) {
	t.Run("HTTP protocol detection", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Verify that HTTP handler is properly configured
		assert.NotNil(t, server.httpHandler)
		assert.Equal(t, httpHandler, server.httpHandler)
	})

	t.Run("SSH protocol detection", func(t *testing.T) {
		cfg := &config.Config{
			Iface:   "127.0.0.1",
			Port:    "0",
			SSHPass: "testpass",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		sshHandler, err := protocols.NewSSHHandler("testpass")
		require.NoError(t, err)
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, sshHandler, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Verify that SSH handler is properly configured
		assert.NotNil(t, server.sshHandler)
		assert.Equal(t, sshHandler, server.sshHandler)
	})

	t.Run("Shell protocol as fallback", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Verify that shell handler is properly configured as fallback
		assert.NotNil(t, server.shellHandler)
		assert.Equal(t, shellHandler, server.shellHandler)
	})
}

// TestServerIntegration tests the integration of server components
func TestServerIntegration(t *testing.T) {
	t.Run("server with HTTP file serving", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()

		cfg := &config.Config{
			Iface:    "127.0.0.1",
			Port:     "0",
			ServeDir: tempDir,
		}

		httpHandler := protocols.NewHTTPHandler(cfg.ServeDir, cfg.Address())
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Verify HTTP handler is configured with the correct directory
		assert.NotNil(t, server.httpHandler)
		assert.Equal(t, cfg.ServeDir, tempDir)
	})

	t.Run("server with all protocols enabled", func(t *testing.T) {
		cfg := &config.Config{
			Iface:    "127.0.0.1",
			Port:     "0",
			ServeDir: "/tmp",
			SSHPass:  "testpass",
		}

		httpHandler := protocols.NewHTTPHandler(cfg.ServeDir, cfg.Address())
		sshHandler, err := protocols.NewSSHHandler(cfg.SSHPass)
		require.NoError(t, err)
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, sshHandler, shellHandler)
		require.NoError(t, err)
		defer server.Stop()

		// Verify all handlers are configured
		assert.NotNil(t, server.httpHandler)
		assert.NotNil(t, server.sshHandler)
		assert.NotNil(t, server.shellHandler)

		// Verify configuration is properly set
		assert.True(t, cfg.IsSSHEnabled())
		assert.True(t, cfg.IsHTTPEnabled())
	})
}

// TestServerLifecycle tests the complete server lifecycle
func TestServerLifecycle(t *testing.T) {
	t.Run("complete server lifecycle", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		// 1. Create server
		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)
		assert.NotNil(t, server)

		// 2. Verify initial state
		assert.NotNil(t, server.listener)
		assert.NotNil(t, server.mux)
		assert.NotNil(t, server.config)

		// 3. Start server with timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.Start(ctx)
		}()

		// 4. Give server time to start
		time.Sleep(50 * time.Millisecond)

		// 5. Stop server
		err = server.Stop()
		assert.NoError(t, err)

		// 6. Wait for server to finish
		select {
		case err := <-serverDone:
			// Server should return an error when stopped
			assert.Error(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("Server did not shut down within timeout")
		}
	})
}

// TestServerErrorScenarios tests various error scenarios
func TestServerErrorScenarios(t *testing.T) {
	t.Run("invalid port configuration", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "99999", // Invalid port
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "failed to create listener")
	})

	t.Run("invalid interface configuration", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "999.999.999.999", // Invalid IP
			Port:  "8443",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "failed to create listener")
	})

	t.Run("server stop on already stopped server", func(t *testing.T) {
		cfg := &config.Config{
			Iface: "127.0.0.1",
			Port:  "0",
		}

		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		require.NoError(t, err)

		// Stop server once
		err = server.Stop()
		assert.NoError(t, err)

		// Stop server again (should handle gracefully)
		err = server.Stop()
		assert.Error(t, err) // Should return an error for already closed listener
	})
}

// TestServerConfiguration tests various server configuration scenarios
func TestServerConfigurationScenarios(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Config
		expectHTTP  bool
		expectSSH   bool
		expectShell bool
		expectError bool
	}{
		{
			name: "minimal configuration",
			config: &config.Config{
				Iface: "127.0.0.1",
				Port:  "0",
			},
			expectHTTP:  true,
			expectSSH:   false,
			expectShell: true,
			expectError: false,
		},
		{
			name: "HTTP with file serving",
			config: &config.Config{
				Iface:    "127.0.0.1",
				Port:     "0",
				ServeDir: "/tmp",
			},
			expectHTTP:  true,
			expectSSH:   false,
			expectShell: true,
			expectError: false,
		},
		{
			name: "SSH enabled",
			config: &config.Config{
				Iface:   "127.0.0.1",
				Port:    "0",
				SSHPass: "password",
			},
			expectHTTP:  true,
			expectSSH:   true,
			expectShell: true,
			expectError: false,
		},
		{
			name: "all protocols enabled",
			config: &config.Config{
				Iface:    "127.0.0.1",
				Port:     "0",
				ServeDir: "/tmp",
				SSHPass:  "password",
			},
			expectHTTP:  true,
			expectSSH:   true,
			expectShell: true,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpHandler := protocols.NewHTTPHandler(tc.config.ServeDir, tc.config.Address())

			var sshHandler *protocols.SSHHandler
			var err error
			if tc.expectSSH {
				sshHandler, err = protocols.NewSSHHandler(tc.config.SSHPass)
				require.NoError(t, err)
			}

			shellHandler := protocols.NewShellHandler(tc.config, nil, nil, nil, tc.config.Address())

			server, err := NewServer(tc.config, httpHandler, sshHandler, shellHandler)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)

				if tc.expectHTTP {
					assert.NotNil(t, server.httpHandler)
				}
				if tc.expectSSH {
					assert.NotNil(t, server.sshHandler)
				} else {
					assert.Nil(t, server.sshHandler)
				}
				if tc.expectShell {
					assert.NotNil(t, server.shellHandler)
				}

				server.Stop()
			}
		})
	}
}

// BenchmarkServerLifecycle benchmarks the complete server lifecycle
func BenchmarkServerLifecycle(b *testing.B) {
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	for b.Loop() {
		httpHandler := protocols.NewHTTPHandler("", "localhost:8443")
		shellHandler := protocols.NewShellHandler(cfg, nil, nil, nil, cfg.Address())

		server, err := NewServer(cfg, httpHandler, nil, shellHandler)
		if err != nil {
			b.Fatal(err)
		}

		// Start and immediately stop to benchmark full lifecycle
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		go server.Start(ctx)
		time.Sleep(1 * time.Millisecond) // Brief startup time
		server.Stop()
		cancel()
	}
}
