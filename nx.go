package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/audibleblink/logerr"
	"github.com/disneystreaming/gomux"
	"github.com/jessevdk/go-flags"
)

var (
	socketDir = filepath.Join(xdg.RuntimeDir, "nx")
	pluginDir = filepath.Join(xdg.ConfigHome, "nx", "plugins")
	session   *gomux.Session
	opts      struct {
		Auto    bool          `long:"auto" description:"Attempt to auto-upgrade to a tty (deprecated: use --exec auto)"`
		Exec    string        `long:"exec" description:"Execute plugin script on connection"`
		Iface   string        `short:"i" long:"host" description:"Interface address on which to bind" default:"0.0.0.0" required:"true"`
		Port    string        `short:"p" long:"port" description:"Port on which to bind" default:"8443" required:"true"`
		Target  string        `short:"t" long:"target" description:"Tmux session name" default:"nx"`
		Verbose bool          `short:"v" long:"verbose" description:"Debug logging"`
		Sleep   time.Duration `long:"sleep" description:"adjust if --auto is failing" default:"500ms"`
	}
)

func main() {
	// create revshell listener
	connStr := fmt.Sprintf("%s:%s", opts.Iface, opts.Port)
	listener, err := net.Listen("tcp", connStr)
	if err != nil {
		logerr.Fatal("listener:", err)
	}
	logerr.Info("listening on:", connStr)

	for {
		logerr.Debug("waiting on new connection")
		conn, err := listener.Accept()
		if err != nil {
			logerr.Fatal("conn:", err)
		}
		logerr.Info("new connection:", conn.RemoteAddr().String())

		// create the unix domain socket filename
		sockF, err := genTempFilename()
		if err != nil {
			logerr.Error("gen filename:", err)
			continue
		}

		// create the unix domain socket
		sockH, err := net.Listen("unix", sockF)
		if err != nil {
			logerr.Error("socket create:", err)
			continue
		}
		logerr.Debug("socket file created:", sockF)

		// background: wait and listen for a connection to the domain socket
		go handleTCPUnix(conn, sockH)

		// create a tmux window for the reverse shell to run in
		window, err := newTmuxWindow(session, sockF)
		if err != nil {
			logerr.Error("tmux window create:", err)
			continue
		}

		// create the tmux command to run in the new window
		// intentional space prefix to keep shell history clean
		cmd := fmt.Sprintf(" socat -d -d stdio unix-connect:'%s'", sockF)
		err = execInWindow(window, cmd)
		if err != nil {
			logerr.Error("tmux exec:", err)
			continue
		}

		logerr.Info("new shell:", conn.RemoteAddr().String())

		// set env var back home for convenience
		time.Sleep(opts.Sleep)
		_ = execInWindow(window, fmt.Sprintf(" export ME=%s", connStr))

		// Handle plugin execution
		if opts.Auto {
			opts.Exec = "auto" // backward compatibility
		}

		if opts.Exec != "" {
			err := executePlugin(opts.Exec, window)
			if err != nil {
				logerr.Error("plugin execution:", err)
			}
		}
	}
}

// handleTCPUnix handles the connection between the network and the unix domain socket
func handleTCPUnix(httpConn net.Conn, domainSocket net.Listener) error {
	log := logerr.Add("handleTCPUnix")
	defer domainSocket.Close()
	netC, sockC := make(chan error), make(chan error)

	socketConn, err := domainSocket.Accept()
	if err != nil {
		log.Warn("socket connection:", err)
		return err
	}
	defer socketConn.Close()

	// stdio from network
	go func() {
		_, err := io.Copy(socketConn, httpConn)
		netC <- err
	}()

	// stdio from us/socat
	go func() {
		_, err := io.Copy(httpConn, socketConn)
		sockC <- err
	}()

	// Wait for either goroutine to finish and return any error
	select {
	case err = <-netC:
		log.Warn("shell died:", err)
	case err = <-sockC:
		log.Warn("tmux died:", err)
	}
	return err
}

// create tempfile name. socket file can't exists when we start
// the listener, so we delete it immediately
// i'm using it for the convenience of getting abs paths
func genTempFilename() (string, error) {
	file, err := os.CreateTemp(socketDir, "*.sock")
	if err != nil {
		return "", err
	}
	file.Close()
	os.Remove(file.Name())
	return file.Name(), err
}

// Handles tmux session existance
func prepareTmux(tmSessName string) (tmux *gomux.Session, err error) {
	log := logerr.Add("prepareTmux")
	exists, err := gomux.CheckSessionExists(tmSessName)
	if err != nil {
		return
	}

	if !exists {
		log.Debug("creating new tmux session")
		return gomux.NewSession(tmSessName)
	}

	// session is in tmux, but not tracked with server yet
	log.Debug("existing tmux session:", opts.Target)
	tmux = &gomux.Session{Name: tmSessName}
	return
}

// newTmuxWindow creates a new tmux window based on the socket file
func newTmuxWindow(session *gomux.Session, socketFile string) (window *gomux.Window, err error) {
	tmWindowName := path.Base(socketFile)
	id := fmt.Sprintf("%s.%d", tmWindowName, session.NextWindowNumber)
	return session.AddWindow(id)
}

// execInWindow executes a command in the tmux window
func execInWindow(window *gomux.Window, cmd string) error {
	logerr.Debug("tmux command:", cmd)
	return window.Panes[0].Exec(cmd) // new windows always have a 0-index pane
}

// executePlugin executes commands from a plugin script
func executePlugin(pluginName string, window *gomux.Window) error {
	log := logerr.Add("executePlugin")

	pluginPath := filepath.Join(pluginDir, pluginName+".sh")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin not found: %s", pluginPath)
	}

	file, err := os.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		log.Debug("executing plugin command:", line)
		err := execInWindow(window, line)
		if err != nil {
			log.Warn("plugin command failed:", err)
		}

		// Default sleep between commands
		time.Sleep(opts.Sleep)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading plugin: %w", err)
	}

	log.Info("Plugin executed:", pluginName)
	return nil
}

func init() {
	var err error
	if _, err := flags.Parse(&opts); err != nil {
		logerr.Fatal(err)
	}

	logerr.DefaultLogger().
		EnableTimestamps().
		EnableColors().
		SetContextSeparator(" ❯ ").
		SetContext("nx").
		SetLogLevel(logerr.LogLevelInfo).
		SetAsGlobal()

	if opts.Verbose {
		logerr.SetLogLevel(logerr.LogLevelDebug)
	}

	// Ensure socket folder exists
	if _, err := os.Stat(socketDir); os.IsNotExist(err) {
		os.Mkdir(socketDir, 0o700)
	}

	// Ensure plugin folder exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		os.MkdirAll(pluginDir, 0o755)
	}

	session, err = prepareTmux(opts.Target)
	if err != nil {
		logerr.Add("tmux").Fatal(err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Start cleanup goroutine
	go func() {
		sig := <-sigChan
		logerr.Info("Received interrupt signal:", sig)
		cleanup()
		os.Exit(0)
	}()
}

// cleanup removes any socket files and performs other cleanup tasks
func cleanup() {
	err := os.RemoveAll(socketDir)
	if err != nil {
		logerr.Error("unable to delete:", err)
	}
}
