# PP Review: Gas & Fees Structure

**Date**: 2026-02-10
**Reviewer**: PP (all 30 gates)
**Scope**: Gas metering, fee routing, pricing models across TypeScript prototype + Cosmos SDK Go chain
**Files Reviewed**: 18 primary files across both stacks

---

## Executive Summary

The gas and fees system implements a dual-layer economic model: **transaction fees** (per-tx gas costs) and **API billing** (Truth Gateway per-query pricing). Both converge on the universal 22% research share deduction. The architecture is sound in concept but has critical cross-stack divergences, a broken invariant test, and floating-point contamination in the billing pipeline.

**Overall Score: 37/50**

| Dimension | Score | Max | Notes |
|-----------|-------|-----|-------|
| TRUTH | 6 | 10 | BPS unit mismatch breaks test; float precision contradicts VULN-016 |
| UNDERSTANDING | 8 | 10 | Clear model, good documentation in code comments |
| BEAUTY | 7 | 10 | Clean separation of concerns; some inconsistency in naming |
| JUSTICE | 7 | 10 | Gas costs proportional to complexity; missing proto mappings |
| CREATIVITY | 9 | 10 | Novel dual-stream model; progressive research ramp |

---

## Architecture Map

```
                    ┌─────────────────────────────────────┐
                    │          TRANSACTION FEES            │
                    │                                     │
  User Tx ──────►  │ LGMGasDecorator (validate gas)      │
  (with fee)       │       ↓                              │
                   │ FeeRouterDecorator (log split)       │
                   │       ↓                              │
                   │ DeductFeeDecorator → fee_collector   │
                   │       ↓                              │
                   │ BeginBlocker: RouteFees()            │
                   │   22% → research_fund                │
                   │   78% → fee_collector (validators)   │
                   └─────────────────────────────────────┘

                    ┌─────────────────────────────────────┐
                    │          API BILLING                 │
                    │                                     │
  Agent Query ──►  │ Truth Gateway: calculateQueryPrice() │
                   │       ↓                              │
                   │ ExecutePayment: CalculateDistribution│
                   │   22% → 5-way research treasury      │
                   │   60% → provider                     │
                   │   30% → knowledge pool (by citation) │
                   │   10% → protocol (50% burn / 50% tr) │
                   └─────────────────────────────────────┘

                    ┌─────────────────────────────────────┐
                    │          BLOCK REWARDS               │
                    │                                     │
  Per Block ────►  │ applyBlockReward()                   │
                   │   22% → 5-way research treasury      │
                   │   78% → block producer               │
                   │   + fee pool sweep to producer        │
                   │                                     │
                   │ Finite fund: decays with API revenue │
                   └─────────────────────────────────────┘
```

### Key Files

| Component | TypeScript | Go (Cosmos SDK) |
|-----------|-----------|-----------------|
| Gas costs | `core/billing/gas.ts` | `app/gas.go` |
| Gas limits | `core/billing/gas.ts:225-237` | `app/gas.go:12-17` |
| Gas price | `core/billing/gas.ts:244-253` | `app/gas.go:20-24` |
| Fee routing | `state/transitions.ts:788-814` | `app/ante.go:315-391` |
| Ante handler | N/A (TS has no ante chain) | `app/ante.go:25-57` |
| BVM gas meter | `core/billing/gas.ts:109-218` (costs) | `x/bvm/vm/gas.go` (meter) |
| API pricing | `core/billing/truth_gateway.ts:208-286` | `x/billing/keeper/pricing.go` |
| Distribution | `core/billing/truth_gateway.ts:329-413` | `x/billing/keeper/distribution.go` |
| Reward router | `core/billing/reward_router.ts` | `app/ante.go:359-391` |
| Block rewards | `state/transitions.ts:4872-4958` | N/A (via distribution module) |
| Tests | `tests/unit/state/transitions.test.ts:805-861` | `app/ante_test.go` |

---

## Findings

### F1: CRITICAL — BPS Unit Mismatch in Ante Test

