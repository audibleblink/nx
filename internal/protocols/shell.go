package protocols

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/audibleblink/logerr"

	"github.com/audibleblink/nx/internal/common"
	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/plugins"
	"github.com/audibleblink/nx/internal/tmux"
	"github.com/audibleblink/nx/pkg/socket"
)

// ShellHandler handles shell connections
type ShellHandler struct {
	config        *config.Config
	tmuxManager   *tmux.Manager
	socketManager *socket.Manager
	pluginManager *plugins.Manager
	connStr       string
	log           logerr.Logger
}

// NewShellHandler creates a new shell handler
func NewShellHandler(
	cfg *config.Config,
	tmuxMgr *tmux.Manager,
	socketMgr *socket.Manager,
	pluginMgr *plugins.Manager,
	connStr string,
) *ShellHandler {
	return &ShellHandler{
		config:        cfg,
		tmuxManager:   tmuxMgr,
		socketManager: socketMgr,
		pluginManager: pluginMgr,
		connStr:       connStr,
		log:           logerr.Add("shell"),
	}
}

// Match checks if the connection data matches shell protocol (fallback)
// NOTE: This method is not used in production - cmux handles protocol detection
// It exists only for backward compatibility with tests
func (h *ShellHandler) Match(data []byte) bool {
	// Accept everything else as shell input
	return true
}

// Match checks if the connection data matches shell protocol (fallback)
// Handle processes shell connections
func (h *ShellHandler) Handle(conn net.Conn) error {
	return h.handleShellConnection(conn)
}

// HandleListener handles incoming shell connections on a listener
func (h *ShellHandler) HandleListener(ctx context.Context, listener net.Listener) error {
	// Custom handler that includes debug logging
	handler := func(conn net.Conn) error {
		h.log.Debug("incoming:", conn.RemoteAddr().String())
		return h.handleShellConnection(conn)
	}

	return common.HandleListenerLoop(ctx, listener, handler, h.log, "Shell")
}

// handleShellConnection processes a single shell connection
func (h *ShellHandler) handleShellConnection(conn net.Conn) error {
	// Check if required managers are available (required for shell functionality)
	if h.socketManager == nil {
		return fmt.Errorf("socket manager not initialized - shell functionality disabled")
	}
	if h.tmuxManager == nil {
		return fmt.Errorf("tmux manager not initialized - shell functionality disabled")
	}

	// Generate unique socket filename
	socketFile, err := h.socketManager.GenerateTempFilename()
	if err != nil {
		return fmt.Errorf("failed to generate socket filename: %w", err)
	}

	// Create Unix domain socket
	unixListener, err := h.socketManager.CreateUnixListener(socketFile)
	if err != nil {
		return fmt.Errorf("failed to create unix listener: %w", err)
	}

	h.log.Debug("socket file created:", socketFile)

	// Bridge TCP connection to Unix socket in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to receive bridge completion
	bridgeDone := make(chan error, 1)

	go func() {
		err := h.socketManager.BridgeConnections(ctx, conn, unixListener)
		bridgeDone <- err
	}()

	// Create tmux window for the reverse shell
	window, err := h.tmuxManager.CreateWindow(socketFile)
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Execute socat command in tmux window
	// Intentional space prefix to keep shell history clean
	socatCmd := fmt.Sprintf(" socat -d -d stdio unix-connect:'%s'", socketFile)
	if err := h.tmuxManager.ExecuteInWindow(window, socatCmd); err != nil {
		return fmt.Errorf("failed to execute socat command: %w", err)
	}

	tmuxLoc := fmt.Sprintf("%s:%d.0", h.tmuxManager.GetSessionName(), window.Number)
	h.log.Infof("new shell on %s: %s", tmuxLoc, conn.RemoteAddr().String())

	// Set environment variables for convenience
	time.Sleep(h.config.Sleep)
	envCmd := fmt.Sprintf(
		" export ME=%s all_proxy=http://%s http_proxy=http://%s https_proxy=http://%s ; clear",
		h.connStr, h.connStr, h.connStr, h.connStr,
	)
	_ = h.tmuxManager.ExecuteInWindow(window, envCmd)

	// Handle plugin execution
	execPlugin := h.config.Exec
	if h.config.Auto {
		execPlugin = "auto" // backward compatibility
	}

	if execPlugin != "" {
		if err := h.pluginManager.Execute(execPlugin, window); err != nil {
			h.log.Error("plugin execution:", err)
		}
	}

	// Wait for bridge connection to complete
	err = <-bridgeDone
	if err != nil && err != context.Canceled {
		h.log.Warn("bridge connection error:", err)
	}

	return nil
}
