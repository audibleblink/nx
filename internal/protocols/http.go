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
	server := &http.Server{Handler: http.HandlerFunc(h.routeRequest)}
	return server.Serve(&singleConnListener{conn: conn})
}

// HandleListener handles connections from a listener (for HTTPS CONNECT)
func (h *HTTPHandler) HandleListener(ctx context.Context, listener net.Listener) error {
	handler := func(conn net.Conn) error {
		h.log.Debug("incoming:", conn.RemoteAddr().String())
		if err := h.Handle(conn); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		h.log.Debug("conn closed")
		return nil
	}
	return common.HandleListenerLoop(ctx, listener, handler, h.log, "HTTP")
}

// routeRequest routes HTTP requests to appropriate handlers based on request type
func (h *HTTPHandler) routeRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		h.log.Info("HTTPS CONNECT request:", r.Host)
		h.handleHTTPSProxy(w, r)
		return
	}

	if h.shouldServeLocally(r) {
		h.handleLocalFile(w, r)
		return
	}

	h.log.Info("HTTP proxy request:", r.Method, r.URL.String())
	h.handleProxy(w, r)
}

// shouldServeLocally determines if a request should be handled as a direct file request
func (h *HTTPHandler) shouldServeLocally(r *http.Request) bool {
	if !r.URL.IsAbs() || r.URL.Host == h.serverAddress {
		return true
	}
	return r.Header.Get("Proxy-Connection") == "" && r.Header.Get("Proxy-Authorization") == ""
}

// handleHTTPSProxy handles HTTPS CONNECT requests for tunneling
func (h *HTTPHandler) handleHTTPSProxy(w http.ResponseWriter, r *http.Request) {
	targetConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

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

	go func() {
		defer targetConn.Close()
		defer clientConn.Close()
		io.Copy(targetConn, clientConn)
	}()

	io.Copy(clientConn, targetConn)
}

// handleProxy handles HTTP proxy requests
func (h *HTTPHandler) handleProxy(w http.ResponseWriter, r *http.Request) {
	if !r.URL.IsAbs() {
		http.Error(w, "Request URL must be absolute for proxy", http.StatusBadRequest)
		return
	}

	proxyReq, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.copyHeaders(r.Header, proxyReq.Header)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	h.copyHeaders(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// copyHeaders copies HTTP headers, excluding hop-by-hop headers
func (h *HTTPHandler) copyHeaders(src, dst http.Header) {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Proxy-Connection":    true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Upgrade":             true,
	}

	for name, values := range src {
		if hopByHopHeaders[name] {
			continue
		}
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

// handleLocalFile handles a request as a local file request
func (h *HTTPHandler) handleLocalFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.IsAbs() {
		h.log.Info("HTTP file request (absolute URL converted):", r.Method, path)
	} else {
		h.log.Info("HTTP file request:", r.Method, path)
	}

	if h.serveDir == "" {
		h.log.Warn("HTTP request received but no serve directory specified")
		http.Error(w, "File serving not enabled", http.StatusServiceUnavailable)
		return
	}

	relativeReq := r.Clone(r.Context())
	relativeReq.URL.Scheme = ""
	relativeReq.URL.Host = ""
	relativeReq.URL.Path = path

	fileServer := http.FileServer(http.Dir(h.serveDir))
	fileServer.ServeHTTP(w, relativeReq)
}

// singleConnListener wraps net.Conn to implement net.Listener
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

// Accept satisfies the net.Listener interface
func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.accepted {
		<-l.closed
		return nil, &net.OpError{Op: "accept", Net: "tcp", Err: io.EOF}
	}

	l.accepted = true
	if l.closed == nil {
		l.closed = make(chan struct{})
	}

	return &connWrapper{Conn: l.conn, listener: l}, nil
}

// Close method for connWrapper
func (c *connWrapper) Close() error {
	err := c.Conn.Close()

	if c.listener.closed != nil {
		select {
		case <-c.listener.closed:
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
		default:
			close(l.closed)
		}
	}
	return l.conn.Close()
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
