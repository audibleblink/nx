package protocols

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/audibleblink/logerr"

	"github.com/audibleblink/nx/internal/common"
)

// HTTPHandler handles HTTP file serving and proxy requests
type HTTPHandler struct {
	serveDir string
	log      logerr.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(serveDir string) *HTTPHandler {
	return &HTTPHandler{
		serveDir: serveDir,
		log:      logerr.Add("http"),
	}
}

// Match checks if the connection data matches HTTP protocol
// NOTE: This method is not used in production - cmux handles protocol detection
// It exists only for backward compatibility with tests
func (h *HTTPHandler) Match(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Check for HTTP methods
	return string(data[:4]) == "GET " ||
		string(data[:4]) == "POST" ||
		string(data[:4]) == "PUT " ||
		string(data[:4]) == "DELE" ||
		string(data[:4]) == "HEAD" ||
		string(data[:4]) == "OPTI" ||
		string(data[:4]) == "PATC" ||
		string(data[:4]) == "CONN"
}

// Handle processes HTTP connections
func (h *HTTPHandler) Handle(conn net.Conn) error {
	server := h.createServer()
	return server.Serve(&singleConnListener{conn: conn})
}

// HandleListener handles connections from a listener (for HTTPS CONNECT)
func (h *HTTPHandler) HandleListener(ctx context.Context, listener net.Listener) error {
	// Custom handler that includes EOF handling
	handler := func(conn net.Conn) error {
		h.log.Debug("incoming:", conn.RemoteAddr().String())
		if err := h.Handle(conn); err != nil {
			if errors.Is(err, io.EOF) {
				h.log.Debug("conn closed")
				return nil
			}
			return err
		}
		return nil
	}

	return common.HandleListenerLoop(ctx, listener, handler, h.log, "HTTP")
}

// handleHTTPSProxy handles HTTPS CONNECT requests for tunneling
func (h *HTTPHandler) handleHTTPSProxy(w http.ResponseWriter, r *http.Request) {
	// Extract target host and port from the request
	targetConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Send 200 Connection Established response
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Hijack the connection to get raw TCP access
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Start bidirectional copying
	go func() {
		defer targetConn.Close()
		defer clientConn.Close()
		io.Copy(targetConn, clientConn)
	}()

	io.Copy(clientConn, targetConn)
}

// createServer creates an HTTP server with combined file serving and proxy functionality
func (h *HTTPHandler) createServer() *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle HTTPS CONNECT requests
		if r.Method == "CONNECT" {
			h.log.Info("HTTPS CONNECT request:", r.Host)
			h.handleHTTPSProxy(w, r)
			return
		}

		// Check if this is an HTTP proxy request (absolute URL)
		if r.URL.IsAbs() {
			h.log.Info("HTTP proxy request:", r.Method, r.URL.String())
			h.handleProxy(w, r)
			return
		}

		// Check for proxy-related headers
		if r.Header.Get("Proxy-Connection") != "" || r.Header.Get("Proxy-Authorization") != "" {
			h.log.Info("HTTP proxy request (headers):", r.Method, r.URL.String())
			h.handleProxy(w, r)
			return
		}

		// Regular HTTP request - handle file serving or return error
		if h.serveDir != "" {
			h.log.Info("HTTP file request:", r.Method, r.URL.Path)
			fileServer := http.FileServer(http.Dir(h.serveDir))
			fileServer.ServeHTTP(w, r)
		} else {
			h.log.Warn("HTTP request received but no serve directory specified")
			http.Error(w, "File serving not enabled", http.StatusServiceUnavailable)
		}
	})

	return &http.Server{
		Handler: handler,
		// ReadTimeout:  30 * time.Second,
		// WriteTimeout: 30 * time.Second,
		// IdleTimeout:  60 * time.Second,
	}
}

// handleProxy handles HTTP proxy requests
func (h *HTTPHandler) handleProxy(w http.ResponseWriter, r *http.Request) {
	// For HTTP proxy, the URL should be absolute
	if !r.URL.IsAbs() {
		http.Error(w, "Request URL must be absolute for proxy", http.StatusBadRequest)
		return
	}

	// Create a new request to the target server
	proxyReq, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers, but remove hop-by-hop headers
	for name, values := range r.Header {
		// Skip hop-by-hop headers
		if name == "Connection" || name == "Proxy-Connection" ||
			name == "Proxy-Authenticate" || name == "Proxy-Authorization" ||
			name == "Te" || name == "Trailers" || name == "Upgrade" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the request to the target server
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, let the client handle them
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// singleConnListener wraps net.Conn to implement net.Listener.
type singleConnListener struct {
	conn     net.Conn
	accepted bool
	closed   chan struct{}
}

// connWrapper wraps the connection to ensure proper cleanup
type connWrapper struct {
	net.Conn
	listener *singleConnListener
}

// Accept is here to satisfy the net.Listener interface
func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.accepted {
		// Wait for the connection to be handled, then return EOF
		<-l.closed
		return nil, &net.OpError{Op: "accept", Net: "tcp", Err: io.EOF}
	}
	l.accepted = true
	if l.closed == nil {
		l.closed = make(chan struct{})
	}

	// Return a wrapped connection that will signal completion when closed
	return &connWrapper{
		Conn:     l.conn,
		listener: l,
	}, nil
}

// Close method for connWrapper
func (c *connWrapper) Close() error {
	// Close the underlying connection
	err := c.Conn.Close()

	// Signal the listener that we're done
	if c.listener.closed != nil {
		select {
		case <-c.listener.closed:
			// Already closed
		default:
			close(c.listener.closed)
		}
	}

	return err
}

func (l *singleConnListener) Close() error {
	if l.closed != nil {
		select {
		case <-l.closed:
			// Already closed
		default:
			close(l.closed)
		}
	}
	return l.conn.Close()
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
