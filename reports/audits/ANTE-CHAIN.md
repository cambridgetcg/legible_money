# Ante Handler Chain Audit

> **Date**: 2026-02-18
> **Scope**: `app/ante.go` (506 lines), `app/ante_legible.go` (398 lines), `app/sybil_funding_decorator.go` (76 lines), `app/gas.go` (387 lines)
> **Previous review**: `.claude/memory/ANTE_HANDLER_REVIEW.md` (2026-02-10, 8 issues found — most fixed)
> **This review**: Delta audit on current code, focused on security, ordering, and newly added decorators

---

## 1. Complete Decorator Chain (21 Decorators)

The comment says "16 decorators" but the actual chain has **21**. Here is the full execution order:

| # | Decorator | Source | Phase | Validates/Enforces | Modifies | Can Reject? |
|---|-----------|--------|-------|-------------------|----------|-------------|
| 1 | `ibcante.RedundantRelayDecorator` | IBC-Go | IBC | Filters duplicate IBC relay packets | Nothing | Yes — duplicate relay |
| 2 | `ante.SetUpContextDecorator` | Cosmos SDK | Gas Init | Nothing | Sets gas meter on ctx | No |
| 3 | `BootstrapGasFreeDecorator` | Custom | Bootstrap | Checks block < 480,000 && all msgs eligible | Replaces gas meter with InfiniteGasMeter | No (passthrough) |
| 4 | `EmergencyHaltDecorator` | Custom | Emergency | Chain halt state; allows only emergency msgs | Nothing | Yes — `ErrChainHalted` |
| 5 | `LGMGasDecorator` | Custom | LGM Pre-Auth | Gas limit vs block/tx limits; min gas per msg type; min fee | Nothing | Yes — gas/fee insufficient |
| 6 | `FeeRouterDecorator` | Custom | LGM Pre-Auth | Nothing (logging only) | Nothing | No |
| 7 | `ante.ExtensionOptionsDecorator` | Cosmos SDK | Standard | Rejects unknown extension options | Nothing | Yes — unknown extensions |
| 8 | `ante.ValidateBasicDecorator` | Cosmos SDK | Standard | Calls `ValidateBasic()` on msgs | Nothing | Yes — validation fail |
| 9 | `ante.TxTimeoutHeightDecorator` | Cosmos SDK | Standard | Tx timeout height vs current block | Nothing | Yes — expired tx |
| 10 | `ante.ValidateMemoDecorator` | Cosmos SDK | Standard | Memo length limit | Nothing | Yes — memo too long |
| 11 | `ante.ConsumeGasForTxSizeDecorator` | Cosmos SDK | Standard | None | Consumes gas proportional to tx bytes | Yes — OOG |
| 12 | `ante.DeductFeeDecorator` | Cosmos SDK | Standard | Fee payer has sufficient balance | Deducts fee from account | Yes — insufficient funds |
| 13 | `ante.SetPubKeyDecorator` | Cosmos SDK | Standard | None | Sets pubkey on account if first tx | No |
| 14 | `ante.ValidateSigCountDecorator` | Cosmos SDK | Standard | Signature count within limit | Nothing | Yes — too many sigs |
| 15 | `ante.SigGasConsumeDecorator` | Cosmos SDK | Standard | None | Consumes gas per signature | Yes — OOG |
| 16 | `ante.SigVerificationDecorator` | Cosmos SDK | **AUTH** | Cryptographic signature verification | Nothing | Yes — bad signature |
| 17 | `ante.IncrementSequenceDecorator` | Cosmos SDK | Standard | None | Increments account sequence (nonce) | No |
| 18 | `SybilFundingDecorator` | Custom | Post-Auth | None | Records MsgSend sender→recipient funding graph | No |
| 19 | `LegibleDIDDecorator` | Custom | Post-Auth | DID format, resolution, signer match | Emits `did_reference` event | Yes — DID resolution fail |
| 20 | `LegibleAccountDecorator` | Custom | Post-Auth | Frozen account check | Updates `LastActiveBlock` | Yes — `ErrAccountFrozen` |
| 21 | `LegibleCapabilityDecorator` | Custom | Post-Auth | Session key capabilities vs msg type | Nothing | Yes — capability denied |

