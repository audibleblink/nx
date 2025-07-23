package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/audibleblink/logerr"
	"github.com/disneystreaming/gomux"
	"github.com/gliderlabs/ssh"
	"github.com/jessevdk/go-flags"
	"github.com/soheilhy/cmux"
	gossh "golang.org/x/crypto/ssh"
)

// Embed the plugins directory
//
//go:embed plugins/*
var bundledPlugins embed.FS

var (
	socketDir = filepath.Join(xdg.RuntimeDir, "nx")
	pluginDir = filepath.Join(xdg.ConfigHome, "nx", "plugins")
	session   *gomux.Session
	opts      struct {
		Auto           bool          `long:"auto" description:"Attempt to auto-upgrade to a tty (uses --exec auto)"`
		Exec           string        `long:"exec" description:"Execute plugin script on connection"`
		InstallPlugins bool          `long:"install-plugins" description:"Install bundled plugins to config directory"`
		Iface          string        `short:"i" long:"host" description:"Interface address on which to bind" default:"0.0.0.0" required:"true"`
		Port           string        `short:"p" long:"port" description:"Port on which to bind" default:"8443" required:"true"`
		ServeDir       string        `short:"d" long:"serve-dir" description:"Directory to serve files from over HTTP"`
		Target         string        `short:"t" long:"target" description:"Tmux session name" default:"nx"`
		Sleep          time.Duration `long:"sleep" description:"adjust if --auto is failing" default:"500ms"`
		Verbose        bool          `short:"v" long:"verbose" description:"Debug logging"`
		SSHPass        string        `short:"s" long:"ssh-pass" description:"SSH password (empty = no auth)"`
	}
)

// installBundledPlugins copies bundled plugins from the embedded filesystem to the user's config directory
func installBundledPlugins() error {
	log := logerr.Add("installBundledPlugins")

	entries, err := bundledPlugins.ReadDir("plugins")
	if err != nil {
		return fmt.Errorf("failed to read bundled plugins: %w", err)
	}

	installedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := bundledPlugins.ReadFile(filepath.Join("plugins", entry.Name()))
		if err != nil {
			log.Warn("Failed to read plugin:", entry.Name(), err)
			continue
		}

		destPath := filepath.Join(pluginDir, entry.Name())

		// Check if plugin already exists
		if _, err := os.Stat(destPath); err == nil {
			log.Info("Plugin already exists, skipping:", entry.Name())
			continue
		}

		// Write plugin file
		err = os.WriteFile(destPath, content, 0755)
		if err != nil {
			log.Warn("Failed to install plugin:", entry.Name(), err)
			continue
		}

		log.Info("Installed plugin:", entry.Name())
		installedCount++
	}

	if installedCount == 0 {
		log.Info("No new plugins to install")
	} else {
		log.Info("Successfully installed", installedCount, "plugin(s) to:", pluginDir)
	}

	return nil
}

func main() {
	// Validate serve directory if provided
	if opts.ServeDir != "" {
		if _, err := os.Stat(opts.ServeDir); os.IsNotExist(err) {
			logerr.Fatal("Serve directory does not exist:", opts.ServeDir)
		}
		logerr.Info("File serving enabled from:", opts.ServeDir)
	}

	// create listener
	connStr := fmt.Sprintf("%s:%s", opts.Iface, opts.Port)
	listener, err := net.Listen("tcp", connStr)
	if err != nil {
		logerr.Fatal("listener:", err)
	}
	logerr.Info("listening on:", connStr)

	mux := cmux.New(listener)
	mux.SetReadTimeout(2 * time.Second) // Set a read timeout to avoid hanging on protocol detection

	// Create matchers for different protocols
	httpL := mux.Match(cmux.HTTP1Fast())
	sshL := mux.Match(cmux.PrefixMatcher("SSH-"))
	shellL := mux.Match(cmux.Any())

	if opts.ServeDir != "" {
		httpServer := setupHTTPServer(opts.ServeDir)
		go func() {
			logerr.Info("HTTP server starting")
			if err := httpServer.Serve(httpL); err != nil {
				logerr.Error("HTTP server error:", err)
			}
		}()
	} else {
		go func() {
			for {
				conn, err := httpL.Accept()
				if err != nil {
					logerr.Error("HTTP listener error:", err)
					continue
				}
				logerr.Warn("HTTP request received but no serve directory specified")
				conn.Close()
			}
		}()
	}

	// Setup SSH tunneling server
	if sshServer, err := setupSSHServer(opts.SSHPass); err == nil {
		go func() {
			logerr.Info("[SSH] Tunneling enabled (pass:", opts.SSHPass != "")
			if err := sshServer.Serve(sshL); err != nil {
				logerr.Error("SSH server error:", err)
			}
		}()
	} else {
		logerr.Error("Failed to setup SSH server:", err)
	}

	go handleShellListener(shellL, connStr)

	logerr.Info("Starting connection multiplexer")
	if err := mux.Serve(); err != nil {
		logerr.Fatal("mux serve:", err)
	}
}

