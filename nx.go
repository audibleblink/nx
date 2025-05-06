package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"

	"github.com/disneystreaming/gomux"
	"github.com/jessevdk/go-flags"
	"github.com/sumup-oss/go-pkgs/logger"
)

var (
	session *gomux.Session
	opts    struct {
		Iface   string `short:"i" long:"host" description:"Interface address on which to bind" default:"0.0.0.0" required:"true"`
		Port    string `short:"p" long:"port" description:"Port on which to bind" default:"8443" required:"true"`
		Target  string `short:"t" long:"target" description:"Tmux session name" default:"nx"`
		Verbose bool   `short:"v" long:"verbose" description:"Debug logging"`
	}
)

func main() {
	// create revshell listener
	connStr := fmt.Sprintf("%s:%s", opts.Iface, opts.Port)
	listener, err := net.Listen("tcp", connStr)
	if err != nil {
		logger.Fatal("listener: ", err)
	}
	logger.Info("listening on: ", connStr)

	for {
		logger.Debug("waiting on new connection")
		conn, err := listener.Accept()
		if err != nil {
			logger.Fatal("conn: ", err)
		}
		logger.Info(fmt.Sprintf("new connection: %s", conn.RemoteAddr().String()))

		// create the unix domain socket filename
		sockF, err := genTempFilename("nx")
		if err != nil {
			logger.Error("gen filename: ", err)
			continue
		}

		// create the unix domain socket
		sockH, err := net.Listen("unix", sockF)
		if err != nil {
			logger.Error("socket create: ", err)
			continue
		}
		logger.Debug(fmt.Sprintf("socket file created: %s", sockF))

		// background: wait and listen for a connection to the domain socket
		go handleTCPUnix(conn, sockH)

		// create a tmux window for the reverse shell to run in
		window, err := newTmuxWindow(session, sockF)
		if err != nil {
			logger.Error("tmux window create: ", err)
			continue
		}

		// create the tmux command to run in the new window
		cmd := fmt.Sprintf("socat -d -d stdio unix-connect:'%s'", sockF)
		err = execInWindow(window, cmd)
		if err != nil {
			logger.Error("tmux exec: ", err)
			continue
		}

		logger.Info("new shell: ", conn.RemoteAddr().String())
	}
}

// handleTCPUnix handles the connection between the network and the unix domain socket
func handleTCPUnix(httpConn net.Conn, domainSocket net.Listener) error {
	defer domainSocket.Close()
	netC, sockC := make(chan error), make(chan error)

	socketConn, err := domainSocket.Accept()
	if err != nil {
		logger.Warn("socket connection: ", err)
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
		logger.Warn("shell died: ", err)
	case err = <-sockC:
		logger.Warn("tmux died: ", err)
	}
	return err
}

// create tempfile name. socket file can't exists when we start
// the listener, so we delete it immediately
// i'm using it for the convenience of getting abs paths
func genTempFilename(stub string) (string, error) {
	// TODO: configurable state path or XDG
	file, err := os.CreateTemp(".state", fmt.Sprintf("%s.*.sock", stub))
	if err != nil {
		err = fmt.Errorf("temp file failed: %w", err)
		return "", err
	}
	file.Close()
	os.Remove(file.Name())
	return file.Name(), err
}

// Handles tmux session existance
func prepareTmux(tmSessName string) (tmux *gomux.Session, err error) {
	exists, err := gomux.CheckSessionExists(tmSessName)
	if err != nil {
		return
	}

	if !exists {
		logger.Debug("creating new tmux session")
		return gomux.NewSession(tmSessName)
	}

	// session is in tmux, but not tracked with server yet
	logger.Debug("tracking existing tmux sessions: ", opts.Target)
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
	logger.Debug("sent to tmux: ", cmd)
	return window.Panes[0].Exec(cmd) // new windows always have a 0-index pane
}

func init() {
	_, err := flags.Parse(&opts)
	switch err {
	case flags.ErrHelp:
		os.Exit(0)
	case nil:
	default:
		logger.Fatal("opts: ", err)
	}

	if opts.Verbose {
		logger.SetLevel(logger.DebugLevel)
	}

	// Ensure socket folder exists
	if _, err := os.Stat(".state"); os.IsNotExist(err) {
		os.Mkdir(".state", 0o700)
	}

	session, err = prepareTmux(opts.Target)
	if err != nil {
		logger.Fatal("tmux: ", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Start cleanup goroutine
	go func() {
		sig := <-sigChan
		logger.Info("Received interrupt signal, ", sig)
		cleanup()
		os.Exit(0)
	}()
}

// cleanup removes any socket files and performs other cleanup tasks
func cleanup() {
	os.RemoveAll(".state")
	// gomux.KillSession(opts.Target)
}
