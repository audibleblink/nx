package cmd

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/audibleblink/logerr"

	"github.com/audibleblink/nx/internal/plugins"
	"github.com/audibleblink/nx/internal/tmux"
)

const (
	DefaultSessionName = "nx"
	DefaultSleep       = 500 * time.Millisecond
)

// Managers holds commonly used manager instances
type Managers struct {
	Tmux   *tmux.Manager
	Plugin *plugins.Manager
}

// NewManagers creates a new set of managers with common defaults
func NewManagers(sessionName string, sleep time.Duration, bundledPlugins embed.FS) (*Managers, error) {
	if sessionName == "" {
		sessionName = DefaultSessionName
	}
	if sleep == 0 {
		sleep = DefaultSleep
	}

	tmuxManager, err := tmux.NewManager(sessionName)
	if err != nil {
		return nil, err
	}

	pluginManager := plugins.NewManager(bundledPlugins, sleep, tmuxManager)

	return &Managers{
		Tmux:   tmuxManager,
		Plugin: pluginManager,
	}, nil
}

// showAvailablePlugins displays available plugins to the user
func showAvailablePlugins(pluginManager *plugins.Manager) {
	plugins, err := pluginManager.ListPlugins()
	if err != nil {
		return
	}

	if len(plugins) > 0 {
		fmt.Println("Available plugins:")
		for _, plugin := range plugins {
			fmt.Printf("  - %s\n", plugin)
		}
	} else {
		fmt.Println("No plugins are currently installed.")
		fmt.Println("Run 'nx --install-plugins' to install bundled plugins.")
	}
}

// showAvailablePanes displays available tmux panes to the user
func showAvailablePanes(tmuxManager *tmux.Manager) {
	panes, err := tmuxManager.ListPanes()
	if err != nil {
		return
	}

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
}

// fatalWithSuggestions logs an error and shows helpful suggestions before exiting
func fatalWithSuggestions(msg string, showPlugins, showPanes bool, managers *Managers) {
	logerr.Error(msg)
	fmt.Println()

	if showPlugins && managers != nil {
		showAvailablePlugins(managers.Plugin)
		fmt.Println()
	}

	if showPanes && managers != nil {
		showAvailablePanes(managers.Tmux)
		fmt.Println()
	}

	os.Exit(1)
}
