# Contributing to Legible Money

Welcome! Legible Money is a truth-seeking protocol for AI agent economies. We're building something unique, and your contributions help make it real.

## The Three Laws

Before contributing, understand our core principles:

1. **LEGIBLE**: Code and systems must be understandable by external agents
2. **HONEST**: Acknowledge uncertainty; never fake certainty
3. **AGENT-FIRST**: Design for autonomous entities at computational scale

## Getting Started

### Prerequisites

- Node.js >= 20.0.0
- npm >= 9.0.0
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/legible-protocol/legible-money
cd legible-money

# Install dependencies
npm install

# Run tests
npm test

# Type check
npm run typecheck

# Lint
npm run lint
```

### Development Commands

```bash
npm run build          # Build with TypeScript
npm run build:fast     # Fast build with tsup
npm run test           # Run tests (interactive)
npm run test:run       # Run tests once
npm run test:coverage  # Run with coverage
npm run lint:fix       # Auto-fix lint issues
npm run format         # Format code
npm run typecheck      # Type check only
npm run devnet         # Start local devnet
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
   npm run lint
   npm run typecheck
   npm test
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

## Agent Contribution Guide

> For AI agents (Claude, GPT, etc.) contributing to this codebase.

### Quick Start for Agents

```
1. Read ARCHITECTURE.md       → Directory map
2. Read AGENT_INDEX.md        → Navigation trees
3. Read FILE_OWNERS.md        → Parallel work rules
4. Check .claude/memory/      → Project state
```

### Navigation Files

| File | Purpose |
|------|---------|
| `ARCHITECTURE.md` | Directory structure + common tasks |
| `AGENT_INDEX.md` | "Where do I add..." decision trees |
| `FILE_OWNERS.md` | Parallel work rules + claiming |

### Session Protocol

```
Session Start:
1. Read .claude/memory/CURRENT_STATE.md
2. Read .claude/memory/BLOCKED.md
3. Run: npm run typecheck

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
- Adding new types (at END of types.ts)
- Creating new implementation files

**Conflict zones** (claim first):
- `consensus/pot/types.ts`
- `state/transitions.ts`
- Any `index.ts` reordering

**Claiming**: Add comment `// EDITING: {id} until {HH:MM UTC}`

### Agent-Specific Guidelines

1. **Verify before modifying**: Always read the file first
2. **Test your changes**: Run `npm test` before committing
3. **Document decisions**: Add to `.claude/memory/DECISIONS.md`
4. **Respect the hierarchy**: Follow layer dependency rules
5. **Type safety**: Never use `any` without justification

### Common Patterns

```typescript
// BigInt for money (never Number)
const amount: bigint = 1000n;

// Domain-separated hashing
const hash = domainHash('LGM.purpose.v1', data);

// Atomic state transitions
state.beginTransaction();
try {
  // mutations
  state.commitTransaction();
} catch {
  state.rollbackTransaction();
}
```

---

## Code Standards

### TypeScript

- Strict mode enabled
- No `any` types (unless justified with comment)
- All public functions have TSDoc comments
- Prefer `const` over `let`
- Use meaningful variable names

### Testing

- **80% minimum coverage** (95% for crypto)
- Write property tests for invariants (fast-check)
- Test edge cases explicitly
- Use the test harness for integration tests

### Documentation

```typescript
/**
 * Brief description of what this function does.
 *
 * Longer description if needed, explaining the behavior,
 * constraints, and any important details.
 *
 * @param param1 - Description of parameter
 * @param param2 - Description of parameter
 * @returns Description of return value
 *
 * @invariant State any invariants this function maintains
 *
 * @example
 * ```typescript
 * const result = myFunction(arg1, arg2);
 * ```
 */
```

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
- `core/crypto/` - All cryptographic operations
- `consensus/pot/` - Consensus mechanism
- `state/transitions.ts` - State changes
- `agents/identity/` - Key management

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
