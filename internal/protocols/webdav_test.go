package protocols

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebDAVHandlerIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nx-webdav-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handler := NewWebDAVHandler(tempDir)

	t.Run("PUT creates new file", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/test.txt", strings.NewReader("content"))
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify file was created
		data, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content", string(data))
	})

	t.Run("PUT overwrites existing file", func(t *testing.T) {
		// Create initial file
		testFile := filepath.Join(tempDir, "existing.txt")
		err := os.WriteFile(testFile, []byte("old"), 0o644)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/existing.txt", strings.NewReader("new"))
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusCreated, w.Code) // stdlib returns 201 for updates

		// Verify file was updated
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "new", string(data))
	})

	t.Run("GET retrieves file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tempDir, "get-test.txt")
		err := os.WriteFile(testFile, []byte("get content"), 0o644)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/get-test.txt", nil)
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "get content", w.Body.String())
	})

	t.Run("DELETE removes file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tempDir, "delete-test.txt")
		err := os.WriteFile(testFile, []byte("delete me"), 0o644)
		require.NoError(t, err)

		req := httptest.NewRequest("DELETE", "/delete-test.txt", nil)
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify file was deleted
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("MKCOL creates directory", func(t *testing.T) {
		req := httptest.NewRequest("MKCOL", "/newdir", nil)
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify directory was created
		dirPath := filepath.Join(tempDir, "newdir")
		info, err := os.Stat(dirPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("PROPFIND lists directory", func(t *testing.T) {
		// Create test files
		err := os.WriteFile(filepath.Join(tempDir, "prop1.txt"), []byte("prop1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "prop2.txt"), []byte("prop2"), 0o644)
		require.NoError(t, err)

		req := httptest.NewRequest("PROPFIND", "/", nil)
		req.Header.Set("Depth", "1")
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusMultiStatus, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "xml")

		// Check that response contains our test files
		body := w.Body.String()
		assert.Contains(t, body, "prop1.txt")
		assert.Contains(t, body, "prop2.txt")
	})

	t.Run("Path traversal protection", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/../outside", strings.NewReader("hack"))
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		// stdlib normalizes path and creates file within serve directory
		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify file was created inside tempDir as "outside", not outside tempDir
		outsideFile := filepath.Join(tempDir, "outside")
		data, err := os.ReadFile(outsideFile)
		require.NoError(t, err)
		assert.Equal(t, "hack", string(data))

		// Verify no file was created outside tempDir
		actualOutside := filepath.Join(filepath.Dir(tempDir), "outside")
		_, err = os.Stat(actualOutside)
		assert.True(t, os.IsNotExist(err), "File should not be created outside serve directory")
	})

	t.Run("OPTIONS returns allowed methods", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/", nil)
		w := httptest.NewRecorder()
		handler.HandleRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		allow := w.Header().Get("Allow")
		// stdlib webdav includes standard WebDAV methods
		assert.Contains(t, allow, "OPTIONS")
		assert.Contains(t, allow, "DELETE")
		assert.Contains(t, allow, "PROPFIND")
		// Note: stdlib may not include GET/PUT in OPTIONS response but still supports them
	})
}
