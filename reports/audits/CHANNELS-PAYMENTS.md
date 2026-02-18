# x/channels Payment Channels — Security Audit Report

**Session**: CHANNELS-PAYMENTS
**Date**: 2026-02-18
**Scope**: State machine, dispute resolution, timeout security, fund safety, off-chain state

---

## 1. Complete State Machine Diagram

### States

| Status     | String Constant | Actually Used? |
|------------|-----------------|----------------|
| `open`     | `ChannelStatusOpen`    | YES — initial state |
| `closing`  | `ChannelStatusClosing` | **NO — dead code, never set by any handler** |
| `disputed` | `ChannelStatusDisputed`| YES — set by DisputeChannel |
| `settled`  | `ChannelStatusSettled` | **NO — dead code, never set by any handler** |
| `closed`   | `ChannelStatusClosed`  | YES — terminal state |

### Transition Diagram

```
                     ┌──────────────────────────────────────────────────┐
                     │                                                  │
                     │   OpenChannel(deposit, timeout)                  │
                     │          │                                       │
                     │          ▼                                       │
                     │    ┌──────────┐                                  │
                     │    │   OPEN   │◄─────────────────────┐           │
                     │    └──────────┘                      │           │
                     │     │  │  │  │                       │           │
                     │     │  │  │  │ DisputeChannel        │           │
                     │     │  │  │  │ (nonce > current)     │           │
                     │     │  │  │  ▼                       │           │
                     │     │  │  │ ┌──────────┐  BeginBlock │           │
                     │     │  │  │ │ DISPUTED │──(timeout)──┘           │
                     │     │  │  │ └──────────┘                         │
                     │     │  │  │      │                               │
                     │     │  │  │      │ SettleChannel                 │
                     │     │  │  │      │ (after deadline)              │
                     │     │  │  │      ▼                               │
                     │     │  │  │ ┌──────────┐                         │
                     │     │  │  └►│  CLOSED  │◄────────────────────┐   │
                     │     │  │    └──────────┘                     │   │
                     │     │  │         ▲  ▲                        │   │
                     │     │  │         │  │                        │   │
                     │     │  │  CloseChannel (cooperative)         │   │
                     │     │  │  SettleChannel (from open)          │   │
                     │     │  │  ClaimExpired (after grace)         │   │
                     │     │  │                                     │   │
                     │     │  │  UpdateChannel / UpdateState        │   │
                     │     │  └──(nonce++, spent updated)──►OPEN    │   │
                     │     │                                        │   │
                     │     │  DepositChannel                        │   │
                     │     └──(payer only, adds funds)──►OPEN       │   │
                     │                                              │   │
                     │          ┌────────────────┐                  │   │
                     │          │ "closing"      │  NEVER USED      │   │
                     │          │ "settled"      │  DEAD CODE       │   │
                     │          └────────────────┘                  │   │
                     └──────────────────────────────────────────────────┘
```

### Message → Transition Map

| Message | From Status | To Status | Guard |
|---------|------------|-----------|-------|
| `OpenChannel` | (new) | `open` | deposit >= MinDeposit, timeout <= MaxDuration, channels < Max |
| `UpdateChannel` | `open` | `open` | sender is party, sequence > nonce, newBalance <= deposited |
| `UpdateState` | `open` | `open` | sender is party, nonce > current, spent <= deposited |
| `DepositChannel` | `open` | `open` | depositor is payer |
| `DisputeChannel` | `open`/`closing`/`disputed` | `disputed` | disputer is party, sequence > existing dispute nonce |
| `CloseChannel` | `open` | `closed` | closer is party, finalBalance <= deposited |
| `SettleChannel` | `open`/`disputed` | `closed` | authority is party, if disputed: block > deadline |
| `ClaimExpired` | `open` | `closed` | claimer is payer, block > expiry + grace |
| `BeginBlock` | `disputed` | `open` | dispute deadline passed (auto-resolve) |
| `SpendFromChannel` | `open` | `open` | amount <= available (cross-module) |

---

## 2. Dispute Resolution Flow

### Initiation
1. Either party calls `DisputeChannel` with their claimed nonce + balance
2. Channel moves to `disputed` status
3. `DisputeDeadline = currentBlock + DisputePeriodBlocks` (default: 1000 blocks ≈ 42 min)
4. A `ChannelDispute` record is created

