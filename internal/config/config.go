package config

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// Config holds all application configuration
// ServerConfig interface defines the methods needed by the server
type ServerConfig interface {
	Address() string
	IsSSHEnabled() bool
	GetSSHPass() string
}

// ShellConfig interface defines the methods needed by the shell handler
type ShellConfig interface {
	GetSleep() time.Duration
	GetExec() string
	GetAuto() bool
}

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
	// return c.SSHPass != ""
	return true
}

// IsAutoUpgradeEnabled returns true if auto TTY upgrade is enabled
func (c *Config) IsAutoUpgradeEnabled() bool {
	return c.Auto
}

func (c *Config) GetSSHPass() string {
	return c.SSHPass
}

func (c *Config) GetSleep() time.Duration {
	return c.Sleep
}

func (c *Config) GetExec() string {
	return c.Exec
}

func (c *Config) GetAuto() bool {
	return c.Auto
}

func (c *Config) GetTarget() string {
	return c.Target
}

// ServerCommand represents the configuration for the server subcommand (default behavior)
type ServerCommand struct {
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
	Server ServerCommand `command:"server" description:"Start nx server (default)"`
	Exec   ExecCommand   `command:"exec"   description:"Execute script on existing panes"`
}

// ServerCommand methods to maintain compatibility with existing Config interface
func (c *ServerCommand) Validate() error {
	// Reuse existing validation logic
	cfg := &Config{
		Auto:           c.Auto,
		Exec:           c.Exec,
		InstallPlugins: c.InstallPlugins,
		Iface:          c.Iface,
		Port:           c.Port,
		ServeDir:       c.ServeDir,
		Target:         c.Target,
		Sleep:          c.Sleep,
		Verbose:        c.Verbose,
		SSHPass:        c.SSHPass,
	}
	return cfg.Validate()
}

func (c *ServerCommand) Address() string {
	cfg := &Config{Iface: c.Iface, Port: c.Port}
	return cfg.Address()
}

func (c *ServerCommand) IsHTTPEnabled() bool {
	return c.ServeDir != ""
}

func (c *ServerCommand) IsSSHEnabled() bool {
	return c.SSHPass != ""
}

func (c *ServerCommand) IsAutoUpgradeEnabled() bool {
	return c.Auto
}

func (c *ServerCommand) GetSSHPass() string {
	return c.SSHPass
}

func (c *ServerCommand) GetSleep() time.Duration {
	return c.Sleep
}

func (c *ServerCommand) GetExec() string {
	return c.Exec
}

func (c *ServerCommand) GetAuto() bool {
	return c.Auto
}
