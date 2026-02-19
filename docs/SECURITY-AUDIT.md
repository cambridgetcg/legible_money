# Pre-Testnet Security Audit

**Date**: 2026-02-19
**Scope**: Full Go chain (`app/`, `x/`, `scripts/`) — ante chain, genesis params, message validation, IBC, panic recovery
**Chain Version**: Cosmos SDK v0.50.10, CometBFT v0.38, IBC-Go v8.5.1

---

## Findings Summary

| Severity | Count | Fixed | Deferred |
|----------|-------|-------|----------|
| Critical | 5 | 2 | 3 |
| High | 12 | 4 | 8 |
| Medium | 8 | 1 | 7 |
| Low | 4 | 0 | 4 |
| **Total** | **29** | **7** | **22** |

---

## Critical Findings

### C-1: MsgAmendLIP Missing ValidateBasic — **FIXED**

**Location**: `x/gov/types/types.go` (was missing entirely)
**Risk**: Proto-generated `MsgAmendLIP` had no `ValidateBasic()`, allowing unbounded `amended_specification` fields (MB+) to enter the mempool and state.
**Fix**: Added `ValidateBasic()` with address validation, non-empty check, and 50KB length bound on `amended_specification`.

### C-2: BVM Bytecode Unbounded in ValidateBasic — **FIXED**

**Location**: `x/bvm/types/types.go:38-46`
**Risk**: `MsgDeployContract.ValidateBasic()` only checked `len(bytecode) > 0` with no upper bound. Attacker could submit 100MB+ bytecode, causing OOM during interpreter init.
**Fix**: Added `MaxDeployBytecodeSize = 1MB` stateless bound. Params-based `MaxBytecodeSize` still checked in keeper (stateful).

### C-3: Empty Treasury Addresses in Genesis — DEFERRED (config)

**Location**: `scripts/testnet-genesis.sh` lines 143, 581
**Risk**: `billing.params.foundation_address` and `vesting_rewards.params.research_treasury_address` are empty strings. Block rewards and research distributions will be lost/stuck.
**Recommendation**: Populate with multisig addresses before genesis init. Add `InitGenesis` validation to reject empty treasury addresses.

### C-4: No BVM Deployment Rate Limiting — DEFERRED (param change)

**Location**: BVM module params
**Risk**: No per-account or per-block rate limiting on contract deployments. `deploy_gas_cost = 200000` is trivial (~27 ulgm). Unlimited state spam.
**Recommendation**: Add `max_deployments_per_account_per_epoch` param (suggest 10). Increase deploy gas cost.

### C-5: No Claim Submission Rate Limiting — DEFERRED (param change)

**Location**: Knowledge module params
**Risk**: Unlimited claim submissions per account force all validators to participate in verification rounds. Combined with low/zero min stake, creates verification spam DoS.
**Recommendation**: Add `max_claims_per_account_per_epoch` param (suggest 10). Enforce minimum claim stake in genesis.

---

## High Findings

### H-1: Zero-Fee Bypass After Bootstrap — DEFERRED (design decision)

**Location**: `app/ante.go:125`
**Risk**: `LGMGasDecorator` skips min gas price check when `feeCoins.IsZero()`. After bootstrap period (480k blocks), a dishonest proposer could include zero-fee txs.
**Note**: Standard Cosmos SDK behavior relies on CometBFT `CheckTx` for gas price enforcement. The ante handler is a secondary defense. Risk is mitigated because proposers must cooperate.

### H-2: Bootstrap InfiniteGasMeter — DEFERRED (by design)

**Location**: `app/ante_legible.go` `BootstrapGasFreeDecorator`
**Risk**: During first 480k blocks (~14 days), eligible tx types get `InfiniteGasMeter`. Block gas limit enforcement may be undermined during bootstrap.
**Mitigation**: Bootstrap is time-limited and only applies to 5 message types. Block-level gas enforcement in `PrepareProposal` provides additional protection.

### H-3: Unbounded Claim Text (Knowledge) — **FIXED**

**Location**: `x/knowledge/types/types.go:408-422`
**Risk**: `MsgSubmitClaim.ClaimText` had no upper bound in `ValidateBasic()`.
**Fix**: Added `MaxClaimTextLength = 10000` (10KB) bound.

### H-4: Unbounded Reason/CounterClaim (Knowledge) — **FIXED**

**Location**: `x/knowledge/types/types.go` (MsgChallengeProvisionalFact, MsgSubmitContradiction)
**Risk**: Challenge reasons and counter-claims could be megabytes.
**Fix**: Added `MaxReasonLength = 5000`, `MaxCounterClaimLength = 10000`, `MaxEvidenceIdsPerMsg = 50`.

### H-5: Unbounded Specification/Justification (Gov) — **FIXED**

