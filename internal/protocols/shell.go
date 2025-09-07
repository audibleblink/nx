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
	"github.com/disneystreaming/gomux"
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

// Handle processes shell connections
func (h *ShellHandler) Handle(conn net.Conn) error {
	return h.handleShellConnection(conn)
}

// HandleListener handles incoming shell connections on a listener
func (h *ShellHandler) HandleListener(ctx context.Context, listener net.Listener) error {
	handler := func(conn net.Conn) error {
		h.log.Debug("incoming:", conn.RemoteAddr().String())
		return h.handleShellConnection(conn)
	}

	return common.HandleListenerLoop(ctx, listener, handler, h.log, "Shell")
}

// handleShellConnection processes a single shell connection
func (h *ShellHandler) handleShellConnection(conn net.Conn) error {
	if err := h.validateManagers(); err != nil {
		return err
	}

	socketFile, unixListener, err := h.createUnixSocket()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridgeDone := make(chan error, 1)
	go func() {
		bridgeDone <- h.socketManager.BridgeConnections(ctx, conn, unixListener)
	}()

	window, err := h.createTmuxWindow(socketFile, conn)
	if err != nil {
		return err
	}

	h.setupEnvironment(window)
	h.executePlugins(window)

	err = <-bridgeDone
	if err != nil && err != context.Canceled {
		h.log.Warn("bridge connection error:", err)
	}

	return nil
}

// validateManagers checks if required managers are available
func (h *ShellHandler) validateManagers() error {
	if h.socketManager == nil {
		return fmt.Errorf("socket manager not initialized - shell functionality disabled")
	}
	if h.tmuxManager == nil {
		return fmt.Errorf("tmux manager not initialized - shell functionality disabled")
	}
	return nil
}

// createUnixSocket creates a Unix domain socket for the shell connection
func (h *ShellHandler) createUnixSocket() (string, net.Listener, error) {
	socketFile, err := h.socketManager.GenerateTempFilename()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate socket filename: %w", err)
	}

	unixListener, err := h.socketManager.CreateUnixListener(socketFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create unix listener: %w", err)
	}

	h.log.Debug("socket file created:", socketFile)
	return socketFile, unixListener, nil
}

// createTmuxWindow creates a tmux window and executes the bridge command
func (h *ShellHandler) createTmuxWindow(socketFile string, conn net.Conn) (*gomux.Window, error) {
	window, err := h.tmuxManager.CreateWindow(socketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	ptyCmd := fmt.Sprintf(" nx bridge %q", socketFile)
	if err := h.tmuxManager.ExecuteInWindow(window, ptyCmd); err != nil {
		return nil, fmt.Errorf("failed to execute bridge command: %w", err)
	}

	tmuxLoc := fmt.Sprintf("%s:%d.0", h.tmuxManager.GetSessionName(), window.Number)
	h.log.Infof("new shell on %s: %s", tmuxLoc, conn.RemoteAddr().String())

	return window, nil
}

// setupEnvironment sets up environment variables for convenience
func (h *ShellHandler) setupEnvironment(window *gomux.Window) {
	time.Sleep(h.config.Sleep)
	envCmd := fmt.Sprintf(
		" export ME=%s all_proxy=http://%s http_proxy=http://%s https_proxy=http://%s ",
		h.connStr, h.connStr, h.connStr, h.connStr,
	)
	_ = h.tmuxManager.ExecuteInWindow(window, envCmd)
}

// executePlugins handles plugin execution
func (h *ShellHandler) executePlugins(window *gomux.Window) {
	scripts := h.config.GetExecScripts()
	if h.config.Auto {
		scripts = []string{"auto"}
	}

	if len(scripts) > 0 {
		if err := h.pluginManager.ExecuteMultiple(scripts, window, false); err != nil {
			h.log.Error("plugin execution:", err)
		}
	}
}
