# Invariants: [Feature/Component Name]

> **Purpose**: Define invariants BEFORE implementation. These guide testing and review.

## Component Overview

<!-- Brief description of what this component does -->

## Invariants

### Core Invariants (Must Never Violate)

#### INV-1: [Name]

**Statement**:
```
[Formal or semi-formal statement of the invariant]
```

**Rationale**: [Why this invariant matters]

**Test Strategy**:
- [ ] Property test with fast-check
- [ ] Unit test edge cases
- [ ] Integration test

**Example Violation**: [What would break this?]

---

#### INV-2: [Name]

**Statement**:
```
[Formal or semi-formal statement]
```

**Rationale**: [Why this invariant matters]

**Test Strategy**:
- [ ] Property test
- [ ] Unit test
- [ ] Integration test

**Example Violation**: [What would break this?]

---

### Safety Invariants (Security-Critical)

#### SAFE-1: [Name]

**Statement**:
```
[Statement]
```

**Attack Vector**: [What attack does this prevent?]

**Verification Method**: [How do we verify this holds?]

---

### Liveness Invariants (System Progress)

#### LIVE-1: [Name]

**Statement**:
```
[Statement - usually "eventually X happens"]
```

**Failure Mode**: [What happens if this fails?]

**Recovery**: [How to recover from failure?]

---

## State Machine (if applicable)

```
[Initial State] --> [State 1] --> [State 2] --> [Final State]
                        |
                        v
                   [Error State]
```

### Valid Transitions

| From | To | Trigger | Guard |
|------|-----|---------|-------|
| | | | |

### Invalid Transitions (Must Reject)

| From | To | Why Invalid |
|------|-----|-------------|
| | | |

---

## Property Tests

```typescript
// Skeleton for property tests - implement in tests/property/

import fc from 'fast-check';

// INV-1 Property Test
test.prop([/* arbitraries */])('INV-1: [name]', (inputs) => {
  // Setup
  // Action
  // Assert invariant holds
});

// INV-2 Property Test
test.prop([/* arbitraries */])('INV-2: [name]', (inputs) => {
  // ...
});
```

---

## Edge Cases

| Case | Expected Behavior | Test? |
|------|-------------------|-------|
| Empty input | | [ ] |
| Maximum values | | [ ] |
| Minimum values | | [ ] |
| Concurrent access | | [ ] |
| Network partition | | [ ] |

---

## The Three Laws Check

- [ ] **LEGIBLE**: Can the invariants be understood by an external agent?
- [ ] **HONEST**: Do the invariants acknowledge uncertainty appropriately?
- [ ] **AGENT-FIRST**: Do the invariants assume computational scale?

---

## Sign-off

- [ ] Invariants reviewed by: [Name]
- [ ] Property tests implemented
- [ ] Ready for implementation
