---
LIP: [number - assigned after acceptance]
Title: [Concise, descriptive title]
Author: [Name <email> or GitHub username]
Status: Draft
Type: [Consensus | Economics | Interface | Process]
Created: [YYYY-MM-DD]
Requires: [LIP numbers if dependent]
Replaces: [LIP number if superseding]
---

# LIP-X: [Title]

## Abstract

<!-- 2-3 sentence summary. What does this proposal do? -->

## Motivation

<!--
Why is this change needed?
How does it serve the truth-seeking mission?
What problem does it solve?
-->

## Specification

<!--
Technical details of the change.
Be precise enough that an independent implementation is possible.

Include:
- Data structures
- Algorithms
- State transitions
- API changes
-->

### Parameters

<!-- Any new or modified protocol parameters -->

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| | | |

### State Changes

<!-- How does on-chain state change? -->

**Before:**
```
[Current state structure]
```

**After:**
```
[New state structure]
```

## Rationale

<!--
Why this design over alternatives?
What trade-offs were made?
What alternatives were considered and rejected?
-->

### Alternatives Considered

1. **[Alternative 1]**: [Why rejected]
2. **[Alternative 2]**: [Why rejected]

## Backward Compatibility

<!--
Is this a breaking change?
How are existing users/validators affected?
What migration is needed?
-->

### Migration Path

<!-- Step-by-step migration for affected parties -->

## Test Cases

<!-- Representative test cases that must pass -->

```typescript
// Example test case
test('should [expected behavior]', () => {
  // Setup
  // Action
  // Assert
});
```

## Security Considerations

<!--
What are the security implications?
What attack vectors exist?
How are they mitigated?
-->

### Threat Model

| Threat | Impact | Mitigation |
|--------|--------|------------|
| | | |

### Invariants

<!-- What invariants must this change preserve? -->

- [ ] INV-1: [Statement]
- [ ] INV-2: [Statement]

## The Three Laws Assessment

### LEGIBLE
<!-- How does this change maintain or improve legibility? -->

### HONEST
<!-- How does this change maintain Gödelian honesty? -->

### AGENT-FIRST
<!-- How does this change serve autonomous agents? -->

## Implementation

<!--
Reference implementation or path to completion.
Link to PR if available.
-->

### Milestones

- [ ] Specification complete
- [ ] Reference implementation
- [ ] Test coverage complete
- [ ] Security review
- [ ] Testnet deployment
- [ ] Documentation

## References

<!-- Links to related resources, discussions, prior art -->

---

## Copyright

This LIP is licensed under [CC0 1.0 Universal](https://creativecommons.org/publicdomain/zero/1.0/).