### Data Flow Diagram

```
TX arrives
  │
  ▼
[1] IBC: Filter duplicate relay ──reject──→ DROP
  │ pass
  ▼
[2] SetUpContext: Initialize gas meter (ctx.GasMeter)
  │
  ▼
[3] Bootstrap: block ≤ 480k && all msgs eligible? → InfiniteGasMeter
  │
  ▼
[4] Emergency: chain halted? → only emergency msgs pass
  │
  ▼
[5] LGMGas: gas ≤ BlockGasLimit, gas ≤ TxGasLimit, gas ≥ Σ(msgGas), fee ≥ gas*price
  │
  ▼
[6] FeeRouter: (log only)
  │
  ▼
[7-10] SDK validation: extensions, ValidateBasic, timeout, memo
  │
  ▼
[11] ConsumeGasForTxSize: consume gas proportional to tx byte size
  │
  ▼
[12] DeductFee: deduct fee from payer account
  │
  ▼
[13-15] Signature setup: set pubkey, validate sig count, consume sig gas
  │
  ▼
[16] SigVerification: ═══ AUTHENTICATION BOUNDARY ═══
  │
  ▼
[17] IncrementSequence: bump nonce (replay protection)
  │
  ▼
[18] SybilFunding: record MsgSend/MsgMultiSend funding relationships
  │
  ▼
[19] DID: validate did:lgm: memo, resolve to signer
  │
  ▼
[20] Account: check frozen, update LastActiveBlock
  │
  ▼
[21] Capability: enforce session key permissions (default deny)
  │
  ▼
MSG HANDLER
```

---

## 2. Custom Decorator Deep Dive

### BootstrapGasFreeDecorator (`ante_legible.go:22-57`)

**Purpose**: During the first ~14 days (480,000 blocks), essential PoT transactions pay no gas, allowing the network to bootstrap truth verification before any LGM is minted.

**Eligible message types** (`gas.go:343-349`):
- `MsgSubmitClaim`, `MsgSubmitCommitment`, `MsgSubmitReveal`
- `MsgRegisterValidator`, `MsgRegisterAccount`

**Behavior**:
- ALL messages in the tx must be eligible (one non-eligible msg → normal gas)
- Sets `ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())` — all downstream gas consumption is free
- The tx still needs a declared gas limit for `LGMGasDecorator` (see P2-1)
- Skips for blocks > `BootstrapEndBlock`

**Assessment**: Clean implementation. The all-or-nothing check prevents abuse of packaging non-eligible msgs with eligible ones.

### EmergencyHaltDecorator (`ante_legible.go:59-93`)

**Purpose**: When chain is in emergency halt state, blocks all non-emergency transactions.

**Behavior**:
- Queries `ek.IsHalted(ctx)` — if false, passthrough
- If halted: every message must satisfy `emergencytypes.IsEmergencyMsg(msg)` (the 6 emergency msg types: ProposeHalt, VoteHalt, ProposeRevert, VoteRevert, ProposeResume, VoteResume)
- Mixed txs (emergency + non-emergency) are rejected
- Skips in simulation mode

**Assessment**: Correctly placed early (position 4) so halted txs don't consume gas or fees. The simulation skip is appropriate.

### LGMGasDecorator (`ante.go:65-131`)

**Purpose**: Enforces per-message-type minimum gas limits from the LGM gas cost table.

**Gas validation logic**:
1. `gasLimit > BlockGasLimit (33,333,333)` → reject
2. `gasLimit > TxGasLimit (11,111,111)` → reject
3. Sum gas requirements per message type from `msgTypeURLToGas` map
4. Clamp sum to `MinGasLimit (22,222)` minimum
5. `gasLimit < totalRequiredGas` → reject
6. If fee > 0: `feeAmount < gasLimit * MinGasPrice (1)` → reject

**Gas cost table** (`ante.go:146-441`): Maps **120+ proto message type URLs** to gas costs. Unknown types fall back to `MinGasLimit = 22,222`.

**Fee check**: `if !feeCoins.IsZero()` — zero fee txs skip the price check. This enables bootstrap gas-free txs (gas declared but fee=0).

