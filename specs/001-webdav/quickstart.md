# WebDAV Quickstart Guide

**Feature**: WebDAV Protocol Support | **Date**: 2025-10-06 | **Version**: 1.0.0

## Overview

This quickstart guide demonstrates how to use nx's WebDAV support to access and manage files through standard WebDAV clients on the same multiplexed port that handles HTTP, SSH, and shell connections.

## Prerequisites

- nx binary built with WebDAV support
- Directory to serve via WebDAV (e.g., `/tmp/webdav-test`)
- WebDAV client (Windows Explorer, macOS Finder, or curl)

## Quick Setup

### 1. Prepare Test Directory

```bash
# Create test directory and sample files
mkdir -p /tmp/webdav-test
cd /tmp/webdav-test
echo "Hello WebDAV" > hello.txt
echo "Sample content" > sample.txt
mkdir documents
echo "Document content" > documents/readme.txt
```

### 2. Start nx with WebDAV Support

```bash
# Start nx with WebDAV enabled on port 8443
./nx server --port 8443 --serve-dir /tmp/webdav-test --webdav

# Expected output:
# [INFO] Starting nx server on port 8443
# [INFO] Serving directory: /tmp/webdav-test
# [INFO] WebDAV support enabled
# [INFO] Listening for connections...
```

## Testing WebDAV Operations

### Using curl (Command Line)

#### List Directory Contents (PROPFIND)
```bash
# List root directory
curl -X PROPFIND http://localhost:8443/ \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?><propfind xmlns="DAV:"><allprop/></propfind>'

# Expected: XML response with file listings including hello.txt, sample.txt, documents/
```

#### Download File (GET)
```bash
# Download a file
curl http://localhost:8443/hello.txt

# Expected output: "Hello WebDAV"
```

#### Upload File (PUT)
```bash
# Upload new file
echo "New file content" | curl -X PUT http://localhost:8443/newfile.txt \
  -H "Content-Type: text/plain" \
  --data-binary @-

# Verify upload
curl http://localhost:8443/newfile.txt
# Expected output: "New file content"
```

#### Create Directory (MKCOL)
```bash
# Create new directory
curl -X MKCOL http://localhost:8443/uploads

# Verify directory exists
curl -X PROPFIND http://localhost:8443/ -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?><propfind xmlns="DAV:"><allprop/></propfind>'
```

#### Delete File (DELETE)
```bash
# Delete a file
curl -X DELETE http://localhost:8443/sample.txt

# Verify deletion (should return 404)
curl -I http://localhost:8443/sample.txt
```

#### Copy File (COPY)
```bash
# Copy file to new location
curl -X COPY http://localhost:8443/hello.txt \
  -H "Destination: http://localhost:8443/hello_copy.txt" \
  -H "Overwrite: T"

# Verify both files exist
curl http://localhost:8443/hello.txt
curl http://localhost:8443/hello_copy.txt
```

#### Move File (MOVE)
```bash
# Move file to new location
curl -X MOVE http://localhost:8443/hello_copy.txt \
  -H "Destination: http://localhost:8443/moved_hello.txt" \
  -H "Overwrite: T"

# Verify move (original should be gone, new should exist)
curl -I http://localhost:8443/hello_copy.txt  # Should return 404
curl http://localhost:8443/moved_hello.txt    # Should return content
```

### Using Windows Explorer

1. Open Windows Explorer
2. In the address bar, type: `\\localhost@8443\DavWWWRoot\`
3. Press Enter
4. You should see the contents of `/tmp/webdav-test`
5. You can now:
   - Browse files and folders
   - Copy files to/from the WebDAV share
   - Create new folders
   - Delete files and folders

### Using macOS Finder

1. Open Finder
2. Press `Cmd + K` to open "Connect to Server"
3. Enter: `http://localhost:8443`
4. Click Connect
5. The WebDAV share will mount and you can:
   - Browse files and folders  
   - Drag and drop files
   - Create folders
   - Delete items

### Using Linux (davfs2)

```bash
# Install davfs2 if not present
sudo apt-get install davfs2  # Ubuntu/Debian
sudo yum install davfs2      # CentOS/RHEL

# Create mount point
sudo mkdir /mnt/webdav

# Mount WebDAV share
sudo mount -t davfs http://localhost:8443 /mnt/webdav

# Use like any filesystem
ls /mnt/webdav
cp localfile.txt /mnt/webdav/
mkdir /mnt/webdav/newfolder

# Unmount when done
sudo umount /mnt/webdav
```

## Verification Scenarios

### Scenario 1: Multiple Protocol Support
Test that WebDAV doesn't interfere with other nx protocols:

