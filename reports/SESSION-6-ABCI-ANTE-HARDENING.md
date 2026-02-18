# Session 6: ABCI++ and Ante Handler Security Hardening

**Date**: 2026-02-18
**Branch**: `feat-cicd-opensource-flow`
**Scope**: `app/abci.go`, `app/vote_extensions.go`, `app/ante.go`, `app/ante_legible.go`, `x/knowledge/keeper/rounds.go`, `x/knowledge/types/commitment.go`

## Summary

Resolved all P0 findings from the ABCI and Ante Chain security audits, added panic recovery to all ABCI handlers, and enforced VEX injection size limits. Total: **7 security fixes** across **7 files**.

## Findings Addressed

### ABCI P0-1: False Equivocation from Multi-Block Commitment (CRITICAL)
**File**: `app/vote_extensions.go` (lines 161-174)
**Root Cause**: `handleCommitPhase` generated a new random salt every block. If a validator was VRF-selected in consecutive blocks, each call produced a different commitment hash. `StoreCommitmentInRound` treated the second hash as equivocation and slashed 100% of the validator's stake.
**Fix**: Check `LocalCommitmentStore` for an existing commitment before generating a new one. If found, re-emit the same commitment hash, making the duplicate idempotent rather than conflicting.
**Impact**: Prevented innocent validators from being slashed during normal multi-block commit phases.

### ABCI P0-2: StoreRevealInRound Missing Commitment Hash Verification (CRITICAL)
**Files**: `x/knowledge/types/commitment.go` (NEW), `x/knowledge/keeper/rounds.go` (lines 181-188)
**Root Cause**: `StoreRevealInRound` verified the validator committed but never checked that the reveal data (verdict, confidence, salt) actually produced the stored commitment hash. A malicious proposer could tamper with reveal data in the VEX injection tx (e.g., flip verdict from "accept" to "reject") and `StoreRevealInRound` would accept it.
**Fix**:
1. Created `x/knowledge/types/commitment.go` with canonical `ComputeCommitmentHash()` and `VerifyCommitmentHash()` functions using domain-separated SHA-256 (`LGM.commitment.v1:` prefix with length-prefixed fields).
2. Added `types.VerifyCommitmentHash()` call in `StoreRevealInRound` after the commitment existence check.
3. Delegated `app/abci.go:ComputeCommitmentHash()` to the canonical implementation in `knowledgetypes`.
**Impact**: Proposer tampering of reveal data is now detected and rejected at the keeper level.

### ANTE P0-1: msgWithSigners Bypass for SDK v0.50 Proto Types (CRITICAL)
**File**: `app/ante_legible.go` (lines 38-79)
**Root Cause**: `LegibleAccountDecorator`, `LegibleCapabilityDecorator`, and `LegibleDIDDecorator` used a per-message `msgWithSigners` type assertion to extract signer addresses. SDK v0.50 proto-generated types (`MsgSend`, `MsgDelegate`, etc.) do NOT implement this interface. This silently skipped:
- Frozen account checks for all standard Cosmos SDK messages
- Session key capability enforcement for all standard Cosmos SDK messages
- DID-to-signer validation for all standard Cosmos SDK messages
**Fix**: Replaced per-message `msgWithSigners` with tx-level signer extraction via `authsigning.SigVerifiableTx.GetSignaturesV2()`. Two new helpers:
- `getSignerAddresses(tx)` — extracts signer addresses from tx signatures
- `getTxSigners(tx)` — extracts signer addresses AND pubkeys (for session key matching)

All three decorators now use these tx-level extractors, which work for ALL message types.
**Impact**: Frozen accounts and session key capabilities now enforced for standard Cosmos SDK messages (MsgSend, MsgDelegate, MsgUndelegate, etc.).

### ANTE P1-1: ICA Type URL Mismatch
**File**: `app/ante_legible.go` (lines 357-360)
**Root Cause**: `isICAMsg()` used gRPC service paths (`/legible.icaauth.v1.Msg/RegisterInterchainAccount`) but `sdk.MsgTypeURL()` returns proto message names (`/legible.icaauth.v1.MsgRegisterInterchainAccount`). Result: ICA capability enforcement was dead code.
**Fix**: Updated URLs to proto message names.

### ANTE P1-2: uint64 Overflow in Gas Calculation
**File**: `app/ante.go` (lines 107-110)
**Root Cause**: `totalRequiredGas += requiredGas` in the gas accumulation loop had no overflow protection. A transaction with >~843 messages of max gas could overflow uint64, wrapping to a small value and bypassing gas validation.
**Fix**: Added overflow detection: `if totalRequiredGas < requiredGas` after addition.

