# PP Review: Ante Handler Chain

> **Review Date**: 2026-02-10
> **Scope**: `app/ante.go`, `app/ante_legible.go`, `app/gas.go`, wiring in `app/app.go:971`
> **Method**: Full PP protocol (30 gates across 5 dimensions)

---

## 1. Territory Map

### File Layout

| File | Lines | Purpose |
|------|-------|---------|
| `app/ante.go` | 383 | Chain composition, LGMGasDecorator, FeeRouterDecorator, msg→gas map, RouteFees() |
| `app/ante_legible.go` | 290 | LegibleAccountDecorator, LegibleCapabilityDecorator, LegibleDIDDecorator |
| `app/gas.go` | 271 | Constants (BlockGasLimit, MinGasPrice, BondDenom), TransactionGasCosts map, fee routing BPS |
| `app/app.go:971` | 1 | `app.SetAnteHandler(NewAnteHandler(app))` |

### Chain Composition (`ante.go:25-52`)

```
┌──────────────────────────────────────────────────────────────────┐
│                    ANTE HANDLER CHAIN ORDER                       │
├──────┬───────────────────────────────┬───────────────────────────┤
│  #   │ Decorator                     │ Phase                     │
├──────┼───────────────────────────────┼───────────────────────────┤
│  1   │ ibcante.RedundantRelay        │ IBC                       │
│  2   │ LGMGasDecorator               │ LGM Pre-Auth              │
│  3   │ FeeRouterDecorator            │ LGM Pre-Auth              │
│  4   │ LegibleDIDDecorator           │ LGM Pre-Auth              │
│  5   │ ante.SetUpContext              │ Standard Cosmos SDK       │
│  6   │ ante.ExtensionOptions          │ Standard Cosmos SDK       │
│  7   │ ante.ValidateBasic             │ Standard Cosmos SDK       │
│  8   │ ante.TxTimeoutHeight           │ Standard Cosmos SDK       │
│  9   │ ante.ValidateMemo              │ Standard Cosmos SDK       │
│ 10   │ ante.ConsumeGasForTxSize       │ Standard Cosmos SDK       │
│ 11   │ ante.DeductFee                 │ Standard Cosmos SDK       │
│ 12   │ ante.SetPubKey                 │ Standard Cosmos SDK       │
│ 13   │ ante.ValidateSigCount          │ Standard Cosmos SDK       │
│ 14   │ ante.SigGasConsume             │ Standard Cosmos SDK       │
│ 15   │ ante.SigVerification           │ Standard Cosmos SDK       │
│ 16   │ ante.IncrementSequence         │ Standard Cosmos SDK       │
│ 17   │ LegibleAccountDecorator        │ LGM Post-Auth             │
│ 18   │ LegibleCapabilityDecorator     │ LGM Post-Auth             │
└──────┴───────────────────────────────┴───────────────────────────┘
```

---

## 2. TRUTH Gates

### Gate 1: Grounding — PASS with issues

All claims below are grounded in the actual source code.

### Gate 2: Calibration — PASS

Confidence is HIGH for ordering issues, MEDIUM for fee routing (depends on runtime behavior).

### Gate 3: Counter-Consideration — findings below

### Gate 4: Completeness — all relevant code read

### Gate 5: Representation — see severity ratings

---

## 3. Findings

### ISSUE-1 (HIGH): LGMGasDecorator runs before SetUpContext — gas meter not initialized

**Location**: `ante.go:26-36` — LGMGasDecorator is at position 2, SetUpContextDecorator at position 5.

**Problem**: `ante.NewSetUpContextDecorator()` initializes the gas meter on the context (`ctx.WithGasMeter()`). The LGMGasDecorator at position 2 runs *before* the gas meter exists. While LGMGasDecorator doesn't consume gas directly (it only reads `feeTx.GetGas()` and validates against limits), any downstream gas consumption in early decorators will panic or silently fail without a gas meter.

**Why it matters**: The standard Cosmos SDK ordering puts `SetUpContext` **first** (after IBC relay check). Moving LGM decorators ahead means:
- `FeeRouterDecorator` (position 3) calls `ctx.Logger()` which should work but gas tracking is absent
- `LegibleDIDDecorator` (position 4) does state reads (`lak.GetAddressForDID`) without gas metering — those reads are free
- If any of these three decorators ever add gas consumption, it will panic

**Recommendation**: Move `ante.NewSetUpContextDecorator()` to position 2 (right after IBC), before the LGM pre-auth decorators. This is the standard Cosmos SDK convention.

