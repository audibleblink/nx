package socket

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/audibleblink/logerr"
)

// Manager handles Unix domain socket operations
type Manager struct {
	socketDir string
}

// NewManager creates a new socket manager
func NewManager() *Manager {
	socketDir := filepath.Join(xdg.RuntimeDir, "nx")
	// Ensure socket directory exists
	os.MkdirAll(socketDir, 0755)

	return &Manager{
		socketDir: socketDir,
	}
}

// GenerateTempFilename creates a unique socket filename
func (m *Manager) GenerateTempFilename() (string, error) {
	file, err := os.CreateTemp(m.socketDir, "*.sock")
	if err != nil {
		return "", fmt.Errorf("failed to create temp socket file: %w", err)
	}
	file.Close()
	os.Remove(file.Name())
	return file.Name(), nil
}

// CreateUnixListener creates a Unix domain socket listener
func (m *Manager) CreateUnixListener(socketPath string) (net.Listener, error) {
	// Remove existing socket file if it exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unix listener: %w", err)
	}

	return listener, nil
}

// BridgeConnections bridges a TCP connection to a Unix domain socket
func (m *Manager) BridgeConnections(ctx context.Context, tcpConn net.Conn, unixListener net.Listener) error {
	log := logerr.Add("BridgeConnections")
	defer unixListener.Close()

	// Accept connection from Unix socket
	unixConn, err := unixListener.Accept()
	if err != nil {
		log.Warn("socket connection:", err)
		return fmt.Errorf("failed to accept unix connection: %w", err)
	}
	defer unixConn.Close()

	// Create channels for error handling
	tcpToUnix := make(chan error, 1)
	unixToTcp := make(chan error, 1)

	// Copy data from TCP to Unix socket
	go func() {
		_, err := io.Copy(unixConn, tcpConn)
		tcpToUnix <- err
	}()

	// Copy data from Unix socket to TCP
	go func() {
		_, err := io.Copy(tcpConn, unixConn)
		unixToTcp <- err
	}()

	// Wait for either goroutine to finish or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-tcpToUnix:
		if err != nil {
			log.Warn("TCP to Unix copy error:", err)
		}
		return err
	case err = <-unixToTcp:
		if err != nil {
			log.Warn("Unix to TCP copy error:", err)
		}
		return err
	}
}

// Cleanup removes socket files and directories
func (m *Manager) Cleanup() error {
	return os.RemoveAll(m.socketDir)
}

// GetSocketDir returns the socket directory path
func (m *Manager) GetSocketDir() string {
	return m.socketDir
}