**Location**: `x/gov/types/types.go` (MsgSubmitLIP, MsgSubmitDisbursement, MsgAttachUpgradePlan)
**Risk**: `specification`, `justification`, and `info` fields had no length bounds.
**Fix**: Added `MaxSpecificationLength = 51200` (50KB), `MaxReasonLength = 5000`, `MaxUpgradeInfoLength = 10000`.

### H-6: Empty Emergency Genesis Council — DEFERRED (config)

**Location**: `scripts/testnet-genesis.sh:278`
**Risk**: `emergency.params.genesis_council = []`. Emergency halt mechanism cannot be triggered because council is empty.
**Recommendation**: Populate with 3+ validator addresses before genesis.

### H-7: Low Apprentice Validator Stake — DEFERRED (config)

**Location**: `scripts/testnet-genesis.sh:351`
**Risk**: Apprentice min_stake = 111000 ulgm (~0.111 LGM). Sybil attack cost is trivial.
**Recommendation**: Increase 1000x to 111000000 ulgm (~111 LGM) for testnet.

### H-8: Staking Description Fields Unbounded — **FIXED**

**Location**: `x/staking/types/types.go:131-142`
**Risk**: Validator `moniker`, `identity`, `website`, `details` had no length bounds.
**Fix**: Added bounds (moniker 70, identity 70, website 140, details 2000).

### H-9: Unset Tree Treasury Addresses — DEFERRED (config)

**Location**: `scripts/testnet-genesis.sh:572-573`
**Risk**: `research_fund_addr` and `protocol_treasury_addr` are empty. Revenue distributions will be lost.

### H-10: Schedule Module Max Active Too High — DEFERRED (config)

**Location**: `scripts/testnet-genesis.sh:549`
**Risk**: `max_active_per_account = 100`. One account can exhaust the 2M gas/block schedule budget.
**Recommendation**: Reduce to 10.

### H-11: Tree Module Panics in Get Methods — DEFERRED (large refactor)

**Location**: `x/tree/keeper/state.go` (~50 panics)
**Risk**: Only module where `Get*()` methods panic on `store.Get` errors. All other modules return `(nil, false)`. Corrupted state data will crash the validator.
**Recommendation**: Convert all Get/Iterate methods to return `(nil, false)` on error. This is the single most impactful safety refactor.

### H-12: No Channel Creation Rate Limiting — DEFERRED (param change)

**Location**: `scripts/testnet-genesis.sh:192`
**Risk**: 50 channels per payer with 1 ulgm deposit. State bloat vector.
**Recommendation**: Reduce `max_channels_per_payer` to 5, increase `min_deposit` to 100 ulgm.

---

## Medium Findings

### M-1: Session Key CanStake Includes MsgRegisterValidator

**Location**: `app/ante_legible.go` `isStakeMsg()`
**Risk**: Session key with `CanStake` capability can register validators — very privileged.
**Recommendation**: Separate `CanRegisterValidator` from `CanStake`, or require operational key for registration.

### M-2: Session Key CanSubmitClaims Includes MsgChallengeFact

**Location**: `app/ante_legible.go` `isClaimMsg()`
**Risk**: Session key with `CanSubmitClaims` can trigger slashing via challenges.
**Recommendation**: Separate `CanChallenge` from `CanSubmitClaims`.

### M-3: ICA Registration Cooldown Unenforced

**Location**: `x/icaauth/keeper/msg_server.go:28-90`
**Risk**: `RegistrationCooldown = 100 blocks` is defined but never checked in `RegisterInterchainAccount`.
**Recommendation**: Add cooldown enforcement check.

### M-4: ICA No Per-Message Rate Limiting

**Location**: `x/icaauth/keeper/msg_server.go:111-162`
**Risk**: No rate limiting on ICA message execution. Governance spam via ICA votes.
**Recommendation**: Add per-ICA message rate limit per epoch.

### M-5: ICA No Timeout Validation

**Location**: `x/icaauth/keeper/msg_server.go:147`
**Risk**: `msg.TimeoutNanos` passed directly to `SendTx()` without bounds checking.
**Recommendation**: Add min/max timeout bounds (60s to 7 days).

### M-6: No Tree Revenue BPS Sum Validation

**Location**: `x/tree/keeper/genesis.go`
**Risk**: Revenue BPS (evidence_tax + contributors + research_fund + protocol_treasury) not validated to sum to 1000000.
**Recommendation**: Add sum validation in `InitGenesis`.

### M-7: No BPS Upper Bound Validation (Knowledge)

**Location**: `x/knowledge/keeper/genesis.go`
**Risk**: Slash BPS values not validated to be ≤ 1000000.
**Recommendation**: Add BPS range validation in `InitGenesis`.

### M-8: Stake/Amount Strings Not BigInt-Validated

**Location**: Multiple modules (staking, gov)
**Risk**: Malformed stake strings (e.g., "999abc") pass ValidateBasic, fail later in keeper.
**Recommendation**: Add `big.Int.SetString()` validation in ValidateBasic for all stake/amount fields.