### FeeRouterDecorator (`ante.go:443-480`)

**Purpose**: Logging only. Documents the 7%/93% research/validator fee split.

The actual fee routing is performed by `vesting_rewards.BeginBlock → keeper.RouteFees()` (`x/vesting_rewards/module.go:150-154`), which sweeps 7% of `fee_collector` to `research_fund` before `x/distribution` sends the remaining 93% to validators.

### SybilFundingDecorator (`sybil_funding_decorator.go:10-75`)

**Purpose**: Intercepts MsgSend/MsgMultiSend to record sender→recipient funding relationships. These records feed into `ComputeSybilDecayBPS()` which applies vote-weight decay to accounts with shared funding sources (sybil resistance for quadratic voting).

**Behavior**:
- Only records for `ulgm` denomination transfers
- For MsgMultiSend: records each input→output pair (O(n*m) records)
- Nil-safe: skips if gov keeper not wired
- Skips simulation (recording is a state mutation)

**Assessment**: Well-designed. Passive recording with no rejection. The correlation engine (`sybil.go:201-257`) is separate from the recording.

### LegibleDIDDecorator (`ante_legible.go:317-397`)

**Purpose**: Validates DID references in transaction memos. If memo starts with `did:lgm:`, validates the DID resolves to one of the message signers.

**Behavior**:
- Only triggers on memos starting with `did:lgm:`
- Validates DID format: `did:lgm:{32 or 64 hex chars}`
- Resolves DID to bech32 address via `lak.GetAddressForDID()`
- Verifies resolved address matches a message signer
- Emits `did_reference` event for indexing
- Non-DID transactions pass through unaffected

**Assessment**: Now correctly placed post-auth (position 19). State reads are metered and authenticated. The 72-character truncation (`ante_legible.go:352-354`) correctly handles `did:lgm:` (8) + 64 hex = 72.

### LegibleAccountDecorator (`ante_legible.go:95-153`)

**Purpose**: Frozen account enforcement + LastActiveBlock tracking.

**Behavior**:
- Iterates messages, extracts signers via `msgWithSigners` interface
- If signer has a registered LegibleAccount and it's frozen → reject
- Updates `LastActiveBlock` to current height (even if tx later fails in msg handler)

**Critical limitation**: See **P0-1** below.

### LegibleCapabilityDecorator (`ante_legible.go:155-315`)

**Purpose**: Enforces session key capability restrictions.

**Behavior**:
- For each message signer, checks if the signing pubkey matches a registered session key
- If session key: enforces capabilities (CanTransfer, CanStake, CanSubmitClaims, CanVote, CanIca)
- Default deny: unrecognized message types are rejected for session keys
- Non-session-key signatures (operational/identity key) pass unrestricted

**Capabilities mapped**:
| Capability | Message Types |
|-----------|---------------|
| `CanTransfer` | MsgSend, MsgMultiSend, IBC MsgTransfer |
| `CanStake` | MsgDelegate, MsgUndelegate, MsgBeginRedelegate, MsgRegisterValidator |
| `CanSubmitClaims` | MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgChallengeFact |
| `CanVote` | MsgVote, MsgCastVote, MsgVoteHalt, MsgVoteRevert, MsgVoteResume |
| `CanIca` | ICA registration (always denied for session keys), ICA send tx |

**Critical limitation**: See **P0-1** below.

---

## 3. Findings

### P0-1 (CRITICAL): `msgWithSigners` bypass — Frozen accounts and session key capabilities not enforced for standard SDK messages

**Location**: `ante_legible.go:18-21, 120-124, 187-191`

```go
type msgWithSigners interface {
    GetSigners() []sdk.AccAddress
}
```

Both `LegibleAccountDecorator` and `LegibleCapabilityDecorator` extract signers via:
```go
ms, ok := msg.(msgWithSigners)
if !ok {
    continue  // ← SILENTLY SKIPS standard SDK messages
}
```

