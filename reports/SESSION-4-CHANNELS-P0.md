# Session 4: Payment Channels P0 Security Fixes

**Date**: 2026-02-18
**Module**: `x/channels`
**Branch**: `feat-cicd-opensource-flow`
**Severity**: P0 (Critical)

---

## Summary

This session resolves the critical P0-1 finding from the channels security audit: **all signature fields were accepted but never verified**, making off-chain state completely unverifiable on-chain. Additionally, the dead `ChannelStatusSettled` state was removed and replaced with a proper `closing` state machine, and dispute window enforcement was hardened.

### Results

| Metric | Value |
|--------|-------|
| Tests passing | **80/80** |
| New tests added | **22** |
| Adversarial tests updated | **5 of 12** (OC-CH-2,3,4,5,9) |
| Files created | 1 (`x/channels/types/signature.go`) |
| Files modified | 3 (`msg_server.go`, `keeper_test.go`, `openclaw_channels_test.go`) |
| Build status | `go build ./...` clean |
| Vet status | Pre-existing proto lock-copy warnings only (not introduced) |

---

## Fix 1: P0-1 — Signature Verification (CRITICAL)

### Problem

All four channel state mutation handlers accepted signature fields but **never verified them**:
- `UpdateChannel` — `msg.Signature` (bytes) ignored
- `CloseChannel` — `msg.CounterpartySignature` (bytes) ignored
- `DisputeChannel` — `msg.ProofSignature` (bytes) ignored
- `UpdateState` — `msg.Update.PayerSignature` and `msg.Update.ProviderSignature` (hex strings) ignored

**Impact**: Any channel party (or third party who passes the party check) could submit arbitrary state without counterparty agreement. Off-chain payment channel security was completely bypassed.

### Solution

Created `x/channels/types/signature.go` with Ed25519 signature verification infrastructure:

```
ChannelSigningPayload(operation, channelId, sequence, spent) → SHA-256 digest
PackSignature(pubkey, sig) → [32-byte pubkey || 64-byte signature]
VerifyPackedSignature(payload, packed, expectedAddr) → error
VerifyHexPackedSignature(payload, hexSig, expectedAddr) → error
```

**Canonical payload format**:
```
SHA-256("legible.channels.v1:{operation}" | "|" | channelId | "|" | BE(sequence) | "|" | spent)
```

**Address derivation**: `SHA-256(Ed25519_pubkey)[:20]` → bech32 with "lgm" prefix

**Verification model**: The transaction sender is authenticated by the Cosmos ante handler. The counterparty's agreement is proven by their Ed25519 signature over the canonical payload. This means:

| Handler | Sender (ante) | Verified Signature |
|---------|---------------|-------------------|
| `UpdateChannel` | payer or provider | counterparty's packed sig |
| `CloseChannel` | payer or provider | counterparty's packed sig |
| `DisputeChannel` | payer or provider | counterparty's packed sig (proof) |
| `UpdateState` | payer or provider | **both** payer + provider hex sigs |

### Files Changed

- **NEW** `x/channels/types/signature.go` — Signing payload builder, packed signature format, Ed25519 verification
- `x/channels/keeper/msg_server.go:116-140` — UpdateChannel signature verification
- `x/channels/keeper/msg_server.go:194-201` — DisputeChannel signature verification
- `x/channels/keeper/msg_server.go:383-385` — CloseChannel signature verification
- `x/channels/keeper/msg_server.go:512-519` — UpdateState dual signature verification

---

## Fix 2: Dead State Machine — `closing` Status

### Problem

`ChannelStatusSettled` was defined in types but never set by any handler. `ChannelStatusClosing` was defined but never set. `SettleChannel` on an open channel immediately distributed funds without any challenge period, bypassing dispute protections.

### Solution

- **Removed** `ChannelStatusSettled` (dead code)
- **Activated** `ChannelStatusClosing` as unilateral close initiation state
- **New state machine** for `SettleChannel`:

```
open → closing (challenge period starts, DisputeDeadline set, zero distribution)
closing → closed (after deadline, funds distributed based on last agreed state)
disputed → closed (after deadline, disputed state used for distribution)
```

- Closing channels **can be disputed** (closing → disputed), protecting the counterparty
- Normal state updates are **blocked during closing** (UpdateChannel rejects non-open channels)

### Files Changed

- `x/channels/types/types.go:17` — Removed `ChannelStatusSettled`
- `x/channels/keeper/msg_server.go:252-358` — Complete SettleChannel rewrite with 3-path state machine
- `x/channels/keeper/msg_server.go:185` — DisputeChannel now accepts `closing` status

---

## Fix 3: Dispute Window Enforcement

### Problem

