package cmd

import (
	"github.com/spf13/cobra"

	"github.com/audibleblink/nx/internal/config"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the nx multiplexed server",
	Long: `Start the nx multiplexed server with HTTP file serving, SSH access,
and shell command execution capabilities. This is the default behavior when
no subcommand is specified.

The server provides:
- HTTP file serving (when --serve-dir is specified)
- SSH server with optional password authentication
- Shell command execution with tmux integration
- Plugin-based automation system`,
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
	serverCmd.Flags().Bool("auto", false, "Attempt to auto-upgrade to a tty (uses --exec auto)")
	serverCmd.Flags().String("exec", "", "Execute plugin script on connection")
	serverCmd.Flags().StringP("host", "i", "0.0.0.0", "Interface address on which to bind")
	serverCmd.Flags().StringP("port", "p", "8443", "Port on which to bind")
	serverCmd.Flags().StringP("serve-dir", "d", "", "Directory to serve files from over HTTP")
	serverCmd.Flags().StringP("target", "t", DefaultSessionName, "Tmux session name")
	serverCmd.Flags().Duration("sleep", DefaultSleep, "adjust if --auto is failing")
	serverCmd.Flags().BoolP("verbose", "v", false, "Debug logging")
	serverCmd.Flags().StringP("ssh-pass", "s", "", "SSH password (empty = no auth)")
}