**Problem**: In Cosmos SDK v0.50.10, proto-generated message types (MsgSend, MsgDelegate, MsgVote, etc.) **do NOT implement `GetSigners() []sdk.AccAddress`**. This method was removed in the v0.47→v0.50 migration in favor of the `cosmos.msg.v1.signer` proto annotation. Verified by inspecting the SDK source:
- `/Users/yu/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.50.10/x/bank/types/msgs.go` — NO `GetSigners` method
- `/Users/yu/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.50.10/x/bank/types/tx.pb.go` — NO `GetSigners` method

**Impact**:

1. **Frozen account bypass**: A frozen LegibleAccount can send MsgSend, MsgDelegate, MsgVote, and ALL standard Cosmos SDK messages. The `msg.(msgWithSigners)` assertion fails, the `continue` skips the frozen check, and the transaction proceeds normally. Account freezing only blocks Legible-specific hand-written message types.

2. **Session key capability bypass**: A session key with `CanTransfer=false` can send MsgSend because the capability check is never reached. The `continue` skips the `findSessionByPubKey` + `checkCapability` calls entirely. Session key restrictions only apply to Legible-specific messages.

**Severity**: CRITICAL — Account freeze is a core security mechanism for compromised keys and abuse prevention. If a frozen account can still transfer funds via MsgSend, the freeze is effectively useless for its primary purpose.

**Fix**: Extract signers from the transaction's signature set instead of per-message `GetSigners()`:

```go
// Option A: Use the tx-level signer extraction (SDK v0.50 pattern)
sigTx, ok := tx.(authsigning.SigVerifiableTx)
if !ok {
    return next(ctx, tx, simulate)
}
signers, err := sigTx.GetSignaturesV2()
// Extract signer addresses from signatures...

// Option B: Use the signing context from app configuration
// app.txConfig.SigningContext().GetSigners(tx)
```

---

### P1-1: ICA type URL mismatch makes `CanIca` capability dead code

**Location**: `ante_legible.go:304-306`

```go
func isICAMsg(msgType string) bool {
    return msgType == "/legible.icaauth.v1.Msg/RegisterInterchainAccount" ||
        msgType == "/legible.icaauth.v1.Msg/SendICATransaction"
}
```

But `sdk.MsgTypeURL(msg)` returns proto message format:
```
/legible.icaauth.v1.MsgRegisterInterchainAccount
/legible.icaauth.v1.MsgSendICATransaction
```

Note the difference: `/Msg/Register...` (gRPC service path) vs `.MsgRegister...` (proto message name). These will **never match**.

**Impact**: The `CanIca` session key capability field has no effect. ICA messages always fall to the `default` case in `checkCapability()`, which returns `ErrSessionCapabilityDenied`. The fail-closed behavior is **secure** (ICA is denied), but the feature is broken — there's no way to enable ICA for session keys even with `CanIca=true`.

**Fix**:
```go
func isICAMsg(msgType string) bool {
    return msgType == "/legible.icaauth.v1.MsgRegisterInterchainAccount" ||
        msgType == "/legible.icaauth.v1.MsgSendICATransaction"
}
```

---

### P1-2: `totalRequiredGas` addition has no overflow protection

**Location**: `ante.go:102-107`

```go
var totalRequiredGas uint64
for _, msg := range msgs {
    msgType := sdk.MsgTypeURL(msg)
    requiredGas := lookupMsgGas(msgType)
    totalRequiredGas += requiredGas
}
```

**Problem**: No overflow check on the `+=`. A tx with many messages could theoretically overflow `uint64`. Practically limited by:
- Max tx size (~1MB default) limits message count
- Each message requires at least ~100 bytes of proto encoding
- So max ~10,000 messages per tx → max ~2 billion gas (well under uint64)

**Impact**: Theoretical only. The overflow would wrap to a small number, causing the `gasLimit < totalRequiredGas` check to pass with insufficient gas. But the practical message count makes this unexploitable.

**Fix**: Add a saturating addition or check:
```go
totalRequiredGas += requiredGas
if totalRequiredGas < requiredGas { // overflow
    return ctx, errors.Wrap(sdkerrors.ErrOutOfGas, "gas requirement overflow")
}
```

---

### P2-1: Bootstrap txs require declared gas limit despite free execution

