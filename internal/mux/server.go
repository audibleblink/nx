package mux

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/audibleblink/logerr"
	"github.com/soheilhy/cmux"

	"github.com/audibleblink/nx/internal/common"
	"github.com/audibleblink/nx/internal/config"
	"github.com/audibleblink/nx/internal/protocols"
)

// Server manages connection multiplexing
type Server struct {
	config       *config.Config
	listener     net.Listener
	mux          cmux.CMux
	httpHandler  *protocols.HTTPHandler
	sshHandler   *protocols.SSHHandler
	shellHandler *protocols.ShellHandler
	log          logerr.Logger
}

// NewServer creates a new multiplexer server
func NewServer(
	cfg *config.Config,
	httpHandler *protocols.HTTPHandler,
	sshHandler *protocols.SSHHandler,
	shellHandler *protocols.ShellHandler,
) (*Server, error) {
	// Create TCP listener
	listener, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Create connection multiplexer
	mux := cmux.New(listener)
	mux.SetReadTimeout(1 * time.Second) // Set read timeout to avoid hanging on protocol detection

	return &Server{
		config:       cfg,
		listener:     listener,
		mux:          mux,
		httpHandler:  httpHandler,
		sshHandler:   sshHandler,
		shellHandler: shellHandler,
		log:          logerr.Add("mux"),
	}, nil
}

// Start starts the multiplexer server
func (s *Server) Start(ctx context.Context) error {
	// Create matchers for different protocols
	sshL := s.mux.Match(cmux.PrefixMatcher("SSH-"))

	// HTTP matcher includes standard HTTP and WebDAV methods
	httpMethods := []string{
		"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", // Standard HTTP
		"PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", // WebDAV
	}
	httpL := s.mux.Match(cmux.HTTP1Fast(httpMethods...))

	shellL := s.mux.Match(cmux.Any())

	// Start HTTP handler (handles file serving, HTTP proxy, and HTTPS CONNECT)
	if s.httpHandler != nil {
		go func() {
			s.log.Debug("HTTP handler starting")
			if err := s.httpHandler.HandleListener(ctx, httpL); err != nil {
				if errors.Is(err, context.Canceled) {
					s.log.Debug("HTTP handler shutting down")
					return
				}
				s.log.Error("HTTP handler error:", err)
			}
		}()
	}

	// Start SSH server if enabled
	if s.config.IsSSHEnabled() && s.sshHandler != nil {
		go func() {
			s.log.Debug("SSH Tunneling enabled. pass:", s.config.SSHPass != "")
			if err := common.HandleListenerLoop(ctx, sshL, s.sshHandler.Handle, s.log, "SSH"); err != nil {
				s.log.Debug("SSH listener loop ended:", err)
			}
		}()
	}

	// Start shell handler
	go func() {
		if err := s.shellHandler.HandleListener(ctx, shellL); err != nil {
			if errors.Is(err, context.Canceled) {
				s.log.Debug("Shell handler shutting down")
				return
			}
			s.log.Error("Shell handler error:", err)
		}
	}()

	s.log.Info("Starting connection multiplexer on:", s.config.Address())
	err := s.mux.Serve()
	return s.log.Add("Start").Wrap(err)
}

// Stop stops the multiplexer server
func (s *Server) Stop() error {
	return s.listener.Close()
}
