package main

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/audibleblink/logerr"
	"github.com/jessevdk/go-flags"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/mux"
	"github.com/audibleblink/nx/internal/plugins"
	"github.com/audibleblink/nx/internal/protocols"
	"github.com/audibleblink/nx/internal/tmux"
	"github.com/audibleblink/nx/pkg/socket"
)

// Embed the plugins directory
//
//go:embed plugins/*
var bundledPlugins embed.FS

func init() {
	// Set up logging
	logerr.SetContextSeparator(" ❯ ")
	logerr.SetLogLevel(logerr.LogLevelInfo)
	logerr.EnableColors()
	logerr.EnableTimestamps()
	logerr.SetContext("nx")
}

func main() {
	// Parse command line arguments
	var cfg config.Config
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		logerr.Fatal("Failed to parse arguments:", err)
	}

	if err := cfg.Validate(); err != nil {
		logerr.Fatal("Invalid configuration:", err)
	}

	if cfg.Verbose {
		logerr.SetLogLevel(logerr.LogLevelDebug)
	}

	if cfg.ServeDir != "" {
		if _, err := os.Stat(cfg.ServeDir); os.IsNotExist(err) {
			logerr.Fatal("Serve directory does not exist:", cfg.ServeDir)
		}
		logerr.Debug("File serving enabled from:", cfg.ServeDir)
	}

	// Initialize components
	socketManager := socket.NewManager()

	tmuxManager, err := tmux.NewManager(cfg.Target)
	if err != nil {
		logerr.Fatal("Failed to initialize tmux manager:", err)
	}

	pluginManager := plugins.NewManager(bundledPlugins, cfg.Sleep, tmuxManager)

	// Install bundled plugins if requested
	if cfg.InstallPlugins {
		if err := pluginManager.InstallBundledPlugins(); err != nil {
			logerr.Fatal("Failed to install plugins:", err)
		}
		logerr.Info("Plugins installed successfully")
		return
	}

	// Create protocol handlers
	httpHandler := protocols.NewHTTPHandler(cfg.ServeDir)

	var sshHandler *protocols.SSHHandler
	if cfg.IsSSHEnabled() {
		sshHandler, err = protocols.NewSSHHandler(cfg.SSHPass)
		if err != nil {
			logerr.Fatal("Failed to create SSH handler:", err)
		}
	}

	shellHandler := protocols.NewShellHandler(
		&cfg,
		tmuxManager,
		socketManager,
		pluginManager,
		cfg.Address(),
	)

	// Create multiplexer server
	server, err := mux.NewServer(&cfg, httpHandler, sshHandler, shellHandler)
	if err != nil {
		logerr.Fatal("Failed to create server:", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logerr.Info("Received shutdown signal, cleaning up...")
		cancel()
		server.Stop()
		socketManager.Cleanup()
	}()

	// Start the server
	if err := server.Start(ctx); err != nil {
		// Check if this is a normal shutdown error
		if !strings.Contains(err.Error(), "use of closed network connection") {
			logerr.Fatal("Server error:", err)
		}
		// Otherwise, this is a normal shutdown - no need to log as fatal
	}
}
