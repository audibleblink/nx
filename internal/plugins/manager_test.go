package plugins

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/disneystreaming/gomux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/audibleblink/nx/internal/tmux"
)

//go:embed testdata
var testPluginsFS embed.FS

// testPlugins provides the embedded filesystem for tests
var testPlugins = testPluginsFS

// MockTmuxManager is a mock implementation of tmux.TmuxManager for testing
type MockTmuxManager struct {
	mock.Mock
}

func (m *MockTmuxManager) CreateWindow(socketFile string) (*gomux.Window, error) {
	args := m.Called(socketFile)
	return args.Get(0).(*gomux.Window), args.Error(1)
}

func (m *MockTmuxManager) ExecuteInWindow(window *gomux.Window, command string) error {
	args := m.Called(window, command)
	return args.Error(0)
}

func (m *MockTmuxManager) GetSession() *gomux.Session {
	args := m.Called()
	return args.Get(0).(*gomux.Session)
}

func (m *MockTmuxManager) GetSessionName() string {
	args := m.Called()
	return args.String(0)
}

func TestNewManager(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	sleepDuration := 100 * time.Millisecond

	manager := NewManager(testPlugins, sleepDuration, mockTmux)
	require.NotNil(t, manager)

	// Check that plugin directory is set and exists
	pluginDir := manager.GetPluginDir()
	assert.NotEmpty(t, pluginDir)
	assert.Contains(t, pluginDir, "nx")
	assert.Contains(t, pluginDir, "plugins")

	// Directory should exist
	_, err := os.Stat(pluginDir)
	assert.NoError(t, err)

	// Cleanup
	defer os.RemoveAll(pluginDir)
}

func TestManagerInstallBundledPlugins(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	t.Run("installs bundled plugins successfully", func(t *testing.T) {
		err := manager.InstallBundledPlugins("testdata")
		require.NoError(t, err)

		// Check that plugins were installed
		pluginDir := manager.GetPluginDir()
		entries, err := os.ReadDir(pluginDir)
		require.NoError(t, err)

		// Should have installed test plugins
		pluginNames := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				pluginNames = append(pluginNames, entry.Name())
			}
		}

		assert.Contains(t, pluginNames, "test_plugin.sh")
		assert.Contains(t, pluginNames, "empty_plugin.sh")

		// Check file permissions
		testPluginPath := filepath.Join(pluginDir, "test_plugin.sh")
		info, err := os.Stat(testPluginPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode())
	})

	t.Run("skips existing plugins", func(t *testing.T) {
		// Install plugins first time
		err := manager.InstallBundledPlugins("testdata")
		require.NoError(t, err)

		// Get modification time of a plugin
		testPluginPath := filepath.Join(manager.GetPluginDir(), "test_plugin.sh")
		info1, err := os.Stat(testPluginPath)
		require.NoError(t, err)

		// Wait a bit to ensure different modification time if file was rewritten
		time.Sleep(10 * time.Millisecond)

		// Install again - should skip existing
		err = manager.InstallBundledPlugins("testdata")
		require.NoError(t, err)

		// Check that file wasn't modified
		info2, err := os.Stat(testPluginPath)
		require.NoError(t, err)
		assert.Equal(t, info1.ModTime(), info2.ModTime())
	})
}

func TestManagerExecute(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 10*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	t.Run("executes plugin successfully", func(t *testing.T) {
		// Create a mock window
		mockWindow := &gomux.Window{}

		// Set up expectations for tmux commands
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Hello from test plugin\"").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "ls -la").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Plugin execution complete\"").Return(nil)

		err := manager.Execute("test_plugin", mockWindow)
		require.NoError(t, err)

		// Verify all expected calls were made
		mockTmux.AssertExpectations(t)
	})

	t.Run("handles empty plugin", func(t *testing.T) {
		mockWindow := &gomux.Window{}

		// Empty plugin should not call any tmux commands
		err := manager.Execute("empty_plugin", mockWindow)
		require.NoError(t, err)

		// No tmux calls should have been made
		mockTmux.AssertNotCalled(t, "ExecuteInWindow")
	})

	t.Run("returns error for non-existent plugin", func(t *testing.T) {
		mockWindow := &gomux.Window{}

		err := manager.Execute("non_existent_plugin", mockWindow)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin not found")
	})

	t.Run("handles tmux execution errors gracefully", func(t *testing.T) {
		mockWindow := &gomux.Window{}

		// Set up expectation for command that will fail
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Hello from test plugin\"").
			Return(assert.AnError)
		mockTmux.On("ExecuteInWindow", mockWindow, "ls -la").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Plugin execution complete\"").Return(nil)

		// Plugin execution should continue despite individual command failures
		err := manager.Execute("test_plugin", mockWindow)
		require.NoError(t, err)

		mockTmux.AssertExpectations(t)
	})
}

