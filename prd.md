# Product Requirements Document: Script Execution on Existing Panes

## 1. Overview

### 1.1 Purpose
Enable nx users to execute plugin scripts on existing tmux panes, extending beyond the current capability of only running scripts on new shell connections.

### 1.2 Background
Currently, nx supports executing scripts via `--exec` flag when catching new shell connections. Users have expressed the need to run scripts on already-established panes

## 2. Goals & Objectives

### 2.1 Primary Goals
- Provide a clean, intuitive interface for targeting existing tmux panes
- Maintain consistency with existing nx architecture and plugin system
- Enable both interactive and automated script execution workflows

### 2.2 Success Metrics
- Zero breaking changes to existing `--exec` functionality

## 3. User Stories

### 3.1 Security Analyst
"As a security analyst, I want to execute monitoring scripts across all active sessions, so I can gather system state during incident response."

### 3.3 Developer
"As a developer, I want to run environment setup scripts on existing shells after configuration changes, so I can update multiple environments efficiently."

## 4. Functional Requirements

### 4.1 Command Interface
```bash
nx exec <script_name> --on <target>
```

### 4.2 Target Specification
- **Full format**: `session:window.pane` (e.g., `nx:0.1`)

### 4.3 Core Features
1. **Single pane execution**
   ```bash
   nx exec auto --on nx:0.1
   ```

3. **Default behavior**
   ```bash
   nx exec auto  # Error stating --on is required
   ```

### 4.4 Options
- `--dry-run`: Show what would be executed without running

## 5. Technical Requirements

### 5.1 Architecture
- Implement as new subcommand using existing go-flags structure
- Reuse existing plugin manager for script execution
- Leverage tmux manager for pane identification and command sending

### 5.2 Error Handling
- **Invalid target**: Clear error message with available panes list
- **Missing script**: Suggest available scripts from plugins directory


## 6. User Experience

### 6.1 Command Flow
```bash
# List available panes

# Run script
$ nx exec auto --on nx:0.1
[nx] Running 'auto' on nx:0.1...
[nx] Script completed successfully

# Dry run
$ nx exec cleanup --on nx:0.1 --dry-run
[nx] Would run 'cleanup' on nx:0.1:
[nx] cd /app
[nx] rm *.bak
```

### 6.2 Help Text
```
Usage:
  nx exec <script> <--on <target>> [OPTIONS]

Arguments:
  script    Name of plugin script to execute

Options:
  --on      Target pane(s) using tmux notation (required)
  --dry-run Preview execution without running

Examples:
  nx exec auto --on nx:0.1
```

### 6.3 Error feedbad

```
nx exec <enter>

Missing script argument. Available options:
- auto
- cleanup

{remaining usage text}

```

## 7. Implementation Phases

### Phase 1: Core Functionality (Week 1-2)
- Subcommand structure
- Single pane targeting
- Basic script execution
- Dry-run mode


## 10. Success Criteria

- [ ] Users can execute scripts on specific panes without reconnection
- [ ] Command interface is intuitive and consistent with nx patterns  
- [ ] Error messages are helpful and guide users to success
- [ ] No breaking changes to existing functionality

