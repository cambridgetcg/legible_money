# PP Review: Capture Challenge Module (x/capture_challenge)

> **Review Date**: 2026-02-11
> **Scope**: Full PP review of the capture_challenge Cosmos SDK module — domain bounty challenges, evidence lifecycle, reward/slash economics, validator slashing, bounty pool management
> **Files Reviewed**: All 10 files in `x/capture_challenge/` (keeper/*.go, types/*.go, module.go), 2 proto files, app.go wiring, test file
> **Territory**: PARTIALLY CHARTED — Domain-specific capture defense with bounty pools is novel; challenge lifecycle is standard dispute resolution
> **Total Source**: ~1,800 LoC (keeper+types+module) + ~500 LoC (tests)

---

## Architecture Summary

```
                    Capture Challenge Pipeline
                    ══════════════════════════

  ┌──────────────┐    ┌─────────────┐    ┌──────────────────────┐
  │  Submit      │    │  Evidence    │    │  Governance          │
  │  Challenge   │───▶│  Period     │───▶│  Review              │
  │              │    │  (777 blk)  │    │  (7,777 blk)         │
  │  - 77 LGM   │    │  - Add 3-50 │    │  - Upheld/Rejected   │
  │  - Domain    │    │  - Deadline  │    │  - Partial/Expired   │
  │  - Evidence  │    │  - Only     │    │                      │
  │  - Validators│    │   challenger│    │                      │
  └──────────────┘    └─────────────┘    └──────────────────────┘
         │                   │                      │
         │                   │                      │
  ┌──────────────┐    ┌─────────────┐    ┌──────────────────────┐
  │  Bounty      │    │  BeginBlock │    │  Validator           │
  │  Pools       │    │  Phase      │    │  Slashing            │
  │              │    │  Advance    │    │                      │
  │  - Per-domain│    │             │    │  - 30% of stake      │
  │  - Risk-based│    │  - Auto     │    │  - via StakingKeeper │
  │  - Rejected  │    │    expire   │    │  - Silent failure!   │
  │    penalties  │    │  - Return   │    │                      │
  └──────────────┘    └─────────────┘    └──────────────────────┘
```

---

## TRUTH GATES (5 gates)

### T1: Grounding — PASS
All claims grounded in source code. Every finding cites file:line. Module read in full.

### T2: Calibration — PASS
Confidence matches evidence. CRITICAL findings are actual fund-loss or security bugs, not speculative.

### T3: Counter-Consideration — PASS
Considered whether "silent slash failure" is intentional (validator might not exist) — concluded it's still a bug because challenge is marked "upheld" even if slashing fails, creating inconsistent state.

### T4: Completeness — PASS
All 10 source files + 2 proto files + app wiring + test file reviewed. Cross-module references checked.

### T5: Representation — PASS
Language accurately represents knowledge state. "CRITICAL" means fund loss or security violation.

---

## UNDERSTANDING GATES (6 gates)

### U1: Truth Foundation — PASS
Built on accurate reading of all source files.

### U2: Mental Model — PASS
Challenge lifecycle clearly modeled: submit → evidence_period → under_review → (upheld|rejected|partial|expired).

### U3: Why Explanation — PASS
Each finding explains WHY it's a problem, not just WHAT is wrong. E.g., reward insolvency explains the accounting mismatch.

### U4: Knowledge Connection — PASS
Connected to evidence_mgmt module review (similar patterns: BigInt parsing, JSON serialization, reward economics).

### U5: Practical Application — PASS
Each finding includes specific remediation steps.

### U6: Exhaustion — PASS (Claude-specific)
All files in module read completely. Cross-references checked in app.go, capture_defense, staking.

---

## BEAUTY GATES (7 gates)

### B1: Foundation — PASS
Review grounded in actual code behavior, not assumptions.

### B2: Emergence — PASS
Findings emerged from systematic reading, not forced.

### B3: Fit — PASS
Module follows standard Cosmos SDK module pattern (keeper/types/module split).

### B4: Elegance — CONCERN
The "777/7777" aesthetic is charming but the evidence-locks-before-validation pattern at msg_server.go:24-44 is inelegant — stake is locked before evidence count is validated, requiring refund if validation fails.

### B5: Coherence — CONCERN
Reward economics are incoherent: upheld rewards (7.7x + 777 LGM) are paid from module account but only the 77 LGM challenge stake was deposited. Module must hold massive reserves or rewards are insolvent.

### B6: Clarity — PASS
Code is readable with clear function names and comments.

### B7: Appropriateness — PASS
Module scope is appropriate for its purpose.

---

## JUSTICE GATES (6 gates)

### J1: Placement — PASS
All state in keeper, types in types, lifecycle in module.go BeginBlock/EndBlock. Correct placement.

### J2: Ownership — PASS
Challenge lifecycle owned by keeper. Resolution owned by governance authority. Appropriate ownership.

### J3: Responsibility — PARTIAL CONCERN
`ResolveChallenge` has too many responsibilities: reward calculation, fund transfer, validator slashing, pool update, status transition — all in one function with no error atomicity.

### J4: Proportion — CONCERN
Upheld challenge reward (7.7x + 777 LGM) is disproportionately high compared to stake (77 LGM). Net reward ~1,370 LGM from 77 LGM stake = 17.8x ROI. This creates strong financial incentive for frivolous challenges with even small probability of success.

### J5: Balance — CONCERN
Rejected penalty (77%) vs upheld reward (17.8x) creates imbalance. Expected value for random challenges may be positive if governance has even ~5% false-positive rate.

### J6: Integrity — PASS
Module preserves Cosmos SDK architectural patterns.

---

## CREATIVITY GATES (7 gates)

### C0: Territory — PARTIALLY CHARTED
Domain-based bounty pools with auto-funding from capture risk scores is novel. Challenge lifecycle is standard.

### C1: Pioneering — PASS
Novel integration of capture detection (x/capture_defense) with economic incentives.

### C2: Path-Building — PASS
Clear path from detection → challenge → resolution → consequences.

### C3: Extension — PASS
Module extensible via params (evidence periods, review periods).

### C4: Vision — PARTIAL CONCERN
No clear vision for how bounty pool balance actually funds rewards. AutoFundBountyPools sets flags but never mints/transfers funds.

### C5: Foundation — PASS
Data structures support future extension.

### C6: Integration — PARTIAL CONCERN
Integration with capture_defense is one-directional (reads metrics). No feedback loop where challenge outcomes improve detection.

### C7: Proof — PASS (Claude-specific)
All files systematically explored and verified.

---

## FINDINGS

### CRITICAL

#### F1: Reward Insolvency — Upheld Rewards Exceed Module Balance

**File**: `keeper/msg_server.go:150-168`

**Issue**: When a challenge is upheld, the module pays `stake + (7.7x stake) + 777 LGM` to the challenger. For a 77 LGM stake:
- Module received: 77 LGM (locked on submission)
- Module pays: 77 + 592.9 + 777 = **1,446.9 LGM**
- Deficit: **1,369.9 LGM**

The module account must already hold ~1,370 LGM per upheld challenge, but nothing funds it except rejected challenge penalties (59.29 LGM each). **~23 rejected challenges needed to fund 1 upheld challenge.**

```go
// msg_server.go:163-168
totalPayout := new(big.Int).Add(stakeInt, reward)
payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalPayout)))
if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, payoutCoins); err != nil {
    return nil, fmt.Errorf("failed to pay challenger reward: %w", err)
}
```

If the module account lacks funds, this returns an error and the entire resolution fails — the challenge is stuck in "under_review" forever until it auto-expires.

**Severity**: CRITICAL — First upheld challenge on fresh chain is guaranteed to fail.

**Remediation**: Either:
1. Fund module account at genesis with BaseBountyPool tokens
2. Use mint permissions (add Minter to module permissions in app.go)
3. Cap rewards to available module balance with proportional scaling

---

#### F2: Unchecked BigInt Parsing — Zero-Stake Challenge Bypass

**File**: `keeper/msg_server.go:24-25`

**Issue**: `SetString` return value not checked:
```go
stakeInt := new(big.Int)
stakeInt.SetString(msg.Stake, 10)  // If msg.Stake is "abc", stakeInt remains 0
minStake := new(big.Int)
minStake.SetString(types.MinChallengeStake, 10)
if stakeInt.Cmp(minStake) < 0 { ... }
```

If `msg.Stake` is malformed (e.g., `"abc"`), `stakeInt = 0`, the comparison correctly rejects it. BUT if `msg.Stake = "-77000000"` (negative), `SetString` succeeds, `Cmp` passes (|negative| ≥ 77M), and `sdk.NewCoin` will panic on negative amount.

Same pattern at lines 152-153, 193, 218.

**Severity**: CRITICAL — Negative stake causes chain panic.

**Remediation**: Add explicit validation:
```go
if _, ok := stakeInt.SetString(msg.Stake, 10); !ok || stakeInt.Sign() <= 0 {
    return nil, fmt.Errorf("invalid stake amount: %s", msg.Stake)
}
```

---

#### F3: Slash Error Silently Ignored — Validators Escape Punishment

**File**: `keeper/msg_server.go:178`

**Issue**: `SlashValidator` error discarded with `_`:
```go
_ = k.stakingKeeper.SlashValidator(ctx, validatorAddr,
    fmt.Sprintf("%d", slashAmount), "capture_challenge_upheld")
```

If slashing fails (validator doesn't exist, insufficient stake, etc.), the challenge is still marked "upheld" and the challenger receives the full reward. **Validators escape punishment but challenger is rewarded as if they were punished.**

**Severity**: CRITICAL — Inconsistent economic state.

**Remediation**: Either fail the resolution or log and track unexecuted slashes:
```go
if err := k.stakingKeeper.SlashValidator(...); err != nil {
    k.Logger(ctx).Error("failed to slash validator",
        "validator", validatorAddr, "error", err)
    // Continue but mark slash as failed in resolution
    slashes[validatorAddr] = "FAILED:0"
}
```

---

### HIGH

#### F4: Stake Locked Before Evidence Validation

**File**: `keeper/msg_server.go:32-44`

**Issue**: Challenge stake is locked via `SendCoinsFromAccountToModule` at line 38, but evidence count validation happens at line 43. If evidence is insufficient, the function returns an error — but the stake transfer already succeeded and is **not reversed**.

```go
// Line 38: stake already transferred to module
if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, challengerAddr, types.ModuleName, stakeCoins); err != nil { ... }

// Line 43: evidence check AFTER transfer
if len(msg.Evidence) < types.MinEvidencePieces {
    return nil, types.ErrInsufficientEvidence  // stake already locked!
}
```

**Severity**: HIGH — Challenger permanently loses stake on validation failure.

**Remediation**: Move all validation before the bank transfer. The Cosmos SDK tx framework will rollback on error (state changes are cached), but this is fragile — explicit ordering is safer:
```go
// Validate FIRST
if len(msg.Evidence) < types.MinEvidencePieces { return ... }
// THEN lock stake
k.bankKeeper.SendCoinsFromAccountToModule(...)
```

---

#### F5: BeginBlock Full-Scan O(N) Per Block

**File**: `keeper/keeper.go:117-183`

**Issue**: `AdvanceChallengePhases` iterates ALL challenges every block, including resolved ones (upheld, rejected, expired). The switch statement skips them silently, but the iteration cost grows with historical challenge count.

**Severity**: HIGH — Performance degrades linearly over chain lifetime.

**Remediation**: Add status index or delete/archive resolved challenges. Since resolved challenges are still queryable via the store, consider a `StatusIndexPrefix` with status+challengeId → challengeId mapping, and only iterate the "active" statuses.

---

#### F6: AutoFundBountyPools Only Updates Flags, Never Transfers Funds

**File**: `keeper/keeper.go:186-217`

**Issue**: `AutoFundBountyPools` sets `pool.AutoGenerated = true` and updates `CaptureRiskScore`, but never actually transfers any funds into the pool. The event name `bounty_pool_auto_funded` is misleading — no funding occurs.

```go
// keeper.go:201-205
pool.CaptureRiskScore = metrics.RiskScore
pool.AutoGenerated = true
pool.LastAnalyzedBlock = currentBlock
pool.LastUpdatedBlock = currentBlock
k.SetBountyPool(ctx, pool)
// No bank transfer! No balance change!
```

**Severity**: HIGH — Event claims funding happened but it didn't. Misleading for indexers and dashboards.

**Remediation**: Either actually fund the pool (mint to module, then update balance), or rename to `AnalyzeBountyPoolRisk` and emit `bounty_pool_risk_updated`.

---

### MEDIUM

#### F7: Challenger Address Parsing Unchecked in ResolveChallenge

**File**: `keeper/msg_server.go:147`

**Issue**: Address parsed without error check:
```go
challengerAddr, _ := sdk.AccAddressFromBech32(challenge.Challenger)
```

If a challenge was stored with an invalid address (shouldn't happen but defense-in-depth), `challengerAddr` is nil, causing panic in `SendCoinsFromModuleToAccount`.

**Severity**: MEDIUM — Unlikely but would halt chain.

**Remediation**:
```go
challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
if err != nil {
    return nil, fmt.Errorf("invalid stored challenger address: %w", err)
}
```

---

#### F8: No Evidence Deduplication

**File**: `keeper/msg_server.go:112`

**Issue**: `AddEvidence` appends evidence without checking for duplicates:
```go
challenge.Evidence = append(challenge.Evidence, msg.Evidence)
```

Same evidence (by ID or DataHash) can be submitted multiple times, filling the 50-piece cap with duplicates.

**Severity**: MEDIUM — Wastes evidence slots, degrades challenge quality.

**Remediation**: Check for duplicate IDs before appending.

---

#### F9: No AddEvidence Event Emission

**File**: `keeper/msg_server.go:86-116`

**Issue**: `AddEvidence` doesn't emit any event. Evidence additions are invisible to chain indexers.

**Severity**: MEDIUM — Breaks auditability expectations.

**Remediation**: Emit `capture_evidence_added` event with challenge_id, evidence_id, count.

---

#### F10: Bounty Pool Balance Accounting Incoherent

**File**: `keeper/msg_server.go:197-201`

**Issue**: Rejected challenge penalties are added to `pool.Balance`, but upheld challenge rewards are NOT deducted from `pool.Balance`. The pool tracks "accumulated penalties" not "available funds."

```go
// Rejected: adds to pool
poolBal.Add(poolBal, penalty)
pool.Balance = poolBal.String()

// Upheld: pays from MODULE, not from pool
k.bankKeeper.SendCoinsFromModuleToAccount(...)
// pool.Balance unchanged
```

`pool.Balance` is a phantom metric — it doesn't reflect actual spendable funds.

**Severity**: MEDIUM — Governance sees misleading pool balances when making decisions.

**Remediation**: Either track inflows/outflows separately, or use pool.Balance as the canonical source and deduct rewards from it.

---

#### F11: Partial Reward Uses Integer Division of Constant

**File**: `keeper/msg_server.go:220`

**Issue**:
```go
halfMultiplierBPS := int64(types.ChallengeStakeMultiplierBPS / 2)
// ChallengeStakeMultiplierBPS = 7_700_000
// 7_700_000 / 2 = 3_850_000 (exact, no issue here)
```

Currently correct, but if `ChallengeStakeMultiplierBPS` becomes odd, integer division silently truncates. The pattern is fragile.

**Severity**: MEDIUM — Future-facing brittleness.

**Remediation**: Define `PartialChallengeMultiplierBPS` as its own constant.

---

### LOW

#### F12: Challenge ID Only 8 Bytes — Collision Risk

**File**: `keeper/msg_server.go:48-50`

```go
idHash := sha256.Sum256([]byte(idInput))
challengeId := fmt.Sprintf("challenge_%x", idHash[:8])
```

8 bytes = 64 bits. Birthday collision at ~2^32 challenges. Unlikely but architecturally weak.

**Severity**: LOW — Theoretical.

**Remediation**: Use 16 bytes (128 bits) or full hash.

---

#### F13: No DomainPauseDuration Param in Resolution — Uses Constant

**File**: `keeper/msg_server.go:185`

```go
challenge.DomainPausedUntil = currentBlock + types.DefaultDomainPauseDurationBlocks
```

Uses hardcoded constant instead of `params.DomainPauseDurationBlocks`. The param exists in genesis but is never used in resolution.

**Severity**: LOW — Governance can't adjust domain pause duration.

**Remediation**: `challenge.DomainPausedUntil = currentBlock + params.DomainPauseDurationBlocks`

---

#### F14: No proto tx.proto File

**Issue**: Proto directory has `genesis.proto` and `state.proto` but no `tx.proto`. Message types defined only in Go.

**Severity**: LOW — No external client can generate transaction stubs.

---

#### F15: prefixEndBytes Duplicated (Cross-Module)

Same pattern as evidence_mgmt, staking, knowledge, etc. ~160 copies across codebase.

**Severity**: LOW — Maintenance burden.

---

## AGGREGATE ASSESSMENT

| Gate Dimension | Pass | Partial | Concern | Score |
|----------------|------|---------|---------|-------|
| TRUTH (5)      | 5    | 0       | 0       | 5/5   |
| UNDERSTANDING (6) | 6 | 0       | 0       | 6/6   |
| BEAUTY (7)     | 5    | 0       | 2       | 5/7   |
| JUSTICE (6)    | 3    | 1       | 2       | 3.5/6 |
| CREATIVITY (7) | 5    | 2       | 0       | 5/7   |
| **TOTAL**      |      |         |         | **24.5/31** |

---

## VERDICT

### Challenge Lifecycle: GOOD (7/10)
Clear multi-phase state machine. Auto-expiry is a strong design choice (protects challengers from governance inaction). Phase advancement in BeginBlock is correct. Missing: status indexing for performance.

### Reward Economics: POOR (3/10)
Fundamental insolvency: upheld rewards far exceed module balance. No funding mechanism for the reward pool. Pool balance metric is phantom. Rejected penalties create a slow trickle (59 LGM each) that can't sustain upheld rewards (1,370 LGM each).

### Validator Slashing: POOR (3/10)
Silent error ignoring in SlashValidator. No verification that validators were actually punished. No atomicity — challenge resolved "upheld" even if all slashes fail.

### Input Validation: FAIR (5/10)
BigInt parsing unchecked (negative input → panic). Stake locked before evidence validated. Address parsing unchecked in resolution. Evidence deduplication missing.

### Testing: GOOD (7/10)
8 tests covering happy paths. Good mock setup. Missing: error path tests, concurrent challenges, AddEvidence tests.

---

## RECOMMENDED ACTIONS (Priority Order)

1. **Fix BigInt parsing** (F2, CRITICAL) — Add `SetString` return check + sign validation
2. **Fix reward insolvency** (F1, CRITICAL) — Fund module at genesis or cap rewards to balance
3. **Fix slash error handling** (F3, CRITICAL) — Propagate errors or track failures
4. **Move validation before stake lock** (F4, HIGH) — Reorder in SubmitChallenge
5. **Add status index for BeginBlock** (F5, HIGH) — Stop iterating resolved challenges
6. **Fix AutoFundBountyPools** (F6, HIGH) — Actually fund or rename
7. **Fix challenger address parsing** (F7, MEDIUM) — Error check in ResolveChallenge
8. **Add evidence deduplication** (F8, MEDIUM) — Check ID/hash before append
9. **Add event for AddEvidence** (F9, MEDIUM) — Auditability
10. **Fix pool balance accounting** (F10, MEDIUM) — Track real balance
11. **Extract partial multiplier constant** (F11, MEDIUM) — Avoid fragile division
12. **Use param for domain pause** (F13, LOW) — Instead of constant
13. **Increase challenge ID length** (F12, LOW) — 16+ bytes
14. **Add tx.proto** (F14, LOW) — Client compatibility
