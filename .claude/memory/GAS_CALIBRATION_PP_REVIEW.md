# PP Review: Gas Protocol & Calibration (Full 30-Gate)

**Date**: 2026-02-11
**Reviewer**: PP (all 30 gates, 5 dimensions)
**Scope**: Gas metering, fee routing, API pricing, distribution, calibration across TS + Go stacks
**Prior Review**: `GAS_FEES_REVIEW.md` (2026-02-10, score 37/50, 13 findings)
**Files Reviewed**: 22 primary files across both stacks

---

## Executive Summary

The gas and fees system implements a dual-layer economic model: **transaction fees** (per-tx gas costs) and **API billing** (Truth Gateway per-query pricing). Both converge on the universal 22% research share deduction.

This review supersedes the prior `GAS_FEES_REVIEW.md` (37/50). Of the original 13 findings, **zero have been fixed**. This review adds **7 new findings**, including one new CRITICAL: the 22% research tax routing function is defined but **never called** from any block lifecycle hook. All transaction fees stay in `fee_collector` on the Go chain.

**Overall Score: 33/50** (down from 37)

| Dimension | Score | Max | Change | Notes |
|-----------|-------|-----|--------|-------|
| TRUTH | 5 | 10 | -1 | Dead RouteFees + unchecked SetString |
| UNDERSTANDING | 7 | 10 | -1 | Stated model (22% tax) not implemented on Go chain |
| BEAUTY | 7 | 10 | 0 | Clean separation; pricing sign confusion |
| JUSTICE | 6 | 10 | -1 | Zero-fee bypass; BVM gas isolation gap |
| CREATIVITY | 8 | 10 | -1 | Model sound but per-fact cost can go negative |

---

## Architecture Map

```
                    ┌─────────────────────────────────────┐
                    │          TRANSACTION FEES            │
                    │                                     │
  User Tx ──────►  │ LGMGasDecorator (validate gas)      │
  (with fee)       │       ↓                              │
                   │ FeeRouterDecorator (LOG ONLY!)       │
                   │       ↓                              │
                   │ DeductFeeDecorator → fee_collector   │
                   │                                      │
                   │ ⚠ RouteFees() NEVER CALLED ⚠         │
                   │   22% research tax NOT executing     │
                   │   100% stays in fee_collector        │
                   └──────────────────────────────────────┘

                    ┌─────────────────────────────────────┐
                    │          API BILLING (Go)            │
                    │                                     │
  Agent Query ──►  │ pricing.go: CalculateQueryPrice()    │
                   │  Bell curve confidence, novelty,     │
                   │  freshness — all big.Int             │
                   │       ↓                              │
                   │ distribution.go: CalculateDistrib()  │
                   │   22% → 5-way research (ramped)      │
                   │   60% → provider                     │
                   │   30% → knowledge pool               │
                   │   10% → protocol (50% burn/50% tr.)  │
                   └──────────────────────────────────────┘

                    ┌─────────────────────────────────────┐
                    │       API BILLING (TS prototype)     │
                    │                                     │
  Agent Query ──►  │ truth_gateway.ts: calculateQueryPrice│
                   │  ⚠ FLOAT intermediates (Math.floor)  │
                   │       ↓                              │
                   │ calculateDistribution()              │
                   │  ⚠ Math.round for BPS conversion     │
                   │  ⚠ Per-fact cost can go negative     │
                   └──────────────────────────────────────┘
```

---

## Findings

### F1: CRITICAL — BPS Test Invariant Always Fails [PRIOR, UNFIXED]

**File**: `app/ante_test.go:177-182`
**Also**: `app/gas.go:228-232`

```go
// gas.go: Uses 1,000,000 = 100% scale
ResearchTaxBPS  uint64 = 220000  // 22%
ValidatorFeeBPS uint64 = 780000  // 78%

// ante_test.go: Tests against 10,000 = 100% scale
func TestFeeRoutingBPSSumTo10000(t *testing.T) {
    if ResearchTaxBPS+ValidatorFeeBPS != 10000 {  // 220000+780000 = 1000000 ≠ 10000
```

