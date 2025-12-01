package protocols

import (
	"net/http"

	"github.com/audibleblink/logerr"
	"golang.org/x/net/webdav"
)

// WebDAVHandler wraps the standard library WebDAV handler
type WebDAVHandler struct {
	handler *webdav.Handler
	logger  logerr.Logger
}

// NewWebDAVHandler creates a new WebDAV handler using stdlib implementation
func NewWebDAVHandler(serveDir string) *WebDAVHandler {
	logger := logerr.Add("webdav")

	return &WebDAVHandler{
		handler: &webdav.Handler{
			FileSystem: webdav.Dir(serveDir),
			LockSystem: webdav.NewMemLS(), // Required by stdlib WebDAV interface (locking disabled by design)
			Logger: func(r *http.Request, err error) {
				if err != nil {
					logger.Error("WebDAV error for %s %s: %v", r.Method, r.URL.Path, err)
				}
			},
		},
		logger: logger,
	}
}

// HandleRequest processes a WebDAV HTTP request using stdlib handler
func (h *WebDAVHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	h.logger.Debugf("%s request for path: %s", r.Method, r.URL.Path)
	h.handler.ServeHTTP(w, r)
}
