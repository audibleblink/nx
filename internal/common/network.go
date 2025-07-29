package common

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/audibleblink/logerr"
)

// IsShutdownError checks if an error is related to server shutdown
func IsShutdownError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "closed") ||
		strings.Contains(errStr, "server closed")
}

// HandleListenerLoop handles the common pattern of accepting connections in a loop
// with proper context cancellation and shutdown error handling
func HandleListenerLoop(
	ctx context.Context,
	listener net.Listener,
	handler func(net.Conn) error,
	log logerr.Logger,
	componentName string,
) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check for context cancellation or shutdown
			select {
			case <-ctx.Done():
				log.Info(componentName + " shutting down")
				return ctx.Err()
			default:
			}

			// Check if this is a shutdown-related error
			if IsShutdownError(err) {
				log.Info(componentName + " listener closed, shutting down")
				return err
			}

			log.Error(componentName+" listener accept:", err)
			// Add a small delay for other errors to prevent tight loops
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Handle connection in goroutine
		go func(conn net.Conn) {
			defer conn.Close()
			if err := handler(conn); err != nil {
				log.Error(componentName+" connection error:", err)
			}
		}(conn)
	}
}
