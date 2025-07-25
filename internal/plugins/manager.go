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

// InstallBundledPlugins installs bundled plugins to the config directory
func (m *Manager) InstallBundledPlugins() error {
	log := m.log.Add("InstallBundledPlugins")

	entries, err := m.bundledPlugins.ReadDir("plugins")
	if err != nil {
		return fmt.Errorf("failed to read bundled plugins: %w", err)
	}

	installedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := m.bundledPlugins.ReadFile(filepath.Join("plugins", entry.Name()))
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

// InstallBundledPluginsFromPath installs bundled plugins from a specific embedded path
func (m *Manager) InstallBundledPluginsFromPath(embeddedPath string) error {
	log := m.log.Add("InstallBundledPluginsFromPath")

	entries, err := m.bundledPlugins.ReadDir(embeddedPath)
	if err != nil {
		return fmt.Errorf("failed to read bundled plugins from %s: %w", embeddedPath, err)
	}

	installedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := m.bundledPlugins.ReadFile(filepath.Join(embeddedPath, entry.Name()))
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
	log := m.log.Add("Execute")

	pluginPath := filepath.Join(m.pluginDir, pluginName+".sh")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin not found: %s", pluginPath)
	}

	file, err := os.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}
	defer file.Close()

	log.Add("Plugin").Info("Running:", pluginName)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		log.Debug("command:", line)
		err := m.tmuxManager.ExecuteInWindow(window, line)
		if err != nil {
			log.Warn("plugin command failed:", err)
		}

		// Default sleep between commands
		time.Sleep(m.sleepDuration)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading plugin: %w", err)
	}

	log.Add("Plugin").Info("Done:", pluginName)
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
