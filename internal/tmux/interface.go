package tmux

import "github.com/disneystreaming/gomux"

// TmuxManager defines the interface for tmux operations
type TmuxManager interface {
	CreateWindow(socketFile string) (*gomux.Window, error)
	ExecuteInWindow(window *gomux.Window, command string) error
	GetSession() *gomux.Session
	GetSessionName() string
}

// PaneTarget represents a specific tmux pane
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

// ExtendedTmuxManager extends TmuxManager with pane targeting capabilities
type ExtendedTmuxManager interface {
	TmuxManager
	ParseTarget(target string) (*PaneTarget, error)
	ListPanes() ([]PaneInfo, error)
	ExecuteOnPane(target *PaneTarget, command string) error
	ValidatePane(target *PaneTarget) error
}