**Impact**: Test always fails. Masked by proto redeclaration build errors. When build is fixed, this is a red test that hides the actual invariant (sum = 1,000,000).

**Fix**: `if ResearchTaxBPS+ValidatorFeeBPS != 1000000 {`

---

### F14: CRITICAL — RouteFees Never Called (Research Tax Dead Code)

**File**: `app/ante.go:359-391`

`RouteFees()` is defined to sweep 22% of `fee_collector` balance to `research_fund`. The `FeeRouterDecorator.AnteHandle()` at line 328-357 explicitly says:

```go
// Fee routing happens post-execution in EndBlocker via distribution module.
// Phase 2: Implement BeginBlock fee sweep from fee_collector to research_fund.
// For now, all fees go to fee_collector and distribution handles validator rewards.
```

There is **no BeginBlocker or EndBlocker** that calls `RouteFees()`. The function is dead code. The 22% research tax — the core economic flywheel of the protocol — **never executes** on the Go chain. 100% of transaction fees go to validators via the standard Cosmos distribution module.

**Impact**: The entire research funding model (bounties, IRL verification, knowledge sustainability loop) receives $0 from transaction fees on the Go chain. Only API billing (x/billing) properly routes the research share.

**Fix**: Wire `RouteFees(ctx, app.BankKeeper)` into `app.go`'s `BeginBlocker` or the tokens module's `BeginBlock`. Verify `research_fund` is in `maccPerms`.

---

### F2: HIGH — Float Precision in TS Truth Gateway [PRIOR, UNFIXED]

**File**: `core/billing/truth_gateway.ts:246, 258, 267, 366-370, 422`

```typescript
// truth_gateway.ts:246 — float intermediate before BigInt
const confidencePremium =
    (baseCost * BigInt(Math.floor((confidenceMultiplier - 1) * 1000))) / 1000n;

// truth_gateway.ts:366-370 — Math.round for BPS
const providerBps = BigInt(Math.round(billingParams.apiProviderShare * 10000));

// truth_gateway.ts:422 — float weight → BigInt
const shareBps = BigInt(Math.round(share * 10000));
```

**Impact**: Rounding errors for large payments. Go `distribution.go` correctly uses pure `big.Int`.

---

### F3: HIGH — Cross-Stack Gas Cost Divergence [PRIOR, UNFIXED]

**Files**: `core/billing/gas.ts` vs `app/gas.go`

| Transaction | TS Gas | Go Gas | Ratio |
|-------------|--------|--------|-------|
| `contractDeploy` / `deploy_contract` | 500,000 | 200,000 | 2.5x |
| `disputeInitiate` / `initiate_dispute` | 250,000 | 100,000 | 2.5x |
| `disputeSettle` / `settle_dispute` | 200,000 | 80,000 | 2.5x |
| `bountyCreate` / `create_bounty` | 150,000 | 80,000 | 1.9x |
| `governancePropose` / `governance_propose` | 150,000 | 100,000 | 1.5x |

Additionally, naming differs: TS `contractDeploy` vs Go `deploy_contract`, TS `disputeInitiate` vs Go `initiate_dispute`.

**Impact**: Agents trained on DevNet (TS) pricing will underpay on production (Go).

---

### F4: HIGH — System Transaction Exemption Divergence [PRIOR, UNFIXED]

**Files**: `state/transitions.ts:296-301` vs `app/gas.go:241-250`

| Exemption | TS | Go |
|-----------|----|----|
| `add_fact` | YES | NO |
| `epoch_transition` | Via `sender === "system"` | YES |
| `emergency_halt/revert/resume` | Via `sender === "system"` | YES |
| `block_reward` | Via `sender === "system"` | YES |

**Impact**: `add_fact` passes fee-free on TS DevNet but requires fees on Go chain.

---

### F15: HIGH — Zero-Fee Bypass in LGMGasDecorator

**File**: `app/ante.go:115-123`

```go
feeCoins := feeTx.GetFee()
if !feeCoins.IsZero() {
    minFee := sdk.NewCoin(BondDenom, math.NewIntFromUint64(gasLimit*MinGasPrice))
    if feeCoins.AmountOf(BondDenom).LT(minFee.Amount) {
        return ctx, errors.Wrapf(sdkerrors.ErrInsufficientFee, ...)
    }
}
```

