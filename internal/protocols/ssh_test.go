package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSHHandler(t *testing.T) {
	t.Run("creates handler with password", func(t *testing.T) {
		handler, err := NewSSHHandler("testpassword")
		require.NoError(t, err)
		require.NotNil(t, handler)

		assert.Equal(t, "testpassword", handler.password)
		assert.NotNil(t, handler.server)
		assert.NotNil(t, handler.log)
	})

	t.Run("creates handler without password (empty)", func(t *testing.T) {
		handler, err := NewSSHHandler("")
		require.NoError(t, err)
		require.NotNil(t, handler)

		assert.Equal(t, "", handler.password)
		assert.NotNil(t, handler.server)
	})

	t.Run("creates handler with complex password", func(t *testing.T) {
		complexPass := "p@$$w0rd!#%^&*()_+-=[]{}|;':\",./<>?"
		handler, err := NewSSHHandler(complexPass)
		require.NoError(t, err)
		require.NotNil(t, handler)

		assert.Equal(t, complexPass, handler.password)
	})
}

func TestSSHHandlerGenerateHostKey(t *testing.T) {
	t.Run("generates valid RSA host key", func(t *testing.T) {
		handler := &SSHHandler{}
		signer, err := handler.generateHostKey()
		require.NoError(t, err)
		require.NotNil(t, signer)

		// Verify the signer has a valid public key
		pubKey := signer.PublicKey()
		require.NotNil(t, pubKey)

		// Verify key type is RSA
		assert.Equal(t, "ssh-rsa", pubKey.Type())
	})

	t.Run("generates unique keys on each call", func(t *testing.T) {
		handler := &SSHHandler{}

		key1, err := handler.generateHostKey()
		require.NoError(t, err)

		key2, err := handler.generateHostKey()
		require.NoError(t, err)

		// Public keys should be different
		pub1 := key1.PublicKey().Marshal()
		pub2 := key2.PublicKey().Marshal()
		assert.NotEqual(t, pub1, pub2, "each call should generate a unique key")
	})
}

func TestSSHHandlerPasswordAuthentication(t *testing.T) {
	t.Run("accepts correct password", func(t *testing.T) {
		handler, err := NewSSHHandler("correctpassword")
		require.NoError(t, err)

		// Get the password handler from the server
		pwHandler := handler.server.PasswordHandler
		require.NotNil(t, pwHandler)

		// Test with correct password
		result := pwHandler(nil, "correctpassword")
		assert.True(t, result, "correct password should be accepted")
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		handler, err := NewSSHHandler("correctpassword")
		require.NoError(t, err)

		pwHandler := handler.server.PasswordHandler
		require.NotNil(t, pwHandler)

		// Test with wrong password
		result := pwHandler(nil, "wrongpassword")
		assert.False(t, result, "incorrect password should be rejected")
	})

	t.Run("accepts any password when password is empty", func(t *testing.T) {
		handler, err := NewSSHHandler("")
		require.NoError(t, err)

		pwHandler := handler.server.PasswordHandler
		require.NotNil(t, pwHandler)

		// Any password should work when server password is empty
		assert.True(t, pwHandler(nil, "anypassword"))
		assert.True(t, pwHandler(nil, ""))
		assert.True(t, pwHandler(nil, "randomstring123"))
	})

	t.Run("empty password matches empty input", func(t *testing.T) {
		handler, err := NewSSHHandler("")
		require.NoError(t, err)

		pwHandler := handler.server.PasswordHandler
		result := pwHandler(nil, "")
		assert.True(t, result)
	})

	t.Run("case sensitive password", func(t *testing.T) {
		handler, err := NewSSHHandler("Password123")
		require.NoError(t, err)

		pwHandler := handler.server.PasswordHandler

		assert.True(t, pwHandler(nil, "Password123"))
		assert.False(t, pwHandler(nil, "password123"))
		assert.False(t, pwHandler(nil, "PASSWORD123"))
	})
}

func TestSSHHandlerPortForwardingCallbacks(t *testing.T) {
	t.Run("local port forwarding callback configured", func(t *testing.T) {
		handler, err := NewSSHHandler("test")
		require.NoError(t, err)

		// Verify local port forwarding callback is set
		callback := handler.server.LocalPortForwardingCallback
		require.NotNil(t, callback)

		// Callback should always return true (allowing the forward)
		result := callback(nil, "localhost", 8080)
		assert.True(t, result)

		result = callback(nil, "192.168.1.1", 22)
		assert.True(t, result)
	})

	t.Run("reverse port forwarding callback configured", func(t *testing.T) {
		handler, err := NewSSHHandler("test")
		require.NoError(t, err)

		// Verify reverse port forwarding callback is set
		callback := handler.server.ReversePortForwardingCallback
		require.NotNil(t, callback)

		// Callback should always return true
		result := callback(nil, "0.0.0.0", 9000)
		assert.True(t, result)

		result = callback(nil, "127.0.0.1", 3000)
		assert.True(t, result)
	})
}

func TestSSHHandlerRequestHandlers(t *testing.T) {
	t.Run("tcpip-forward handler configured", func(t *testing.T) {
		handler, err := NewSSHHandler("test")
		require.NoError(t, err)

		requestHandlers := handler.server.RequestHandlers
		require.NotNil(t, requestHandlers)

		_, exists := requestHandlers["tcpip-forward"]
		assert.True(t, exists, "tcpip-forward handler should be configured")
	})

	t.Run("cancel-tcpip-forward handler configured", func(t *testing.T) {
		handler, err := NewSSHHandler("test")
		require.NoError(t, err)

		requestHandlers := handler.server.RequestHandlers
		require.NotNil(t, requestHandlers)

		_, exists := requestHandlers["cancel-tcpip-forward"]
		assert.True(t, exists, "cancel-tcpip-forward handler should be configured")
	})
}

func TestSSHHandlerHandle(t *testing.T) {
	t.Run("accepts connection without error", func(t *testing.T) {
		handler, err := NewSSHHandler("test")
		require.NoError(t, err)

		// Create a pipe to simulate network connection
		server, client := net.Pipe()
		defer client.Close()

		// Handle connection in goroutine (it will block waiting for SSH handshake)
		done := make(chan error, 1)
		go func() {
			done <- handler.Handle(server)
		}()

		// Close client to end the connection
		client.Close()

		// Should complete without panic
		select {
		case err := <-done:
			// Handle returns nil even if connection closes
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Handle did not complete")
		}
	})
}

func TestSSHHandlerCreationConcurrent(t *testing.T) {
	t.Run("can create multiple handlers concurrently", func(t *testing.T) {
		const numHandlers = 10
		results := make(chan error, numHandlers)

		for i := 0; i < numHandlers; i++ {
			go func(idx int) {
				_, err := NewSSHHandler("password")
				results <- err
			}(i)
		}

		// All should succeed
		for i := 0; i < numHandlers; i++ {
			select {
			case err := <-results:
				assert.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for handler creation")
			}
		}
	})
}