```
1. ibcante.RedundantRelay
2. ante.SetUpContext          ← move here
3. LGMGasDecorator
4. FeeRouterDecorator
5. LegibleDIDDecorator
6. ante.ExtensionOptions
...
```

---

### ISSUE-2 (HIGH): FeeRouterDecorator is a no-op — 22% research split never happens in ante

**Location**: `ante.go:294-323`

**Problem**: The `FeeRouterDecorator.AnteHandle()` only *logs* the intended 22/78 split (`ante.go:316-320`). It does not perform any fee routing. The actual routing is delegated to `RouteFees()` (`ante.go:327-357`) which is documented as "Called in BeginBlock to sweep accumulated fees."

However, searching for `RouteFees` in `app/app.go`, there is **no call to `RouteFees`** from `BeginBlocker`:

```go
// app/app.go:1054
func (app *LegibleApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
    return app.ModuleManager.BeginBlock(ctx)
}
```

The `RouteFees` function is defined but **never invoked**. All fees go to `fee_collector` and are distributed entirely to validators by the standard Cosmos distribution module. The 22% research fund allocation documented in comments (`ante.go:17-20`) and the protocol constants (`gas.go:230`: `ResearchTaxBPS = 2200`) are dead code.

**Impact**: Research fund receives 0% of fees instead of the intended 22%. This breaks the economic sustainability loop.

**Recommendation**: Either:
1. Wire `RouteFees()` into BeginBlocker (add a call before `ModuleManager.BeginBlock`)
2. Or implement the split inside `DeductFeeDecorator` by wrapping it with a custom decorator that sends 22% to `research_fund` and 78% to `fee_collector`

---

### ISSUE-3 (MEDIUM): LegibleDIDDecorator reads state without authentication

**Location**: `ante_legible.go:226-289`

**Problem**: The DID decorator runs at position 4, **before** signature verification (position 15). This means an unauthenticated sender can trigger state reads (`lak.GetAddressForDID()`, message signer resolution) by crafting a tx with a memo starting with `did:lgm:`.

While this is not directly exploitable (the tx will still fail signature verification later), it creates a **free state query vector**:
- An attacker can probe which DIDs are registered by sending unsigned transactions with DID memos
- Each such tx performs a KV store read before being rejected at signature verification
- Without gas metering (Issue-1), these reads are completely free

**Recommendation**: Move `LegibleDIDDecorator` to the post-auth section (after position 16), or at minimum after `SetUpContext` and `SigVerification`. DID validation doesn't need to run pre-auth — it's an additional constraint, not a prerequisite.

---

### ISSUE-4 (MEDIUM): LegibleAccountDecorator state writes in ante handler

**Location**: `ante_legible.go:63-66`

```go
// Update last active block
if account.LastActiveBlock < currentHeight {
    account.LastActiveBlock = currentHeight
    lad.lak.SetAccount(ctx, account)
}
```

**Problem**: The ante handler performs a **state write** (`SetAccount`) for every transaction from a registered Legible account. In Cosmos SDK, ante handler state changes are committed only if the entire tx succeeds (`CheckTx` mode) but in `DeliverTx` mode, ante changes persist even if the message handler fails. This means:

1. A transaction that passes ante but fails in `MsgServer` execution will still update `LastActiveBlock` — meaning even failed transactions count as "activity"
2. This is a semantic issue: should a failed transfer still mark the account as "active"?

**Counter-argument**: This is arguably acceptable behavior — the account holder did sign and submit a transaction, which represents activity even if the tx logic failed. But it should be a conscious decision.

**Recommendation**: Document this behavior explicitly. If failed-tx activity tracking is undesirable, move the `LastActiveBlock` update into a `PostHandler` or into the message server handlers themselves.

---

### ISSUE-5 (MEDIUM): Session key capability check has coverage gaps

**Location**: `ante_legible.go:156-180`

The `checkCapability()` function maps message types to four capability buckets:
- `CanTransfer` — bank send, multisend, IBC transfer
- `CanStake` — delegate, undelegate, redelegate, register validator
- `CanSubmitClaims` — submit claim, commitment, reveal, challenge fact
- `CanVote` — gov vote, cast vote, halt/revert/resume votes

**Problem**: Any message type not covered by these four categories **silently passes** — the function returns `nil` for unknown message types (`ante_legible.go:179`). This means a session key with zero capabilities (`CanTransfer=false, CanStake=false, CanSubmitClaims=false, CanVote=false`) can still:

- Deploy contracts (`MsgDeployContract`)
- Create projects (`MsgCreateProject`)
- Open payment channels (`MsgOpenChannel`)
- Submit research (`MsgSubmitResearch`)
- Initiate disputes (`MsgInitiateDispute`)
- Create tokens (`MsgCreateToken`)
- Trigger emergency proposals (`MsgProposeHalt`)
- Deploy services (`MsgDeployService`)
- **And all other ~60 LGM-specific message types**