When `feeCoins.IsZero()`, the minimum fee check is **skipped entirely**. A transaction with `gas=MinGasLimit` (22,222) and `fee=0` passes gas validation. The downstream `DeductFeeDecorator` would then attempt to deduct 0 from the sender, which succeeds. This means **any user can submit zero-fee transactions** that pass ante validation.

The system transaction check only applies to `SystemTxTypes` in `gas.go:241-250`, but the LGMGasDecorator doesn't filter by system tx type — it allows zero fees for ALL transaction types.

**Impact**: Users can submit any transaction without paying fees, bypassing the economic model entirely.

**Fix**: Move the zero-fee check before the IsZero guard:
```go
if feeCoins.IsZero() && !isSystemTx(tx) {
    return ctx, errors.Wrap(sdkerrors.ErrInsufficientFee, "zero fees not allowed")
}
```

---

### F16: HIGH — Stale Evidence_mgmt Proto URLs in Ante

**File**: `app/ante.go:194-198`

```go
"/legible.evidence_mgmt.v1.MsgArchiveEvidence": TransactionGasCosts["archive_evidence"],
"/legible.evidence_mgmt.v1.MsgAuditSubmit":     TransactionGasCosts["audit_submit"],
"/legible.evidence_mgmt.v1.MsgAuditResolve":    TransactionGasCosts["audit_resolve"],
```

The evidence_mgmt module was rewritten (earlier this session). Current messages:
- `MsgSubmitEvidence` (exists, mapped correctly)
- `MsgAuditEvidence` (renamed from MsgAuditSubmit — **NOT mapped**)
- `MsgRegisterOracle` (exists, mapped correctly)
- `MsgSubmitOracleData` (**NEW — NOT mapped**)
- `MsgUpdateParams` (**NEW — NOT mapped**)

**Impact**: `MsgAuditEvidence` and `MsgSubmitOracleData` fall back to MinGasLimit (22,222 gas) instead of intended costs (50,000 and 60,000+). Users pay ~2.3x less than intended.

**Fix**: Replace stale entries:
```go
"/legible.evidence_mgmt.v1.MsgSubmitEvidence":  TransactionGasCosts["submit_evidence"],
"/legible.evidence_mgmt.v1.MsgAuditEvidence":   TransactionGasCosts["audit_submit"],
"/legible.evidence_mgmt.v1.MsgRegisterOracle":  TransactionGasCosts["register_oracle"],
"/legible.evidence_mgmt.v1.MsgSubmitOracleData": TransactionGasCosts["submit_evidence"],
"/legible.evidence_mgmt.v1.MsgUpdateParams":    MinGasLimit,
```

---

### F5: MEDIUM — RouteFees Swallows Module Account Errors [PRIOR, UNFIXED]

**File**: `app/ante.go:380-386`

Research fund routing logs a warning but continues if the module account doesn't exist. **Note**: Since F14 shows RouteFees is never called, this finding is moot until F14 is fixed.

---

### F6: MEDIUM — PrepareProposal Heuristic Gas Estimation [PRIOR, UNFIXED]

**File**: `app/abci.go:184-186`

```go
estimatedGas := uint64(len(tx)) * 10
```

A 200-byte `MsgDeployContract` (200,000 gas actual) estimates as 2,000 gas, allowing 16x more txs than the block supports. Mitigated by actual execution gas limits.

---

### F7: MEDIUM — BVM Gas Not Reconciled with SDK Gas Meter [PRIOR, UNFIXED]

**Files**: `x/bvm/vm/gas.go` + `x/bvm/keeper/msg_server.go`

BVM `GasMeter` operates independently from SDK's `ctx.GasMeter()`. No `ctx.GasMeter().ConsumeGas(bvmMeter.Used(), "bvm_execution")` after BVM execution. Contract calls can bypass block gas limits.

---

### F8: MEDIUM — BPS Scale Inconsistency [PRIOR, UNFIXED]

