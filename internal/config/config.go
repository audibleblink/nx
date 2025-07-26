package config

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// Config holds all application configuration

type Config struct {
	Auto           bool          `long:"auto"            description:"Attempt to auto-upgrade to a tty (uses --exec auto)"`
	Exec           string        `long:"exec"            description:"Execute plugin script on connection"`
	InstallPlugins bool          `long:"install-plugins" description:"Install bundled plugins to config directory"`
	Iface          string        `long:"host"            description:"Interface address on which to bind"                  short:"i" default:"0.0.0.0" required:"true"`
	Port           string        `long:"port"            description:"Port on which to bind"                               short:"p" default:"8443"    required:"true"`
	ServeDir       string        `long:"serve-dir"       description:"Directory to serve files from over HTTP"             short:"d"`
	Target         string        `long:"target"          description:"Tmux session name"                                   short:"t" default:"nx"`
	Sleep          time.Duration `long:"sleep"           description:"adjust if --auto is failing"                                   default:"500ms"`
	Verbose        bool          `long:"verbose"         description:"Debug logging"                                       short:"v"`
	SSHPass        string        `long:"ssh-pass"        description:"SSH password (empty = no auth)"                      short:"s"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate interface address
	if c.Iface != "" {
		if ip := net.ParseIP(c.Iface); ip == nil {
			return fmt.Errorf("invalid interface address: %s", c.Iface)
		}
	}

	// Validate port
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %s", c.Port)
	}

	// Validate sleep duration
	if c.Sleep < 0 {
		return fmt.Errorf("sleep duration cannot be negative: %v", c.Sleep)
	}

	return nil
}

// Address returns the full address string for binding
func (c *Config) Address() string {
	return net.JoinHostPort(c.Iface, c.Port)
}

// IsHTTPEnabled returns true if HTTP file serving is enabled
func (c *Config) IsHTTPEnabled() bool {
	return c.ServeDir != ""
}

// IsSSHEnabled returns true if SSH server is enabled
func (c *Config) IsSSHEnabled() bool {
	return true
}



// ServerCommand represents the configuration for the server subcommand (default behavior)

// ExecCommand represents the configuration for the exec subcommand
type ExecCommand struct {
	Args struct {
		Script string `positional-arg-name:"script" description:"Name of plugin script to execute"`
	} `positional-args:"yes" required:"yes"`
	On     string `                      required:"true" long:"on"      description:"Target pane using tmux notation (session:window.pane)"`
	DryRun bool   `                                      long:"dry-run" description:"Preview execution without running"`
}

// Commands represents the available subcommands
type Commands struct {
	Server Config      `command:"server" description:"Start nx server (default)"`
	Exec   ExecCommand `command:"exec"   description:"Execute script on existing panes"`
}

// ServerCommand methods to maintain compatibility with existing Config interface