// setupHTTPServer creates and configures the HTTP server
func setupHTTPServer(serveDir string) *http.Server {
	fileServer := http.FileServer(http.Dir(serveDir))

	// Add logging middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logerr.Info("HTTP request:", r.Method, r.URL.Path)
		fileServer.ServeHTTP(w, r)
	})

	return &http.Server{
		Handler: handler,
		// ReadTimeout:  10 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  60 * time.Second,
	}
}

// setupSSHServer creates and configures the SSH server for tunneling
func setupSSHServer(password string) (*ssh.Server, error) {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	server := &ssh.Server{
		Handler: func(s ssh.Session) {
			io.WriteString(s, "nx SSH tunneling active\n")
			<-s.Context().Done()
		},

		PasswordHandler: func(ctx ssh.Context, pass string) bool {
			if password == "" {
				return true
			}
			return pass == password
		},

		LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			logerr.Info("[SSH] Local forward: ->", fmt.Sprintf("%s:%d", dhost, dport))
			return true
		},

		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			logerr.Info("[SSH] Remote forward:", fmt.Sprintf("%s:%d <-", bindHost, bindPort))
			return true
		},

		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
	}

	// Simple host key generation
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}
	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	server.AddHostKey(signer)

	return server, nil
}

// handleShellListener handles shell connections from the cmux listener
func handleShellListener(listener net.Listener, connStr string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logerr.Error("shell listener accept:", err)
			continue
		}

		logerr.Info("new shell connection:", conn.RemoteAddr().String())

		// Handle shell connection (existing logic from main)
		go func(conn net.Conn) {
			// create the unix domain socket filename
			sockF, err := genTempFilename()
			if err != nil {
				logerr.Error("gen filename:", err)
				conn.Close()
				return
			}

			// create the unix domain socket
			sockH, err := net.Listen("unix", sockF)
			if err != nil {
				logerr.Error("socket create:", err)
				conn.Close()
				return
			}
			logerr.Debug("socket file created:", sockF)

			// background: wait and listen for a connection to the domain socket
			go handleTCPUnix(conn, sockH)

			// create a tmux window for the reverse shell to run in
			window, err := newTmuxWindow(session, sockF)
			if err != nil {
				logerr.Error("tmux window create:", err)
				conn.Close()
				return
			}

			// create the tmux command to run in the new window
			// intentional space prefix to keep shell history clean
			cmd := fmt.Sprintf(" socat -d -d stdio unix-connect:'%s'", sockF)
			err = execInWindow(window, cmd)
			if err != nil {
				logerr.Error("tmux exec:", err)
				conn.Close()
				return
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
		}(conn)
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

	// Handle plugin installation
	if opts.InstallPlugins {
		if err := installBundledPlugins(); err != nil {
			logerr.Fatal("Failed to install plugins:", err)
		}
		os.Exit(0)
	}

	session, err = prepareTmux(opts.Target)
	if err != nil {
		logerr.Add("tmux").Fatal(err)
	}

	// Check if plugins directory is empty and suggest installation
	pluginFiles, err := os.ReadDir(pluginDir)
	if err == nil && len(pluginFiles) == 0 {
		logerr.Warn("No plugins found. Run 'nx --install-plugins' to install bundled plugins.")
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
