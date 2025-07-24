package tmux

import (
	"fmt"
	"path"

	"github.com/audibleblink/logerr"
	"github.com/disneystreaming/gomux"
)

// Manager handles tmux session and window operations
type Manager struct {
	session *gomux.Session
	log     logerr.Logger
}

// NewManager creates a new tmux manager
func NewManager(sessionName string) (*Manager, error) {
	session, err := prepareSession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tmux session: %w", err)
	}

	return &Manager{
		session: session,
		log:     logerr.Add("tmux"),
	}, nil
}

// prepareSession creates or attaches to an existing tmux session
func prepareSession(sessionName string) (*gomux.Session, error) {
	log := logerr.Add("prepareSession")

	exists, err := gomux.CheckSessionExists(sessionName)
	if err != nil {
		return nil, log.Wrap(err)
	}

	if !exists {
		log.Debug("creating new tmux session")
		return gomux.NewSession(sessionName)
	}

	// Session exists in tmux, but not tracked with server yet
	log.Debug("existing tmux session:", sessionName)
	return &gomux.Session{Name: sessionName}, nil
}

// CreateWindow creates a new tmux window for the given socket file
func (m *Manager) CreateWindow(socketFile string) (*gomux.Window, error) {
	windowName := path.Base(socketFile)
	windowID := fmt.Sprintf("%s.%d", windowName, m.session.NextWindowNumber)

	window, err := m.session.AddWindow(windowID)
	if err != nil {
		return nil, m.log.Wrap(err)
	}

	return window, nil
}

// ExecuteInWindow executes a command in the specified tmux window
func (m *Manager) ExecuteInWindow(window *gomux.Window, command string) error {
	m.log.Debug("send-keys:", command)

	if len(window.Panes) == 0 {
		return m.log.Wrap("window has no panes")
	}

	err := window.Panes[0].Exec(command)
	if err != nil {
		return m.log.Add("pane").Wrap(err)
	}

	return nil
}

// GetSession returns the underlying gomux session
func (m *Manager) GetSession() *gomux.Session {
	return m.session
}

// GetSessionName returns the session name
func (m *Manager) GetSessionName() string {
	return m.session.Name
}