This is a significant permission escalation vector — session keys intended to be restricted can perform the vast majority of protocol actions.

**Recommendation**: Invert the logic. Default to **deny** for session keys, and only allow explicitly permitted capabilities:
```go
// If signing with session key, default deny for unrecognized msg types
return legibleauthtypes.ErrSessionCapabilityDenied
```

Or add more granular capability categories (e.g., `CanManageProjects`, `CanManageContracts`, `CanManageEmergency`).

---

### ISSUE-6 (LOW): FeeRouterDecorator validates FeeTx but SDK already does this

**Location**: `ante.go:305-307`

```go
feeTx, ok := tx.(sdk.FeeTx)
if !ok {
    return next(ctx, tx, simulate)
}
```

Since `ante.NewDeductFeeDecorator()` at position 11 already requires `sdk.FeeTx` and returns an error if the tx doesn't implement it, this check in FeeRouterDecorator is redundant (the tx will be rejected later anyway). Not harmful, but unnecessary code.

---

### ISSUE-7 (LOW): Gas map missing several message types

**Location**: `ante.go:136-279` vs `gas.go:28-226`

The `msgTypeURLToGas` map in `ante.go` maps protobuf message type URLs to gas costs. However, comparing it against `TransactionGasCosts` in `gas.go`, several transaction types defined in `gas.go` have **no corresponding proto message URL mapping**:

- Agent Home: `create_home`, `update_home`, `update_memory_cid`, `home_start_session`, `home_end_session`, `home_register_key`, `home_revoke_key`, `configure_guardian`, `acknowledge_alert`, `set_spending_limit`
- Domain qualification: `request_qualification`, `endorse_validator`, `renew_qualification`, `withdraw_qualification`
- Recovery: `recover_account`, `set_recovery_config`, `initiate_recovery`, `submit_recovery_shard`, `challenge_recovery`, `execute_recovery`
- Other: `freeze_account`, `unfreeze_account`, `add_contributor`, `apply_to_project`, `review_application`, `set_availability`, `subscribe_service`

These message types get `MinGasLimit` (22,222) instead of their intended gas cost from `lookupMsgGas()` fallback (`ante.go:131`). This means some high-cost operations (like `deploy_contract` at 200,000 or `create_home` at 150,000) can be submitted with only 22,222 gas if the proto URL isn't in the map.

**Counter-argument**: These may correspond to message types that don't have proto definitions yet (TypeScript-only concepts).

**Recommendation**: Audit which `TransactionGasCosts` entries correspond to live proto messages and add their URLs to `msgTypeURLToGas`.

---

### ISSUE-8 (LOW): `BondDenom` defined in two places

**Location**: `app/gas.go` does not define `BondDenom` — it's in `app/app.go:197`. But `app/app.go:297` also sets `sdk.DefaultBondDenom = "ulgm"`. This is fine but worth noting for consistency: the canonical source is `app/app.go:197`.

---

### ISSUE-9 (INFO): No ante handler tests

[Searched: `*ante*test*`, `*test*ante*` — 0 results]

There are no test files for the ante handler chain. The custom decorators (LGMGasDecorator, FeeRouterDecorator, LegibleDIDDecorator, LegibleAccountDecorator, LegibleCapabilityDecorator) have zero test coverage. This is the primary transaction validation path — any bug here affects every transaction on the chain.

---

## 4. BEAUTY Gates

### Gate 3: Fit — PARTIAL PASS

The decorator pattern matches Cosmos SDK conventions well. The three-phase structure (Pre-Auth → Standard → Post-Auth) is a clean mental model.

### Gate 4: Elegance — FAIL (FeeRouterDecorator does nothing)

FeeRouterDecorator occupies a position in the chain but performs no work beyond logging. It should either be removed or wired to actually route fees.

### Gate 5: Coherence — FAIL (ordering inconsistency)

LGM decorators running before `SetUpContext` breaks the Cosmos SDK convention that every decorator has a gas meter available. The pre-auth/post-auth split makes sense conceptually, but the pre-auth decorators should still run after gas meter initialization.

### Gate 7: Appropriateness — PASS

The gas cost table is comprehensive (80+ message types) and the constants are well-organized. The fee routing BPS constants are clearly defined.

---

## 5. JUSTICE Gates

### Gate 1: Placement — PASS

`ante.go`, `ante_legible.go`, and `gas.go` are all correctly placed in `app/`. The Legible-specific decorators are properly separated into `ante_legible.go`.

