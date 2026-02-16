# W5: Cross-Stack Golden Vectors

> Two implementations. Same truth. Verify it.

## Why This Path
The TS chain has 4054 tests proving its behavior. The Go chain is meant to replicate that behavior in a production framework. But we've never verified they produce the same results for the same inputs. This is a T3 (Counter-Evidence) gap — we need to actively seek divergence, not assume equivalence.

Blocked by W3 (E2E DevNet Lifecycle) — establishes the Go chain behavioral baseline first.

## Success Criteria
1. [ ] Golden vector set defined: 50+ test vectors covering critical operations
2. [ ] TS generator: runs each vector through TS chain, captures output
3. [ ] Go consumer: runs each vector through Go chain, compares output
4. [ ] Crypto vectors: Ed25519 sign/verify, SHA-512, BLAKE3, VRF, domain hashing
5. [ ] State vectors: account creation, balance transfer, stake/unstake
6. [ ] Consensus vectors: confidence aggregation, decay computation, tier transitions
7. [ ] Any divergence documented with root cause analysis
8. [ ] Zero unexplained divergences across all vector categories

## Vector Categories
| Category | Count | TS Source | Go Source |
|----------|-------|-----------|-----------|
| Crypto primitives | 15 | core/crypto/ | x/knowledge/crypto/ |
| Account operations | 10 | state/accounts.ts | x/auth/keeper/ |
| Staking operations | 8 | state/validators.ts | x/staking/keeper/ |
| Knowledge claims | 8 | state/knowledge.ts | x/knowledge/keeper/ |
| Confidence math | 5 | consensus/pot/confidence.ts | x/knowledge/keeper/confidence.go |
| Validator tiers | 4 | consensus/pot/validator_tiers.ts | x/staking/keeper/ |

## The Truth Argument (T3, T1)
- **T3 (Counter-Evidence)**: Actively seeking where the two stacks disagree
- **T1 (Grounding)**: Every "same behavior" claim backed by matching outputs
- This is the hardest form of truth: not "does it work?" but "does it match?"

## Implementation Approach
```
tests/cross_stack/
├── vectors/
│   ├── crypto.json        # Input→expected output for crypto ops
│   ├── accounts.json      # Account state transitions
│   ├── staking.json       # Staking state transitions
│   ├── knowledge.json     # Claim/fact lifecycle
│   └── confidence.json    # Confidence aggregation math
├── ts_runner.ts           # Runs vectors through TS code
├── go_runner_test.go      # Runs same vectors through Go code
└── compare.sh             # Diffs outputs, reports divergence
```

## Gates Targeted
| Gate | Current | Target | How |
|------|---------|--------|-----|
| T3 (Counter-Evidence) | FAIL | PASS | Actively seeking divergence between stacks |
| T1 (Grounding) | FAIL | PASS | Equivalence claims backed by vector evidence |
| T4 (Completeness) | PARTIAL | PASS | 6 vector categories cover full protocol |
| U4 (Application) | PARTIAL | PASS | Understanding verified by implementation match |

## Mastery Impact
- **Critical** for Truth: T3 is the hardest truth gate — seeking counter-evidence
- **High** for Understanding: divergences reveal where mental models are wrong
- **Foundation** for L3: L3 Truth = "counter-considerations are strengths, not obligations"