```bash
# Terminal 1: Start nx with multiple protocols
./nx server --port 8443 --serve-dir /tmp/webdav-test --webdav

# Terminal 2: Test HTTP file serving still works
curl http://localhost:8443/hello.txt

# Terminal 3: Test WebDAV works alongside HTTP
curl -X PROPFIND http://localhost:8443/ -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?><propfind xmlns="DAV:"><allprop/></propfind>'

# Both should work without interference
```

### Scenario 2: File Operation Validation
Verify all file operations work correctly:

```bash
# Create test sequence
echo "Step 1" > test_sequence.txt
curl -X PUT http://localhost:8443/test_sequence.txt --data-binary @test_sequence.txt

# Read back
curl http://localhost:8443/test_sequence.txt  # Should show "Step 1"

# Update
echo "Step 2" > test_sequence.txt  
curl -X PUT http://localhost:8443/test_sequence.txt --data-binary @test_sequence.txt

# Read updated content
curl http://localhost:8443/test_sequence.txt  # Should show "Step 2"

# Copy and verify
curl -X COPY http://localhost:8443/test_sequence.txt \
  -H "Destination: http://localhost:8443/backup.txt" -H "Overwrite: T"
  
curl http://localhost:8443/backup.txt  # Should show "Step 2"

# Delete original and verify copy remains
curl -X DELETE http://localhost:8443/test_sequence.txt
curl -I http://localhost:8443/test_sequence.txt  # Should return 404
curl http://localhost:8443/backup.txt           # Should still return "Step 2"
```

### Scenario 3: Directory Operations
Test directory creation and listing:

```bash
# Create nested directory structure via WebDAV
curl -X MKCOL http://localhost:8443/projects
curl -X MKCOL http://localhost:8443/projects/webapp
curl -X MKCOL http://localhost:8443/projects/mobile

# Upload files to different directories
echo "Web app code" | curl -X PUT http://localhost:8443/projects/webapp/app.js --data-binary @-
echo "Mobile code" | curl -X PUT http://localhost:8443/projects/mobile/main.swift --data-binary @-

# List directory structure
curl -X PROPFIND http://localhost:8443/projects/ -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?><propfind xmlns="DAV:"><allprop/></propfind>'

# Should show webapp/ and mobile/ directories in XML response
```

### Scenario 4: Error Handling
Verify proper error responses:

```bash
# Test 404 for nonexistent resource
curl -I http://localhost:8443/nonexistent.txt  # Should return 404

# Test 403 for path traversal attempt  
curl -I http://localhost:8443/../etc/passwd     # Should return 403

# Test 405 for unsupported WebDAV method
curl -X LOCK http://localhost:8443/hello.txt   # Should return 405

# Test 400 for malformed PROPFIND
curl -X PROPFIND http://localhost:8443/ -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  -d '<invalid>xml'  # Should return 400
```

## Expected Results Summary

✅ **Success Indicators:**
- WebDAV operations return appropriate HTTP status codes (201, 204, 207, etc.)
- Files uploaded via WebDAV are accessible via regular HTTP GET
- Directory listings show correct file metadata (size, modification time)
- Multiple WebDAV clients can connect simultaneously
- WebDAV operations don't disrupt other nx protocols
- Path validation prevents directory traversal attacks
- Unsupported methods (LOCK/UNLOCK) return 405 errors

❌ **Failure Indicators:**
- WebDAV requests timeout or hang
- File operations fail with 500 errors
- Path traversal attempts succeed (security issue)
- WebDAV interferes with HTTP/SSH/shell protocols
- Malformed requests crash the server
- Authentication prompts appear (should be no auth per requirements)

## Troubleshooting

### Common Issues

**Connection Refused**
- Verify nx server is running: `ps aux | grep nx`
- Check port is correct: `netstat -tulpn | grep 8443`
- Ensure firewall allows port 8443

**403 Forbidden Errors**
- Check file permissions on serve directory: `ls -la /tmp/webdav-test`
- Verify paths don't contain `../` sequences
- Ensure user running nx has read/write access to serve directory

**404 Not Found**
- Verify file exists in serve directory: `ls /tmp/webdav-test`
- Check path case sensitivity (Linux paths are case-sensitive)
- Ensure proper URL encoding for special characters

**WebDAV Client Issues**
- Windows: Try `\\localhost@8443\DavWWWRoot\` instead of HTTP URL
- macOS: Ensure URL includes `http://` prefix
- Linux: Check davfs2 is installed and configured

### Debug Logging

Enable debug logging to troubleshoot issues:

```bash
# Start nx with debug logging
./nx server --port 8443 --serve-dir /tmp/webdav-test --webdav --debug

# Look for WebDAV-specific log entries:
# [DEBUG] WebDAV: PROPFIND request for path: /
# [DEBUG] WebDAV: Returning 207 Multi-Status with 3 resources
# [ERROR] WebDAV: Path validation failed for: /../etc/passwd
```

This quickstart guide validates the WebDAV implementation against all functional requirements and provides a comprehensive testing framework for the feature.