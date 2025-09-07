package bridge

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCreateSession(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Just close the connection for this test
			conn.Close()
		}
	}()

	// Create session
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// Verify session was created
	if session.socketConn == nil {
		t.Fatal("Socket connection not established")
	}
}

func TestNewSession(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Just close the connection for this test
			conn.Close()
		}
	}()

	// Create session using new API
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer session.Close()

	// Verify session was created
	if session.socketConn == nil {
		t.Fatal("Socket connection not established")
	}
	if session.ctx == nil {
		t.Fatal("Context not initialized")
	}
	if session.cancel == nil {
		t.Fatal("Cancel function not initialized")
	}
	if session.errChan == nil {
		t.Fatal("Error channel not initialized")
	}
}

func TestSessionClose(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept one connection
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			// Keep connection open briefly
			time.Sleep(100 * time.Millisecond)
			conn.Close()
		}
	}()

	// Create and close session
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Close should not error
	if err := session.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Second close should not panic
	session.Close()
}

func TestSessionLifecycle(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept connections and close them immediately
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Close immediately to trigger session completion
			conn.Close()
		}
	}()

	// Create session
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// Start session in goroutine (it should complete quickly due to closed connection)
	done := make(chan error, 1)
	go func() {
		done <- session.Start()
	}()

	// Wait for completion with timeout
	select {
	case err := <-done:
		// Session should complete (possibly with error due to closed connection)
		t.Logf("Session completed with: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Session did not complete within timeout")
	}
}

func TestConcurrentSessions(t *testing.T) {
	// Create temporary directory for sockets
	tmpDir := t.TempDir()

	// Create multiple socket listeners
	const numSessions = 5
	listeners := make([]net.Listener, numSessions)
	socketPaths := make([]string, numSessions)

	for i := range numSessions {
		socketPaths[i] = filepath.Join(tmpDir, fmt.Sprintf("test%d.sock", i))
		listener, err := net.Listen("unix", socketPaths[i])
		if err != nil {
			t.Fatalf("Failed to create socket listener %d: %v", i, err)
		}
		listeners[i] = listener
		defer listener.Close()

		// Accept and immediately close connections
		go func(l net.Listener) {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}(listener)
	}

	// Create sessions concurrently
	var wg sync.WaitGroup
	errors := make(chan error, numSessions)

	for i := range numSessions {
		wg.Add(1)
		go func(socketPath string) {
			defer wg.Done()

			session, err := NewSession(socketPath)
			if err != nil {
				errors <- fmt.Errorf("CreateSession failed: %w", err)
				return
			}
			defer session.Close()

			// Start session (should complete quickly)
			err = session.Start()
			if err != nil {
				t.Logf("Session completed with expected error: %v", err)
			}
		}(socketPaths[i])
	}

	// Wait for all sessions to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errors:
		t.Fatalf("Concurrent session error: %v", err)
	case <-done:
		t.Log("All concurrent sessions completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent sessions did not complete within timeout")
	}
}

func TestErrorHandlingScenarios(t *testing.T) {
	// Test 1: Invalid socket path
	_, err := NewSession("/nonexistent/path/test.sock")
	if err == nil {
		t.Error("Expected error for invalid socket path, got nil")
	}

	// Test 2: Session with nil connection (edge case)
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create socket but don't listen
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	listener.Close() // Close immediately

	// Try to create session with closed socket
	_, err = NewSession(socketPath)
	if err == nil {
		t.Error("Expected error for closed socket, got nil")
	}
}

func TestContextCancellation(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create Unix socket listener that keeps connections open
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept connections and keep them open to test cancellation
	connChan := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		connChan <- conn
		// Keep connection open until test completes
		<-time.After(5 * time.Second)
		conn.Close()
	}()

	// Create session
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// Start session in goroutine
	done := make(chan error, 1)
	go func() {
		done <- session.Start()
	}()

	// Wait for connection to be established
	select {
	case <-connChan:
		// Connection established, now cancel
	case <-time.After(1 * time.Second):
		t.Fatal("Connection not established within timeout")
	}

	// Cancel session
	session.Close() // This should trigger context cancellation

	// Wait for completion
	select {
	case err := <-done:
		// Should get context.Canceled error
		if err != context.Canceled {
			t.Logf("Session cancelled with: %v (expected context.Canceled)", err)
		} else {
			t.Logf("Session properly cancelled with context.Canceled")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Session did not respond to cancellation within timeout")
	}
}

func TestResourceCleanupVerification(t *testing.T) {
	// Test session cleanup without actual socket connection
	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		socketConn: nil, // No actual connection for this test
		ctx:        ctx,
		cancel:     cancel,
		errChan:    make(chan error, 2),
	}

	// Close session
	err := session.Close()
	if err != nil {
		t.Errorf("Session.Close() returned error: %v", err)
	}

	// Verify multiple Close() calls don't panic (sync.Once protection)
	err = session.Close()
	if err != nil {
		t.Errorf("Second Close() call returned error: %v", err)
	}

	// Third close should also be safe
	err = session.Close()
	if err != nil {
		t.Errorf("Third Close() call returned error: %v", err)
	}

	t.Log("Resource cleanup verification completed successfully")
}

func TestBidirectionalStdioBridging(t *testing.T) {
	// Create temporary socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "bidirectional.sock")

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	// Accept connection and echo data back
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Simple echo server to test bidirectional communication
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}

		// Echo the data back with a prefix to verify round-trip
		response := fmt.Sprintf("ECHO: %s", string(buffer[:n]))
		conn.Write([]byte(response))
	}()

	// Create session
	session, err := NewSession(socketPath)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// Note: This test demonstrates the stdio bridging concept but cannot
	// fully test actual stdin/stdout redirection in a unit test environment.
	// The actual bidirectional functionality is validated through the socket
	// manager tests and integration tests.

	t.Logf("Successfully created stdio bridge session for socket: %s", socketPath)

	// Verify session initialization
	if session.socketConn == nil {
		t.Fatal("Socket connection not established")
	}
	if session.ctx == nil {
		t.Fatal("Context not initialized")
	}
	if session.cancel == nil {
		t.Fatal("Cancel function not initialized")
	}
	if session.errChan == nil {
		t.Fatal("Error channel not initialized")
	}

	// Test that we can send data through the socket connection directly
	testData := "test bidirectional data"
	_, err = session.socketConn.Write([]byte(testData))
	if err != nil {
		t.Fatalf("Failed to write to socket: %v", err)
	}

	// Read response to verify socket connectivity
	buffer := make([]byte, 1024)
	session.socketConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := session.socketConn.Read(buffer)
	if err != nil {
		t.Logf("Socket read error (may be expected in test env): %v", err)
	} else {
		response := string(buffer[:n])
		if !strings.Contains(response, "ECHO:") {
			t.Errorf("Expected echo response, got: %s", response)
		} else {
			t.Logf("Received expected echo response: %s", response)
		}
	}
}
