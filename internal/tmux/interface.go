package tmux

import "github.com/disneystreaming/gomux"

// TmuxManager defines the interface for tmux operations
type TmuxManager interface {
	CreateWindow(socketFile string) (*gomux.Window, error)
	ExecuteInWindow(window *gomux.Window, command string) error
	GetSession() *gomux.Session
	GetSessionName() string
}
