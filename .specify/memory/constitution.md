<!--
Sync Impact Report - Constitution Update
========================================
Version Change: [Initial] → 1.0.0
Modified Principles: N/A (initial constitution)
Added Sections:
  - Core Principles (5 principles)
  - Architecture Constraints
  - Development Workflow
  - Governance
Removed Sections: N/A
Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section aligns with principles
  ✅ spec-template.md - Requirements structure compatible
  ✅ tasks-template.md - Task categorization reflects test-first principle
Follow-up TODOs: None
-->

# nx Constitution

## Core Principles

### I. Library-First Architecture

Every feature MUST start as a standalone internal package with clear boundaries and single responsibility. Libraries MUST be:
- Self-contained with minimal external dependencies
- Independently testable through comprehensive unit tests
- Documented with clear purpose and public API contracts
- Named according to their domain responsibility (e.g., `protocols`, `mux`, `tmux`)

**Rationale**: Internal package separation enables independent evolution, testing isolation, and prevents circular dependencies. This pattern is evident in nx's architecture where protocols, multiplexing, and session management are cleanly separated.

### II. CLI-First Interface

Every library MUST expose functionality via CLI commands. All interfaces MUST follow:
- Text-based I/O protocol: command-line args → stdout, errors → stderr
- Support for both JSON and human-readable output formats where applicable
- Integration with shell scripting and automation workflows
- Cobra command structure for consistent UX

**Rationale**: CLI-first design ensures debuggability, scriptability, and universal accessibility. nx's primary interface is CLI-based, enabling reverse shell management, file serving, and tunneling through simple commands.

### III. Test-First Development (NON-NEGOTIABLE)

TDD is MANDATORY for all new features and significant changes:
1. Write tests that capture expected behavior
2. Obtain user/stakeholder approval of test scenarios
3. Verify tests FAIL (red state)
4. Implement functionality until tests PASS (green state)
5. Refactor while maintaining green state

**Rationale**: Test-first development prevents regression, documents intent, and ensures features meet requirements before implementation. The red-green-refactor cycle is strictly enforced to maintain code quality and prevent technical debt.

### IV. Integration Testing for Contracts

Integration tests are REQUIRED for:
- New internal package contracts (e.g., protocol handlers, multiplexer routing)
- Changes to existing package interfaces
- Inter-package communication and data flow
- Shared schemas and data structures
- Protocol multiplexing behavior (HTTP, SSH, WebDAV, shell)

**Rationale**: Unit tests verify isolated behavior; integration tests verify system-level correctness. Given nx's multiplexing architecture, integration tests prevent protocol interference and validate end-to-end flows.

### V. Observability Through Simplicity

All code MUST be observable and debuggable:
- Text I/O ensures transparent data flow inspection
- Structured logging REQUIRED via logerr package with appropriate levels:
  - Debug: Request/response details, protocol detection
  - Info: Successful operations, connection events
  - Error: Failures, security violations, unexpected states
- Log context MUST identify the operation domain (e.g., "webdav", "ssh", "mux")
- Avoid hidden state; prefer explicit data flow

**Rationale**: Debugging production issues in reverse shell/tunneling scenarios requires clear observability. Text protocols and structured logging enable rapid diagnosis without additional tooling.

## Architecture Constraints

### Technology Stack

- **Language**: Go 1.24+ (toolchain compatibility required)
- **CLI Framework**: Cobra for command structure
- **Logging**: logerr package with context separation
- **Testing**: Standard Go testing with testify assertions
- **Multiplexing**: cmux for protocol detection and routing
- **Session Management**: gomux for tmux integration
- **Configuration**: XDG-compliant paths via adrg/xdg

**Migration Policy**: Major version bumps in dependencies require justification and testing plan. Breaking changes in public APIs require deprecation period.

### Security Posture

- All file path operations MUST validate against directory traversal attacks
- Authentication is optional by design (user-controlled via flags)
- Security-sensitive operations (file access, command execution) MUST be logged
- Plugin execution MUST be explicitly enabled and documented

### Performance Expectations

- Protocol detection MUST complete within first 512 bytes of connection
- File serving operations: streaming I/O, no size limits imposed by nx
- Concurrent connections: support multiple simultaneous clients without degradation
- Memory: avoid buffering entire files; use streaming where possible

## Development Workflow

### Feature Development Process

1. **Specification Phase**: Create feature spec in `specs/[###-feature]/spec.md` following template
2. **Planning Phase**: Generate implementation plan with constitution compliance check
3. **Test-First Implementation**:
   - Write contract tests for new interfaces
   - Write integration tests for protocol interactions
   - Verify tests fail before implementation
   - Implement until tests pass
   - Refactor without breaking tests
4. **Documentation**: Update README, quickstart guides, and inline godoc
5. **Review**: Verify compliance with all constitution principles before merge

### Branching Strategy

- Feature branches: `###-feature-name` format (e.g., `001-webdav`)
- Commits follow conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`
- PRs require passing tests and constitution compliance verification

### Quality Gates

All PRs MUST pass:
- [ ] All existing tests remain green
- [ ] New tests written and passing
- [ ] Integration tests added for contract changes
- [ ] Logging added at appropriate levels
- [ ] No directory traversal vulnerabilities introduced
- [ ] Documentation updated
- [ ] Constitution principles verified

## Governance

### Amendment Process

This constitution supersedes all other development practices. Amendments require:
1. Documented justification with impact analysis
2. Update to version number according to semantic versioning (see below)
3. Sync impact report identifying affected templates and documentation
4. Migration plan for existing code if principles change
5. Team review and approval

### Versioning Policy

Constitution version follows MAJOR.MINOR.PATCH:
- **MAJOR**: Backward-incompatible governance changes, principle removals, or redefinitions
- **MINOR**: New principles added, sections materially expanded, new mandatory requirements
- **PATCH**: Clarifications, wording improvements, typo fixes, non-semantic refinements

### Compliance Review

- All PRs MUST verify alignment with constitution principles in PR description
- Feature specs MUST include constitution check section (auto-generated in plan.md)
- Complexity introduced MUST be justified in complexity tracking table
- Use `.specify/memory/constitution.md` as source of truth for all governance decisions

### Deferred Decisions

None. All principles are defined and in effect as of ratification date.

**Version**: 1.0.0 | **Ratified**: 2025-11-30 | **Last Amended**: 2025-11-30