**File**: `app/ante_test.go:177-182`
**Also**: `app/gas.go:228-232`

The fee routing constants use a 1,000,000 = 100% scale:
```go
// gas.go:228-232
ResearchTaxBPS  uint64 = 220000  // 22%
ValidatorFeeBPS uint64 = 780000  // 78%
```

But the invariant test checks against 10,000 = 100% (standard BPS):
```go
// ante_test.go:177-179
func TestFeeRoutingBPSSumTo10000(t *testing.T) {
    if ResearchTaxBPS+ValidatorFeeBPS != 10000 {  // 220000+780000 = 1000000 ≠ 10000
```

**Impact**: This test ALWAYS FAILS. The sum is 1,000,000, not 10,000. The test is named "SumTo10000" but the constants use mega-BPS (1M = 100%). This bug is latent because Go tests are currently blocked by proto redeclaration build errors.

**Risk**: When build errors are fixed, this will be a red test that masks the actual invariant — that research + validator = 100% of fees.

**Remediation**: Change the test to:
```go
func TestFeeRoutingBPSSumToFullBasis(t *testing.T) {
    if ResearchTaxBPS+ValidatorFeeBPS != 1000000 {
```

---

### F2: HIGH — Float Precision Contamination in Truth Gateway

**File**: `core/billing/truth_gateway.ts:226-251, 347-358, 394-409`

The VULN-016 fix in `reward_router.ts` correctly converted all monetary math to pure BigInt basis-point arithmetic. But `truth_gateway.ts` still uses `Number` intermediates with `Math.floor()` / `Math.round()`:

```typescript
// truth_gateway.ts:226-232 — float intermediate before BigInt conversion
const confidenceMultiplier = fact.confidence > 0.5
    ? 1 + (fact.confidence - 0.5) * params.confidencePremiumRate
    : 1;
const confidencePremium =
    (baseCost * BigInt(Math.floor((confidenceMultiplier - 1) * 1000))) / 1000n;
```

```typescript
// truth_gateway.ts:347-358 — Math.round to convert float to BPS
const providerBps = BigInt(Math.round(billingParams.apiProviderShare * 10000));
const knowledgeBps = BigInt(Math.round(billingParams.knowledgePoolShare * 10000));
```

```typescript
// truth_gateway.ts:404 — float weight → BigInt conversion
const shareBps = BigInt(Math.round(share * 10000));
```

**Impact**: For large payment amounts (>2^53 wei), `Math.floor` and `Math.round` introduce rounding errors. At 1000 scale (truth_gateway uses `/1000n`), precision loss is ~0.1% per fact. The knowledge pool distribution also uses float division for weight calculation (line 403: `w.weight / totalWeight`), losing precision when weights are close.

**Contrast**: The Go `x/billing/keeper/distribution.go` correctly uses `math/big` throughout, avoiding this class of bug entirely.

**Remediation**: Convert TS truth_gateway pricing to pure BigInt BPS arithmetic. Store confidence and rates as uint64 BPS values. Follow the pattern in `reward_router.ts:155-179`.

---

### F3: HIGH — Cross-Stack Gas Cost Divergence

**Files**: `core/billing/gas.ts` vs `app/gas.go`

The TS and Go gas cost tables have systematic divergences:

| Transaction | TS Gas | Go Gas | Delta |
|-------------|--------|--------|-------|
| `contract_deploy` / `deploy_contract` | 500,000 | 200,000 | 2.5x |
| `dispute_initiate` / `initiate_dispute` | 250,000 | 100,000 | 2.5x |
| `dispute_settle` / `settle_dispute` | 200,000 | 80,000 | 2.5x |
| `bounty_create` / `create_bounty` | 150,000 | 80,000 | 1.9x |
| `governance_propose` / `governance_propose` | 150,000 | 100,000 | 1.5x |
| `challenge` / `challenge_fact` | 80,000 | 80,000 | match |
| `transfer` / `transfer` | 21,000 | 21,000 | match |

