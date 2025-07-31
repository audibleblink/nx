package plugins

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/audibleblink/logerr"
	"github.com/disneystreaming/gomux"

	"github.com/audibleblink/nx/internal/tmux"
)

// Manager handles plugin operations
type Manager struct {
	pluginDir      string
	bundledPlugins embed.FS
	sleepDuration  time.Duration
	tmuxManager    tmux.TmuxManager
	log            logerr.Logger
}

// NewManager creates a new plugin manager
func NewManager(
	bundledPlugins embed.FS,
	sleepDuration time.Duration,
	tmuxMgr tmux.TmuxManager,
) *Manager {
	pluginDir := filepath.Join(xdg.ConfigHome, "nx", "plugins")
	// Ensure plugin directory exists
	os.MkdirAll(pluginDir, 0755)

	return &Manager{
		pluginDir:      pluginDir,
		bundledPlugins: bundledPlugins,
		sleepDuration:  sleepDuration,
		tmuxManager:    tmuxMgr,
		log:            logerr.Add("plugins"),
	}
}

// InstallBundledPlugins installs bundled plugins to the config directory.
// If embeddedPath is provided, it will be used instead of the default "plugins" path.
func (m *Manager) InstallBundledPlugins(embeddedPath ...string) error {
	log := m.log.Add("InstallBundledPlugins")

	// Use default path if none provided
	pluginPath := "plugins"
	if len(embeddedPath) > 0 && embeddedPath[0] != "" {
		pluginPath = embeddedPath[0]
	}

	entries, err := m.bundledPlugins.ReadDir(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read bundled plugins from %s: %w", pluginPath, err)
	}

	installedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := m.bundledPlugins.ReadFile(filepath.Join(pluginPath, entry.Name()))
		if err != nil {
			log.Warn("Failed to read plugin:", entry.Name(), err)
			continue
		}

		destPath := filepath.Join(m.pluginDir, entry.Name())

		// Check if plugin already exists
		if _, err := os.Stat(destPath); err == nil {
			log.Info("Plugin already exists, skipping:", entry.Name())
			continue
		}

		// Write plugin file
		err = os.WriteFile(destPath, content, 0755)
		if err != nil {
			log.Warn("Failed to install plugin:", entry.Name(), err)
			continue
		}

		log.Info("Installed plugin:", entry.Name())
		installedCount++
	}

	if installedCount == 0 {
		log.Info("No new plugins to install")
	} else {
		log.Info("Successfully installed", installedCount, "plugin(s) to:", m.pluginDir)
	}

	return nil
}

// Execute executes a plugin by name in the specified tmux window
func (m *Manager) Execute(pluginName string, window *gomux.Window) error {
	commands, err := m.readPluginCommands(pluginName)
	if err != nil {
		return err
	}

	return m.executeCommands(pluginName, commands, func(command string) error {
		return m.tmuxManager.ExecuteInWindow(window, command)
	})
}

// ExecuteOnPane executes a plugin by name on a specific tmux pane
func (m *Manager) ExecuteOnPane(pluginName string, target *tmux.PaneTarget) error {
	commands, err := m.readPluginCommands(pluginName)
	if err != nil {
		return err
	}

	return m.executeCommands(pluginName, commands, func(command string) error {
		// Create a new tmux manager for the target session
		tmuxManager, err := tmux.NewManager(target.Session)
		if err != nil {
			return fmt.Errorf("failed to get tmux manager: %w", err)
		}
		return tmuxManager.ExecuteOnPane(target, command)
	})
}

// ExecuteMultiple executes multiple plugins sequentially in the specified tmux window
func (m *Manager) ExecuteMultiple(pluginNames []string, window *gomux.Window, continueOnError bool) error {
	log := m.log.Add("ExecuteMultiple")
	
	if len(pluginNames) == 0 {
		return nil
	}
	
	// Validate all plugins exist before executing any
	for _, pluginName := range pluginNames {
		if !m.PluginExists(pluginName) {
			return fmt.Errorf("plugin '%s' not found", pluginName)
		}
	}
	
	log.Infof("Executing %d plugin(s): %v", len(pluginNames), pluginNames)
	
	for i, pluginName := range pluginNames {
		log.Infof("[%d/%d] Executing '%s'...", i+1, len(pluginNames), pluginName)
		
		if err := m.Execute(pluginName, window); err != nil {
			if continueOnError {
				log.Error(fmt.Sprintf("Plugin '%s' failed (continuing): %v", pluginName, err))
				continue
			} else {
				return fmt.Errorf("plugin '%s' failed: %w", pluginName, err)
			}
		}
		
		log.Infof("[%d/%d] Plugin '%s' completed successfully", i+1, len(pluginNames), pluginName)
	}
	
	log.Info("All plugins completed successfully")
	return nil
}

// ExecuteMultipleOnPane executes multiple plugins sequentially on a specific tmux pane
func (m *Manager) ExecuteMultipleOnPane(pluginNames []string, target *tmux.PaneTarget, continueOnError bool) error {
	log := m.log.Add("ExecuteMultipleOnPane")
	
	if len(pluginNames) == 0 {
		return nil
	}
	
	// Validate all plugins exist before executing any
	for _, pluginName := range pluginNames {
		if !m.PluginExists(pluginName) {
			return fmt.Errorf("plugin '%s' not found", pluginName)
		}
	}
	
	log.Infof("Executing %d plugin(s) on pane: %v", len(pluginNames), pluginNames)
	
	for i, pluginName := range pluginNames {
		log.Infof("[%d/%d] Executing '%s'...", i+1, len(pluginNames), pluginName)
		
		if err := m.ExecuteOnPane(pluginName, target); err != nil {
			if continueOnError {
				log.Error(fmt.Sprintf("Plugin '%s' failed (continuing): %v", pluginName, err))
				continue
			} else {
				return fmt.Errorf("plugin '%s' failed: %w", pluginName, err)
			}
		}
		
		log.Infof("[%d/%d] Plugin '%s' completed successfully", i+1, len(pluginNames), pluginName)
	}
	
	log.Info("All plugins completed successfully")
	return nil
}

// readPluginCommands reads and parses commands from a plugin file
func (m *Manager) readPluginCommands(pluginName string) ([]string, error) {
	pluginPath := filepath.Join(m.pluginDir, pluginName+".sh")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin not found: %s", pluginPath)
	}

	file, err := os.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			commands = append(commands, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading plugin: %w", err)
	}

	return commands, nil
}

// executeCommands executes a list of commands using the provided executor
func (m *Manager) executeCommands(pluginName string, commands []string, executor func(string) error) error {
	log := m.log.Add("Execute")
	log.Info("Running plugin:", pluginName)

	for _, command := range commands {
		log.Debug("command:", command)
		if err := executor(command); err != nil {
			log.Warn("plugin command failed:", err)
		}
		time.Sleep(m.sleepDuration)
	}

	log.Info("Plugin completed:", pluginName)
	return nil
}

// ListPlugins returns a list of available plugins
func (m *Manager) ListPlugins() ([]string, error) {
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
			pluginName := strings.TrimSuffix(entry.Name(), ".sh")
			plugins = append(plugins, pluginName)
		}
	}

	return plugins, nil
}

// PluginExists checks if a plugin exists
func (m *Manager) PluginExists(pluginName string) bool {
	pluginPath := filepath.Join(m.pluginDir, pluginName+".sh")
	_, err := os.Stat(pluginPath)
	return err == nil
}

// GetPluginDir returns the plugin directory path
func (m *Manager) GetPluginDir() string {
	return m.pluginDir
}