**Behavior**: `BootstrapGasFreeDecorator` (position 3) sets InfiniteGasMeter, but `LGMGasDecorator` (position 5) still checks `feeTx.GetGas()` against minimum requirements. A bootstrap-eligible tx must declare `gas ≥ 100,000` (for MsgSubmitClaim) even though no gas is consumed.

**Impact**: UX confusion. Bootstrap users must set gas high but fee=0. The `LGMGasDecorator` skips the fee check for zero-fee txs (`!feeCoins.IsZero()`), so fee=0 works. But the gas limit must be set.

**Recommendation**: Consider having `LGMGasDecorator` check if the gas meter is infinite and skip validation, or move `LGMGasDecorator` before `BootstrapGasFreeDecorator` so bootstrap txs short-circuit before the gas check. Alternatively, document this behavior clearly for devnet/testnet usage.

---

### P2-2: FeeRouterDecorator is pure overhead

**Location**: `ante.go:443-480`

The decorator only logs `Debug` messages. Actual fee routing is in `vesting_rewards.BeginBlock`. It adds one function call per tx that does no validation and performs no state changes. Could be removed to simplify the chain from 21 to 20 decorators.

---

### P2-3: LastActiveBlock state write in ante handler persists on failed txs

**Location**: `ante_legible.go:139-148`

As documented in the code comment: `LastActiveBlock` is updated even if the message handler fails. This is a conscious design decision (signing = activity), but it means:
- A sequence of invalid transactions counts as "active"
- Cannot distinguish between successful and failed activity
- Could be exploited to keep an account from being flagged as inactive while only sending no-op txs

**Status**: Previously documented, not changed. Design decision stands.

---

## 4. Decorator Ordering Assessment

### Current Order Analysis

The current ordering is **significantly improved** from the 2026-02-10 review:

| Property | Status | Notes |
|----------|--------|-------|
| Gas meter before gas consumption | PASS | `SetUpContext` at #2, before all custom decorators |
| Authentication before authorization | PASS | `SigVerification` (#16) before all Legible post-auth (#19-21) |
| Cheapest checks first | MOSTLY | IBC relay (#1), gas check (#5) before expensive sig verify (#16) |
| Emergency halt early | PASS | Position #4 — no gas/fees consumed for halted txs |
| Bootstrap before gas check | PASS | Position #3 — infinite meter set before `LGMGasDecorator` (#5) |

### Ordering Issues

1. **EmergencyHalt vs Bootstrap ordering**: Currently Bootstrap (#3) before EmergencyHalt (#4). If the chain is halted AND within bootstrap period, a bootstrap-eligible tx would get an infinite gas meter before being rejected by the halt check. This is harmless but slightly wasteful — EmergencyHalt could come first.

2. **LGMGasDecorator before ValidateBasic**: The gas decorator (#5) runs before `ValidateBasic` (#8). A tx with completely invalid message structure still gets gas-checked before being rejected for basic validation. This is fine — the gas check is cheap (map lookup).

---

## 5. Security Assessment

### Fee Manipulation
- **Min gas enforcement**: Per-message gas costs are enforced in `LGMGasDecorator`. Users cannot underpay for complex operations.
- **Min gas price**: Enforced only when fee > 0. Zero-fee txs are allowed (needed for bootstrap).
- **Fee deduction**: Standard SDK `DeductFeeDecorator` at position #12 deducts from account balance before execution.
- **Fee routing**: 7% research / 93% validators, executed in `vesting_rewards.BeginBlock`.
- **RISK**: Validator-level `min-gas-prices` config could block bootstrap txs. Not a code bug but an operational concern.

### Nonce / Replay Protection
- `IncrementSequenceDecorator` at position #17 bumps the account sequence after signature verification.
- Replay of the same tx fails because the sequence number has already been incremented.
- **No issues found**.

### Signature Verification
- `SigVerificationDecorator` at position #16 verifies all signatures cryptographically.
- `SigGasConsumeDecorator` at position #15 consumes gas per signature (prevents gas-free sig brute force).
- `ValidateSigCountDecorator` at position #14 limits total signature count.
- **No issues found with signature handling**.

### Transaction Size
- `ConsumeGasForTxSizeDecorator` at position #11 charges gas proportional to tx byte size.
- CometBFT enforces `max_tx_bytes` at the mempool level (default 1MB).
- `BlockGasLimit = 33,333,333` caps total gas per block.
- Combined: max ~1MB txs, max ~33M gas per block.
- **No issues found**.

### Mempool DoS
- Invalid txs (bad signatures, insufficient fee) are rejected in `CheckTx` via the ante chain before entering the mempool.
- The ante chain runs in `CheckTx` mode — rejections don't cost the node any persistent state changes.
- Gas consumption during `CheckTx` is bounded by the declared gas limit.
- **Low risk**: An attacker could flood with txs that pass early checks but fail at `SigVerification`. Cost: IBC check + SetUpContext + gas check + basic validation + memo check + gas-for-size + fee deduction attempt. The fee deduction (#12) would catch insufficient balance before the expensive sig verify (#16), limiting the attack surface.

---

## 6. Summary

| # | Severity | Issue | Location | Status |
|---|----------|-------|----------|--------|
| P0-1 | **CRITICAL** | `msgWithSigners` bypass — frozen accounts and session key capabilities not enforced for SDK messages (MsgSend, MsgDelegate, etc.) | `ante_legible.go:120,187` | **NEW — NEEDS FIX** |
| P1-1 | Significant | ICA type URL mismatch — `CanIca` capability is dead code | `ante_legible.go:304-306` | **NEW — NEEDS FIX** |
| P1-2 | Low-risk | `totalRequiredGas` overflow (theoretical) | `ante.go:106` | NEW — recommend fix |
| P2-1 | UX | Bootstrap txs still need declared gas limit | `ante.go:76-131` | Design decision |
| P2-2 | Cleanup | FeeRouterDecorator is logging-only overhead | `ante.go:443-480` | Won't fix (harmless) |
| P2-3 | Design | LastActiveBlock updated on failed txs | `ante_legible.go:145-148` | Previously documented |

### Previous Review Issues — Resolution Status

| Issue | Original Severity | Status |
|-------|-------------------|--------|
| LGMGasDecorator before SetUpContext | HIGH | **FIXED** — SetUpContext now at #2 |
| RouteFees never called | HIGH | **FIXED** — wired in vesting_rewards.BeginBlock |
| Session key default-allow for unknown msgs | MEDIUM | **FIXED** — now default-deny |
| DID reads before authentication | MEDIUM | **FIXED** — DID decorator now post-auth (#19) |
| LastActiveBlock on failed txs | MEDIUM | **DOCUMENTED** — design decision |
| Missing gas URL mappings | LOW | **MOSTLY FIXED** — 120+ URLs now mapped |
| No ante handler tests | INFO | **FIXED** — 22 tests in ante_test.go |

### New Decorators Since Last Review

| Decorator | Purpose | Assessment |
|-----------|---------|------------|
| `BootstrapGasFreeDecorator` | Gas waiver for first ~14 days | Clean implementation |
| `EmergencyHaltDecorator` | Blocks non-emergency txs when halted | Correctly placed early |
| `SybilFundingDecorator` | Records funding graph for vote-weight decay | Well-designed passive recording |

---

## 7. Recommended Fix for P0-1

The `msgWithSigners` interface does not work for SDK v0.50 proto-generated types. The fix must extract signers from the **transaction's signature data** rather than per-message `GetSigners()`:

```go
// Replace the current per-message signer extraction:
//   ms, ok := msg.(msgWithSigners)
//
// With tx-level signer extraction:
func getSignerAddresses(tx sdk.Tx) []sdk.AccAddress {
    sigTx, ok := tx.(authsigning.Tx)
    if !ok {
        return nil
    }
    sigs, err := sigTx.GetSignaturesV2()
    if err != nil {
        return nil
    }
    var addrs []sdk.AccAddress
    for _, sig := range sigs {
        addr := sdk.AccAddress(sig.PubKey.Address())
        addrs = append(addrs, addr)
    }
    return addrs
}
```

Then check all signers against frozen status and session key capabilities **once per tx** instead of per-message. This is both more correct and more efficient.