| Location | Scale | 100% = |
|----------|-------|--------|
| TS `reward_router.ts` | Standard BPS | 10,000 |
| Go `app/gas.go` | Mega-BPS | 1,000,000 |
| Go `x/billing/types/types.go` | Mega-BPS | 1,000,000 |

Same label "BPS" for two different scales. Direct cause of F1.

---

### F17: MEDIUM — Pricing Sign Confusion in Go

**File**: `x/billing/keeper/pricing.go:98-103`

```go
freshnessDiscount.Mul(baseCost, big.NewInt(int64(params.FreshnessDiscountBps)))
freshnessDiscount.Div(freshnessDiscount, bpsBasis)
freshnessDiscount.Neg(freshnessDiscount) // negative = premium
factPrice.Add(factPrice, new(big.Int).Neg(freshnessDiscount))
```

A parameter named `FreshnessDiscountBps` that actually produces a **premium** for fresh facts. Double negation: Neg(Neg(discount)) = addition. The breakdown reports `FreshnessDiscount` as a negative number for premium cases — confusing for API consumers.

**Fix**: Rename to `FreshnessPremiumBps` and remove the double negation:
```go
freshnessPremium.Mul(baseCost, big.NewInt(int64(params.FreshnessPremiumBps)))
freshnessPremium.Div(freshnessPremium, bpsBasis)
factPrice.Add(factPrice, freshnessPremium)
```

---

### F18: MEDIUM — BaseCostPerFact SetString Unchecked

**File**: `x/billing/keeper/pricing.go:31-32`

```go
baseCost := new(big.Int)
baseCost.SetString(params.BaseCostPerFact, 10)
```

Return value of `SetString` is not checked. If `BaseCostPerFact` is corrupted (e.g., via direct state manipulation bypassing `Params.Validate()`), `baseCost` stays at 0. All queries become free.

**Fix**:
```go
if _, ok := baseCost.SetString(params.BaseCostPerFact, 10); !ok {
    return nil, nil, fmt.Errorf("invalid base_cost_per_fact: %s", params.BaseCostPerFact)
}
```

---

### F19: MEDIUM — Per-Fact Cost Can Go Negative in TS

**File**: `core/billing/truth_gateway.ts:272-273`

```typescript
const factCost = baseCost + confidencePremium + fundamentalityPremium - freshnessDiscount;
```

For a well-cited fact (high fundamentalityMultiplier discount → negative `fundamentalityPremium`) with high confidence (negative `confidencePremium`), `factCost` can go negative. The floor at line 301 only guards the **total** cost, not individual fact costs. A query for 10 well-cited + 1 novel fact can be cheaper than just the novel fact alone.

**Fix**: Add per-fact floor: `const factCost = maxBigInt(0n, baseCost + ...)` or `const factCost = factCostRaw < 0n ? 1n : factCostRaw`.

---

### F9: LOW — Dead Priority Multiplier [PRIOR, UNFIXED]

**Files**: `core/billing/gas.ts:252`, `app/gas.go:23`

`PriorityMultiple = 2` / `priorityMultiplier: 2n` defined but never referenced.

---

### F10: LOW — Missing Proto URL Gas Mappings [PRIOR, PARTIALLY ADDRESSED]

**File**: `app/ante.go:306-312`

22+ transaction types fall back to `MinGasLimit`. Agent Home (10 types), Qualification (4), Auth recovery (8), Governance extras (3). Now also evidence_mgmt (3 new — F16).

---

### F11: LOW — Number() on BigInt in Quote Validity [PRIOR, UNFIXED]

**File**: `core/billing/truth_gateway.ts:332`

```typescript
validUntil: Number(currentBlock) + 24,
```

---

### F12: LOW — No TS Runtime Gas Enforcement [PRIOR, UNFIXED]

**Files**: `state/transitions.ts`, `core/billing/gas.ts`

Gas costs defined but never enforced. Any TS transaction with fee >= 0 passes if balance covers it.

---

### F13: LOW — Fee Deduction Timing Divergence [PRIOR, UNFIXED]

**Files**: `state/transitions.ts:787-814` (after handler) vs `app/ante.go:46` (before handler)

TS: fees charged after handler → free retries on failure.
Go: fees charged before handler → paid regardless of outcome.

---

