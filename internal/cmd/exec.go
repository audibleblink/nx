package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/audibleblink/logerr"
	"github.com/spf13/cobra"

	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/plugins"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec <scripts>",
	Short: "Execute plugin script(s) on a tmux pane",
	Long: `Execute one or more plugin scripts on a specified tmux pane. Scripts must be
installed in the plugins directory, and the target pane must exist.

Multiple scripts can be specified as comma-separated values and will be executed sequentially.

The target pane is specified using tmux notation: session:window.pane

Examples:
  nx exec auto --on nx:0.1
  nx exec auto,utils,cleanup --on myapp:1.0 --dry-run`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create exec config from flags and args using cobra's built-in flag access
		cfg := &config.ExecCommand{}
		cfg.Args.Scripts = args[0]
		cfg.On, _ = cmd.Flags().GetString("on")
		cfg.DryRun, _ = cmd.Flags().GetBool("dry-run")

		executeExecCommand(cfg)
	},
}

func init() {
	// Define flags directly without global variables
	execCmd.Flags().String("on", "", "Target pane using tmux notation (session:window.pane)")
	execCmd.Flags().Bool("dry-run", false, "Preview execution without running")
	execCmd.MarkFlagRequired("on")
	
	// Set up flag completions
	execCmd.RegisterFlagCompletionFunc("on", completeTargets)
	execCmd.ValidArgsFunction = completePlugins
}

// executeExecCommand runs the exec command with the given configuration
func executeExecCommand(cfg *config.ExecCommand) {
	// Get scripts from config (handles comma-separated parsing)
	scripts := cfg.GetScripts()
	if len(scripts) == 0 {
		fatalWithSuggestions("Missing script argument", true, false, nil)
	}

	// Initialize managers
	managers, err := NewManagers(DefaultSessionName, DefaultSleep, bundledPlugins)
	if err != nil {
		logerr.Fatal("Failed to initialize managers:", err)
	}

	// Parse and validate target
	target, err := managers.Tmux.ParseTarget(cfg.On)
	if err != nil {
		logerr.Error("Invalid target format:", err)
		fmt.Println()
		fmt.Println("Target format should be: session:window.pane")
		fmt.Println("Examples:")
		fmt.Println("  nx:0.1    - session 'nx', window 0, pane 1")
		fmt.Println("  myapp:2.0 - session 'myapp', window 2, pane 0")
		os.Exit(1)
	}

	if err := managers.Tmux.ValidatePane(target); err != nil {
		fatalWithSuggestions(
			fmt.Sprintf("Target pane '%s' not found", cfg.On),
			false, true, managers,
		)
	}

	// Validate all scripts exist before executing any
	for _, script := range scripts {
		if !managers.Plugin.PluginExists(script) {
			fatalWithSuggestions(
				fmt.Sprintf("Plugin '%s' not found", script),
				true, false, managers,
			)
		}
	}

	// Handle dry-run
	if cfg.DryRun {
		showDryRunMultiple(scripts, cfg.On, managers.Plugin)
		return
	}

	// Execute scripts sequentially
	logerr.Info(fmt.Sprintf("Running %d script(s) on %s...", len(scripts), cfg.On))
	
	if err := managers.Plugin.ExecuteMultipleOnPane(scripts, target, false); err != nil {
		logerr.Fatal("Failed to execute scripts:", err)
	}
	
	logerr.Info("All scripts completed successfully")
}

// showDryRun displays what would be executed in dry-run mode
func showDryRun(script, target string, pluginManager *plugins.Manager) {
	logerr.Info(fmt.Sprintf("Would run '%s' on %s:", script, target))

	pluginPath := filepath.Join(pluginManager.GetPluginDir(), script+".sh")
	content, err := os.ReadFile(pluginPath)
	if err != nil {
		logerr.Fatal("Failed to read plugin file:", err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			logerr.Info("  " + line)
		}
	}
}

// showDryRunMultiple displays what would be executed for multiple scripts in dry-run mode
func showDryRunMultiple(scripts []string, target string, pluginManager *plugins.Manager) {
	logerr.Info(fmt.Sprintf("Would run %d script(s) on %s:", len(scripts), target))
	
	for i, script := range scripts {
		logerr.Info(fmt.Sprintf("[%d/%d] Script '%s':", i+1, len(scripts), script))
		
		pluginPath := filepath.Join(pluginManager.GetPluginDir(), script+".sh")
		content, err := os.ReadFile(pluginPath)
		if err != nil {
			logerr.Error(fmt.Sprintf("Failed to read plugin file '%s': %v", script, err))
			continue
		}

		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				logerr.Info("    " + line)
			}
		}
		
		if i < len(scripts)-1 {
			logerr.Info("") // Add spacing between scripts
		}
	}
}
