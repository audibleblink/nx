package socket

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	require.NotNil(t, manager)

	// Check that socket directory is set
	socketDir := manager.GetSocketDir()
	assert.NotEmpty(t, socketDir)
	assert.Contains(t, socketDir, "nx")

	// Check that directory exists
	_, err := os.Stat(socketDir)
	assert.NoError(t, err)

	// Cleanup
	defer manager.Cleanup()
}

func TestManagerGenerateTempFilename(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	t.Run("generates unique filenames", func(t *testing.T) {
		filename1, err1 := manager.GenerateTempFilename()
		filename2, err2 := manager.GenerateTempFilename()

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, filename1, filename2)

		// Both should be in the socket directory
		assert.Contains(t, filename1, manager.GetSocketDir())
		assert.Contains(t, filename2, manager.GetSocketDir())

		// Both should have .sock extension
		assert.True(t, strings.HasSuffix(filename1, ".sock"))
		assert.True(t, strings.HasSuffix(filename2, ".sock"))

		// Files should not exist (they're removed after creation)
		_, err := os.Stat(filename1)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filename2)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("multiple calls generate different names", func(t *testing.T) {
		filenames := make(map[string]bool)
		for range 10 {
			filename, err := manager.GenerateTempFilename()
			require.NoError(t, err)
			assert.False(t, filenames[filename], "filename %s was generated twice", filename)
			filenames[filename] = true
		}
	})
}

func TestManagerCreateUnixListener(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	t.Run("creates unix listener successfully", func(t *testing.T) {
		socketPath, err := manager.GenerateTempFilename()
		require.NoError(t, err)

		listener, err := manager.CreateUnixListener(socketPath)
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		// Check that socket file exists
		_, err = os.Stat(socketPath)
		assert.NoError(t, err)

		// Check that we can get the address
		addr := listener.Addr()
		assert.Equal(t, "unix", addr.Network())
		assert.Equal(t, socketPath, addr.String())
	})

	t.Run("removes existing socket file", func(t *testing.T) {
		socketPath, err := manager.GenerateTempFilename()
		require.NoError(t, err)

		// Create a file at the socket path
		file, err := os.Create(socketPath)
		require.NoError(t, err)
		file.Close()

		// Verify file exists
		_, err = os.Stat(socketPath)
		require.NoError(t, err)

		// Create listener should succeed and remove the existing file
		listener, err := manager.CreateUnixListener(socketPath)
		require.NoError(t, err)
		defer listener.Close()

		// Socket should now exist as a socket, not a regular file
		info, err := os.Stat(socketPath)
		require.NoError(t, err)
		assert.Equal(t, os.ModeSocket, info.Mode()&os.ModeSocket)
	})

	t.Run("fails with invalid path", func(t *testing.T) {
		invalidPath := "/invalid/path/that/does/not/exist/socket.sock"
		listener, err := manager.CreateUnixListener(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, listener)
		assert.Contains(t, err.Error(), "failed to create unix listener")
	})
}

func TestManagerBridgeConnections(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	t.Run("bridges connections successfully", func(t *testing.T) {
		// Create a Unix socket listener
		socketPath, err := manager.GenerateTempFilename()
		require.NoError(t, err)

		unixListener, err := manager.CreateUnixListener(socketPath)
		require.NoError(t, err)

		// Create a pair of connected TCP connections to simulate client/server
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start bridging in a goroutine
		bridgeErr := make(chan error, 1)
		go func() {
			bridgeErr <- manager.BridgeConnections(ctx, server, unixListener)
		}()

		// Connect to the Unix socket
		unixConn, err := net.Dial("unix", socketPath)
		require.NoError(t, err)
		defer unixConn.Close()

		// Test data flow: client -> server -> unix socket
		testData := "Hello, Unix socket!"
		go func() {
			client.Write([]byte(testData))
			client.Close()
		}()

		// Read from Unix socket
		buffer := make([]byte, len(testData))
		n, err := unixConn.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, testData, string(buffer[:n]))

		// Wait for bridge to complete
		select {
		case err := <-bridgeErr:
			// Bridge should complete when connections close
			assert.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("Bridge operation timed out")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		socketPath, err := manager.GenerateTempFilename()
		require.NoError(t, err)

		unixListener, err := manager.CreateUnixListener(socketPath)
		require.NoError(t, err)

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		// Create context that we'll cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Start bridging
		bridgeErr := make(chan error, 1)
		go func() {
			bridgeErr <- manager.BridgeConnections(ctx, server, unixListener)
		}()

		// Connect to Unix socket
		unixConn, err := net.Dial("unix", socketPath)
		require.NoError(t, err)
		defer unixConn.Close()

		// Cancel context
		cancel()

		// Bridge should return context error
		select {
		case err := <-bridgeErr:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(2 * time.Second):
			t.Fatal("Bridge did not respond to context cancellation")
		}
	})

	t.Run("handles unix listener accept failure", func(t *testing.T) {
		socketPath, err := manager.GenerateTempFilename()
		require.NoError(t, err)

		unixListener, err := manager.CreateUnixListener(socketPath)
		require.NoError(t, err)

		// Close the listener immediately to cause accept to fail
		unixListener.Close()

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = manager.BridgeConnections(ctx, server, unixListener)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to accept unix connection")
	})
}

