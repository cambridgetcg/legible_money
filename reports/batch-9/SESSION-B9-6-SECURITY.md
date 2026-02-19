# B9-6: Pre-Testnet Security Hardening — Session Report

## Summary

Completed a full security audit of the Go chain across 5 attack surfaces: ante handler chain, genesis parameters, message validation (ValidateBasic), IBC security, and panic recovery. Identified 29 findings (5 Critical, 12 High, 8 Medium, 4 Low). Applied 7 fixes to the highest-impact issues. Documented all findings with severity, fix status, and recommendations in `docs/SECURITY-AUDIT.md`.

---

## Audit Scope

| Area | Files Audited | Method |
|------|---------------|--------|
| Ante chain | `app/ante.go`, `app/ante_legible.go`, `app/gas.go`, `app/abci.go`, `app/sybil_funding_decorator.go`, `app/vote_extensions.go` | Manual line-by-line |
| Genesis params | `scripts/testnet-genesis.sh`, all `proto/*/genesis.proto` | Agent-assisted |
| ValidateBasic | `x/knowledge/types/types.go`, `x/bvm/types/types.go`, `x/tokens/types/types.go`, `x/staking/types/types.go`, `x/gov/types/types.go` | Agent-assisted |
| IBC security | `x/ibcratelimit/`, `x/icaauth/`, `app/app.go` IBC wiring | Agent-assisted |
| Panic recovery | All `x/*/keeper/*.go`, `app/*.go` (non-test) | Agent-assisted + grep |

---

## Fixes Applied

### 1. MsgAmendLIP ValidateBasic — CRITICAL

**File**: `x/gov/types/types.go`
**Change**: Added missing `ValidateBasic()` and `GetSigners()` for proto-generated `MsgAmendLIP`. Validates proposer address, non-empty lip_id, non-empty amended_specification, and specification length ≤ 50KB.

### 2. Gov String Length Bounds — HIGH

**File**: `x/gov/types/types.go`
**Change**: Added length constants and bounds checks:
- `MaxTitleLength = 200` (already existed, now using constant)
- `MaxDescriptionLength = 2000` (already existed, now using constant)
- `MaxSpecificationLength = 51200` (NEW — applies to MsgSubmitLIP.Specification, MsgAmendLIP.AmendedSpecification)
- `MaxReasonLength = 5000` (NEW — applies to MsgSubmitDisbursement.Justification)
- `MaxUpgradeInfoLength = 10000` (NEW — applies to MsgAttachUpgradePlan.Info)

### 3. BVM Bytecode Size Cap — CRITICAL

**File**: `x/bvm/types/types.go`
**Change**: Added `MaxDeployBytecodeSize = 1MB` and `MaxInputDataSize = 64KB` stateless bounds. `MsgDeployContract.ValidateBasic()` now rejects bytecode > 1MB. `MsgCallContract.ValidateBasic()` now rejects input_data > 64KB.

### 4. Knowledge Claim Text Bound — HIGH

**File**: `x/knowledge/types/types.go`
**Change**: Added `MaxClaimTextLength = 10000` bound to `MsgSubmitClaim.ValidateBasic()` and `MsgAddFact.ValidateBasic()`.

### 5. Knowledge Challenge Bounds — HIGH

**File**: `x/knowledge/types/types.go`
**Change**: Added bounds to `MsgChallengeProvisionalFact` (reason ≤ 5KB, evidence IDs ≤ 50) and `MsgSubmitContradiction` (counter-claim ≤ 10KB, evidence IDs ≤ 50).

### 6. Staking Description Bounds — HIGH

**File**: `x/staking/types/types.go`
**Change**: Added bounds to `MsgRegisterValidator.ValidateBasic()` for description fields: moniker ≤ 70, identity ≤ 70, website ≤ 140, details ≤ 2000.

---

## Key Findings (Not Fixed — Deferred)

### Critical (Deferred)

| ID | Finding | Module | Reason for Deferral |
|----|---------|--------|---------------------|
| C-3 | Empty treasury addresses in genesis | billing, vesting_rewards | Config change, not code — must be set before genesis init |
| C-4 | No BVM deployment rate limiting | bvm | Requires new params + keeper logic (separate task) |
| C-5 | No claim submission rate limiting | knowledge | Requires new params + keeper logic (separate task) |