### Counter-Dispute
- The counterparty can submit a dispute with a **higher nonce** to supersede
- Lower or equal nonce → `ErrDisputeAlreadyActive`
- The latest dispute with highest nonce becomes the active one

### Resolution: Two Paths

**Path A — Timeout (BeginBlock auto-resolve)**
- `BeginBlock` checks for expired disputes each block
- When `currentBlock > dispute.DeadlineBlock`:
  - Channel `Spent` updated to `dispute.DisputedSpent`
  - Channel `Nonce` updated to `dispute.DisputedNonce`
  - Channel status reverts to **`open`** (channel continues)
  - Dispute marked `Resolved=true, Resolution="timeout"`
- **No funds are transferred** — channel just reopens with updated state

**Path B — Manual Settlement (SettleChannel)**
- Either party calls `SettleChannel` after the dispute deadline
- If disputed and deadline passed:
  - Dispute's `DisputedSpent` becomes `ch.Spent`
  - Funds are distributed: payer gets `deposited - spent`, provider gets `spent`
  - Channel status → `closed`
  - Dispute marked `Resolved=true, Resolution="disputed_state_accepted"` (**FIXED** — was hardcoded "provider_wins")

### What Happens to Funds During Dispute
- Funds remain locked in the channels module account
- No transfers occur until SettleChannel or ClaimExpired
- Channel cannot be updated or closed while disputed (only higher-nonce counter-dispute)

### Escalation
- **No integration with x/disputes module** — disputes are entirely self-contained
- No arbiter involvement, no multi-tier escalation
- Only the nonce-based counter-dispute mechanism exists

---

## 3. Timeout Table

| Timeout | Default Value | Blocks | Real Time (2.521s/block) | Consequence |
|---------|--------------|--------|-------------------------|-------------|
| `MaxChannelDuration` | 1,036,800 | ~30 days | Channel creation capped | `ErrChannelDurationExceeded` |
| `DisputePeriodBlocks` | 1,000 | ~42 min | Counter-dispute window | Auto-resolve in BeginBlock |
| `ExpiryGracePeriod` | 500 | ~21 min | Extra time after expiry | Prevents immediate ClaimExpired |
| Channel expiry | `OpenedAtBlock + TimeoutHeight` | Varies | Up to 30 days | Payer can ClaimExpired after expiry+grace |
| `DefaultSettlementFreq` | 100 | ~4.2 min | Stored but **never enforced** | No auto-settlement |

### Delay Analysis
- **Can a party delay indefinitely?** YES — if neither party calls SettleChannel or ClaimExpired after dispute, the channel stays open forever. BeginBlock only reverts disputed→open, it doesn't force closure.
- **Expired channel with undistributed funds**: Sits in `open` status. Only payer can call `ClaimExpired`. Provider has no mechanism to force closure of an expired channel.
- **No automatic channel expiration**: `BeginBlock` only resolves disputes, not expired channels.

---

## 4. Fund Flow Diagram

```
  ┌──────────┐  OpenChannel     ┌──────────────────┐
  │  Payer   │ ──(deposit)────► │ channels module   │
  │ account  │                  │    account        │
  └──────────┘                  └──────────────────┘
       │                               │
       │  DepositChannel               │
       │  (adds to deposit)            │
       ├──────────────────────────────►│
       │                               │
       │  [Off-chain state updates]    │
       │  (no on-chain fund movement)  │
       │                               │
       │                          CloseChannel / SettleChannel / ClaimExpired
       │                               │
       │  ◄──(deposited - spent)───────┤  payerRefund
       │                               │
  ┌──────────┐                         │
  │ Provider │ ◄──(spent)──────────────┤  providerPayout
  │ account  │                         │
  └──────────┘                         │
                                       │
  ┌──────────┐  SpendFromChannel       │
  │ Other    │ ◄──(amount)─────────────┤  cross-module
  │ module   │   (billing, tree, etc.) │
  └──────────┘                         │
```

### Fund Accounting Invariant
```
deposited = spent + available     (must always hold)
module_account_balance >= sum(available) for all open channels
```

