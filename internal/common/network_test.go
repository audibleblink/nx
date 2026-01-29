package common

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/audibleblink/logerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsShutdownError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "use of closed network connection",
			err:      errors.New("use of closed network connection"),
			expected: true,
		},
		{
			name:     "closed error",
			err:      errors.New("connection closed"),
			expected: true,
		},
		{
			name:     "server closed error",
			err:      errors.New("server closed"),
			expected: true,
		},
		{
			name:     "generic error - not shutdown",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "timeout error - not shutdown",
			err:      errors.New("connection timeout"),
			expected: false,
		},
		{
			name:     "EOF error - not shutdown",
			err:      errors.New("EOF"),
			expected: false,
		},
		{
			name:     "closed in middle of message",
			err:      errors.New("read: connection closed by peer"),
			expected: true,
		},
		{
			name:     "wrapped closed error",
			err:      errors.New("listener error: use of closed network connection"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsShutdownError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// mockListener implements net.Listener for testing
type mockListener struct {
	acceptCh  chan net.Conn
	acceptErr error
	closed    bool
	closeCh   chan struct{}
	mu        sync.Mutex
}

func newMockListener() *mockListener {
	return &mockListener{
		acceptCh: make(chan net.Conn, 10),
		closeCh:  make(chan struct{}),
	}
}

func (m *mockListener) Accept() (net.Conn, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, errors.New("use of closed network connection")
	}
	err := m.acceptErr
	m.mu.Unlock()

	if err != nil {
		return nil, err
	}

	select {
	case conn := <-m.acceptCh:
		return conn, nil
	case <-m.closeCh:
		return nil, errors.New("use of closed network connection")
	}
}

func (m *mockListener) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

// testLogger creates a silent logger for testing
func testLogger() logerr.Logger {
	logger := logerr.DefaultLogger()
	logger.Output = io.Discard
	return *logger
}

func TestHandleListenerLoop(t *testing.T) {
	t.Run("handles connections and calls handler", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		handlerCalled := make(chan struct{}, 1)
		handler := func(conn net.Conn) error {
			handlerCalled <- struct{}{}
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Start listener loop in goroutine
		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "test")
		}()

		// Send a connection
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		listener.acceptCh <- server

		// Wait for handler to be called
		select {
		case <-handlerCalled:
			// Success
		case <-time.After(time.Second):
			t.Fatal("handler was not called")
		}

		// Cancel context and close listener to stop loop
		cancel()
		listener.Close() // Need to close listener to unblock Accept

		select {
		case err := <-done:
			// Should exit with either context error or shutdown error
			assert.True(t, err == context.Canceled || IsShutdownError(err))
		case <-time.After(time.Second):
			t.Fatal("loop did not exit after context cancellation")
		}
	})

	t.Run("exits on context cancellation", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		handler := func(conn net.Conn) error {
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "test-component")
		}()

		// Cancel immediately
		cancel()
		// Close listener to unblock Accept
		listener.Close()

		select {
		case err := <-done:
			// Should exit with either context error or shutdown error
			assert.True(t, err == context.Canceled || IsShutdownError(err))
		case <-time.After(time.Second):
			t.Fatal("loop did not exit after context cancellation")
		}
	})

	t.Run("exits on listener close (shutdown error)", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		handler := func(conn net.Conn) error {
			return nil
		}

		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "test-shutdown")
		}()

		// Close listener to simulate shutdown
		time.Sleep(50 * time.Millisecond)
		listener.Close()

		select {
		case err := <-done:
			assert.True(t, IsShutdownError(err))
		case <-time.After(time.Second):
			t.Fatal("loop did not exit after listener close")
		}
	})

	t.Run("continues after handler errors", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		handlerCallCount := 0
		var mu sync.Mutex
		handler := func(conn net.Conn) error {
			mu.Lock()
			handlerCallCount++
			mu.Unlock()
			return errors.New("handler error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "error-test")
		}()

		// Send multiple connections - loop should continue despite handler errors
		for i := 0; i < 3; i++ {
			server, client := net.Pipe()
			listener.acceptCh <- server
			server.Close()
			client.Close()
			time.Sleep(20 * time.Millisecond)
		}

		// Give time for handlers to run
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		count := handlerCallCount
		mu.Unlock()

		assert.Equal(t, 3, count, "all handlers should be called despite errors")

		// Cleanup
		cancel()
		listener.Close()
		<-done
	})

	t.Run("handles multiple connections concurrently", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		var handlerCount int
		var mu sync.Mutex
		handler := func(conn net.Conn) error {
			mu.Lock()
			handlerCount++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond) // Simulate work
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "concurrent")
		}()

		// Send multiple connections
		const numConns = 5
		for i := 0; i < numConns; i++ {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			listener.acceptCh <- server
		}

		// Wait for all handlers to be called
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		count := handlerCount
		mu.Unlock()

		assert.Equal(t, numConns, count, "all connections should be handled")

		// Cleanup
		cancel()
		listener.Close()
		<-done
	})
}

func TestHandleListenerLoopEdgeCases(t *testing.T) {
	t.Run("connection closed by defer after handler returns", func(t *testing.T) {
		listener := newMockListener()
		logger := testLogger()

		handler := func(conn net.Conn) error {
			// Handler doesn't need to close - loop does it via defer
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- HandleListenerLoop(ctx, listener, handler, logger, "close-test")
		}()

		// Send a connection
		server, client := net.Pipe()
		listener.acceptCh <- server

		// Wait for handler to complete
		time.Sleep(100 * time.Millisecond)

		// Verify client can detect connection close
		client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		buf := make([]byte, 1)
		_, err := client.Read(buf)
		require.Error(t, err) // Should error because server side is closed

		// Cleanup
		cancel()
		listener.Close()
		<-done
	})
}
