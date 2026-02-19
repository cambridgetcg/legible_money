# SESSION-B7-3: PUBKEY-BOOTSTRAP

## Summary

Investigated the reported issue where genesis-funded accounts with `null` pubkey fail with "signature verification failed" on their first non-bank transaction. **The issue does not exist in the current codebase.** The SDK's `SetPubKeyDecorator` is correctly positioned in the ante chain and transparently handles pubkey bootstrapping for ALL message types.

## Root Cause Analysis

### Ante Chain Order (app/ante.go:25-62)

The decorator chain relevant to pubkey handling:

| # | Decorator | Location | Pubkey Relevant? |
|---|-----------|----------|------------------|
| 1 | IBCRedundantRelay | ante.go:27 | No |
| 2 | SetUpContext | ante.go:30 | No |
| 3 | BootstrapGasFree | ante.go:33 | No — checks msg types only |
| 4 | EmergencyHalt | ante.go:36 | No — checks halt state only |
| 5 | LGMGas | ante.go:39 | No — validates gas limits only |
| 6 | FeeRouter | ante.go:40 | No — just logs fee split |
| 7-12 | SDK standard | ante.go:43-48 | No — memo, gas, fee deduction |
| **13** | **SetPubKeyDecorator** | **ante.go:49** | **YES — extracts pubkey from tx signature, sets on account** |
| 14-15 | ValidateSigCount, SigGasConsume | ante.go:50-51 | No |
| **16** | **SigVerificationDecorator** | **ante.go:52** | **YES — reads pubkey from account, verifies signature** |
| 17-21 | Post-auth decorators | ante.go:53-61 | No |

**Key finding**: `SetPubKeyDecorator` (line 49) runs BEFORE `SigVerificationDecorator` (line 52). No custom decorators between them access the account's pubkey. The pubkey is set from the tx's signature data, NOT from the message type. This is message-type agnostic.

### How SetPubKeyDecorator Works (SDK v0.50)

1. Extracts pubkeys from `tx.AuthInfo.SignerInfos[].PublicKey`
2. For each signer, calls `AccountKeeper.GetAccount()`
3. If `account.GetPubKey() == nil`, calls `account.SetPubKey(pk)` and `AccountKeeper.SetAccount()`
4. Writes to the same store context used by subsequent decorators
5. `SigVerificationDecorator` then reads the (now-set) pubkey and verifies the signature

This works identically in both `CheckTx` (mempool) and `DeliverTx` (block execution).

### Custom Decorators Analysis

None of the 5 custom decorators that run BEFORE `SetPubKeyDecorator` access the account's pubkey:

- **BootstrapGasFreeDecorator** (ante_legible.go:93-116): Only checks `BootstrapGasFreeTypes[msgTypeURL]` — no account access
- **EmergencyHaltDecorator** (ante_legible.go:135-152): Only checks `ek.IsHalted(ctx)` — no account access
- **LGMGasDecorator** (ante.go:76-135): Only checks gas limits from `tx.GetGas()` — no account access
- **FeeRouterDecorator** (ante.go:460-484): Only logs fee amounts — no account access
- **SDK's DeductFeeDecorator** (ante.go:48): Gets fee payer from `feeTx.FeePayer()` which derives address from the tx's embedded pubkey (not on-chain pubkey) — no nil-pubkey issue

## Verification on Live Devnet

### Test Protocol

1. Created fresh account `test_bootstrap2` (local keyring only)
2. Funded via `bank send test1 → test_bootstrap2 5000000ulgm`
3. Verified account state: `pub_key: null`, `sequence: 0`, balance: `5000000ulgm`
4. Sent `knowledge submit-claim` as the **first-ever transaction** (NOT a bank send)

### Results

```
=== Account state BEFORE first tx ===
pub_key: null
sequence: 0
balance: 5000000ulgm

=== knowledge submit-claim (first tx ever) ===
CheckTx:  code=0 (PASSED — pubkey set by SetPubKeyDecorator)
DeliverTx: code=0 height=172 (claim submitted successfully)

=== Account state AFTER first tx ===
pub_key: {secp256k1, AwJWDNDGyJ+ccr5+Z5DdhUqjcYp1ZB9JvSa2M0nrmuKB}
sequence: 1
```

**The pubkey was transparently bootstrapped on the first non-bank transaction.** No special handling required.

## Conclusion

### Option B: Already Implemented

`SetPubKeyDecorator` is at `app/ante.go:49`, correctly positioned before `SigVerificationDecorator` at line 52. It handles pubkey bootstrapping for ALL message types (bank, knowledge, staking, etc.) with zero user-facing friction.

### Option A: Not Needed

Adding a pubkey bootstrap phase to devnet/testnet scripts is unnecessary:
- `SetPubKeyDecorator` handles it transparently
- Adding bootstrap sends would slow down init scripts by ~15-20 seconds (3 accounts × 5s wait)
- No functional benefit since the ante chain already works

### Probable Original Issue

The B6-6 testing report of "signature verification failed" was most likely caused by one of:
1. **Insufficient funds** — the error "spendable balance X is smaller than Y: insufficient funds" was observed during our testing when the account didn't have enough ulgm to cover gas fees. This could be misinterpreted as a signature issue.
2. **Earlier codebase version** — if `SetPubKeyDecorator` was missing from an earlier iteration of the ante chain
3. **Stale CLI state** — keyring/account number mismatch from a partial devnet reset

## No Code Changes Required

The ante chain is correctly configured. No scripts need modification.

## Files Examined

| File | Lines | Purpose |
|------|-------|---------|
| `app/ante.go` | 510 | Full ante decorator chain + LGM gas/fee decorators |
| `app/ante_legible.go` | 448 | Custom decorators (Bootstrap, Emergency, Account, DID, Capability) |
| `devnet/scripts/cosmos-devnet.sh` | 361 | Single-node devnet launcher |
| `devnet/scripts/test-lgm-modules.sh` | 867 | Extended module E2E tests |
| `deployment/docker/init-cosmos-docker.sh` | 319 | Docker 3-validator devnet init |
