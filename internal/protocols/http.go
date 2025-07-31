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
// Common constants for protocol handlers
const (
	// HTTP method prefixes for protocol detection
	httpMethodGET     = "GET "
	httpMethodPOST    = "POST"
	httpMethodPUT     = "PUT "
	httpMethodDELETE  = "DELE"
	httpMethodHEAD    = "HEAD"
	httpMethodOPTIONS = "OPTI"
	httpMethodPATCH   = "PATC"
	httpMethodCONNECT = "CONN"
	
	// SSH protocol identifier
	
	// Minimum data length for protocol detection
	minProtocolDataLength = 4
	
	// HTTP timeouts
)



type HTTPHandler struct {
	serveDir      string
	serverAddress string
	log           logerr.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(serveDir, serverAddress string) *HTTPHandler {
	return &HTTPHandler{
		serveDir:      serveDir,
		serverAddress: serverAddress,
		log:           logerr.Add("http"),
	}
}



// Handle processes HTTP connections
func (h *HTTPHandler) Handle(conn net.Conn) error {
	server := h.createServer()
	return server.Serve(&singleConnListener{conn: conn})
}

// HandleListener handles connections from a listener (for HTTPS CONNECT)
func (h *HTTPHandler) HandleListener(ctx context.Context, listener net.Listener) error {
	handler := h.createConnectionHandler()
	return common.HandleListenerLoop(ctx, listener, handler, h.log, "HTTP")
}

// createConnectionHandler creates a connection handler function with proper error handling
func (h *HTTPHandler) createConnectionHandler() func(net.Conn) error {
	return func(conn net.Conn) error {
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
	return &http.Server{
		Handler: http.HandlerFunc(h.routeRequest),
		// Timeouts commented out for compatibility
		// ReadTimeout:  30 * time.Second,
		// WriteTimeout: 30 * time.Second,
		// IdleTimeout:  60 * time.Second,
	}
}

// routeRequest routes HTTP requests to appropriate handlers based on request type
func (h *HTTPHandler) routeRequest(w http.ResponseWriter, r *http.Request) {
	// Handle HTTPS CONNECT requests for tunneling
	if r.Method == "CONNECT" {
		h.log.Info("HTTPS CONNECT request:", r.Host)
		h.handleHTTPSProxy(w, r)
		return
	}

	// Determine if this should be handled as a direct file request or proxy request
	if h.shouldHandleAsDirectRequest(r) {
		h.handleLocalFile(w, r)
		return
	}

	// Handle as HTTP proxy request
	h.log.Info("HTTP proxy request:", r.Method, r.URL.String())
	h.handleProxy(w, r)
}

// shouldHandleAsDirectRequest determines if a request should be handled as a direct file request
func (h *HTTPHandler) shouldHandleAsDirectRequest(r *http.Request) bool {
	// Smart proxy detection: check if this should be handled as a direct request
	// even if the URL is absolute (e.g., curl with http_proxy set)
	if h.isDirectRequest(r) {
		return true
	}

	// If URL is not absolute, it's definitely a direct request
	if !r.URL.IsAbs() {
		return true
	}

	// Check for proxy-related headers - if present, handle as proxy
	if h.hasProxyHeaders(r) {
		return false
	}

	// Default to direct request for non-absolute URLs
	return !r.URL.IsAbs()
}

// hasProxyHeaders checks if the request contains proxy-specific headers
func (h *HTTPHandler) hasProxyHeaders(r *http.Request) bool {
	return r.Header.Get("Proxy-Connection") != "" || 
		   r.Header.Get("Proxy-Authorization") != ""
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

// isDirectRequest determines if a request should be handled as a direct file request
// rather than a proxy request, even if the URL is absolute
func (h *HTTPHandler) isDirectRequest(r *http.Request) bool {
	if !r.URL.IsAbs() {
		return true // Relative URLs are always direct requests
	}

	// If the request is for our server address, treat as direct request
	return r.URL.Host == h.serverAddress
}

// handleLocalFile handles a request as a local file request, converting absolute URLs to relative paths
func (h *HTTPHandler) handleLocalFile(w http.ResponseWriter, r *http.Request) {
	var path string
	if r.URL.IsAbs() {
		// Extract path from absolute URL
		path = r.URL.Path
		h.log.Info("HTTP file request (absolute URL converted):", r.Method, path)
	} else {
		path = r.URL.Path
		h.log.Info("HTTP file request:", r.Method, path)
	}

	if h.serveDir != "" {
		// Create a new request with relative path for the file server
		relativeReq := r.Clone(r.Context())
		relativeReq.URL.Scheme = ""
		relativeReq.URL.Host = ""
		relativeReq.URL.Path = path

		fileServer := http.FileServer(http.Dir(h.serveDir))
		fileServer.ServeHTTP(w, relativeReq)
	} else {
		h.log.Warn("HTTP request received but no serve directory specified")
		http.Error(w, "File serving not enabled", http.StatusServiceUnavailable)
	}
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
