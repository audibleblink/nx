package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration

type Config struct {
	Auto             bool          `long:"auto"              description:"Attempt to auto-upgrade to a tty (uses --exec auto)"`
	Exec             string        `long:"exec"              description:"Execute plugin scripts on connection (comma-separated)"`
	ContinueOnError  bool          `long:"continue-on-error" description:"Continue executing remaining scripts if one fails"`
	ScriptTimeout    time.Duration `long:"script-timeout"    description:"Timeout per script execution" default:"30s"`
	InstallPlugins   bool          `long:"install-plugins"   description:"Install bundled plugins to config directory"`
	Iface            string        `long:"host"              description:"Interface address on which to bind"                  short:"i" default:"0.0.0.0" required:"true"`
	Port             string        `long:"port"              description:"Port on which to bind"                               short:"p" default:"8443"    required:"true"`
	ServeDir         string        `long:"serve-dir"         description:"Directory to serve files from over HTTP"             short:"d"`
	Target           string        `long:"target"            description:"Tmux session name"                                   short:"t" default:"nx"`
	Sleep            time.Duration `long:"sleep"             description:"adjust if --auto is failing"                                   default:"500ms"`
	Verbose          bool          `long:"verbose"           description:"Debug logging"                                       short:"v"`
	SSHPass          string        `long:"ssh-pass"          description:"SSH password (empty = no auth)"                      short:"s"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.validateNetwork(); err != nil {
		return err
	}
	if err := c.validateTiming(); err != nil {
		return err
	}
	if err := c.validateScripts(); err != nil {
		return err
	}
	return nil
}

// validateNetwork validates network-related configuration
func (c *Config) validateNetwork() error {
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

	return nil
}

// validateTiming validates timing-related configuration
func (c *Config) validateTiming() error {
	if c.Sleep < 0 {
		return fmt.Errorf("sleep duration cannot be negative: %v", c.Sleep)
	}
	return nil
}

// validateScripts validates script-related configuration
func (c *Config) validateScripts() error {
	if c.ScriptTimeout < 0 {
		return fmt.Errorf("script timeout cannot be negative: %v", c.ScriptTimeout)
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

// IsSSHEnabled returns true if SSH server should be enabled
// SSH is always enabled but authentication depends on SSHPass being set
func (c *Config) IsSSHEnabled() bool {
	return true // SSH server is always enabled for this application
}

// GetExecScripts returns a slice of script names from the Exec field
// Handles both single scripts and comma-separated multiple scripts
func (c *Config) GetExecScripts() []string {
	if c.Exec == "" {
		return nil
	}
	
	// Split by comma and trim whitespace
	scripts := make([]string, 0)
	for _, script := range strings.Split(c.Exec, ",") {
		script = strings.TrimSpace(script)
		if script != "" {
			scripts = append(scripts, script)
		}
	}
	
	// Return nil if no valid scripts found
	if len(scripts) == 0 {
		return nil
	}
	
	return scripts
}

// HasExecScripts returns true if any execution scripts are configured
func (c *Config) HasExecScripts() bool {
	return len(c.GetExecScripts()) > 0
}



// ServerCommand represents the configuration for the server subcommand (default behavior)

// ExecCommand represents the configuration for the exec subcommand
type ExecCommand struct {
	Args struct {
		Scripts string `positional-arg-name:"scripts" description:"Name(s) of plugin script(s) to execute (comma-separated)"`
	} `positional-args:"yes" required:"yes"`
	On              string        `                      required:"true" long:"on"              description:"Target pane using tmux notation (session:window.pane)"`
	DryRun          bool          `                                      long:"dry-run"         description:"Preview execution without running"`
	ContinueOnError bool          `                                      long:"continue-on-error" description:"Continue executing remaining scripts if one fails"`
	ScriptTimeout   time.Duration `                                      long:"script-timeout"  description:"Timeout per script execution" default:"30s"`
}

// Commands represents the available subcommands
type Commands struct {
	Server Config      `command:"server" description:"Start nx server (default)"`
	Exec   ExecCommand `command:"exec"   description:"Execute script on existing panes"`
}

// GetScripts returns a slice of script names from the Scripts field
// Handles both single scripts and comma-separated multiple scripts
func (e *ExecCommand) GetScripts() []string {
	if e.Args.Scripts == "" {
		return nil
	}
	
	// Split by comma and trim whitespace
	scripts := make([]string, 0)
	for _, script := range strings.Split(e.Args.Scripts, ",") {
		script = strings.TrimSpace(script)
		if script != "" {
			scripts = append(scripts, script)
		}
	}
	
	// Return nil if no valid scripts found
	if len(scripts) == 0 {
		return nil
	}
	
	return scripts
}

// ServerCommand methods to maintain compatibility with existing Config interface
