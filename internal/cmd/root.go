package cmd

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/audibleblink/logerr"
	"github.com/spf13/cobra"

	"github.com/audibleblink/nx/internal/common"
	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/mux"
	"github.com/audibleblink/nx/internal/plugins"
	"github.com/audibleblink/nx/internal/protocols"
	"github.com/audibleblink/nx/internal/tmux"
	"github.com/audibleblink/nx/pkg/socket"
)

var bundledPlugins embed.FS

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nx",
	Short: "A multiplexed server for tmux automation and file serving",
	Long: `nx is a multiplexed server that provides HTTP file serving, SSH access,
and shell command execution with tmux integration. It supports plugin-based
automation and can be controlled via various protocols.

By default, nx starts the server. Use subcommands for specific operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read install-plugins flag from root command
		installPlugins, _ := cmd.Flags().GetBool("install-plugins")

		// Default behavior: run server with default config
		cfg := &config.Config{
			Auto:           false,
			Exec:           "",
			InstallPlugins: installPlugins,
			Iface:          "0.0.0.0",
			Port:           "8443",
			ServeDir:       "",
			Target:         DefaultSessionName,
			Sleep:          DefaultSleep,
			Verbose:        false,
			SSHPass:        "",
		}
		executeServerCommand(cfg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(plugins embed.FS) {
	bundledPlugins = plugins

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(completionCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add root-level flags
	rootCmd.Flags().Bool("install-plugins", false, "Install bundled plugins to config directory")
}

// executeServerCommand runs the server with the given configuration
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
	httpHandler := protocols.NewHTTPHandler(cfg.ServeDir, cfg.Address())

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