The dispute window boundary condition needed verification: could settlement occur at the exact deadline block?

### Solution

All deadline checks use **strictly greater than** (`blockHeight > deadline`), meaning:
- At exact deadline: settlement rejected
- One block after deadline: settlement allowed

This is consistent across both closing and disputed states. The check at `msg_server.go:295`:
```go
if uint64(ctx.BlockHeight()) <= ch.DisputeDeadline {
    return nil, types.ErrDisputePeriodActive
}
```

### Test Coverage

`TestDisputeWindow_CannotSettleAtExactDeadline` verifies:
1. At exact deadline block → rejected
2. One block after deadline → accepted

---

## Test Coverage

### New Tests (22 total)

**Signature Verification (6)**:
- `TestSignatureVerification_ValidSignature` — Happy path
- `TestSignatureVerification_WrongAddress` — Pubkey doesn't derive to expected address
- `TestSignatureVerification_TamperedPayload` — Sign one payload, verify against another
- `TestSignatureVerification_TruncatedSignature` — Partial packed sig rejected
- `TestSignatureVerification_EmptySignature` — Nil and empty byte slices rejected
- `TestSignatureVerification_HexRoundtrip` — Hex encoding/decoding fidelity

**Missing Signature Rejection (4)**:
- `TestUpdateState_MissingSignature` — No sigs → rejected
- `TestDisputeChannel_MissingSignature` — No proof sig → rejected
- `TestCloseChannel_MissingSignature` — No counterparty sig → rejected
- `TestCloseChannel_WrongSignerAddress` — Payer signs instead of provider → rejected

**Closing State Machine (5)**:
- `TestSettleChannel_OpensClosingPeriod` — Open → closing with deadline
- `TestSettleChannel_ClosingToFinal` — Closing → closed with fund distribution
- `TestSettleChannel_ClosingBeforeDeadline` — Cannot finalize before deadline
- `TestSettleChannel_AfterDispute` — Disputed → closed with disputed state
- `TestStateMachine_DisputeDuringClosing` — Closing can be disputed

**State Transition Guards (2)**:
- `TestStateMachine_CannotUpdateDuringClosing` — UpdateChannel rejects closing channels
- `TestDisputeWindow_CannotSettleAtExactDeadline` — Boundary condition test

**Signature Integration in Existing Tests (5 OC-CH updated)**:
- `OC-CH-2` (NonceRollback) — Valid sigs added, nonce guard tested
- `OC-CH-3` (PrematureSettlement) — Valid dispute sig added
- `OC-CH-4` (Overspend) — Valid sig added, overspend guard tested (not masked by sig check)
- `OC-CH-5` (NonceFreshness) — Valid sigs for all 4 UpdateState calls
- `OC-CH-9` (DepositToClosedChannel) — Valid close sig for cooperative close

### Updated Existing Tests

All existing tests that call `UpdateChannel`, `CloseChannel`, `DisputeChannel`, or `UpdateState` now provide valid Ed25519 signatures generated from deterministic test key pairs.

**Test Key Infrastructure** (`keeper_test.go:36-65`):
- `testParty` struct: Ed25519 keypair + derived bech32 address
- `newTestParty(name)`: Deterministic key from `SHA-256("test_seed:" + name)`
- `signPacked(party, op, channelId, seq, spent)`: Packed signature helper
- `signHex(party, op, channelId, seq, spent)`: Hex-encoded signature helper

---

## Audit Finding Status

| ID | Finding | Severity | Status |
|----|---------|----------|--------|
| P0-1 | ALL signature fields ignored | P0 | **FIXED** |
| P0-2 | Channel amount validation missing | P0 | FIXED (prior session) |
| P0-3 | Party authorization gaps | P0 | FIXED (prior session) |
| P0-4 | Dispute deadline enforcement | P0 | FIXED (prior session) |
| Dead States | ChannelStatusSettled/Closing unused | P1 | **FIXED** |
| Dispute Window | Boundary condition at exact deadline | P1 | **FIXED** |

---

## Known Issues (Not in Scope)

1. **Proto lock-copy warnings**: `go vet` reports ~50 warnings about protobuf `sync.Mutex` copy. These are pre-existing in all proto-generated types and would require changing the entire keeper interface to use pointer receivers. Not a runtime issue — protobuf MessageState mutex is for lazy initialization, not concurrent access.

2. **BeginBlock only resolves disputes, not closing channels**: Closing channels require manual `SettleChannel` call after deadline. A future enhancement could auto-resolve closing channels in BeginBlock.

3. **No signature replay protection across channels**: The canonical payload includes `channelId`, preventing cross-channel replay. However, there's no chain-ID domain separation in the signing payload. Consider adding for mainnet.