---

## 5. Off-Chain State Format and Verification

### ChannelStateUpdate (proto: state.pb.go:182)

| Field | Type | Purpose |
|-------|------|---------|
| `channel_id` | string | Identifies channel |
| `nonce` | uint64 | Monotonically increasing sequence |
| `spent` | string | Cumulative amount paid to provider (bigint) |
| `state_hash` | string | Hash of off-chain application state |
| `payer_signature` | string | Payer's signature over the update |
| `provider_signature` | string | Provider's signature over the update |

### Signature Verification: **COMPLETELY ABSENT**

**Neither `UpdateState`, `UpdateChannel`, `CloseChannel`, nor `DisputeChannel` verify any signature fields.** All four handlers accept the raw values without cryptographic proof.

Proto fields that exist but are ignored:
- `MsgUpdateChannel.Signature` (bytes, field 6) — **NOT CHECKED** (msg_server.go:114-162)
- `MsgCloseChannel.CounterpartySignature` (bytes, field 6) — **NOT CHECKED** (msg_server.go:302-371)
- `MsgDisputeChannel.ProofSignature` (bytes, field 6) — **NOT CHECKED** (msg_server.go:164-217)
- `ChannelStateUpdate.PayerSignature` (string, field 5) — **NOT CHECKED** (msg_server.go:425-473)
- `ChannelStateUpdate.ProviderSignature` (string, field 6) — **NOT CHECKED** (msg_server.go:425-473)

### Replay Protection
- **Nonce monotonicity**: Strictly enforced via `newNonce > ch.Nonce`
- **Same-nonce replay**: Rejected (`<=` check)
- **Rollback**: Rejected (lower nonce rejected)

---

## 6. Security Assessment

### 6.1 P0 Findings (Critical — Must Fix Before Production)

#### P0-1: ALL signature fields ignored — off-chain state is unverifiable

**Files**: `x/channels/keeper/msg_server.go:114-162` (UpdateChannel), `:302-371` (CloseChannel), `:164-217` (DisputeChannel), `:425-473` (UpdateState)

**Impact**: The entire payment channel security model relies on signatures to prove both parties agreed to a state. Without verification:
- **Payer can steal**: Call `CloseChannel` with `FinalBalanceReceiver: "0"`, get full refund despite services rendered
- **Provider can steal**: Call `DisputeChannel` with `ClaimedBalanceReceiver: <full deposit>`, wait for timeout, call `SettleChannel`
- **Either party can fabricate state**: Submit `UpdateChannel`/`UpdateState` with arbitrary spent amounts

**Severity**: CRITICAL — Payment channels without signature verification provide zero security guarantees. The tx signer authentication (Cosmos ante handler) only proves who submitted the message, not that the counterparty agreed to the state.

