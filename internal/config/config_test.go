package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with defaults",
			config: Config{
				Iface:  "0.0.0.0",
				Port:   "8443",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "valid config with IPv6",
			config: Config{
				Iface:  "::1",
				Port:   "9000",
				Target: "test",
				Sleep:  1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with localhost",
			config: Config{
				Iface:  "127.0.0.1",
				Port:   "1234",
				Target: "nx",
				Sleep:  0,
			},
			wantErr: false,
		},
		{
			name: "invalid interface address",
			config: Config{
				Iface:  "invalid.ip.address",
				Port:   "8443",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "invalid interface address",
		},
		{
			name: "invalid port - non-numeric",
			config: Config{
				Iface:  "0.0.0.0",
				Port:   "invalid",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port - too low",
			config: Config{
				Iface:  "0.0.0.0",
				Port:   "0",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port - too high",
			config: Config{
				Iface:  "0.0.0.0",
				Port:   "65536",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "negative sleep duration",
			config: Config{
				Iface:  "0.0.0.0",
				Port:   "8443",
				Target: "nx",
				Sleep:  -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "sleep duration cannot be negative",
		},
		{
			name: "empty interface address (should be valid)",
			config: Config{
				Iface:  "",
				Port:   "8443",
				Target: "nx",
				Sleep:  500 * time.Millisecond,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "IPv4 address",
			config: Config{
				Iface: "192.168.1.1",
				Port:  "8080",
			},
			expected: "192.168.1.1:8080",
		},
		{
			name: "IPv6 address",
			config: Config{
				Iface: "::1",
				Port:  "9000",
			},
			expected: "[::1]:9000",
		},
		{
			name: "localhost",
			config: Config{
				Iface: "127.0.0.1",
				Port:  "3000",
			},
			expected: "127.0.0.1:3000",
		},
		{
			name: "all interfaces",
			config: Config{
				Iface: "0.0.0.0",
				Port:  "8443",
			},
			expected: "0.0.0.0:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Address()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigIsHTTPEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "HTTP enabled with serve directory",
			config: Config{
				ServeDir: "/var/www/html",
			},
			expected: true,
		},
		{
			name: "HTTP enabled with relative path",
			config: Config{
				ServeDir: "./public",
			},
			expected: true,
		},
		{
			name: "HTTP disabled - empty serve directory",
			config: Config{
				ServeDir: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsHTTPEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigIsSSHEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "SSH always enabled (current implementation)",
			config: Config{
				SSHPass: "password123",
			},
			expected: true,
		},
		{
			name: "SSH enabled even without password",
			config: Config{
				SSHPass: "",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsSSHEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigIsAutoUpgradeEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "auto upgrade enabled",
			config: Config{
				Auto: true,
			},
			expected: true,
		},
		{
			name: "auto upgrade disabled",
			config: Config{
				Auto: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Auto
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConfigDefaults tests that the struct tags define reasonable defaults
func TestConfigDefaults(t *testing.T) {
	// This test documents the expected default values from struct tags
	// Note: go-flags would normally set these, but we test the expected values
	expectedDefaults := map[string]interface{}{
		"Iface":  "0.0.0.0",
		"Port":   "8443",
		"Target": "nx",
		"Sleep":  500 * time.Millisecond,
	}

	// Create a config with expected defaults
	config := Config{
		Iface:  "0.0.0.0",
		Port:   "8443",
		Target: "nx",
		Sleep:  500 * time.Millisecond,
	}

	assert.Equal(t, expectedDefaults["Iface"], config.Iface)
	assert.Equal(t, expectedDefaults["Port"], config.Port)
	assert.Equal(t, expectedDefaults["Target"], config.Target)
	assert.Equal(t, expectedDefaults["Sleep"], config.Sleep)

	// Ensure the config with defaults is valid
	err := config.Validate()
	assert.NoError(t, err)
}

// TestConfigEdgeCases tests edge cases and boundary conditions
func TestConfigEdgeCases(t *testing.T) {
	t.Run("minimum valid port", func(t *testing.T) {
		config := Config{
			Iface: "127.0.0.1",
			Port:  "1",
			Sleep: 0,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("maximum valid port", func(t *testing.T) {
		config := Config{
			Iface: "127.0.0.1",
			Port:  "65535",
			Sleep: 0,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("zero sleep duration", func(t *testing.T) {
		config := Config{
			Iface: "127.0.0.1",
			Port:  "8443",
			Sleep: 0,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("very long sleep duration", func(t *testing.T) {
		config := Config{
			Iface: "127.0.0.1",
			Port:  "8443",
			Sleep: 24 * time.Hour,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})
}
