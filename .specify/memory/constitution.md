<!--
Sync Impact Report:
Version: 0.0.0 → 1.0.0
Change Type: MAJOR (Initial constitution establishment)
Modified Principles: N/A (initial creation)
Added Sections:
  - Core Principles (5 principles)
  - Development Standards
  - Governance
Removed Sections: N/A
Templates Status:
  ✅ .specify/templates/plan-template.md - Aligned with constitution principles
  ✅ .specify/templates/spec-template.md - Aligned with requirements standards
  ✅ .specify/templates/tasks-template.md - Aligned with TDD and testing principles
  ✅ .opencode/command/*.md - No agent-specific references found
  ✅ AGENTS.md - Serves as runtime guidance, aligned with principles
Follow-up TODOs: None
-->

# nx Constitution

## Core Principles

### I. Code Quality & Standards (NON-NEGOTIABLE)
All code MUST adhere to standard Go conventions and formatting:
- Use `go fmt` for all code - no exceptions
- Import organization: standard library, third-party, internal packages with blank line separation
- Naming conventions strictly enforced: PascalCase for exported, camelCase for unexported
- Error handling: Always check errors, use `fmt.Errorf` with `%w` for proper wrapping
- Context cancellation: Respect `context.Context` for all shutdown operations

**Rationale**: Consistency enables maintainability. Security tools require predictable,
auditable code. Standard formatting eliminates bikeshedding and enables automated tooling.

### II. Testing Discipline (NON-NEGOTIABLE)
Test-driven development is mandatory for all features:
- Table-driven tests using testify for all new functionality
- Tests MUST be written before implementation (Red-Green-Refactor)
- Contract tests required for protocol handlers (HTTP, SSH, shell)
- Integration tests required for multiplexing and plugin system
- Test files end with `_test.go` and live alongside source
- Test data in `testdata/` directories

**Rationale**: nx handles security-sensitive operations (reverse shells, SSH tunneling).
Bugs can create vulnerabilities. TDD ensures behavior is specified before implementation
and provides regression protection.

### III. Security First
Security considerations MUST be addressed in all code:
- Input validation for all user-supplied data (ports, paths, commands)
- No sensitive information in error messages or logs
- Proper cleanup of connections, sessions, and resources
- Signal handling (SIGINT, SIGTERM) for graceful shutdown
- Dependencies checked for known vulnerabilities

**Rationale**: nx is a security tool used in sensitive environments. Security failures
can compromise entire systems. Defense-in-depth requires security at every layer.

### IV. Simplicity & Maintainability
Favor simple, direct solutions over complex abstractions:
- YAGNI principle: Implement only what is needed now
- No premature abstraction - wait for 3+ use cases before extracting patterns
- Clear separation of concerns by package (internal/cmd, internal/protocols, etc.)
- Interfaces defined close to usage, not speculatively
- Complexity MUST be justified in documentation

**Rationale**: Security tools must be auditable. Complex code hides bugs and
vulnerabilities. Simple code is easier to review, test, and maintain.

### V. Observability
All components MUST provide visibility into their operation:
- Use `github.com/audibleblink/logerr` for structured logging
- Log levels: Debug (verbose), Info (normal), Error (failures), Fatal (unrecoverable)
- Context-aware logging with proper component identification
- No logging of sensitive data (passwords, keys, session tokens)
- Performance-critical paths instrumented for debugging

**Rationale**: Network tools operate in complex environments. Debugging requires
visibility. Structured logging enables troubleshooting without compromising security.

## Development Standards

### Code Organization
- Internal packages in `internal/` - not importable by external projects
- Public APIs in `pkg/` - stable interfaces for external use
- Bundled resources use `embed.FS` for single-binary distribution
- Plugin system uses XDG base directory specification

### Dependency Management
- Minimize external dependencies - each adds attack surface
- Pin versions in `go.mod` for reproducible builds
- Review dependency licenses for compatibility
- Use standard library when sufficient

### Documentation
- Package-level documentation for all packages
- Exported functions and types MUST have doc comments
- README.md kept current with feature additions
- AGENTS.md updated when development patterns change

## Governance

### Amendment Process
1. Proposed changes documented with rationale
2. Impact assessment on existing code and templates
3. Version bump following semantic versioning
4. Update all dependent templates and documentation
5. Migration plan for breaking changes

### Versioning Policy
- **MAJOR**: Backward incompatible principle removals or redefinitions
- **MINOR**: New principles added or materially expanded guidance
- **PATCH**: Clarifications, wording improvements, typo fixes

### Compliance Review
- All pull requests MUST verify constitutional compliance
- Constitution violations require explicit justification
- Complexity additions documented in implementation plans
- AGENTS.md serves as runtime development guidance

### Enforcement
- Constitution supersedes all other development practices
- Automated checks (go fmt, go vet, tests) gate all commits
- Code review verifies adherence to principles
- Violations without justification block merges

**Version**: 1.0.0 | **Ratified**: 2025-05-02 | **Last Amended**: 2025-10-06
