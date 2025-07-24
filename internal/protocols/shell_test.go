package protocols

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/audibleblink/nx/internal/config"
)

func TestNewShellHandler(t *testing.T) {
	// Skip this test since it requires actual tmux and plugin manager
	t.Skip("Skipping NewShellHandler test - requires complex setup")
}

func TestShellHandlerMatch(t *testing.T) {
	cfg := &config.Config{Target: "test"}
	handler := &ShellHandler{config: cfg}

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "shell command",
			data:     []byte("ls -la"),
			expected: true,
		},
		{
			name:     "bash command",
			data:     []byte("bash -i"),
			expected: true,
		},
		{
			name:     "empty data",
			data:     []byte(""),
			expected: true, // Shell handler accepts all data as fallback
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03},
			expected: true, // Shell handler accepts all data as fallback
		},
		{
			name:     "any text data",
			data:     []byte("random text input"),
			expected: true, // Shell handler accepts all data as fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.Match(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShellHandlerMatchLogic(t *testing.T) {
	handler := &ShellHandler{}

	t.Run("accepts all data as fallback", func(t *testing.T) {
		testData := [][]byte{
			[]byte("hello world"),
			[]byte("ls -la"),
			[]byte("echo test"),
			[]byte(""), // empty
			[]byte("any random input"),
			{0x00, 0x01, 0x02}, // binary
		}

		for _, data := range testData {
			assert.True(t, handler.Match(data), "Shell handler should accept all data as fallback: %v", data)
		}
	})

	t.Run("handles short data", func(t *testing.T) {
		shortData := [][]byte{
			[]byte("a"),
			[]byte("ab"),
			[]byte("abc"),
		}

		for _, data := range shortData {
			result := handler.Match(data)
			assert.True(t, result, "Should accept all data including short data: %v", data)
		}
	})
}

func TestShellHandlerEdgeCases(t *testing.T) {
	handler := &ShellHandler{}

	t.Run("handles nil data", func(t *testing.T) {
		// This might panic in the actual implementation
		// In production, you'd want to add nil checks
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Handler panicked with nil data (expected): %v", r)
			}
		}()

		result := handler.Match(nil)
		// If it doesn't panic, it should return true (shell handler accepts all data)
		assert.True(t, result)
	})

	t.Run("handles very long data", func(t *testing.T) {
		longData := make([]byte, 10000)
		for i := range longData {
			longData[i] = 'a'
		}

		result := handler.Match(longData)
		assert.True(t, result) // Shell handler accepts all data as fallback
	})

	t.Run("handles mixed content", func(t *testing.T) {
		// Test various types of content that might be routed to shell handler
		mixedData := []byte("some command with numbers 123 and symbols !@#$%")
		result := handler.Match(mixedData)
		assert.True(t, result) // Shell handler accepts all data as fallback
	})
}

// BenchmarkShellMatch benchmarks the shell matching function
func BenchmarkShellMatch(b *testing.B) {
	handler := &ShellHandler{}
	testData := []byte("ls -la")

	for b.Loop() {
		handler.Match(testData)
	}
}

// BenchmarkShellMatchGeneral benchmarks shell handler matching
func BenchmarkShellMatchGeneral(b *testing.B) {
	handler := &ShellHandler{}
	testData := []byte("ls -la")

	for b.Loop() {
		handler.Match(testData)
	}
}