---

## Low Findings

### L-1: IBC Rate Limit Zero-Amount Packets

**Location**: `x/ibcratelimit/keeper/middleware.go:130-133`
**Risk**: Zero or negative amount IBC packets pass through untracked.
**Recommendation**: Return error for zero/negative amounts.

### L-2: BVM Contract Address Not Format-Validated

**Location**: `x/bvm/types/types.go` MsgCallContract
**Risk**: Contract address only checked for non-empty, no format validation.

### L-3: IBC Rate Limit State Panics

**Location**: `x/ibcratelimit/keeper/state.go:20-23`
**Risk**: Store operations panic on error (e.g., disk full).
**Recommendation**: Convert to error returns for production.

### L-4: Qualification Min Accuracy Too Low

**Location**: `scripts/testnet-genesis.sh:519`
**Risk**: `min_accuracy_bps = 7700` (0.77%) allows ~99% failure rate. Should be ≥ 600000 (60%) for production.

---

## Panic Audit Summary

| Category | Count | Risk | Action |
|----------|-------|------|--------|
| init()/startup/genesis | ~90 | Acceptable | Standard Cosmos pattern |
| Interface stubs (BVM) | 2 | Acceptable | Never called |
| **tree Get/Iterate panics** | **~50** | **Critical** | **Refactor to return errors** |
| BeginBlocker/EndBlocker Set panics | ~100 | Risky | Convert to error returns |
| MsgServer Set panics | ~200 | Low | Recovered by SDK baseapp |

**Key insight**: The `x/tree/keeper/state.go` is the only module where `Get*()` methods panic on `store.Get` errors. Every other module safely returns `(nil, false)`. BeginBlocker panics are NOT recovered by the SDK.

---

## IBC Security Assessment

### Rate Limiting (x/ibcratelimit)
- **Status**: Well-implemented sliding window algorithm with epoch carry-over
- **Coverage**: Transfer module + ICA controller + ICA host all wrapped
- **Tests**: 43 tests including boundary attack scenarios
- **Defaults**: 10,000 LGM/day per channel (may need increase for production)

### ICA Authorization (x/icaauth)
- **Status**: Dual-layer allowlist (global + per-account)
- **MsgTransfer intentionally excluded** (prevents rate limit bypass via ICA)
- **Tests**: 26 tests including allowlist enforcement
- **Gaps**: Registration cooldown unenforced, no per-message rate limiting

### Middleware Stack
```
Transfer:  transfer → fee → rate limit → channel
ICA ctrl:  icacontroller → rate limit → ICA middleware
ICA host:  icahost → rate limit → ICA middleware
```
All IBC modules properly wrapped with rate limiting.

---

## Ante Chain Assessment

### Decorator Chain (16 decorators)
1. IBCDedup → 2. SetUpContext → 3. Bootstrap → 4. EmergencyHalt → 5. LGMGas → 6. FeeRouter → 7-13. SDK standard → 14. SybilFunding → 15. DID → 16. Account → 17. Capability

### Strengths
- Tx-level signer extraction via `getSignerAddresses()` (SDK v0.50 compatible)
- Default-deny session key capabilities
- VRF proof re-verification in `ProcessVoteExtInjection`
- Panic recovery in all ABCI++ handlers
- MaxVEXInjectionBytes (2MB) size limit
- Commitment hash verification in reveal storage

### Risks
- Zero-fee bypass after bootstrap (H-1)
- InfiniteGasMeter during bootstrap (H-2)
- Session key scope too broad (M-1, M-2)

---

## Remaining Known Risks for Testnet

1. **State spam**: No per-account rate limits on claims, BVM deployments, or channel creation
2. **Tree module panics**: Get methods can crash validators on corrupted state
3. **Empty treasuries**: Block rewards and revenue distributions lost until addresses configured
4. **No emergency council**: Kill switch unavailable until genesis council populated
5. **Low validator stake**: Sybil attack cost is trivial at current apprentice minimums
6. **BeginBlocker panics**: ~100 Set method panics across all modules not recovered by SDK

---

## Pre-Testnet Checklist

- [ ] Populate treasury addresses in genesis script (C-3, H-9)
- [ ] Populate emergency genesis council (H-6)
- [ ] Increase apprentice validator min_stake 1000x (H-7)
- [ ] Reduce schedule max_active_per_account to 10 (H-10)
- [ ] Reduce channels max_channels_per_payer to 5 (H-12)
- [ ] Add BPS sum validation to tree InitGenesis (M-6)
- [ ] Add BPS range validation to knowledge InitGenesis (M-7)
- [ ] Enforce ICA registration cooldown (M-3)
- [ ] Refactor tree/keeper/state.go Get panics (H-11) — large but highest-impact safety improvement