### F20: LOW — Message Billing Float Multipliers

**File**: `core/billing/message_billing.ts:194, 227`

```typescript
depthMultipliers: [1.0, 1.25, 1.5, 2.0, 2.5],
complexityThresholds: { low: 500, medium: 2000, high: 5000, veryHigh: 10000 },
```

Complexity and depth multipliers stored as floats. While this is config (not monetary math), the eventual cost computation `totalCost = baseCost * complexityMultiplier` requires float→BigInt conversion.

---

## Test Coverage Assessment

| Area | TS Tests | Go Tests | Gaps |
|------|----------|----------|------|
| Fee routing (22/78 split) | 1 test | 0 integration | **No test verifies RouteFees is actually called (F14)** |
| Gas cost lookup | 0 explicit | 2 tests | — |
| Gas limits invariant | 0 | 1 test | — |
| BPS sum invariant | 0 | 1 test **(ALWAYS FAILS - F1)** | Critical |
| LGMGasDecorator | 0 | 6 tests | **No test for zero-fee bypass (F15)** |
| Proto URL completeness | 0 | 2 tests | Tests exist but don't catch stale entries |
| API pricing formula | 0 | 0 | **No tests for CalculateQueryPrice** |
| API distribution | 0 | 0 | **No tests for CalculateDistribution** |
| Per-fact negative cost | 0 | 0 | **No boundary test (F19)** |
| BVM gas reconciliation | 0 | 0 | **No test (F7)** |
| Cross-stack consistency | 0 | 0 | **No cross-stack validation (F3)** |

---

## Positive Findings

1. **P1**: Clean architecture — Gas costs centralized in `gas.ts` / `gas.go`. Single source of truth.
2. **P2**: Comprehensive Go ante handler chain — 15 decorators in correct order with clear comments.
3. **P3**: Init-time gas validation — `app/ante.go:409-416` panics at startup if any gas cost exceeds `TxGasLimit`.
4. **P4**: Progressive research ramp — `GetEffectiveResearchShareBps()` in both stacks. 5%→22% linear.
5. **P5**: 5-way treasury split with remainder handling — Reserve gets remainder to avoid dust loss (both stacks).
6. **P6**: Go `distribution.go` uses pure `big.Int` throughout — no float contamination in production billing.
7. **P7**: Thorough Go ante tests — 18 test functions covering gas lookup, limits, decorators, fee calculation.
8. **P8**: Marginal Information Value pricing — economically sound model: confidence bell curve, novelty discount, freshness premium. Prices knowledge at its actual marginal value to agents.
9. **P9**: Default-deny session capabilities — `LegibleCapabilityDecorator` rejects unrecognized message types for session keys, preventing capability bypass.
10. **P10**: Emergency halt gate before gas processing — halted txs don't consume gas or fees.

---

## Resolution Priority

| Finding | Severity | Effort | Priority |
|---------|----------|--------|----------|
| **F14**: RouteFees dead code (research tax not executing) | CRITICAL | 15 min | **P0** — Wire into BeginBlocker |
| **F1**: BPS test mismatch | CRITICAL | 5 min | **P0** — Fix test constant |
| **F15**: Zero-fee bypass | HIGH | 15 min | **P0** — Add system tx guard |
| **F16**: Stale evidence_mgmt proto URLs | HIGH | 10 min | **P1** — Update ante.go |
| F2: Float precision in TS gateway | HIGH | 2 hr | P1 — Before production |
| F3: Cross-stack gas divergence | HIGH | 1 hr | P1 — Before migration |
| F4: System tx exemption mismatch | HIGH | 30 min | P1 — Before migration |
| F7: BVM gas reconciliation | MEDIUM | 1 hr | P2 |
| F18: Unchecked SetString | MEDIUM | 10 min | P2 |
| F19: Negative per-fact cost | MEDIUM | 15 min | P2 |
| F17: Pricing sign confusion | MEDIUM | 30 min | P2 — Rename |
| F6: PrepareProposal heuristic | MEDIUM | 30 min | P2 |
| F8: BPS scale inconsistency | MEDIUM | 1 hr | P2 — Documentation |
| F5: RouteFees swallows errors | MEDIUM | 15 min | P2 (after F14) |
| F9: Dead priority multiplier | LOW | 15 min | P3 |
| F10: Missing proto mappings | LOW | CI | P3 |
| F11-F13, F20 | LOW | Various | P3-P4 |

