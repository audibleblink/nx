// Package bridge provides stdio-to-socket bridging functionality that replaces
// the previous socat dependency. It enables bidirectional communication between
// stdin/stdout and Unix domain sockets through managed sessions with proper
// resource cleanup and error handling. The actual PTY functionality is provided
// by tmux; this package only handles the stdio bridging.
package bridge

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

// Session represents an active stdio-to-socket bridge session
type Session struct {
	socketConn net.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	errChan    chan error
	closeOnce  sync.Once
}

// NewSession creates a new stdio bridge session connected to the specified Unix socket
func NewSession(socketPath string) (*Session, error) {
	socketConn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket %s: %w", socketPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	session := &Session{
		socketConn: socketConn,
		ctx:        ctx,
		cancel:     cancel,
		errChan:    make(chan error, 2),
	}

	return session, nil
}

// Start begins the bidirectional communication between stdin/stdout and the socket.
func (s *Session) Start() error {
	go s.copyStdinToSocket()
	go s.copySocketToStdout()

	select {
	case err := <-s.errChan:
		return err
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

// Close cleans up all resources associated with the session
func (s *Session) Close() error {
	if s.cancel != nil {
		s.cancel()
	}

	var closeErr error
	s.closeOnce.Do(func() {
		if s.socketConn != nil {
			closeErr = s.socketConn.Close()
		}
	})

	return closeErr
}

// copyStdinToSocket copies data from stdin to the socket connection.
// This enables input from the local terminal to be sent through the socket.
func (s *Session) copyStdinToSocket() {
	defer s.closeOnce.Do(func() {
		if s.socketConn != nil {
			s.socketConn.Close()
		}
	})
	_, err := io.Copy(s.socketConn, os.Stdin)
	s.errChan <- err
}

// copySocketToStdout copies data from the socket connection to stdout.
// This enables output from the socket to be displayed on the local terminal.
func (s *Session) copySocketToStdout() {
	defer s.closeOnce.Do(func() {
		if s.socketConn != nil {
			s.socketConn.Close()
		}
	})
	_, err := io.Copy(os.Stdout, s.socketConn)
	s.errChan <- err
}
