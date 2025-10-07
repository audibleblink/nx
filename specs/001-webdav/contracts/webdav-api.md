# WebDAV API Contracts

**Version**: 1.0.0 | **Date**: 2025-10-06 | **Protocol**: WebDAV over HTTP

## Overview

WebDAV (Web Distributed Authoring and Versioning) extends HTTP with additional methods for file management. This contract defines the exact HTTP interactions expected by the nx WebDAV implementation.

## Authentication

**Type**: None (unauthenticated access per requirement FR-011)

## Base Configuration

**Protocol**: HTTP/1.1
**Port**: Same as nx multiplexed port (e.g., 8443)
**Root Path**: Configured via --serve-dir flag
**Content-Type**: application/xml for WebDAV responses, varies for file content

## WebDAV Methods

### PROPFIND - Directory Listing

**Purpose**: Retrieve properties and directory listings

**Request**:
```http
PROPFIND /path/to/directory HTTP/1.1
Host: localhost:8443
Depth: 1
Content-Type: application/xml
Content-Length: 142

<?xml version="1.0" encoding="utf-8" ?>
<propfind xmlns="DAV:">
  <allprop/>
</propfind>
```

**Response (Success)**:
```http
HTTP/1.1 207 Multi-Status
Content-Type: application/xml; charset=utf-8
Content-Length: 1024

<?xml version="1.0" encoding="utf-8" ?>
<multistatus xmlns="DAV:">
  <response>
    <href>/path/to/directory/</href>
    <propstat>
      <status>HTTP/1.1 200 OK</status>
      <prop>
        <resourcetype><collection/></resourcetype>
        <getlastmodified>Wed, 06 Oct 2025 10:00:00 GMT</getlastmodified>
      </prop>
    </propstat>
  </response>
  <response>
    <href>/path/to/directory/file.txt</href>
    <propstat>
      <status>HTTP/1.1 200 OK</status>
      <prop>
        <resourcetype/>
        <getcontentlength>1024</getcontentlength>
        <getlastmodified>Wed, 06 Oct 2025 09:30:00 GMT</getlastmodified>
        <getcontenttype>text/plain</getcontenttype>
      </prop>
    </propstat>
  </response>
</multistatus>
```

**Headers**:
- `Depth`: 0 (current resource), 1 (immediate children), infinity (all descendants)

**Status Codes**:
- 207 Multi-Status: Successful directory listing
- 404 Not Found: Directory doesn't exist
- 403 Forbidden: Access denied (outside serve-dir)

### GET - Download File

**Purpose**: Download file content

**Request**:
```http
GET /path/to/file.txt HTTP/1.1
Host: localhost:8443
```

**Response (Success)**:
```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 1024
Last-Modified: Wed, 06 Oct 2025 09:30:00 GMT

[file content here]
```

**Status Codes**:
- 200 OK: File downloaded successfully
- 404 Not Found: File doesn't exist
- 403 Forbidden: Access denied

### PUT - Upload File

**Purpose**: Upload or update file content

**Request**:
```http
PUT /path/to/file.txt HTTP/1.1
Host: localhost:8443
Content-Type: text/plain
Content-Length: 1024

[file content here]
```

**Response (Success)**:
```http
HTTP/1.1 201 Created
Content-Length: 0
```

**Response (Update)**:
```http
HTTP/1.1 204 No Content
Content-Length: 0
```

**Status Codes**:
- 201 Created: New file created
- 204 No Content: Existing file updated
- 403 Forbidden: Access denied
- 507 Insufficient Storage: Disk full

### DELETE - Remove File/Directory

**Purpose**: Delete file or empty directory

**Request**:
```http
DELETE /path/to/file.txt HTTP/1.1
Host: localhost:8443
```

**Response (Success)**:
```http
HTTP/1.1 204 No Content
Content-Length: 0
```

**Status Codes**:
- 204 No Content: Successfully deleted
- 404 Not Found: File/directory doesn't exist
- 403 Forbidden: Access denied
- 409 Conflict: Directory not empty

### MKCOL - Create Directory

**Purpose**: Create new directory

**Request**:
```http
MKCOL /path/to/newdir HTTP/1.1
Host: localhost:8443
Content-Length: 0
```

**Response (Success)**:
```http
HTTP/1.1 201 Created
Content-Length: 0
```

**Status Codes**:
- 201 Created: Directory created successfully
- 403 Forbidden: Access denied
- 405 Method Not Allowed: Resource already exists
- 409 Conflict: Parent directory doesn't exist

### COPY - Copy File/Directory

**Purpose**: Copy resource to new location

**Request**:
```http
COPY /path/to/source.txt HTTP/1.1
Host: localhost:8443
Destination: http://localhost:8443/path/to/destination.txt
Overwrite: T
```

**Response (Success)**:
```http
HTTP/1.1 201 Created
Content-Length: 0
```

**Headers**:
- `Destination`: Target URL for copy operation (must be absolute URL)
- `Overwrite`: T (overwrite) or F (fail if exists)

**Status Codes**:
- 201 Created: Resource copied successfully
- 204 No Content: Resource copied, overwriting existing
- 403 Forbidden: Access denied
- 409 Conflict: Destination parent doesn't exist
- 412 Precondition Failed: Overwrite=F and destination exists

### MOVE - Move File/Directory

**Purpose**: Move resource to new location

**Request**:
```http
MOVE /path/to/source.txt HTTP/1.1
Host: localhost:8443
Destination: http://localhost:8443/path/to/destination.txt
Overwrite: T
```

**Response (Success)**:
```http
HTTP/1.1 201 Created
Content-Length: 0
```

**Headers**: Same as COPY
**Status Codes**: Same as COPY

## Error Responses

### Standard WebDAV Error Format

```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/xml; charset=utf-8

<?xml version="1.0" encoding="utf-8" ?>
<error xmlns="DAV:">
  <cannot-modify-protected-property/>
</error>
```

### Common Error Conditions

- **400 Bad Request**: Malformed WebDAV request or XML
- **403 Forbidden**: Path outside configured serve-dir
- **404 Not Found**: Resource doesn't exist
- **405 Method Not Allowed**: Unsupported WebDAV method
- **409 Conflict**: Resource state prevents operation
- **412 Precondition Failed**: Required headers missing or invalid
- **422 Unprocessable Entity**: Valid syntax but logical error
- **507 Insufficient Storage**: Disk space exhausted

## WebDAV Compliance Headers

All responses include:
```http
DAV: 1, 2
Server: nx/[version]
```

## Path Handling

- All paths are relative to configured --serve-dir
- Paths are cleaned using filepath.Clean()
- Directory traversal attempts (../) return 403 Forbidden
- URL decoding applied to handle special characters

## Content Type Detection

- Uses Go's http.DetectContentType() for file uploads
- Common types: text/plain, application/octet-stream, image/jpeg, etc.
- Directory listings always use application/xml

## Limitations (By Design)

- **No Locking**: LOCK/UNLOCK methods return 405 Method Not Allowed
- **No Authentication**: All requests processed without credentials
- **No Versioning**: No support for WebDAV versioning extensions
- **Sequential Operations**: One operation per client connection at a time