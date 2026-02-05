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
- [ ] Property tests cover invariants (fast-check)
- [ ] Edge cases are tested

### Security (if touching crypto/consensus/state/identity)
- [ ] No non-deterministic operations
- [ ] No `Date.now()` or similar in state transitions
- [ ] No unordered iterations
- [ ] Domain separation used for hashing
- [ ] No hardcoded secrets
- [ ] Security review requested (label: `security-review`)

### Quality
- [ ] All tests pass locally (`npm test`)
- [ ] Type check passes (`npm run typecheck`)
- [ ] Lint passes (`npm run lint`)
- [ ] Coverage meets threshold (80%+)
- [ ] No new TypeScript `any` types (or justified in comments)

### Documentation
- [ ] Code has TSDoc comments for public APIs
- [ ] README/docs updated if user-facing
- [ ] CHANGELOG updated if user-facing

---

## Test Plan

<!-- How was this tested? What scenarios were covered? -->

## Screenshots (if applicable)

<!-- For UI changes or visual debugging output -->

---

## Reviewer Notes

<!-- Anything reviewers should pay special attention to? -->
