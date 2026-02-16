# W3: E2E DevNet Lifecycle

> Truth is grounded in evidence. Run the chain. Send the transactions. Observe the results.

## Why This Path
4054 unit tests prove components work in isolation. Zero E2E tests prove the chain works as a system. This is a T1 (Grounding) failure — we claim the devnet works, but we haven't verified it end-to-end against the Cosmos binary.

This waypoint is already **40% complete**: `devnet/scripts/test-tx-lifecycle.sh` exists with 23 tests across 5 phases. What remains is execution, iteration, and extension.

## Success Criteria
1. [x] test-tx-lifecycle.sh created (23 tests, 5 phases)
2. [x] Makefile targets: test-tx-lifecycle, test-e2e-cosmos
3. [x] CI workflow: optional e2e-cosmos job (workflow_dispatch)
4. [ ] All 23 tests pass against live devnet
5. [ ] Any failures fixed in chain code or test expectations
6. [ ] Custom module query tests added (Phase 2 — gRPC direct)
7. [ ] E2E run documented in session log with evidence

## Phases Covered
| Phase | Tests | Dimension |
|-------|-------|-----------|
| Bank Operations | 6 | Happy path (D1) |
| Staking Operations | 5 | Happy path + state (D1, D3) |
| Governance Operations | 4 | Happy path + time (D1, D5) |
| Authorization & Errors | 5 | Error handling + security (D2, D4) |
| Economic Invariants | 3 | Conservation laws (D6) |

## The Truth Argument (T1, T4, T5)
- **T1 (Grounding)**: Every claim about chain behavior backed by observed tx results
- **T4 (Completeness)**: 6 testing dimensions covered — not just happy path
- **T5 (Representation)**: Supply conservation, fee deduction, staking pool — the system as a whole

## What "Pass" Looks Like
```
═══ Phase 1: Bank Operations ═══
  PASS T1.1: Send tokens (validator → test1)
  PASS T1.2: Recipient balance increased by send amount
  ...
╔═══════════════════════════════════════════════════════╗
║          Transaction Lifecycle Test Results           ║
║  Passed:  23/23                                      ║
║  Failed:  0/23                                       ║
╚═══════════════════════════════════════════════════════╝
ALL TESTS PASSED
```

## Gates Targeted
| Gate | Current | Target | How |
|------|---------|--------|-----|
| T1 (Grounding) | FAIL | PASS | Claims verified by live chain evidence |
| T4 (Completeness) | PARTIAL | PASS | 6 dimensions, 23 tests, all passing |
| T5 (Representation) | PARTIAL | PASS | Economic invariants verified empirically |

## Mastery Impact
- **Critical** for Truth: transforms claims into evidence
- **Foundation** for W5 (Cross-Stack): establishes Go chain behavioral baseline
- **Already in progress**: highest velocity path