func TestManagerCleanup(t *testing.T) {
	manager := NewManager()
	socketDir := manager.GetSocketDir()

	// Create some test files in the socket directory
	testFile := filepath.Join(socketDir, "test.sock")
	file, err := os.Create(testFile)
	require.NoError(t, err)
	file.Close()

	// Verify file exists
	_, err = os.Stat(testFile)
	require.NoError(t, err)

	// Cleanup should remove the entire directory
	err = manager.Cleanup()
	assert.NoError(t, err)

	// Directory should no longer exist
	_, err = os.Stat(socketDir)
	assert.True(t, os.IsNotExist(err))
}

func TestManagerGetSocketDir(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	socketDir := manager.GetSocketDir()
	assert.NotEmpty(t, socketDir)
	assert.Contains(t, socketDir, "nx")

	// Directory should exist
	info, err := os.Stat(socketDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestBridgeConnectionsBidirectional tests that data flows in both directions
func TestBridgeConnectionsBidirectional(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	socketPath, err := manager.GenerateTempFilename()
	require.NoError(t, err)

	unixListener, err := manager.CreateUnixListener(socketPath)
	require.NoError(t, err)

	// Create TCP connection pair
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start bridging
	bridgeErr := make(chan error, 1)
	go func() {
		bridgeErr <- manager.BridgeConnections(ctx, server, unixListener)
	}()

	// Connect to Unix socket
	unixConn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer unixConn.Close()

	// Test bidirectional communication
	testData1 := "Client to Unix"
	testData2 := "Unix to Client"

	// Send from client to unix
	go func() {
		client.Write([]byte(testData1))
	}()

	// Read at unix socket
	buffer1 := make([]byte, len(testData1))
	n, err := unixConn.Read(buffer1)
	require.NoError(t, err)
	assert.Equal(t, testData1, string(buffer1[:n]))

	// Send from unix to client
	go func() {
		unixConn.Write([]byte(testData2))
	}()

	// Read at client
	buffer2 := make([]byte, len(testData2))
	n, err = client.Read(buffer2)
	require.NoError(t, err)
	assert.Equal(t, testData2, string(buffer2[:n]))

	// Close connections to end bridge
	client.Close()
	unixConn.Close()

	// Wait for bridge to complete
	select {
	case <-bridgeErr:
		// Bridge completed
	case <-time.After(2 * time.Second):
		t.Fatal("Bridge operation timed out")
	}
}

// TestConcurrentSocketOperations tests that multiple socket operations can run concurrently
func TestConcurrentSocketOperations(t *testing.T) {
	manager := NewManager()
	defer manager.Cleanup()

	const numOperations = 5
	results := make(chan error, numOperations)

	// Run multiple socket operations concurrently
	for range numOperations {
		go func() {
			socketPath, err := manager.GenerateTempFilename()
			if err != nil {
				results <- err
				return
			}

			listener, err := manager.CreateUnixListener(socketPath)
			if err != nil {
				results <- err
				return
			}
			listener.Close()
			results <- nil
		}()
	}

	// Wait for all operations to complete
	for range numOperations {
		select {
		case err := <-results:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}