### Task 5a: Panic Recovery for ABCI Handlers
**Files**: `app/abci.go`, `app/vote_extensions.go`
**Motivation**: An unrecovered panic in any ABCI handler crashes the node. In a BFT network this can halt consensus if >1/3 of validators crash simultaneously.
**Fix**: Added `defer func() { if r := recover() ... }()` to all 5 ABCI entry points:
- `PrepareProposalHandler` — returns empty proposal on panic
- `ProcessProposalHandler` — rejects proposal on panic
- `ExtendVoteHandler` — returns empty extension on panic
- `VerifyVoteExtensionHandler` — accepts extension on panic (preserves liveness)
- `ProcessVoteExtInjection` — skips injection on panic

### Task 5b: VEX Injection Size Limits
**File**: `app/abci.go`
**Motivation**: Without size limits, a malicious proposer could create an oversized VEX injection tx, bloating blocks.
**Fix**:
- `MaxVEXInjectionBytes = 2MB` constant (already existed, now enforced)
- PrepareProposal: drops injection if encoded bytes exceed limit
- ProcessProposal: rejects proposal if VEX injection tx exceeds limit

## Test Results

```
go build ./...                     ✅ Clean
go test ./app/... -count=1         ✅ 42/42 PASS (0.6s)
go test ./x/knowledge/... -count=1 ✅ All PASS (0.4s)
go test ./x/... -count=1           ✅ All PASS (2 pre-existing flaky tests in x/ibcratelimit pass in isolation)
```

### Test Updates
Three existing tests in `x/knowledge/keeper/keeper_test.go` needed updates for P0-2 (commitment hash verification):
- **TestCommitRevealLifecycle**: Updated to compute proper commitment hashes from reveal data using `types.ComputeCommitmentHash()`
- **TestRevealStoresSalt**: Same
- **TestRevealEquivocation_DifferentVerdict**: Updated to expect hash mismatch error (P0-2 catches tampered reveal before equivocation logic)

## Files Changed

| File | Lines Changed | Description |
|------|--------------|-------------|
| `app/vote_extensions.go` | +20 | P0-1 fix (commitment re-use) + panic recovery for ExtendVote/VerifyVoteExtension |
| `x/knowledge/types/commitment.go` | +43 (NEW) | Canonical ComputeCommitmentHash / VerifyCommitmentHash |
| `x/knowledge/keeper/rounds.go` | +8 | P0-2 fix (hash verification in StoreRevealInRound) |
| `app/abci.go` | +45 | Hash delegation, panic recovery (PrepareProposal, ProcessProposal, ProcessVoteExtInjection), VEX size enforcement |
| `app/ante_legible.go` | +42 | P0-1 tx-level signer extraction, P1-1 ICA URL fix |
| `app/ante.go` | +3 | P1-2 overflow detection |
| `x/knowledge/keeper/keeper_test.go` | +25, -20 | Updated 3 tests for P0-2 compatibility |

## Remaining Items (Not Addressed)

| ID | Priority | Description | Reason |
|----|----------|-------------|--------|
| P1-3 | P1 | VRF seed uses per-block prevBlockHash | Requires protocol-level change to VRF seed generation; documented in crypto/vrf.go |
| P2-1 | P2 | ProcessProposal halt check skips VEX injection | Intentional — VEX pseudo-txs bypass halt check to preserve PoT data continuity |
| P2-2 | P2 | PrepareProposal gas estimation heuristic | Acceptable for devnet; production should use SDK's AnteHandler gas estimation |
| P2-4 | P2 | LocalCommitmentStore cleanup age hardcoded to 100 | Low risk; cleanup happens every block, 100 is conservative |
| P2-1 (Ante) | P2 | Bootstrap txs require declared gas limit | UX improvement, not a security issue |
| P2-2 (Ante) | P2 | FeeRouterDecorator logging only | Documentation placeholder; fee routing happens in BeginBlocker |

## Security Impact

| Finding | Before | After |
|---------|--------|-------|
| Multi-block commitment | Innocent validators slashed 100% | Idempotent re-emission, no slash |
| Proposer reveal tampering | Undetected, arbitrary verdict changes | Rejected by hash verification |
| Frozen account bypass | MsgSend/MsgDelegate skip frozen check | All msg types checked |
| Session key bypass | MsgSend/MsgDelegate skip capability check | All msg types checked |
| ICA capability check | Dead code (URL mismatch) | Correctly matches proto URLs |
| Gas overflow | Theoretical uint64 wrap → bypass | Saturating addition with error |
| ABCI panic | Node crash, potential consensus halt | Graceful degradation |
| VEX injection DoS | Unbounded block bloat | 2MB limit enforced |
