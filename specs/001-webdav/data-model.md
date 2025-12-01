# Data Model: WebDAV Protocol Support

**Phase**: 1 | **Date**: 2025-10-06 | **Feature**: WebDAV Protocol Support

## Core Entities

### WebDAVHandler
**Purpose**: Main protocol handler for WebDAV requests within nx multiplexing architecture

**Fields**:
- `serveDir string` - Root directory for WebDAV operations (from --serve-dir flag)
- `logger *logerr.Logger` - Structured logging instance
- `mutex sync.Mutex` - Ensures sequential operations per client requirement

**Methods**:
- `Handle(conn net.Conn) error` - Main entry point for WebDAV connections
- `parseWebDAVRequest(*http.Request) (*WebDAVRequest, error)` - Parse incoming WebDAV requests
- `handlePROPFIND(*WebDAVRequest) *http.Response` - Handle directory listing requests
- `handleGET(*WebDAVRequest) *http.Response` - Handle file download requests  
- `handlePUT(*WebDAVRequest) *http.Response` - Handle file upload requests
- `handleDELETE(*WebDAVRequest) *http.Response` - Handle file deletion requests
- `handleMKCOL(*WebDAVRequest) *http.Response` - Handle directory creation requests
- `handleCOPY(*WebDAVRequest) *http.Response` - Handle file copy requests
- `handleMOVE(*WebDAVRequest) *http.Response` - Handle file move requests

**Validation Rules**:
- serveDir must be an absolute path within filesystem
- serveDir must exist and be readable
- All file operations must stay within serveDir bounds (no directory traversal)

**State Transitions**: Stateless handler - no persistent state between requests

### WebDAVRequest
**Purpose**: Parsed representation of incoming WebDAV HTTP request

**Fields**:
- `Method string` - HTTP method (PROPFIND, GET, PUT, DELETE, MKCOL, COPY, MOVE)
- `Path string` - Cleaned and validated file path relative to serveDir
- `Headers map[string]string` - WebDAV-specific headers (Depth, Destination, Overwrite)
- `Body io.ReadCloser` - Request body for file uploads
- `ContentLength int64` - Size of request body (for uploads)

**Validation Rules**:
- Method must be valid WebDAV HTTP method
- Path must not contain '..' or other traversal attempts
- Path must resolve within configured serveDir
- Depth header must be '0', '1', or 'infinity' for PROPFIND requests
- Destination header required for COPY/MOVE operations
- ContentLength must be reasonable for file operations

**Relationships**: Created from http.Request, consumed by WebDAVHandler methods

### WebDAVResponse  
**Purpose**: WebDAV-compliant HTTP response with proper status codes and XML formatting

**Fields**:
- `StatusCode int` - HTTP status (200, 207, 404, 422, etc.)
- `Headers map[string]string` - Response headers (Content-Type, DAV capabilities)
- `Body []byte` - Response body (XML for PROPFIND, file data for GET)
- `XMLNamespace string` - DAV namespace for XML responses

**Methods**:
- `WriteXMLResponse(properties []FileProperty) error` - Generate XML for PROPFIND
- `WriteFileResponse(file io.Reader) error` - Stream file content for GET
- `WriteErrorResponse(code int, message string) error` - Generate error responses

**Validation Rules**:
- StatusCode must be valid HTTP status code
- XML responses must be well-formed and WebDAV-compliant  
- Content-Type header must match response body type

### FileProperty
**Purpose**: Metadata about files/directories for WebDAV PROPFIND responses

**Fields**:
- `Name string` - File or directory name
- `Path string` - Full path relative to serveDir root
- `IsDirectory bool` - Whether item is a directory
- `Size int64` - File size in bytes (0 for directories)
- `ModTime time.Time` - Last modification time
- `ContentType string` - MIME type for files

**Methods**:
- `ToXMLProperty() *XMLProperty` - Convert to XML representation for responses
- `ValidatePath(serveDir string) error` - Ensure path is within allowed directory

**Validation Rules**:
- Name must not be empty or contain path separators
- Path must be relative and not contain traversal attempts
- Size must be non-negative
- ModTime must be valid timestamp
- ContentType must be valid MIME type for files

**Relationships**: Collected in arrays for WebDAVResponse XML generation

### WebDAVConfig
**Purpose**: Configuration for WebDAV feature within nx application

**Fields**:
- `Enabled bool` - Whether WebDAV is enabled (from --webdav flag)  
- `ServeDir string` - Root directory path (from --serve-dir flag)
- `MaxFileSize int64` - Maximum allowed file upload size
- `AllowedMethods []string` - Enabled WebDAV methods

**Validation Rules**:
- Enabled only when ServeDir is specified (requirement FR-017)
- ServeDir must exist and be accessible if WebDAV enabled
- MaxFileSize must be positive value
- AllowedMethods must contain only valid WebDAV HTTP methods

**State Transitions**: Set once during application startup, immutable during runtime

## Entity Relationships

```
WebDAVConfig
    ↓ (configures)
WebDAVHandler
    ↓ (processes)
WebDAVRequest
    ↓ (generates)  
WebDAVResponse
    ↓ (contains)
FileProperty[]
```

## Data Flow

1. **Request Reception**: Multiplexer detects WebDAV method → routes to WebDAVHandler
2. **Request Parsing**: WebDAVHandler.parseWebDAVRequest() → creates WebDAVRequest
3. **Path Validation**: WebDAVRequest validates path within serveDir bounds
4. **Operation Execution**: Handler method executes file operation (read/write/delete)
5. **Response Generation**: Create WebDAVResponse with appropriate status and data
6. **XML Serialization**: For PROPFIND, convert FileProperty[] to XML format
7. **Response Delivery**: Send WebDAVResponse back through HTTP connection

## Persistence

**No Database**: All data operations directly on filesystem via Go standard library
**File Operations**: os.Open, os.Create, os.Remove, os.Mkdir, filepath.Walk
**Configuration**: Stored in memory during runtime, configured via CLI flags
**Logging**: Structured logs to stdout via logerr package

## Concurrency

**Sequential Operations**: sync.Mutex ensures one WebDAV operation per client at a time
**No File Locking**: Multiple clients can access same files (last write wins)
**Connection Handling**: Each client connection handled in separate goroutine
**Shared State**: Only WebDAVConfig read-only after startup, FileProperty created per-request

## Error Handling

**Path Validation Errors**: Return 404 Not Found or 403 Forbidden
**File Operation Errors**: Return 500 Internal Server Error or 422 Unprocessable Entity  
**Malformed Requests**: Return 400 Bad Request
**Method Not Allowed**: Return 405 Method Not Allowed
**XML Parsing Errors**: Return 400 Bad Request with error details

All errors logged via logerr.Error() with context for debugging.