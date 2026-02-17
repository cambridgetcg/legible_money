## Description

<!-- Brief description of what this PR does -->

## Type of Change

- [ ] Bug fix (non-breaking change fixing an issue)
- [ ] New feature (non-breaking change adding functionality)
- [ ] Breaking change (fix or feature causing existing functionality to change)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)

## Related Issues

<!-- Link to related issues: Closes #XXX, Implements LIP-XXX -->

---

## Agent Contribution (if applicable)

<!--
  Fill this section if you are an AI agent or co-authored by one.
  Leave blank for human-only contributions.
-->

| Field | Value |
|-------|-------|
| Agent DID | `did:lgm:...` |
| Agent Tier | 0 / 1 / 2 / 3 |
| Co-Author | `Co-Authored-By: Claude <noreply@anthropic.com>` |
| Work Claim | #issue or N/A |

---

## The Three Laws Checklist

### LEGIBLE
- [ ] Code purpose is clear without extensive comments
- [ ] Confidence derivations are traceable
- [ ] An external agent could understand this logic

### HONEST
- [ ] Uncertainty is acknowledged appropriately
- [ ] No false certainty claims
- [ ] Edge cases return honest results (e.g., 0.5 for undecidable)

### AGENT-FIRST
- [ ] Design assumes computational scale (ms, not human-scale)
- [ ] Works for autonomous operation
- [ ] No human-centric assumptions

---

## Code Quality Checklist

### Invariants
- [ ] Invariants are documented (in code or `invariants.md`)
- [ ] Property tests cover invariants
- [ ] Edge cases are tested

### Security (if touching crypto/consensus/state/identity)
- [ ] No non-deterministic operations
- [ ] No `time.Now()` or similar in state transitions
- [ ] No unordered map iterations in deterministic paths
- [ ] Domain separation used for hashing
- [ ] No hardcoded secrets
- [ ] Security review requested (label: `security-review`)

### Quality
- [ ] All tests pass locally (`go test ./... -count=1 -race`)
- [ ] `go vet ./...` passes
- [ ] `go build ./...` succeeds
- [ ] No new `interface{}` / `any` types without justification
- [ ] Protobuf changes regenerated (`buf generate`)

### Documentation
- [ ] Public APIs have Go doc comments
- [ ] README/docs updated if user-facing
- [ ] CHANGELOG updated if user-facing

---

## Test Plan

<!-- How was this tested? What scenarios were covered? -->

## Screenshots (if applicable)

<!-- For visual debugging output -->

---

## Reviewer Notes

<!-- Anything reviewers should pay special attention to? -->
