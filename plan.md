## Execution Plan: Script Execution on Existing Panes

### Overview
This plan outlines the implementation of the `nx exec` subcommand feature that allows executing plugin scripts on existing tmux panes, as specified in the PRD.

### Phase 1: Command Structure Refactoring (Week 1, Days 1-3)

#### 1.1 Restructure Command Parsing
**Files to modify:**
- `nx.go` - Refactor to support subcommands
- `internal/config/config.go` - Split into base config and subcommand configs

**Implementation approach:**
1. Create new command structure using go-flags subcommands:
   ```go
   type Commands struct {
       Server ServerCommand `command:"server" description:"Start nx server (default)"`
       Exec   ExecCommand   `command:"exec" description:"Execute script on existing panes"`
   }
   ```

2. Move current Config fields to ServerCommand
3. Create ExecCommand struct with fields:
   - Script (positional argument)
   - On (required flag for target specification)
   - DryRun (optional flag)

#### 1.2 Maintain Backward Compatibility
- Make server command the default when no subcommand is specified
- Preserve all existing flags and behavior for server mode

### Phase 2: Tmux Pane Targeting (Week 1, Days 3-5)

#### 2.1 Extend Tmux Manager
**Files to create/modify:**
- `internal/tmux/pane.go` - New file for pane-specific operations
- `internal/tmux/interface.go` - Add new methods to interface
- `internal/tmux/session.go` - Add pane targeting methods

**New functionality:**
1. Parse target format "session:window.pane"
2. List available panes method
3. Execute command on specific pane method
4. Validate pane exists method

**Implementation details:**
```go
type PaneTarget struct {
    Session string
    Window  int
    Pane    int
}

func (m *Manager) ParseTarget(target string) (*PaneTarget, error)
func (m *Manager) ListPanes() ([]PaneInfo, error)
func (m *Manager) ExecuteOnPane(target *PaneTarget, command string) error
func (m *Manager) ValidatePane(target *PaneTarget) error
```

#### 2.2 Gomux Integration
Since gomux may not support direct pane targeting, implement using tmux commands:
- Use `tmux send-keys -t session:window.pane` for command execution
- Use `tmux list-panes -a -F` for pane listing

### Phase 3: Exec Command Implementation (Week 2, Days 1-3)

#### 3.1 Create Exec Command Handler
**Files to create:**
- `internal/commands/exec.go` - Main exec command logic
- `internal/commands/exec_test.go` - Unit tests

**Core functionality:**
1. Validate script exists using plugin manager
2. Parse and validate target specification
3. Read script contents
4. Execute commands on target pane(s)
5. Handle dry-run mode

#### 3.2 Plugin Manager Extensions
**Files to modify:**
- `internal/plugins/manager.go` - Add ExecuteOnPane method

**New method:**
```go
func (m *Manager) ExecuteOnPane(pluginName string, target *tmux.PaneTarget) error
```

### Phase 4: User Experience & Error Handling (Week 2, Days 3-4)

#### 4.1 Help System
**Implementation:**
- Comprehensive help text for exec command
- List available scripts when script arg is missing
- Show available panes when target is invalid

#### 4.2 Error Messages
**Error scenarios to handle:**
1. Missing script argument → List available scripts
2. Invalid target format → Show correct format with examples
3. Non-existent pane → List available panes
4. Script not found → Suggest available scripts
5. Missing --on flag → Clear error message

### Phase 5: Testing & Documentation (Week 2, Days 4-5)

#### 5.1 Test Coverage
**Test files to create/modify:**
- `internal/commands/exec_test.go` - Command logic tests
- `internal/tmux/pane_test.go` - Pane targeting tests
- `internal/plugins/manager_test.go` - Add ExecuteOnPane tests

#### 5.2 Integration Tests
- Test script execution on real tmux panes
- Test error scenarios
- Test dry-run functionality

### Technical Considerations

1. **Tmux Command Execution**: Since gomux v0.0.0 may not support direct pane targeting, implement using raw tmux commands via exec.Command

2. **Target Parsing**: Use regex or string splitting to parse "session:window.pane" format

3. **Script Execution**: Reuse existing plugin execution logic but redirect to specific pane

4. **Concurrency**: Consider future support for multiple pane execution (not in Phase 1)

### File Structure After Implementation
```
nx.go (modified - subcommand support)
internal/
  commands/
    exec.go (new)
    exec_test.go (new)
    server.go (new - moved from nx.go)
  config/
    config.go (modified - base config only)
    server.go (new - server-specific config)
    exec.go (new - exec-specific config)
  tmux/
    pane.go (new)
    pane_test.go (new)
    session.go (modified)
    interface.go (modified)
  plugins/
    manager.go (modified)
```

### Risk Mitigation

1. **Gomux Limitations**: If gomux doesn't support needed functionality, implement direct tmux command execution as fallback

2. **Breaking Changes**: Extensive testing of existing functionality to ensure backward compatibility

3. **Performance**: Pane listing could be slow with many sessions - implement caching if needed

[Tool: todowrite]

---


good plan. 1 exception, if you're unsure of what gomux supports, look it up. do your thing


