package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/audibleblink/logerr"
	"github.com/jessevdk/go-flags"

	"github.com/audibleblink/nx/internal/common"
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
	// Parse command line arguments with subcommands
	var commands config.Commands
	parser := flags.NewParser(&commands, flags.Default)

	// Parse arguments and determine which command was used
	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		logerr.Fatal("Failed to parse arguments:", err)
	}

	// Determine which command to execute
	if parser.Active != nil {
		switch parser.Active.Name {
		case "exec":
			executeExecCommand(&commands.Exec)
			return
		case "server":
			executeServerCommand(&commands.Server)
			return
		}
	}

	// Default behavior: if no subcommand specified, run server with default config
	// This maintains backward compatibility
	executeServerCommand(&commands.Server)
}

func executeServerCommand(cfg *config.Config) {
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
		cfg,
		tmuxManager,
		socketManager,
		pluginManager,
		cfg.Address(),
	)

	// Create multiplexer server
	server, err := mux.NewServer(cfg, httpHandler, sshHandler, shellHandler)
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
		if !common.IsShutdownError(err) {
			logerr.Fatal("Server error:", err)
		}
		// Otherwise, this is a normal shutdown - no need to log as fatal
	}
}

func executeExecCommand(cfg *config.ExecCommand) {
	// Debug: print the parsed values
	// fmt.Printf("DEBUG: Parsed script: '%s', on: '%s', dry-run: %v\n", cfg.Args.Script, cfg.On, cfg.DryRun)

	// Validate that we have a script argument
	if cfg.Args.Script == "" {
		showExecUsageAndExit("Missing script argument")
	}

	// Parse the target
	tmuxManager, err := tmux.NewManager("nx") // Use default session name for now
	if err != nil {
		logerr.Fatal("Failed to initialize tmux manager:", err)
	}

	target, err := tmuxManager.ParseTarget(cfg.On)
	if err != nil {
		logerr.Error("Invalid target format:", err)
		fmt.Println()
		fmt.Println("Target format should be: session:window.pane")
		fmt.Println("Examples:")
		fmt.Println("  nx:0.1    - session 'nx', window 0, pane 1")
		fmt.Println("  myapp:2.0 - session 'myapp', window 2, pane 0")
		os.Exit(1)
	}

	// Validate that the target pane exists
	if err := tmuxManager.ValidatePane(target); err != nil {
		// If validation fails, show available panes
		panes, listErr := tmuxManager.ListPanes()
		if listErr != nil {
			logerr.Fatal("Target pane not found and failed to list available panes:", listErr)
		}

		logerr.Error(fmt.Sprintf("Target pane '%s' not found", cfg.On))
		fmt.Println()
		if len(panes) > 0 {
			fmt.Println("Available panes:")
			for _, pane := range panes {
				status := ""
				if pane.Active {
					status = " (active)"
				}
				fmt.Printf("  - %s:%d.%d - %s%s\n",
					pane.Target.Session, pane.Target.Window, pane.Target.Pane,
					pane.WindowName, status)
			}
		} else {
			fmt.Println("No tmux panes are currently available.")
			fmt.Println("Make sure tmux is running and has active sessions.")
		}
		os.Exit(1)
	}

	// Initialize plugin manager
	pluginManager := plugins.NewManager(bundledPlugins, 500*time.Millisecond, tmuxManager)

	// Check if the plugin exists
	if !pluginManager.PluginExists(cfg.Args.Script) {
		availablePlugins, err := pluginManager.ListPlugins()
		if err != nil {
			logerr.Fatal("Plugin not found and failed to list available plugins:", err)
		}

		logerr.Error(fmt.Sprintf("Plugin '%s' not found", cfg.Args.Script))
		fmt.Println()
		if len(availablePlugins) > 0 {
			fmt.Println("Available plugins:")
			for _, plugin := range availablePlugins {
				fmt.Printf("  - %s\n", plugin)
			}
		} else {
			fmt.Println("No plugins are currently installed.")
			fmt.Println("Run 'nx server --install-plugins' to install bundled plugins.")
		}
		os.Exit(1)
	}

	// Handle dry-run mode
	if cfg.DryRun {
		logerr.Info(fmt.Sprintf("Would run '%s' on %s:", cfg.Args.Script, cfg.On))

		// Read and display the plugin contents
		pluginPath := filepath.Join(pluginManager.GetPluginDir(), cfg.Args.Script+".sh")
		content, err := os.ReadFile(pluginPath)
		if err != nil {
			logerr.Fatal("Failed to read plugin file:", err)
		}

		lines := strings.SplitSeq(string(content), "\n")
		for line := range lines {
			line = strings.TrimSpace(line)
			// Skip empty lines and comments for dry-run display
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			logerr.Info("  " + line)
		}
		return
	}

	// Execute the plugin on the target pane
	logerr.Info(fmt.Sprintf("Running '%s' on %s...", cfg.Args.Script, cfg.On))

	if err := pluginManager.ExecuteOnPane(cfg.Args.Script, target); err != nil {
		logerr.Fatal("Failed to execute plugin:", err)
	}

	logerr.Info("Script completed successfully")
}

// showExecUsageAndExit displays helpful usage information for the exec command and exits
func showExecUsageAndExit(errorMsg string) {
	if errorMsg != "" {
		logerr.Error(errorMsg)
		fmt.Println()
	}

	// Try to get available plugins
	tmuxManager, err := tmux.NewManager("nx")
	if err == nil {
		pluginManager := plugins.NewManager(bundledPlugins, 500*time.Millisecond, tmuxManager)
		availablePlugins, err := pluginManager.ListPlugins()
		if err == nil && len(availablePlugins) > 0 {
			fmt.Println("Available plugins:")
			for _, plugin := range availablePlugins {
				fmt.Printf("  - %s\n", plugin)
			}
			fmt.Println()
		}
	}

	fmt.Println("Usage:")
	fmt.Println("  nx exec <script> --on <target> [OPTIONS]")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  script    Name of plugin script to execute")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --on      Target pane using tmux notation (session:window.pane)")
	fmt.Println("  --dry-run Preview execution without running")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  nx exec auto --on nx:0.1")
	fmt.Println("  nx exec cleanup --on myapp:1.0 --dry-run")

	os.Exit(1)
}