Additionally, the TS type names use camelCase/snake_case mix (`contractDeploy` → `contract_call`) while Go uses pure snake_case (`deploy_contract`). Some names differ: TS `disputeInitiate` vs Go `initiate_dispute`.

**Impact**: Economic models built on TS gas costs produce different fee estimates than the Go chain. Agents that learned pricing from DevNet (TS) will underpay on production chain (Go).

**Remediation**: Establish a single canonical gas table (recommend Go `app/gas.go` as source of truth since it's the production chain) and generate TS constants from it. Add a cross-stack consistency test.

---

### F4: HIGH — System Transaction Exemption Inconsistency

**Files**: `state/transitions.ts:296-301` vs `app/gas.go:241-250`

TS exemptions:
```typescript
const isSystemTx = tx.sender === "system" ||
    (tx.fee === 0n && ["add_fact", "verification_reward", "slash_validator"].includes(tx.type));
```

Go exemptions:
```go
var SystemTxTypes = map[string]bool{
    "verification_reward": true, "slash_validator": true,
    "epoch_transition": true, "emergency_halt": true,
    "emergency_revert": true, "emergency_resume": true,
    "block_reward": true,
}
```

**Divergences**:
- TS exempts `add_fact` — Go does NOT
- Go exempts `epoch_transition`, `emergency_*`, `block_reward` — TS uses `sender === "system"` check instead
- TS exemption is condition-based (fee=0 AND type match); Go is type-only

**Impact**: A consensus node running TS prototype would accept fee-free `add_fact` transactions; the Go chain would reject them without proper fees. This breaks migration compatibility.

**Remediation**: Align exemption lists. `add_fact` is consensus-issued (system tx) — add to Go's `SystemTxTypes`. Add `epoch_transition`, `emergency_*`, `block_reward` to TS's explicit list.

---

### F5: MEDIUM — RouteFees Silently Swallows Module Account Errors

**File**: `app/ante.go:377-386`

```go
err := bk.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, ResearchFundName, researchCoins)
if err != nil {
    // Research fund module account may not exist yet — log and continue.
    ctx.Logger().Warn("failed to route fees to research fund", ...)
}
```

**Impact**: If `research_fund` module account is not registered in `maccPerms` (app.go), 22% of ALL transaction fees silently disappear into the void. The chain continues operating but the economic model breaks — no research funding, no bounties, no sustainability loop.

**Remediation**: Make this a hard error during `InitGenesis` — verify all treasury module accounts exist. Convert the `Warn` to an `Error` + return the error from `RouteFees`.

---

### F6: MEDIUM — PrepareProposal Heuristic Gas Estimation

**File**: `app/abci.go:184-186`

```go
estimatedGas := uint64(len(tx)) * 10
if totalGas+estimatedGas > BlockGasLimit {
    break
}
```

**Impact**: A small-bytesize transaction with expensive operations (e.g., `MsgDeployContract` at 200,000 gas but only ~200 bytes → estimated 2,000 gas) could allow 16x more txs than the block gas limit supports. The actual execution would then exceed `BlockGasLimit`, causing tx failures.

**Note**: This is mitigated by actual gas metering during execution — txs beyond the limit would fail, not be processed. But it means block space is wasted on txs that will fail.

**Remediation**: Use `lookupMsgGas()` for actual gas estimation in PrepareProposal:
```go
for _, msg := range decodedTx.GetMsgs() {
    estimatedGas += lookupMsgGas(sdk.MsgTypeURL(msg))
}
```

---

### F7: MEDIUM — BVM Gas Meter Not Reconciled with SDK Gas Meter

**Files**: `x/bvm/vm/gas.go` vs Cosmos SDK gas meter

The BVM implements its own `GasMeter` (line 8-40) with EIP-style opcode costs (line 43-72). The `MsgCallContract` message carries a `GasLimit` field (x/bvm/types/types.go:67), but there is no evidence that consumed BVM gas is reported back to the SDK's gas meter via `ctx.GasMeter().ConsumeGas()`.

**Impact**: BVM contract executions may consume unlimited SDK gas, bypassing the block gas limit. A contract making 10,000 `sstore` operations (200M BVM gas) would be metered by the BVM but not charged at the SDK level.

**Remediation**: After BVM execution, add `ctx.GasMeter().ConsumeGas(bvmMeter.Used(), "bvm_execution")` in the msg_server handler to reconcile the two meters.

---

### F8: MEDIUM — BPS Scale Inconsistency Across Stacks

**Files**: Multiple

The codebase uses two different BPS scales:

| Location | Scale | 100% = | Example |
|----------|-------|--------|---------|
| TS `reward_router.ts` | Standard BPS | 10,000 | `2200n` = 22% |
| TS `core/constants.ts` | Standard BPS | 10,000 | `7700n` = 77% |
| Go `app/gas.go` | Mega-BPS | 1,000,000 | `220000` = 22% |
| Go `x/billing/types/types.go` | Mega-BPS | 1,000,000 | `220000` = 22% |
| Go `app/ante.go:375` | Mega-BPS | 1,000,000 | `.QuoRaw(1000000)` |
| Var name | Both labeled "BPS" | — | Confusing |

**Impact**: Developers converting between stacks may use wrong divisor (10000 vs 1000000), producing 100x errors. The broken ante_test.go (F1) is a direct result of this confusion.

**Remediation**: Standardize on one scale. Recommend 1,000,000 for Go (higher precision) and 10,000 for TS (BigInt handles precision). Add explicit `const BPS_BASIS = 1_000_000` and `const BPS_BASIS = 10000n` constants with documentation.

---

### F9: MEDIUM — No Dynamic Fee Adjustment

**Files**: `core/billing/gas.ts:244-253`, `app/gas.go:20-24`

Both stacks define `baseGasPrice` / `BaseGasPrice` as constants with a `PriorityMultiple` of 2. The priority multiplier is defined but **never referenced** anywhere in the codebase.

**Impact**: During congestion, all transactions pay the same price. There is no incentive for users to bid higher fees, no mechanism to prioritize urgent knowledge verification over routine transfers. The mempool sorts by `feePerByte` (consensus/mempool/pool.ts:105) but the fee is a flat rate from the gas table.

**Note**: This is a design choice, not a bug. The protocol may intentionally avoid EIP-1559 complexity. However, the dead `PriorityMultiple` constant is confusing.

**Remediation**: Either implement priority pricing (multiply base gas price for tip inclusion) or remove the `PriorityMultiple` / `priorityMultiplier` constants with a comment explaining why dynamic fees are not used.

---

### F10: LOW — Missing Proto URL Gas Mappings

**File**: `app/ante.go:306-312`

```go
// NOTE: The following gas cost keys in TransactionGasCosts have no proto URL
// mapping yet because their tx.proto files have not been created:
//   - Agent Home (10 types): create_home, update_home, etc.
//   - Qualification (4 types): request_qualification, etc.
//   - Auth recovery (8 types): recover_account, freeze_account, etc.
//   - Governance extras: governance_stake_lip, switch_vote, finalize_review
```

22+ transaction types fall back to `MinGasLimit` (22,222 gas) because they lack proto message definitions. Some of these are expensive operations (e.g., `create_home` = 150,000 gas in the cost table, but charged only 22,222 via fallback).

**Impact**: Users submitting these transactions pay ~7x less than intended. Not currently exploitable since the proto files don't exist, but will be a fee undercharge when protos are added if the URL mapping isn't updated simultaneously.

**Remediation**: Add a CI check that ensures every entry in `TransactionGasCosts` has a corresponding entry in `msgTypeURLToGas`, or that unmapped types are explicitly documented with a TODO.

---

### F11: LOW — Quote Validity Uses Number() on BigInt

**File**: `core/billing/truth_gateway.ts:314`

```typescript
validUntil: Number(currentBlock) + 24, // ~1 minute at 2521ms/block
```

**Impact**: After block ~9 quadrillion (2^53), `Number(currentBlock)` loses precision. Not a practical concern at 2521ms/block (~716 million years), but violates the principle of not using `Number()` on BigInt values.

**Remediation**: Change to `Number(currentBlock + 24n)` or store as string.

---

### F12: LOW — TS Prototype Has No Runtime Gas Metering

**Files**: `state/transitions.ts`, `consensus/pot/types.ts:23-113`

The TS prototype defines gas costs (`core/billing/gas.ts`) but never enforces them at runtime. Transaction execution in `state/transitions.ts` only checks `tx.fee <= balance` — it does not validate `tx.fee >= estimateTransactionGas(tx.type)`. The `ConsensusParams` interface and `BlockHeader` have no gas-related fields at all.

In contrast, the Go chain enforces gas via `LGMGasDecorator` (checks gasLimit >= required gas for msg types) and `DeductFeeDecorator` (deducts from balance).

**Impact**: On TS DevNet, users can submit any transaction with `fee: 0n` (only system txs are truly exempted, but any zero-fee tx where the handler doesn't fail passes). On Go production chain, zero-fee txs are rejected by the ante handler. This means agents trained on DevNet behavior will fail on production.

**Remediation**: Add a pre-handler gas validation step in TS `executeTransaction()` that checks `tx.fee >= estimateTransactionGas(tx.type) * GAS_PRICE.minimumGasPrice`, or accept this as intentional for DevNet prototyping and document it.

---

### F13: LOW — Fee Deduction Timing in TS vs Go

**Files**: `state/transitions.ts:788-814` vs `app/ante.go:46`

In the TS prototype, fees are deducted **after** the transaction handler succeeds (line 788-814, inside the try block after the switch statement). In the Cosmos SDK chain, fees are deducted **before** message execution (via `DeductFeeDecorator` at ante.go:46, which runs before message handlers).

**Impact**: In TS, if a transaction fails, the fee is not charged (the transaction rolls back). In Go, fees are always charged regardless of execution outcome. This is standard Cosmos SDK behavior and is the correct approach for production — but it means DevNet (TS) gives free retries while mainnet (Go) does not.

**Remediation**: Document this behavioral difference. Consider adding fee deduction before handler execution in TS for testing parity, or accept the divergence as intentional for DevNet.

---

## Test Coverage Assessment

| Area | TS Tests | Go Tests | Gaps |
|------|----------|----------|------|
| Fee routing (22/78 split) | 1 test (transitions.test.ts:805-861) | 0 integration tests | No Go integration test |
| Gas cost lookup | 0 explicit | 2 tests (ante_test.go:13-52) | No TS gas cost tests |
| Gas limits invariant | 0 | 1 test (ante_test.go:159-175) | — |
| BPS sum invariant | 0 | 1 test **(BROKEN - F1)** | Critical |
| LGMGasDecorator | 0 | 6 tests (ante_test.go:350-437) | Good Go coverage |
| GetMinimumFee | 0 | 3 tests (ante_test.go:234-276) | Good Go coverage |
| Proto URL completeness | 0 | 2 tests (ante_test.go:280-327) | Good Go coverage |
| System tx exemptions | 1 test (transitions.test.ts) | 1 test (ante_test.go:200-217) | — |
| API pricing formula | 0 | 0 | **No tests for calculateQueryPrice** |
| Cross-stack consistency | 0 | 0 | **No cross-stack validation** |
| BVM gas reconciliation | 0 | 0 | **No test for BVM↔SDK meter** |

---

## Positive Findings

1. **P1: Clean architecture** — Gas costs, limits, and prices are centralized in dedicated files (`gas.ts`, `gas.go`), not scattered across modules.

2. **P2: Comprehensive Go ante handler chain** — 17 decorators in correct order with clear comments. Emergency halt gate placed before gas validation.

3. **P3: Init-time gas validation** — `app/ante.go:409-416` panics at startup if any gas cost exceeds `TxGasLimit`, preventing misconfiguration.

4. **P4: Progressive research ramp** — `GetEffectiveResearchShareBps()` in both stacks implements linear ramp from 5%→22% to avoid discouraging early adopters. Well-tested in Go.

5. **P5: 5-way treasury split with remainder handling** — Go `distribution.go:79-84` gives remainder to reserve fund to avoid rounding loss. TS `splitEvidenceTax()` does the same.

6. **P6: Atomic fee distribution** — Both stacks wrap fee operations in transactions (`beginTransaction`/`commitTransaction` in TS, bank module atomicity in Go).

7. **P7: Thorough Go ante tests** — 18 test functions covering gas lookup, limits, decorators, fee calculation, proto URL completeness.

---

## Resolution Priority

| Finding | Severity | Effort | Priority |
|---------|----------|--------|----------|
| F1: BPS test mismatch | CRITICAL | 5 min | P0 — Fix immediately |
| F2: Float precision in gateway | HIGH | 2 hr | P1 — Before production |
| F3: Cross-stack gas divergence | HIGH | 1 hr | P1 — Before migration |
| F4: System tx exemption mismatch | HIGH | 30 min | P1 — Before migration |
| F5: RouteFees swallows errors | MEDIUM | 15 min | P2 |
| F6: PrepareProposal gas heuristic | MEDIUM | 30 min | P2 |
| F7: BVM gas reconciliation | MEDIUM | 1 hr | P2 |
| F8: BPS scale inconsistency | MEDIUM | 1 hr | P2 — Documentation |
| F9: Dead priority multiplier | MEDIUM | 15 min | P3 |
| F10: Missing proto mappings | LOW | CI setup | P3 |
| F11: Number() on BigInt | LOW | 5 min | P3 |
| F12: No TS runtime gas metering | LOW | 30 min | P3 |
| F13: Fee deduction timing | LOW | Documentation | P4 |

---

## Gate Scores (Detailed)

### TRUTH (6/10)
- **Grounding**: Gas costs grounded in EIP conventions and computational complexity. ✓
- **Calibration**: BPS unit mismatch (F1, F8) shows miscalibration between stacks. ✗
- **Counter-Consideration**: No dynamic pricing considered — acceptable if intentional. ~
- **Completeness**: 22+ unmapped proto URLs (F10). Missing test coverage for pricing. ✗
- **Representation**: Float contamination (F2) contradicts VULN-016 fix. ✗

### UNDERSTANDING (8/10)
- **Mental Model**: Clear dual-stream model (tx fees + API billing). ✓
- **Why**: Research funding rationale well-documented in code comments. ✓
- **Connection**: Fee→research→bounty→knowledge loop clearly traced. ✓
- **Boundary**: System tx exemptions defined (though inconsistent — F4). ~
- **Application**: Good test coverage in Go; weak in TS pricing. ~

### BEAUTY (7/10)
- **Emergence**: Gas system emerges naturally from computational costs. ✓
- **Fit**: Architecture fits Cosmos SDK patterns well. ✓
- **Elegance**: Clean separation: gas.ts/gas.go for costs, ante.go for validation. ✓
- **Coherence**: BPS scale inconsistency (F8) breaks coherence. ✗
- **Clarity**: Good naming except cross-stack divergence (F3). ~

### JUSTICE (7/10)
- **Placement**: Gas costs properly placed in dedicated files. ✓
- **Proportion**: Gas costs proportional to operation complexity. ✓
- **Balance**: 22%/78% split well-justified by protocol goals. ✓
- **Responsibility**: RouteFees error handling too lenient (F5). ✗
- **Integrity**: BVM gas isolation lacks SDK reconciliation (F7). ✗

### CREATIVITY (9/10)
- **Territory**: Novel dual-stream economic model. ✓
- **Pioneering**: Progressive research ramp is innovative. ✓
- **Path-Building**: Citation-weighted knowledge pool distribution. ✓
- **Extension**: Sustainability condition (API revenue > block decay). ✓
- **Vision**: Clear path from TS prototype to Go production. ✓