### Gate 6: Integrity — FAIL (Issue-3)

Running DID state reads before authentication violates the principle that unauthenticated requests should not trigger state access.

---

## 6. CREATIVITY Gates

Not applicable — this is charted territory (Cosmos SDK ante handler pattern).

---

## 7. Summary of Issues by Priority

| # | Severity | Issue | Effort | Status |
|---|----------|-------|--------|--------|
| 1 | **HIGH** | LGMGasDecorator runs before gas meter init (SetUpContext) | Reorder chain (1 line) | **FIXED** |
| 2 | **HIGH** | RouteFees() never called — 22% research fund is dead code | Wire into BeginBlocker | **FIXED** |
| 5 | **MEDIUM** | Session key capabilities silently pass unknown msg types | Invert default to deny | **FIXED** |
| 3 | **MEDIUM** | DID decorator reads state without authentication | Move after sig verification | **FIXED** |
| 4 | **MEDIUM** | LastActiveBlock updated even on failed txs | Document or move to PostHandler | **DOCUMENTED** |
| 7 | **LOW** | ~30 gas costs have no proto URL mapping | Add missing URL→gas entries | **FIXED** (15 tree msgs added; ~23 await tx.proto) |
| 6 | **LOW** | FeeRouterDecorator FeeTx check is redundant | Minor cleanup | Won't fix (harmless) |
| 9 | **INFO** | Zero ante handler test coverage | Add test file | **ADDED** (22 tests in ante_test.go; blocked by proto redeclarations) |
| 8 | **INFO** | BondDenom single source of truth is app.go:197 | Informational | N/A |

---

## 8. Recommended Chain Order (Fixed)

```go
return sdk.ChainAnteDecorators(
    // --- IBC ---
    ibcante.NewRedundantRelayDecorator(app.IBCKeeper),

    // --- Gas Meter Init (must be before any gas consumption) ---
    ante.NewSetUpContextDecorator(),

    // --- LGM Pre-Auth (gas meter available) ---
    NewLGMGasDecorator(),
    NewFeeRouterDecorator(app.BankKeeper),  // or remove if wiring RouteFees in BeginBlock

    // --- Standard Cosmos SDK Decorators ---
    ante.NewExtensionOptionsDecorator(nil),
    ante.NewValidateBasicDecorator(),
    ante.NewTxTimeoutHeightDecorator(),
    ante.NewValidateMemoDecorator(app.AccountKeeper),
    ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
    ante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, nil),
    ante.NewSetPubKeyDecorator(app.AccountKeeper),
    ante.NewValidateSigCountDecorator(app.AccountKeeper),
    ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
    ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
    ante.NewIncrementSequenceDecorator(app.AccountKeeper),

    // --- Legible Money Post-Auth (signer authenticated) ---
    NewLegibleDIDDecorator(app.LegibleAuthKeeper),    // moved post-auth
    NewLegibleAccountDecorator(app.LegibleAuthKeeper),
    NewLegibleCapabilityDecorator(app.LegibleAuthKeeper, app.AccountKeeper),
)
```

---

## 9. Key Insight for Future Work

The ante handler chain is the **single point of transaction validation** for the entire Legible Money chain. Every transaction — knowledge claims, staking, governance, disputes, payments — flows through these 18 decorators. The chain currently has no test coverage, and two of the three custom pre-auth decorators have semantic issues (dead fee routing, unauthenticated state reads). Fixing these is prerequisite to any mainnet launch.

---

## 10. Delta Review (2026-02-18)

> See `reports/ANTE-CHAIN.md` for full audit report.

### Chain now has 21 decorators (was 18)
New additions: BootstrapGasFreeDecorator (#3), EmergencyHaltDecorator (#4), SybilFundingDecorator (#18).

### P0: `msgWithSigners` interface bypass (CRITICAL — UNFIXED)
SDK v0.50.10 removed `GetSigners() []sdk.AccAddress` from proto-generated types. `LegibleAccountDecorator` and `LegibleCapabilityDecorator` use `msg.(msgWithSigners)` which FAILS for all standard SDK messages (MsgSend, MsgDelegate, MsgVote).
- **Frozen accounts can send MsgSend** (freeze is bypassed)
- **Session keys bypass capability checks** for standard SDK messages
- **Fix**: Extract signers from tx signatures, not per-message GetSigners

### P1: ICA type URL mismatch
`isICAMsg()` uses gRPC path format (`/Msg/Register...`) but `sdk.MsgTypeURL` returns proto name format (`.MsgRegister...`). `CanIca` capability is dead code (fail-closed, so secure but broken).