---

## Gate Scores (Detailed)

### TRUTH (5/10)
- **T1 Grounding**: Gas costs grounded in EIP conventions and computational complexity. ✓
- **T2 Calibration**: BPS test broken (F1). RouteFees dead code (F14). ✗✗
- **T3 Counter-Consideration**: No dynamic pricing — acceptable if intentional, but dead `PriorityMultiple` confusing. ○
- **T4 Completeness**: 22+ unmapped proto URLs (F10). Stale evidence_mgmt URLs (F16). Missing pricing tests. ✗
- **T5 Representation**: Float contamination in TS (F2). Unchecked SetString in Go (F18). Per-fact negative (F19). ✗

### UNDERSTANDING (7/10)
- **U1 Mental Model**: Clear dual-stream model (tx fees + API billing). ✓
- **U2 Why**: Research funding rationale well-documented in code comments. ✓
- **U3 Connection**: Fee→research→bounty loop defined but **not actually executing** on Go chain (F14). ✗
- **U4 Boundary**: System tx exemptions inconsistent (F4). Zero-fee bypass (F15). ✗
- **U5 Application**: Good Go ante tests. Weak TS pricing tests. Missing cross-stack validation. ○

### BEAUTY (7/10)
- **B1 Emergence**: Gas system emerges naturally from computational costs. ✓
- **B2 Fit**: Architecture fits Cosmos SDK patterns well (ante chain, module accounts). ✓
- **B3 Elegance**: Clean separation: gas.ts/gas.go for costs, ante.go for validation, billing/ for pricing. ✓
- **B4 Coherence**: BPS scale inconsistency (F8). Pricing sign confusion (F17). ✗
- **B5 Naming**: Cross-stack naming divergence (F3). `FreshnessDiscount` that's actually a premium. ✗
- **B6 Simplicity**: Good — minimal abstractions, direct computation. ✓
- **B7 Clarity**: Well-commented. Progressive ramp logic readable. ✓

### JUSTICE (6/10)
- **J1 Placement**: Gas costs properly placed in dedicated files. ✓
- **J2 Proportion**: Gas costs proportional to operation complexity. ✓
- **J3 Balance**: 22%/78% split well-justified — but **not executing** (F14). ✗
- **J4 Responsibility**: RouteFees error handling too lenient (F5). ✗
- **J5 Integrity**: BVM gas isolation lacks SDK reconciliation (F7). Zero-fee bypass (F15). ✗
- **J6 Fairness**: Knowledge pool weighted by confidence × log2(citations) — sound economics. ✓

### CREATIVITY (8/10)
- **C1 Territory**: Novel dual-stream economic model. ✓
- **C2 Pioneering**: Progressive research ramp is innovative. ✓
- **C3 Path-Building**: Citation-weighted knowledge pool distribution. ✓
- **C4 Extension**: Sustainability condition (API revenue > block decay). ✓
- **C5 Vision**: Clear path from TS prototype to Go production. ✓
- **C6 Boundaries**: Per-fact cost can go negative (F19) — model boundary not enforced. ✗
- **C7 Integration**: Message billing integrates complexity/depth/streaming — forward-thinking. ✓

---

## Summary of Changes from Prior Review

| Aspect | Prior (2026-02-10) | Current (2026-02-11) |
|--------|-------------------|---------------------|
| Score | 37/50 | **33/50** |
| Findings | 13 | **20** (13 prior + 7 new) |
| CRITICAL | 1 (F1) | **2** (+F14) |
| HIGH | 3 (F2-F4) | **6** (+F15, F16) |
| MEDIUM | 4 (F5-F8) | **7** (+F17, F18, F19) |
| LOW | 5 (F9-F13) | **5** (+F20, same count) |
| Prior findings fixed | — | **0 of 13** |

**Most urgent action**: Wire `RouteFees()` into BeginBlocker (F14). Without this, the protocol's core economic model is inactive on the production chain.
