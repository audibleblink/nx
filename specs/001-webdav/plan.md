
# Implementation Plan: WebDAV Protocol Support

**Branch**: `001-webdav-i-want` | **Date**: 2025-10-06 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/red/code/nx/specs/001-webdav-i-want/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Add WebDAV protocol support to nx's multiplexed port functionality, allowing users to access and manage files through standard WebDAV clients (Windows Explorer, macOS Finder, curl) on the same port that handles HTTP, SSH, and reverse shell connections. WebDAV operations will be sequential with no locking mechanism, serving the same directory specified by --serve-dir flag.

## Technical Context
**Language/Version**: Go 1.24.1  
**Primary Dependencies**: cmux (connection multiplexing), cobra (CLI), logerr (logging), testify (testing)  
**Storage**: Filesystem operations via standard library (os, filepath packages)  
**Testing**: Go testing with testify framework, table-driven tests  
**Target Platform**: Cross-platform (Linux, macOS, Windows) command-line tool
**Project Type**: single - CLI multiplexing tool  
**Performance Goals**: Sequential WebDAV operations (one at a time), support for variable file sizes  
**Constraints**: Must not interfere with existing protocols, no authentication required, no file locking  
**Scale/Scope**: Single-user tool, concurrent protocol support on one port, filesystem-limited storage

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Code Quality & Standards**
- [x] Go formatting (`go fmt`) will be applied to all code
- [x] Import organization follows standard library → third-party → internal pattern
- [x] Naming conventions documented (PascalCase exported, camelCase unexported)
- [x] Error handling uses `fmt.Errorf` with `%w` for wrapping
- [x] Context cancellation planned for shutdown operations

**II. Testing Discipline**
- [x] Tests will be written BEFORE implementation (TDD)
- [x] Table-driven test patterns planned using testify
- [x] Contract tests identified for protocol handlers
- [x] Integration tests planned for multiplexing/plugins
- [x] Test files will use `_test.go` suffix

**III. Security First**
- [x] Input validation identified for all user-supplied data (WebDAV paths, HTTP headers)
- [x] Error messages reviewed to avoid leaking sensitive information
- [x] Resource cleanup planned (connections, sessions, files)
- [x] Signal handling (SIGINT, SIGTERM) included in design
- [x] Dependencies checked for known vulnerabilities

**IV. Simplicity & Maintainability**
- [x] YAGNI principle applied - no speculative features (no locking, no auth)
- [x] Abstractions justified (following existing protocol handler pattern)
- [x] Package separation follows existing structure (internal/protocols/webdav.go)
- [x] Complexity additions documented in Complexity Tracking section

**V. Observability**
- [x] Structured logging using `logerr` package planned
- [x] Log levels appropriate (Debug/Info/Error/Fatal)
- [x] Component identification in log messages
- [x] No sensitive data in logs (passwords, keys, tokens)
- [x] Performance-critical paths identified for instrumentation

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/
├── protocols/
│   ├── webdav.go          # New WebDAV protocol handler
│   ├── webdav_test.go     # Unit tests for WebDAV handler
│   ├── http.go            # Existing HTTP handler (may need updates)
│   ├── http_test.go       # Existing HTTP tests
│   ├── shell.go           # Existing shell handler
│   ├── shell_test.go      # Existing shell tests
│   └── ssh.go             # Existing SSH handler
├── mux/
│   ├── server.go          # May need updates for WebDAV detection
│   ├── server_test.go     # Integration tests for protocol detection
│   └── integration_test.go # End-to-end multiplexing tests
├── cmd/
│   ├── server.go          # May need WebDAV flag addition
│   └── common.go          # Shared command functionality
└── config/
    ├── config.go          # May need WebDAV configuration
    └── config_test.go     # Configuration validation tests

pkg/
└── socket/                # Existing socket utilities (unchanged)

specs/001-webdav-i-want/
├── contracts/             # WebDAV API contracts (Phase 1)
├── research.md            # Phase 0 output
├── data-model.md          # Phase 1 output  
├── quickstart.md          # Phase 1 output
└── tasks.md               # Phase 2 output (from /tasks command)
```

**Structure Decision**: Single project structure following existing nx architecture. WebDAV handler will be added to internal/protocols/ alongside existing protocol handlers (HTTP, SSH, shell). The multiplexer in internal/mux/ will be extended to detect and route WebDAV requests. Configuration and CLI commands will be enhanced to support WebDAV enablement.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh opencode`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P] 
- Each user story → integration test task
- Implementation tasks to make tests pass

**Ordering Strategy**:
- TDD order: Tests before implementation 
- Dependency order: Models before services before UI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: 25-30 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
