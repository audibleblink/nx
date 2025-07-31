package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(nx completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ nx completion bash > /etc/bash_completion.d/nx
  # macOS:
  $ nx completion bash > $(brew --prefix)/etc/bash_completion.d/nx

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ nx completion zsh > "${fpath[1]}/_nx"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ nx completion fish | source

  # To load completions for each session, execute once:
  $ nx completion fish > ~/.config/fish/completions/nx.fish

PowerShell:

  PS> nx completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> nx completion powershell > nx.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	// Set up custom completions using simplified functions
	execCmd.RegisterFlagCompletionFunc("on", completeTargets)
	execCmd.ValidArgsFunction = completePlugins
	serveCmd.RegisterFlagCompletionFunc("serve-dir", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	})
	serveCmd.RegisterFlagCompletionFunc("exec", completePlugins)
	serveCmd.RegisterFlagCompletionFunc("target", completeTargets)
}

// completeTargets provides completion for tmux targets
func completeTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	managers, err := NewManagers("", 0, bundledPlugins)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	panes, err := managers.Tmux.ListPanes()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	var targets []string
	for _, pane := range panes {
		target := fmt.Sprintf("%s:%d.%d", pane.Target.Session, pane.Target.Window, pane.Target.Pane)
		targets = append(targets, target)
	}
	return targets, cobra.ShellCompDirectiveNoFileComp
}

// completePlugins provides completion for plugin names
func completePlugins(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	managers, err := NewManagers("", 0, bundledPlugins)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	plugins, err := managers.Plugin.ListPlugins()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	return plugins, cobra.ShellCompDirectiveNoFileComp
}
