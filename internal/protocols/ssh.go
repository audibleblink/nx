package protocols

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"

	"github.com/audibleblink/logerr"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// SSHHandler handles SSH tunneling connections
type SSHHandler struct {
	password string
	server   *ssh.Server
	log      logerr.Logger
}

// NewSSHHandler creates a new SSH handler
func NewSSHHandler(password string) (*SSHHandler, error) {
	handler := &SSHHandler{
		password: password,
		log:      logerr.Add("ssh"),
	}

	server, err := handler.createServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	handler.server = server
	return handler, nil
}

// Handle processes SSH connections
func (h *SSHHandler) Handle(conn net.Conn) error {
	h.log.Debugf("connection from %s", conn.RemoteAddr().String())
	h.server.HandleConn(conn)
	return nil
}

// createServer creates and configures the SSH server
func (h *SSHHandler) createServer() (*ssh.Server, error) {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	server := &ssh.Server{
		Handler: func(s ssh.Session) {
			io.WriteString(s, "nx SSH tunneling active\n")
			<-s.Context().Done()
		},

		PasswordHandler: func(ctx ssh.Context, pass string) bool {
			return h.password == "" || pass == h.password
		},

		LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			h.log.Info("Local forward: ->", fmt.Sprintf("%s:%d", dhost, dport))
			return true
		},

		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			h.log.Info("Remote forward:", fmt.Sprintf("%s:%d <-", bindHost, bindPort))
			return true
		},

		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
	}

	hostKey, err := h.generateHostKey()
	if err != nil {
		return nil, err
	}

	server.AddHostKey(hostKey)
	return server, nil
}

// generateHostKey generates and returns an SSH host key
func (h *SSHHandler) generateHostKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}

	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}