func TestManagerListPlugins(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	t.Run("lists installed plugins", func(t *testing.T) {
		// Install plugins first
		err := manager.InstallBundledPlugins("testdata")
		require.NoError(t, err)

		plugins, err := manager.ListPlugins()
		require.NoError(t, err)

		// Should return plugin names without .sh extension
		assert.Contains(t, plugins, "test_plugin")
		assert.Contains(t, plugins, "empty_plugin")

		// Should not contain file extensions
		for _, plugin := range plugins {
			assert.False(t, strings.HasSuffix(plugin, ".sh"))
		}
	})

	t.Run("returns empty list when no plugins", func(t *testing.T) {
		// Create a new manager with empty plugin directory and unique path
		emptyManager := NewManager(embed.FS{}, 100*time.Millisecond, mockTmux)
		defer os.RemoveAll(emptyManager.GetPluginDir())

		// Ensure the plugin directory is empty by creating a fresh one
		os.RemoveAll(emptyManager.GetPluginDir())
		os.MkdirAll(emptyManager.GetPluginDir(), 0755)

		plugins, err := emptyManager.ListPlugins()
		require.NoError(t, err)
		assert.Empty(t, plugins)
	})

	t.Run("ignores non-shell files", func(t *testing.T) {
		// Create a separate manager for this test to avoid interference
		testManager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
		defer os.RemoveAll(testManager.GetPluginDir())

		// Install plugins first
		err := testManager.InstallBundledPlugins("testdata")
		require.NoError(t, err)

		// Create a non-shell file
		pluginDir := testManager.GetPluginDir()
		nonShellFile := filepath.Join(pluginDir, "readme.txt")
		err = os.WriteFile(nonShellFile, []byte("This is not a plugin"), 0644)
		require.NoError(t, err)

		// Create a directory
		subDir := filepath.Join(pluginDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		plugins, err := testManager.ListPlugins()
		require.NoError(t, err)

		// Should only contain .sh files
		for _, plugin := range plugins {
			assert.NotEqual(t, "readme", plugin)
			assert.NotEqual(t, "subdir", plugin)
		}
	})
}

func TestManagerPluginExists(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	t.Run("returns true for existing plugin", func(t *testing.T) {
		exists := manager.PluginExists("test_plugin")
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent plugin", func(t *testing.T) {
		exists := manager.PluginExists("non_existent_plugin")
		assert.False(t, exists)
	})
}

func TestManagerGetPluginDir(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	pluginDir := manager.GetPluginDir()
	assert.NotEmpty(t, pluginDir)
	assert.Contains(t, pluginDir, "nx")
	assert.Contains(t, pluginDir, "plugins")

	// Directory should exist
	info, err := os.Stat(pluginDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestPluginExecutionTiming tests that sleep duration is respected between commands
func TestPluginExecutionTiming(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	sleepDuration := 50 * time.Millisecond
	manager := NewManager(testPlugins, sleepDuration, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	mockWindow := &gomux.Window{}

	// Track timing of calls
	var callTimes []time.Time
	mockTmux.On("ExecuteInWindow", mockWindow, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
		}).
		Return(nil)

	start := time.Now()
	err = manager.Execute("test_plugin", mockWindow)
	require.NoError(t, err)

	// Should have made 3 calls (non-comment, non-empty lines)
	assert.Len(t, callTimes, 3)

	// Check that there was appropriate delay between calls
	if len(callTimes) >= 2 {
		timeBetweenCalls := callTimes[1].Sub(callTimes[0])
		assert.GreaterOrEqual(t, timeBetweenCalls, sleepDuration)
	}

	// Total execution time should be at least 2 * sleepDuration (2 sleeps between 3 commands)
	totalTime := time.Since(start)
	expectedMinTime := 2 * sleepDuration
	assert.GreaterOrEqual(t, totalTime, expectedMinTime)
}

// TestManagerExecuteMultiple tests the ExecuteMultiple method
func TestManagerExecuteMultiple(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 10*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	t.Run("executes multiple plugins successfully", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		// Set up expectations for both plugins
		// test_plugin commands
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Hello from test plugin\"").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "ls -la").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Plugin execution complete\"").Return(nil)
		
		// empty_plugin has no commands, so no expectations needed

		pluginNames := []string{"test_plugin", "empty_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, false)
		require.NoError(t, err)

		mockTmux.AssertExpectations(t)
	})

	t.Run("handles empty plugin list", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		err := manager.ExecuteMultiple([]string{}, mockWindow, false)
		require.NoError(t, err)
		
		// No tmux calls should have been made
		mockTmux.AssertNotCalled(t, "ExecuteInWindow")
	})

	t.Run("stops on first error when continueOnError is false", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		// Note: The current executeCommands implementation doesn't propagate command failures
		// It only logs them and continues. So we test with a non-existent plugin instead.
		pluginNames := []string{"non_existent_plugin", "empty_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin 'non_existent_plugin' not found")

		// No tmux calls should have been made due to early validation
		mockTmux.AssertNotCalled(t, "ExecuteInWindow")
	})

	t.Run("continues on error when continueOnError is true", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		// First plugin will fail, but execution should continue
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Hello from test plugin\"").Return(assert.AnError)
		mockTmux.On("ExecuteInWindow", mockWindow, "ls -la").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Plugin execution complete\"").Return(nil)
		
		pluginNames := []string{"test_plugin", "empty_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, true)
		require.NoError(t, err)

		mockTmux.AssertExpectations(t)
	})

	t.Run("returns error for non-existent plugin", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		pluginNames := []string{"non_existent_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin 'non_existent_plugin' not found")
	})
}

