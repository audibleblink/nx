package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/audibleblink/nx/internal/bridge"
	"github.com/spf13/cobra"
	"github.com/audibleblink/logerr"
)

var bridgeCmd = &cobra.Command{
	Use:   "bridge <socket-path>",
	Short: "Connect to Unix socket and bridge stdio",
	Long:  `Connect to a Unix domain socket and bridge stdin/stdout through it.`,
	Args:  cobra.ExactArgs(1),
	Run:   runBridgeConnect,
}

func runBridgeConnect(cmd *cobra.Command, args []string) {
	if err := runBridgeConnectWithError(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBridgeConnectWithError(socketPath string) error {
	session, err := bridge.NewSession(socketPath)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sessionDone := make(chan error, 1)
	go func() {
		sessionDone <- session.Start()
	}()

	select {
	case err := <-sessionDone:
		if err != nil {
			return fmt.Errorf("session ended with error: %w", err)
		}
		return nil
	case sig := <-sigChan:
		logerr.Info("Received signal, shutting down:", sig)
		session.Close()
		return fmt.Errorf("interrupted by signal %v", sig)
	}
}

func init() {
	rootCmd.AddCommand(bridgeCmd)
}
