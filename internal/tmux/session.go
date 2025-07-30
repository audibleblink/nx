package tmux

import (
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/audibleblink/logerr"
	"github.com/disneystreaming/gomux"
)

// Manager handles tmux session and window operations
// PaneTarget represents a specific tmux pane
// TmuxManager defines the interface for tmux operations
type TmuxManager interface {
	CreateWindow(socketFile string) (*gomux.Window, error)
	ExecuteInWindow(window *gomux.Window, command string) error
	GetSession() *gomux.Session
	GetSessionName() string
}

type PaneTarget struct {
	Session string
	Window  int
	Pane    int
}

// PaneInfo contains information about a tmux pane
type PaneInfo struct {
	Target      PaneTarget
	Active      bool
	WindowName  string
	SessionName string
}

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

	// Use direct tmux send-keys instead of gomux Exec to avoid quote escaping
	// Get the pane target from the window
	targetStr := fmt.Sprintf("%s:%d.0", m.session.Name, window.Number)
	
	// Use tmux send-keys directly, same as ExecuteOnPane
	cmd := exec.Command("tmux", "send-keys", "-t", targetStr, command, "C-m")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command in window %s: %w", targetStr, err)
	}

	m.log.Debug("Executed command in window", targetStr, ":", command)
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

// ParseTarget parses a target string in the format "session:window.pane"
func (m *Manager) ParseTarget(target string) (*PaneTarget, error) {
	// Regular expression to match session:window.pane format
	re := regexp.MustCompile(`^([^:]+):(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(target)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid target format '%s', expected 'session:window.pane'", target)
	}

	window, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid window number '%s'", matches[2])
	}

	pane, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid pane number '%s'", matches[3])
	}

	return &PaneTarget{
		Session: matches[1],
		Window:  window,
		Pane:    pane,
	}, nil
}

// ListPanes returns information about all available panes
func (m *Manager) ListPanes() ([]PaneInfo, error) {
	// Use tmux list-panes command to get all panes
	cmd := exec.Command(
		"tmux",
		"list-panes",
		"-a",
		"-F",
		"#{session_name}:#{window_index}.#{pane_index}|#{pane_active}|#{window_name}",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	var panes []PaneInfo
	lines := strings.SplitSeq(strings.TrimSpace(string(output)), "\n")

	for line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}

		target, err := m.ParseTarget(parts[0])
		if err != nil {
			continue // Skip invalid targets
		}

		active := parts[1] == "1"
		windowName := parts[2]

		panes = append(panes, PaneInfo{
			Target:      *target,
			Active:      active,
			WindowName:  windowName,
			SessionName: target.Session,
		})
	}

	return panes, nil
}

// ExecuteOnPane executes a command on a specific pane
func (m *Manager) ExecuteOnPane(target *PaneTarget, command string) error {
	targetStr := fmt.Sprintf("%s:%d.%d", target.Session, target.Window, target.Pane)

	// Use tmux send-keys to execute the command on the specific pane
	// We need to be careful with quote handling - pass the command as a single argument
	// to avoid Go's exec package from escaping quotes
	cmd := exec.Command("tmux", "send-keys", "-t", targetStr, command, "C-m")
	
	// Set the command to not escape shell metacharacters
	// by ensuring the command string is passed exactly as-is
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command on pane %s: %w", targetStr, err)
	}

	m.log.Debug("Executed command on pane", targetStr, ":", command)
	return nil
}

// ValidatePane checks if a pane exists and is accessible
func (m *Manager) ValidatePane(target *PaneTarget) error {
	targetStr := fmt.Sprintf("%s:%d.%d", target.Session, target.Window, target.Pane)

	// Use tmux display-message to check if the pane exists
	cmd := exec.Command("tmux", "display-message", "-t", targetStr, "-p", "#{pane_id}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"pane %s does not exist or is not accessible: %s",
			targetStr,
			string(output),
		)
	}

	// Check if we got a valid pane ID (should start with %)
	outputStr := strings.TrimSpace(string(output))
	if !strings.HasPrefix(outputStr, "%") {
		return fmt.Errorf("pane %s returned invalid pane ID: %s", targetStr, outputStr)
	}

	return nil
}
