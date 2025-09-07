package protocols

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/pkg/socket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShellHandler(t *testing.T) {
	cfg := &config.Config{
		Iface: "127.0.0.1",
		Port:  "0",
	}

	socketMgr := socket.NewManager()
	defer socketMgr.Cleanup()

	handler := NewShellHandler(cfg, nil, socketMgr, nil, "test-conn")

	require.NotNil(t, handler)
	assert.Equal(t, cfg, handler.config)
	assert.Equal(t, socketMgr, handler.socketManager)
	assert.Equal(t, "test-conn", handler.connStr)
}

func TestShellHandlerValidation(t *testing.T) {
	cfg := &config.Config{}

	t.Run("validates required socket manager", func(t *testing.T) {
		handler := NewShellHandler(cfg, nil, nil, nil, "test")

		// Create a mock connection for testing validation
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		err := handler.Handle(server)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "socket manager not initialized")
	})

	t.Run("validates required tmux manager", func(t *testing.T) {
		socketMgr := socket.NewManager()
		defer socketMgr.Cleanup()

		handler := NewShellHandler(cfg, nil, socketMgr, nil, "test")

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		err := handler.Handle(server)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tmux manager not initialized")
	})
}

func TestShellHandlerSocketCreation(t *testing.T) {
	cfg := &config.Config{}
	socketMgr := socket.NewManager()
	defer socketMgr.Cleanup()

	handler := NewShellHandler(cfg, nil, socketMgr, nil, "test")

	t.Run("creates unix socket successfully", func(t *testing.T) {
		socketFile, unixListener, err := handler.createUnixSocket()

		require.NoError(t, err)
		require.NotEmpty(t, socketFile)
		require.NotNil(t, unixListener)
		defer unixListener.Close()

		// Verify socket file path
		assert.Contains(t, socketFile, ".sock")

		// Verify listener is functional
		assert.Equal(t, "unix", unixListener.Addr().Network())
		assert.Equal(t, socketFile, unixListener.Addr().String())
	})
}

func TestShellHandlerBridgeIntegration(t *testing.T) {
	t.Run("demonstrates bridge command integration concept", func(t *testing.T) {
		// This test shows how the shell handler integrates with the nx bridge command
		// without requiring actual tmux setup

		socketMgr := socket.NewManager()
		defer socketMgr.Cleanup()

		// Create a socket like the shell handler would
		socketPath, err := socketMgr.GenerateTempFilename()
		require.NoError(t, err)

		unixListener, err := socketMgr.CreateUnixListener(socketPath)
		require.NoError(t, err)
		defer unixListener.Close()

		// Simulate what happens when tmux runs "nx bridge <socketPath>"
		// The bridge command would connect to this socket and bridge stdio
		bridgeCmd := fmt.Sprintf("nx bridge %q", socketPath)

		// Verify the command format is correct
		assert.Contains(t, bridgeCmd, "nx bridge")
		assert.Contains(t, bridgeCmd, socketPath)
		assert.Contains(t, bridgeCmd, ".sock")

		t.Logf("Bridge command that would be executed in tmux: %s", bridgeCmd)
	})
}

func TestShellHandlerContextHandling(t *testing.T) {
	t.Run("respects context cancellation in bridge operations", func(t *testing.T) {
		socketMgr := socket.NewManager()
		defer socketMgr.Cleanup()

		// Create a socket
		socketPath, err := socketMgr.GenerateTempFilename()
		require.NoError(t, err)

		unixListener, err := socketMgr.CreateUnixListener(socketPath)
		require.NoError(t, err)
		defer unixListener.Close()

		// Create network connection pair
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		// Test context cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start bridge operation that should be cancelled
		err = socketMgr.BridgeConnections(ctx, server, unixListener)

		// Should return context error due to timeout
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}
