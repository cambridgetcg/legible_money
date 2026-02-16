# Contributing to Legible Money

Welcome! Legible Money is a truth-seeking protocol for AI agent economies. We're building something unique, and your contributions help make it real.

## The Three Laws

Before contributing, understand our core principles:

1. **LEGIBLE**: Code and systems must be understandable by external agents
2. **HONEST**: Acknowledge uncertainty; never fake certainty
3. **AGENT-FIRST**: Design for autonomous entities at computational scale

## Getting Started

### Prerequisites

- Go >= 1.22
- [buf](https://buf.build/) (for protobuf)
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/legible-protocol/legible-money
cd legible-money

# Build
go build ./...

# Run tests
go test ./... -count=1 -race

# Vet
go vet ./...
```

### Development Commands

```bash
go build ./...                       # Build all packages
go test ./... -count=1 -race         # Run tests with race detector
go test ./x/knowledge/... -count=1   # Run specific module tests
go vet ./...                         # Static analysis
buf generate                         # Regenerate protobuf (from proto/)
buf lint                             # Lint proto definitions
govulncheck ./...                    # Security vulnerability scan
```

## How to Contribute

### Types of Contributions

| Type | Process |
|------|---------|
| Bug fix | Open issue → PR |
| Small feature | Open issue → PR |
| Large feature | Discussion → LIP → PR |
| Documentation | Direct PR |
| Security fix | Private disclosure (see below) |

### Contribution Workflow

1. **Check existing issues** to avoid duplicates
2. **Open an issue** describing what you want to do
3. **Fork the repository**
4. **Create a branch**: `git checkout -b fix-issue-123` or `feat-description`
5. **Write invariants first** (for new features)
6. **Write tests first** (TDD)
7. **Implement the change**
8. **Ensure all checks pass**:
   ```bash
   go vet ./...
   go test ./... -count=1 -race
   go build ./...
   ```
9. **Submit a PR** using the template
10. **Address review feedback**
11. **Squash and merge** (maintainer action)

### Commit Messages

```
<type>(<scope>): <subject>

[body]

[footer]

Types:
- feat: New feature
- fix: Bug fix
- docs: Documentation
- style: Formatting
- refactor: Code restructuring
- test: Adding tests
- chore: Maintenance

Example:
feat(consensus): implement commit-reveal verification

Adds commit-reveal mechanism to prevent validator copying.

Implements LIP-7.
Closes #123.
```

### Branch Naming

- `lip-XXX-description` - LIP implementations
- `fix-issue-XXX` - Bug fixes
- `feat-description` - Features
- `refactor-description` - Refactoring
- `docs-description` - Documentation

## Project Structure

```
legible-money/
├── app/                     # Cosmos SDK application wiring
├── x/                       # Cosmos SDK modules (24 custom modules)
│   ├── auth/                # Agent authentication & session keys
│   ├── bvm/                 # Bytecode Virtual Machine
│   ├── claiming_pot/        # Proof of Truth claim processing
│   ├── knowledge/           # Knowledge graph & evaluation pipeline
│   ├── staking/             # Validator staking & delegation
│   ├── tokens/              # LGM token & expirable balances
│   └── ...                  # See x/ for all modules
├── proto/                   # Protobuf definitions
│   └── legible/             # Per-module .proto files
├── cmd/                     # CLI entrypoints
└── docs/                    # Documentation
```

## Agent Contribution Guide

> For AI agents (Claude, GPT, etc.) contributing to this codebase.

### Quick Start for Agents

```
1. Read ARCHITECTURE.md       → Directory map
2. Read AGENT_INDEX.md        → Navigation trees
3. Read FILE_OWNERS.md        → Parallel work rules
4. Check .claude/memory/      → Project state
```

### Session Protocol

```
Session Start:
1. Read .claude/memory/CURRENT_STATE.md
2. Read .claude/memory/BLOCKED.md
3. Run: go build ./...

During Session:
- Update CURRENT_STATE.md on progress
- Log decisions to DECISIONS.md

Session End:
- Update CURRENT_STATE.md
- Clear resolved blockers
```

### Parallel Work Rules

**Safe zones** (no coordination):
- Adding new test files
- Adding new types within a module
- Creating new implementation files

**Conflict zones** (claim first):
- `x/knowledge/keeper/` — evaluation pipeline
- `app/app.go` — module wiring
- Proto files affecting multiple modules

**Claiming**: Add comment `// EDITING: {id} until {HH:MM UTC}`

### Agent-Specific Guidelines

1. **Verify before modifying**: Always read the file first
2. **Test your changes**: Run `go test ./... -count=1` before committing
3. **Document decisions**: Add to `.claude/memory/DECISIONS.md`
4. **Respect the hierarchy**: Follow layer dependency rules
5. **Type safety**: Use concrete types; avoid `interface{}` without justification

---

## Code Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- No `interface{}` / `any` without justification
- All exported functions have Go doc comments
- Use `errors.New` / `fmt.Errorf` with `%w` for wrapping
- Prefer table-driven tests

### Testing

- **80% minimum coverage** (95% for crypto)
- Use `-race` flag for all test runs
- Use `-count=1` to disable test caching
- Test edge cases explicitly

### Protobuf

- All state and genesis types defined in `.proto` files
- Run `buf generate` after proto changes
- Run `buf lint` to validate proto definitions
- Keep generated `.pb.go` files in version control

---

## LIP Process

For significant changes, write a Legible Improvement Proposal:

1. Copy `docs/templates/LIP_TEMPLATE.md`
2. Fill out all sections
3. Submit as PR to `docs/lips/`
4. Gather community feedback
5. Address concerns
6. Move to "Accepted" status
7. Implement

See [DEVELOPMENT_FRAMEWORK.md](docs/DEVELOPMENT_FRAMEWORK.md) for full details.

## Security

### Reporting Vulnerabilities

**NEVER open public issues for security vulnerabilities.**

Report to:
- Email: security@legible.money (when available)
- GitHub: Security Advisories (private)

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security-Sensitive Code

Extra review required for:
- `x/knowledge/` - Evaluation pipeline and knowledge state
- `x/claiming_pot/` - Proof of Truth consensus
- `x/staking/` - Validator economics
- `x/auth/` - Agent identity and session keys
- `x/bvm/` - Virtual machine execution

## Review Process

### What Reviewers Look For

1. **The Three Laws compliance**
2. **Invariants documented and tested**
3. **No security issues**
4. **Code quality and readability**
5. **Test coverage**
6. **Documentation**

### Review Timeline

| Priority | Response Time |
|----------|---------------|
| Security | 4 hours |
| Bug fix | 24 hours |
| Feature | 48 hours |
| Refactor | 72 hours |

## Community

### Getting Help

- Open a [Discussion](https://github.com/legible-protocol/legible-money/discussions)
- Check existing issues and PRs
- Read the [documentation](docs/)

### Code of Conduct

Be respectful. We're building for a future where agents and humans collaborate. Start that collaboration with kindness.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Legible Money. You're helping build the infrastructure for autonomous agent economies.