**Fix**: Before accepting any state transition:
1. `CloseChannel` — require BOTH payer and provider signatures over `(channelId, sequence, finalBalanceReceiver)`
2. `UpdateChannel`/`UpdateState` — require both signatures (or at minimum, the counterparty's signature)
3. `DisputeChannel` — require the disputer to provide a signed state from the counterparty proving the claimed nonce/balance

#### P0-2: DisputeChannel does not validate ClaimedBalanceReceiver <= Deposited — **FIXED**

**File**: `x/channels/keeper/msg_server.go:182-190`

Was storing `ClaimedBalanceReceiver` directly as `DisputedSpent` without bounds check. Now validates `0 <= ClaimedBalanceReceiver <= Deposited` before creating the dispute.

#### P0-3: Negative amounts accepted via big.Int string parsing — **FIXED**

**Files**: `msg_server.go` — UpdateChannel, UpdateState, CloseChannel, DisputeChannel

`big.Int.SetString("-1000000", 10)` returns `ok=true` with negative value, which would break the `deposited = spent + available` invariant. All four handlers now reject `spent.Sign() < 0`.

#### P0-4: SettleChannel resolution always says "provider_wins" regardless of outcome — **FIXED**

**File**: `x/channels/keeper/msg_server.go:258`

Was hardcoded `"provider_wins"` — now changed to `"disputed_state_accepted"` which accurately reflects that the disputer's claimed state was used for settlement. The disputer can be either party, so the old label was misleading.

### 6.2 P1 Findings (Significant — Fix Before Launch)

#### P1-1: Provider cannot initiate settlement of expired channel

**File**: `x/channels/keeper/msg_server.go:490`

```go
if msg.Claimer != ch.Payer {
    return nil, types.ErrNotChannelParty.Wrap("only payer can claim expired channel")
}
```

Only the payer can call `ClaimExpired`. If the payer disappears or is uncooperative, the provider's earned funds (`ch.Spent`) remain locked in the module account indefinitely.

**Fix**: Allow provider to also call `ClaimExpired`. Both parties have funds at stake in an expired channel.

#### P1-2: No automatic channel expiration in BeginBlocker

**File**: `x/channels/module.go:128-155`

`BeginBlock` only resolves expired disputes. Expired channels in `open` status are never auto-closed. This means:
- Funds can sit locked indefinitely if neither party acts
- No event emitted when a channel expires

**Fix**: Add `ExpireFormations` logic to BeginBlock:
```go
// Also expire channels past their ExpiresAtBlock + GracePeriod
channels := am.keeper.GetChannelsByStatus(ctx, types.ChannelStatusOpen)
for _, ch := range channels {
    if ch.ExpiresAtBlock > 0 && currentBlock > ch.ExpiresAtBlock + params.ExpiryGracePeriod {
        // Auto-settle: distribute funds
    }
}
```
Note: This is O(n) over all open channels. Consider a deadline index for efficiency.

#### P1-3: FraudPenaltyBps parameter defined but never applied

**File**: `x/channels/types/genesis.go:13`

`FraudPenaltyBps: 500` (5%) is stored in params but no handler uses it. If a counter-dispute proves the original disputer submitted a lower nonce (fraud), there is no penalty.

**Fix**: When a counter-dispute supersedes an existing dispute with a higher nonce, slash the original disputer by `FraudPenaltyBps` of the channel deposit.

#### P1-4: "closing" and "settled" statuses are dead code

**File**: `x/channels/types/types.go:14-18`

`ChannelStatusClosing` and `ChannelStatusSettled` are defined but no handler ever transitions a channel to these states. `DisputeChannel` even checks for `closing` status (msg_server.go:173), which is unreachable.

**Fix**: Either implement the `closing` state (e.g., as a unilateral close with challenge period) or remove the dead constants.

#### P1-5: SpendFromChannel nonce increment breaks off-chain state sync

**File**: `x/channels/keeper/keeper.go:149`

```go
ch.Nonce++
```

`SpendFromChannel` (called by billing/tree modules) increments the on-chain nonce. Off-chain state updates that were valid at nonce N are now rejected because on-chain nonce jumped. This desynchronizes on-chain and off-chain state.

**Fix**: Either:
- Use a separate counter for cross-module spends (don't touch the nonce used for off-chain updates)
- Or document that off-chain parties must query the on-chain nonce before submitting updates

#### P1-6: SettleChannel can be called on open (non-disputed) channels

**File**: `x/channels/keeper/msg_server.go:220-300`

`SettleChannel` only rejects `closed` and `settled` status. An `open` channel can be settled by either party at any time, using the current on-chain `ch.Spent`. Combined with P0-1 (no signature verification), a party could:
1. Submit `UpdateChannel` with favorable balance
2. Immediately call `SettleChannel` to lock in that state

**Fix**: `SettleChannel` on an `open` channel should either require both signatures or go through a challenge period (the `closing` state that exists but is unused).

### 6.3 P2 Findings (Minor — Track for Hardening)

#### P2-1: SettlementFrequency stored but never enforced

`DefaultSettlementFreq: 100` is set on channel creation (msg_server.go:97) and stored in `ch.SettlementFrequency`, but no handler or BeginBlock logic uses it. `LastSettlementBlock` is similarly unused.

#### P2-2: channelCounterKey (0x20) collides with key prefix namespace

`channelCounterKey = []byte{0x20}` (state.go:239) is outside the documented prefix range (0x01-0x11) but shares the same store. Not a collision today but fragile.

#### P2-3: ExportGenesis channel/dispute ordering is non-deterministic

`IterateChannels` and `IterateDisputes` iterate in KV store order (lexicographic). This is deterministic for the same state but the export ordering may vary across store implementations. For genesis export reproducibility, consider explicit sorting.

#### P2-4: DeleteChannel/DeleteDispute silently ignore errors

```go
_ = store.Delete(types.ChannelKey(ch.ChannelId))
```

Store deletion errors are discarded. In practice, IAVL store Delete never fails, but this masks potential issues.

---

## 7. Cross-Module Integration

### Bank Module
- `OpenChannel` → `SendCoinsFromAccountToModule` (payer → channels module)
- `DepositChannel` → `SendCoinsFromAccountToModule` (payer → channels module)
- `CloseChannel/SettleChannel/ClaimExpired` → `SendCoinsFromModuleToAccount` (channels module → payer/provider)
- `SpendFromChannel` → `SendCoinsFromModuleToModule` (channels → billing/tree/etc.)
- All transfers are **atomic within the message handler** (Cosmos SDK transaction semantics)
- **nil guard on bankKeeper**: All transfers check `if m.Keeper.bankKeeper != nil` — if nil, the transfer is silently skipped while accounting is still updated. This would corrupt state in production.

### Billing Module (via SpendFromChannel)
- `keeper.go:120-161` — `SpendFromChannel(ctx, channelId, amount, recipientModule)`
- Updates channel accounting (spent, available, nonce) and transfers tokens
- Called cross-module via the keeper interface
- **Race condition risk**: If two modules call `SpendFromChannel` in the same block, the available balance could underflow if not properly sequenced. However, Cosmos SDK message handlers are sequential within a block, so this is safe.

### Channel Info Query
- `keeper.go:108-115` — `GetChannelInfo(ctx, channelId)` returns payer, provider, available, status
- Cross-module read-only query interface

---

## 8. Summary

### Architecture Assessment

The channels module implements a **unidirectional payment channel** (payer→provider) with on-chain state anchoring and dispute resolution. The core state machine is sound:
- Monotonic nonce progression prevents replay
- Module account escrow prevents double-spending at the bank level
- Dispute deadline provides a counter-dispute window

### Critical Gap: Signature Verification

The single most critical issue is that **all signature fields are defined in proto but completely ignored in handlers**. This renders the payment channel insecure — any channel party can unilaterally claim any balance distribution. Payment channels without signature verification are functionally equivalent to giving both parties a blank check.

### Fix Status

| Finding | Status | Description |
|---------|--------|-------------|
| **P0-1** | OPEN | Implement signature verification for all state transitions |
| **P0-2** | **FIXED** | Validate DisputedSpent <= Deposited |
| **P0-3** | **FIXED** | Reject negative amount strings (4 handlers) |
| **P0-4** | **FIXED** | Fix dispute resolution label (was hardcoded "provider_wins") |
| P1-1 | OPEN | Allow provider to call ClaimExpired |
| P1-2 | OPEN | Add channel auto-expiration in BeginBlocker |
| P1-3 | OPEN | Implement FraudPenaltyBps slashing |
| P1-5 | OPEN | Fix SpendFromChannel nonce collision |
| P1-6 | OPEN | Add challenge period for SettleChannel on open channels |

### Priority Remaining

1. **P0-1**: Signature verification — requires design decision on signature scheme and key management

### Test Coverage

Existing tests cover the happy paths and basic adversarial probes (12 OpenClaw tests + 40+ keeper tests). The test suite correctly validates:
- Third-party rejection (OC-CH-1, OC-CH-8)
- Nonce rollback/replay (OC-CH-2, OC-CH-5)
- Premature settlement (OC-CH-3)
- Overspend (OC-CH-4, OC-CH-7)
- Deposit to non-open channels (OC-CH-9)
- ClaimExpired timing (OC-CH-10)
- Zero/negative deposits (OC-CH-11)

**Missing test coverage**:
- Signature verification (not tested because not implemented)
- Negative balance string injection
- DisputedSpent > Deposited overflow
- Provider ClaimExpired (rejected by design — P1-1)
- SpendFromChannel + off-chain nonce desync
- Concurrent multi-channel fund accounting

---

*Report generated by Claude Code — CHANNELS-PAYMENTS audit session*