### High (Deferred)

| ID | Finding | Module | Reason for Deferral |
|----|---------|--------|---------------------|
| H-1 | Zero-fee bypass after bootstrap | app/ante.go | By design — CometBFT CheckTx is primary defense |
| H-2 | Bootstrap InfiniteGasMeter | app/ante_legible.go | By design — time-limited to 480k blocks |
| H-6 | Empty emergency genesis council | emergency | Config change, must be set before genesis |
| H-7 | Low apprentice validator stake | staking | Config change, must be set before genesis |
| H-9 | Unset tree treasury addresses | tree | Config change |
| H-10 | Schedule max_active_per_account = 100 | schedule | Config change |
| H-11 | Tree Get method panics (~50) | tree/keeper/state.go | Large refactor — needs dedicated session |
| H-12 | No channel creation rate limiting | channels | Config change |

### Panic Audit Results

- **~441 total panics** in non-test Go code
- **~90 acceptable** (init, startup, genesis — standard Cosmos pattern)
- **~50 CRITICAL** in `x/tree/keeper/state.go` — only module where Get methods panic
- **~100 risky** in BeginBlocker/EndBlocker Set paths (not recovered by SDK)
- **~200 low** in MsgServer paths (recovered by SDK baseapp)

### IBC Assessment

- Rate limiting: Well-architected sliding window with carry-over prevention
- ICA auth: Dual-layer allowlist, MsgTransfer intentionally excluded
- Test coverage: 69 tests across both modules
- Gaps: Registration cooldown unenforced (M-3), no per-message rate limit (M-4)

---

## Files Changed

### Modified
- **`x/gov/types/types.go`** — Added MsgAmendLIP.ValidateBasic(), length constants, bounds on 4 messages
- **`x/bvm/types/types.go`** — Added MaxDeployBytecodeSize, MaxInputDataSize, bounds on 2 messages
- **`x/knowledge/types/types.go`** — Added length constants, bounds on 4 messages
- **`x/staking/types/types.go`** — Added description field bounds on MsgRegisterValidator

### New
- **`docs/SECURITY-AUDIT.md`** — Full security checklist with all 29 findings

---

## Verification

```
go build ./x/gov/types/...        # Pass
go build ./x/bvm/types/...        # Pass
go build ./x/knowledge/types/...  # Pass
go build ./x/staking/types/...    # Pass
go test ./x/gov/... -count=1      # 3 packages pass
go test ./x/bvm/... -count=1      # 3 packages pass
go test ./x/knowledge/... -count=1 # 5 packages pass
go test ./x/staking/... -count=1  # 2 packages pass
```

Zero test regressions across all affected modules.

---

## Ante Chain Strengths (Confirmed)

1. **P0-1 fix verified**: `getSignerAddresses()` extracts signers from tx signatures, not per-message `GetSigners()`. All SDK v0.50 messages properly handled.
2. **Default-deny capabilities**: Session keys cannot do anything not explicitly granted.
3. **VRF re-verification**: `ProcessVoteExtInjection` re-verifies VRF proofs (GAP-2 fix).
4. **Panic recovery**: All 4 ABCI++ handlers wrap in `defer/recover`.
5. **Commitment hash verification**: Reveals must match commitments (prevents proposer tampering).
6. **Deterministic injection**: VEX commitments/reveals sorted before injection.
7. **Size limits**: `MaxVEXInjectionBytes = 2MB` prevents oversized vote extensions.

---

## Recommended Next Steps (Priority Order)

1. **Before genesis**: Set treasury addresses, emergency council, validator stake minimums
2. **Dedicated task**: Refactor `x/tree/keeper/state.go` Get panics → error returns (~50 locations)
3. **Dedicated task**: Add per-account rate limits for claims and BVM deployments
4. **Future**: Convert BeginBlocker Set panics to error returns across all modules
5. **Future**: Add BPS sum validation to tree and knowledge InitGenesis
6. **Future**: Enforce ICA registration cooldown
