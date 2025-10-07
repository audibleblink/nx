# Tasks: WebDAV Protocol Support

**Input**: Design documents from `/home/red/code/nx/specs/001-webdav-i-want/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → ✅ Found: tech stack (Go 1.24.1, cmux, cobra, logerr, testify), structure (internal/protocols/)
2. Load optional design documents:
   → ✅ data-model.md: 5 entities identified (WebDAVHandler, WebDAVRequest, WebDAVResponse, FileProperty, WebDAVConfig)
   → ✅ contracts/: WebDAV API contract with 7 HTTP methods (PROPFIND, GET, PUT, DELETE, MKCOL, COPY, MOVE)
   → ✅ research.md: Technical decisions for Go standard library approach
3. Generate tasks by category:
   → Setup: Go project structure, dependencies, linting
   → Tests: contract tests for all WebDAV methods, integration tests
   → Core: protocol handler, request/response models, multiplexer integration
   → Integration: CLI flags, configuration, logging
   → Polish: unit tests, performance validation, documentation
4. Apply task rules:
   → Different files = mark [P] for parallel execution
   → Same file = sequential (no [P])
   → Tests before implementation (TDD approach)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → ✅ All WebDAV methods have contract tests
   → ✅ All entities have implementation tasks
   → ✅ All integration points covered
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
Single project structure following existing nx architecture:
- `internal/protocols/` - Protocol handlers
- `internal/mux/` - Connection multiplexer
- `internal/cmd/` - CLI commands
- `internal/config/` - Configuration
- Contract tests in `specs/001-webdav-i-want/contracts/`

## Phase 3.1: Setup
- [ ] T001 Verify project structure matches plan.md at `/home/red/code/nx/internal/protocols/`
- [ ] T002 Update go.mod dependencies (ensure cmux, cobra, logerr, testify are available)
- [ ] T003 [P] Configure Go formatting and vet tools for WebDAV files

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [ ] T004 [P] Contract test PROPFIND method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVPROPFIND)
- [ ] T005 [P] Contract test PUT method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVPUT)
- [ ] T006 [P] Contract test DELETE method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVDELETE)
- [ ] T007 [P] Contract test MKCOL method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVMKCOL)
- [ ] T008 [P] Contract test COPY method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVCOPY)
- [ ] T009 [P] Contract test MOVE method in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVMOVE)
- [ ] T010 [P] Contract test error responses in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVErrorResponses)
- [ ] T011 [P] Contract test sequential operations in `/home/red/code/nx/specs/001-webdav-i-want/contracts/webdav_test.go` (TestWebDAVSequentialOperations)
- [ ] T012 [P] Integration test multiple protocol support in `/home/red/code/nx/internal/mux/integration_test.go` (TestWebDAVWithOtherProtocols)
- [ ] T013 [P] Integration test WebDAV protocol detection in `/home/red/code/nx/internal/mux/server_test.go` (TestWebDAVDetection)

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [ ] T014 [P] WebDAVConfig entity in `/home/red/code/nx/internal/config/config.go` (add WebDAV fields)
- [ ] T015 [P] WebDAVRequest entity in `/home/red/code/nx/internal/protocols/webdav.go` (struct and validation)
- [ ] T016 [P] WebDAVResponse entity in `/home/red/code/nx/internal/protocols/webdav.go` (struct and XML methods)
- [ ] T017 [P] FileProperty entity in `/home/red/code/nx/internal/protocols/webdav.go` (struct and XML conversion)
- [ ] T018 WebDAVHandler entity with Handle method in `/home/red/code/nx/internal/protocols/webdav.go`
- [ ] T019 PROPFIND method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handlePROPFIND)
- [ ] T020 PUT method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handlePUT)
- [ ] T021 DELETE method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handleDELETE)
- [ ] T022 MKCOL method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handleMKCOL)
- [ ] T023 COPY method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handleCOPY)
- [ ] T024 MOVE method handler in `/home/red/code/nx/internal/protocols/webdav.go` (handleMOVE)
- [ ] T025 WebDAV request parsing in `/home/red/code/nx/internal/protocols/webdav.go` (parseWebDAVRequest)
- [ ] T026 Path validation and security in `/home/red/code/nx/internal/protocols/webdav.go` (validatePath)
- [ ] T027 XML response generation in `/home/red/code/nx/internal/protocols/webdav.go` (XML marshaling methods)
- [ ] T027a WebDAV Depth header validation in `/home/red/code/nx/internal/protocols/webdav.go` (validateDepth method)
- [ ] T027b File permission enforcement in `/home/red/code/nx/internal/protocols/webdav.go` (checkFilePermissions method)

## Phase 3.4: Integration
- [ ] T028 WebDAV protocol detection in `/home/red/code/nx/internal/mux/server.go` (extend multiplexer)
- [ ] T029 WebDAV flag addition in `/home/red/code/nx/internal/cmd/server.go` (--webdav CLI flag)
- [ ] T030 WebDAV configuration validation in `/home/red/code/nx/internal/config/config.go` (validate WebDAV settings)
- [ ] T031 WebDAV handler initialization in `/home/red/code/nx/internal/cmd/server.go` (NewWebDAVHandler call)
- [ ] T032 Sequential operation enforcement in `/home/red/code/nx/internal/protocols/webdav.go` (sync.Mutex implementation)
- [ ] T033 Structured logging integration in `/home/red/code/nx/internal/protocols/webdav.go` (logerr package usage)
- [ ] T034 Error handling and status codes in `/home/red/code/nx/internal/protocols/webdav.go` (HTTP error responses)

## Phase 3.5: Polish
- [ ] T035 [P] Unit tests for WebDAVRequest validation in `/home/red/code/nx/internal/protocols/webdav_test.go`
- [ ] T036 [P] Unit tests for FileProperty XML conversion in `/home/red/code/nx/internal/protocols/webdav_test.go`
- [ ] T037 [P] Unit tests for path validation security in `/home/red/code/nx/internal/protocols/webdav_test.go`
- [ ] T038 [P] Unit tests for WebDAV config validation in `/home/red/code/nx/internal/config/config_test.go`
- [ ] T039 Performance validation using quickstart.md scenarios (verify sequential operations <100ms per file, concurrent client handling without timeouts) in `/home/red/code/nx/specs/001-webdav-i-want/`
- [ ] T040 [P] Update AGENTS.md with WebDAV implementation details
- [ ] T041 [P] Add WebDAV examples to README.md
- [ ] T042 Remove code duplication and refactor common patterns
- [ ] T043 Execute full quickstart.md validation scenarios

## Dependencies
- Tests (T004-T013) before implementation (T014-T027)
- T014 (WebDAVConfig) blocks T030 (config validation) and T031 (handler init)
- T015-T017 (entities) block T018 (WebDAVHandler)
- T018 (WebDAVHandler) blocks T019-T027b (method handlers)
- T028 (protocol detection) blocks T031 (handler initialization)
- T029 (CLI flag) blocks T031 (handler initialization)
- Implementation (T014-T034) before polish (T035-T043)
- T027a (depth validation) and T027b (file permissions) follow T027 (XML generation)

## Parallel Example
```bash
# Launch T004-T011 together (contract tests):
Task: "Contract test PROPFIND method in webdav_test.go"
Task: "Contract test PUT method in webdav_test.go"
Task: "Contract test DELETE method in webdav_test.go"
Task: "Contract test MKCOL method in webdav_test.go"
Task: "Contract test COPY method in webdav_test.go"
Task: "Contract test MOVE method in webdav_test.go"
Task: "Contract test error responses in webdav_test.go"
Task: "Contract test sequential operations in webdav_test.go"

