# Feature Specification: WebDAV Protocol Support

**Feature Branch**: `001-webdav-i-want`  
**Created**: 2025-10-06  
**Status**: Draft  
**Input**: User description: "webdav. i want to add webdav as a multiplexed option for things to listen/route to"

## Execution Flow (main)
```
1. Parse user description from Input
   → ✅ Feature description provided: WebDAV multiplexing support
2. Extract key concepts from description
   → ✅ Identified: WebDAV protocol, multiplexing, routing
3. For each unclear aspect:
   → Marked ambiguities with [NEEDS CLARIFICATION]
4. Fill User Scenarios & Testing section
   → ✅ User flows defined for file operations
5. Generate Functional Requirements
   → ✅ Requirements generated and testable
6. Identify Key Entities (if data involved)
   → ✅ Entities identified (WebDAV handler, file operations)
7. Run Review Checklist
   → ✅ All clarifications resolved
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers


## Clarifications

### Session 2025-10-06
- Q: What are the WebDAV performance expectations for file operations? → A: Unknown file sizes, only one operation at a time. Uploading client will likely be curl. Browsing client may be an OS file browser

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a user of nx, I want to access and manage files on the server through WebDAV so that I can use standard file management tools (Windows Explorer, macOS Finder, file sync clients) to interact with files on the same port that handles reverse shells, HTTP serving, and SSH tunneling.

### Acceptance Scenarios
1. **Given** nx is running with WebDAV enabled on port 8443, **When** a user connects with a WebDAV client (e.g., Windows network drive, macOS Finder "Connect to Server"), **Then** the connection is accepted and the user can browse the configured directory
2. **Given** a WebDAV connection is established, **When** the user uploads a file through their WebDAV client, **Then** the file is saved to the configured directory on the server
3. **Given** a WebDAV connection is established, **When** the user downloads a file, **Then** the file is transferred from the server to the client
4. **Given** a WebDAV connection is established, **When** the user creates a new folder, **Then** the folder is created in the configured directory
5. **Given** a WebDAV connection is established, **When** the user deletes a file or folder, **Then** the item is removed from the server
6. **Given** nx is running with multiple protocols enabled (HTTP, SSH, shell, WebDAV), **When** a WebDAV request arrives on the multiplexed port, **Then** it is correctly routed to the WebDAV handler without interfering with other protocols
7. **Given** a user wants to upload a file via curl, **When** they execute a WebDAV PUT request using curl, **Then** the file is uploaded successfully to the configured directory
8. **Given** a user wants to browse files, **When** they connect using an OS file browser (Windows Explorer/macOS Finder), **Then** they can navigate and view the directory structure


### Edge Cases
- What happens when a WebDAV client attempts to upload a file that already exists? (System MUST overwrite the existing file with the new content)
- What happens when a WebDAV client attempts to access a file outside the configured directory? (System MUST reject with HTTP 403 Forbidden status code)
- What happens when disk space is full during an upload? (System MUST return appropriate WebDAV error code)
- What happens when a WebDAV request arrives but WebDAV is not enabled? (System MUST route to appropriate fallback handler or reject)
- How does the system handle concurrent WebDAV operations on the same file? (See FR-018: last write wins approach)
- What happens when a reverse shell connection and WebDAV connection arrive simultaneously? (Multiplexer MUST handle both correctly)
- What happens when multiple WebDAV operations are attempted simultaneously from the same client? (System MUST queue and process operations sequentially)


## Requirements *(mandatory)*

### Functional Requirements
- **FR-001**: System MUST support WebDAV protocol on the same multiplexed port as existing protocols (HTTP, SSH, shell)
- **FR-002**: System MUST detect incoming WebDAV requests and route them to the WebDAV handler
- **FR-003**: System MUST allow users to enable/disable WebDAV support via command-line flag
- **FR-004**: System MUST serve the same directory via WebDAV as specified by the --serve-dir flag
- **FR-005**: System MUST support basic WebDAV operations: PROPFIND (list files), GET (download), PUT (upload), DELETE (remove), MKCOL (create directory)
- **FR-006**: System MUST support WebDAV COPY and MOVE operations for file management
- **FR-007**: System MUST return appropriate WebDAV-compliant HTTP status codes and XML responses
- **FR-008**: System MUST handle WebDAV depth headers (0, 1, infinity) for directory listings
- **FR-009**: System MUST validate all file paths to prevent directory traversal attacks
- **FR-010**: System MUST log WebDAV operations using the logerr package with Debug level for request details, Info level for successful operations, and Error level for failures
- **FR-011**: System MUST allow unauthenticated WebDAV access (no authentication required)
- **FR-012**: System MUST handle WebDAV requests without disrupting existing protocol handlers (HTTP file serving, SSH tunneling, reverse shell)
- **FR-013**: System MUST support standard WebDAV clients (Windows Explorer, macOS Finder, davfs2, cadaver)
- **FR-014**: System MUST return HTTP 405 Method Not Allowed with "DAV: 1, 2" header for WebDAV lock requests (LOCK/UNLOCK methods)
- **FR-015**: System MUST respect file permissions of the underlying filesystem
- **FR-016**: System MUST overwrite existing files when WebDAV clients upload files with the same name
- **FR-017**: System MUST only enable WebDAV when the --serve-dir flag is specified
- **FR-018**: System MUST handle concurrent WebDAV operations without file locking (last write wins)
- **FR-019**: System MUST handle WebDAV operations sequentially (one operation at a time per client)
- **FR-020**: System MUST support curl as the primary uploading client
- **FR-021**: System MUST support OS file browsers (Windows Explorer, macOS Finder) for browsing operations
- **FR-022**: System MUST handle files of unknown/variable sizes without predetermined limits


### Key Entities *(include if feature involves data)*
- **WebDAVHandler**: Processes WebDAV-specific HTTP methods (PROPFIND, MKCOL, COPY, MOVE) and returns WebDAV-compliant XML responses
- **WebDAV Request**: Represents an incoming WebDAV operation (upload via curl, download, delete, create directory, copy, move) with associated metadata (path, variable size, timestamp, client type)
- **WebDAV Request**: Incoming request with WebDAV-specific headers (Depth, Destination, Overwrite) and methods
- **Directory Listing**: Collection of file/folder metadata returned in response to PROPFIND requests

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous (except marked items)
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities resolved (4 clarifications addressed)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---
