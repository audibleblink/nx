package cmd

import (
	"github.com/spf13/cobra"

	"github.com/audibleblink/nx/internal/config"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the nx multiplexed server",
	Long: `Start the nx multiplexed server with HTTP file serving, SSH access,
and shell command execution capabilities. This is the default behavior when
no subcommand is specified.

The server provides:
- HTTP file serving (when --serve-dir is specified)
- SSH server with optional password authentication
- Shell command execution with tmux integration
- Plugin-based automation system with multiple script support`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create config from flags using cobra's built-in flag access
		cfg := &config.Config{}
		cfg.Auto, _ = cmd.Flags().GetBool("auto")
		cfg.Exec, _ = cmd.Flags().GetString("exec")
		cfg.InstallPlugins = false
		cfg.Iface, _ = cmd.Flags().GetString("host")
		cfg.Port, _ = cmd.Flags().GetString("port")
		cfg.ServeDir, _ = cmd.Flags().GetString("serve-dir")
		cfg.Target, _ = cmd.Flags().GetString("target")
		cfg.Sleep, _ = cmd.Flags().GetDuration("sleep")
		cfg.Verbose, _ = cmd.Flags().GetBool("verbose")
		cfg.SSHPass, _ = cmd.Flags().GetString("ssh-pass")

		executeServerCommand(cfg)
	},
}

func init() {
	// Define flags directly without global variables
	serveCmd.Flags().Bool("auto", false, "Attempt to auto-upgrade to a tty (uses --exec auto)")
	serveCmd.Flags().String("exec", "", "Execute plugin scripts on connection (comma-separated)")
	serveCmd.Flags().StringP("host", "i", "0.0.0.0", "Interface address on which to bind")
	serveCmd.Flags().StringP("port", "p", "8443", "Port on which to bind")
	serveCmd.Flags().StringP("serve-dir", "d", "", "Directory to serve files from over HTTP")
	serveCmd.Flags().StringP("target", "t", DefaultSessionName, "Tmux session name")
	serveCmd.Flags().Duration("sleep", DefaultSleep, "adjust if --auto is failing")
	serveCmd.Flags().BoolP("verbose", "v", false, "Debug logging")
	serveCmd.Flags().StringP("ssh-pass", "s", "", "SSH password (empty = no auth)")

	// Set up flag completions
	serveCmd.RegisterFlagCompletionFunc(
		"serve-dir",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	)
	serveCmd.RegisterFlagCompletionFunc("exec", completePlugins)
	serveCmd.RegisterFlagCompletionFunc("target", completeTargets)
}