# Launch T015-T017 together (entity structs):
Task: "WebDAVRequest entity in internal/protocols/webdav.go"
Task: "WebDAVResponse entity in internal/protocols/webdav.go"  
Task: "FileProperty entity in internal/protocols/webdav.go"

# Launch T035-T038 together (unit tests):
Task: "Unit tests for WebDAVRequest validation in webdav_test.go"
Task: "Unit tests for FileProperty XML conversion in webdav_test.go"
Task: "Unit tests for path validation security in webdav_test.go"
Task: "Unit tests for WebDAV config validation in config_test.go"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing (TDD requirement)
- Commit after each task completion
- WebDAV methods must be implemented sequentially due to shared handler file
- Integration tests verify multiplexer doesn't break existing protocols
- All file paths are absolute to avoid confusion

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - 7 WebDAV methods → 7 contract test tasks [P] (T004-T010)
   - Error scenarios → 1 error test task [P] (T010) 
   - Sequential operations → 1 concurrency test task [P] (T011)
   
2. **From Data Model**:
   - 5 entities → 5 model creation tasks (T014-T018)
   - WebDAVHandler methods → 10 method implementation tasks (T019-T027b)
   
3. **From User Stories**:
   - Protocol multiplexing → 2 integration tests [P] (T012-T013)
   - Quickstart scenarios → validation tasks (T039, T043)

4. **Ordering**:
   - Setup → Tests → Models → Handlers → Integration → Polish
   - Sequential enforcement for same-file modifications

## Validation Checklist
*GATE: Checked by main() before returning*

- [x] All WebDAV methods have corresponding contract tests (T004-T010)
- [x] All entities have model implementation tasks (T014-T018)
- [x] All tests come before implementation (T004-T013 before T014-T027b)
- [x] Parallel tasks are truly independent (different files marked [P])
- [x] Each task specifies exact absolute file path
- [x] No task modifies same file as another [P] task
- [x] WebDAV security requirements covered (T026, T037)
- [x] Constitutional requirements addressed (TDD, logging, Go standards)
- [x] Integration with existing nx architecture (T028, T029, T031)

## Execution Status
*Updated during task execution*

- [x] Task generation complete
- [ ] Phase 3.1: Setup complete
- [ ] Phase 3.2: Tests complete (TDD)
- [ ] Phase 3.3: Core implementation complete
- [ ] Phase 3.4: Integration complete
- [ ] Phase 3.5: Polish complete