// TestManagerExecuteMultipleOnPane tests the ExecuteMultipleOnPane method
func TestManagerExecuteMultipleOnPane(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 10*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	target := &tmux.PaneTarget{
		Session: "non_existent_session_12345",
		Window:  0,
		Pane:    1,
	}

	t.Run("executes empty plugin on pane successfully", func(t *testing.T) {
		// Empty plugin has no commands, so it should succeed without calling tmux
		pluginNames := []string{"empty_plugin"}
		err := manager.ExecuteMultipleOnPane(pluginNames, target, false)
		require.NoError(t, err)
	})

	t.Run("handles empty plugin list", func(t *testing.T) {
		err := manager.ExecuteMultipleOnPane([]string{}, target, false)
		require.NoError(t, err)
	})
}

// TestManagerMultipleScriptIntegration tests integration scenarios
func TestManagerMultipleScriptIntegration(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 10*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	t.Run("mixed success and failure with continue on error", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		// First plugin succeeds
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Hello from test plugin\"").Return(nil)
		mockTmux.On("ExecuteInWindow", mockWindow, "ls -la").Return(assert.AnError) // This fails
		mockTmux.On("ExecuteInWindow", mockWindow, "echo \"Plugin execution complete\"").Return(nil)
		
		// Second plugin (empty) succeeds (no commands to execute)
		
		pluginNames := []string{"test_plugin", "empty_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, true)
		require.NoError(t, err)

		mockTmux.AssertExpectations(t)
	})

	t.Run("validates all plugins exist before execution", func(t *testing.T) {
		mockWindow := &gomux.Window{}
		
		// Mix of existing and non-existing plugins
		pluginNames := []string{"test_plugin", "non_existent", "empty_plugin"}
		err := manager.ExecuteMultiple(pluginNames, mockWindow, false)
		
		// Should fail on the non-existent plugin
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non_existent")
		
		// No tmux commands should have been executed
		mockTmux.AssertNotCalled(t, "ExecuteInWindow")
	})
}

// TestPluginFilePermissions tests that installed plugins have correct permissions
func TestPluginFilePermissions(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 100*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	pluginPath := filepath.Join(manager.GetPluginDir(), "test_plugin.sh")
	info, err := os.Stat(pluginPath)
	require.NoError(t, err)

	// Should be executable
	assert.Equal(t, os.FileMode(0755), info.Mode())
}

// TestConcurrentPluginOperations tests that plugin operations are thread-safe
func TestConcurrentPluginOperations(t *testing.T) {
	mockTmux := &MockTmuxManager{}
	manager := NewManager(testPlugins, 10*time.Millisecond, mockTmux)
	defer os.RemoveAll(manager.GetPluginDir())

	// Install plugins first
	err := manager.InstallBundledPlugins("testdata")
	require.NoError(t, err)

	const numGoroutines = 5
	results := make(chan error, numGoroutines)

	// Run multiple plugin operations concurrently
	for range numGoroutines {
		go func() {
			plugins, err := manager.ListPlugins()
			if err != nil {
				results <- err
				return
			}

			if len(plugins) == 0 {
				results <- assert.AnError
				return
			}

			exists := manager.PluginExists("test_plugin")
			if !exists {
				results <- assert.AnError
				return
			}

			results <- nil
		}()
	}

	// Wait for all operations to complete
	for range numGoroutines {
		select {
		case err := <-results:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}
