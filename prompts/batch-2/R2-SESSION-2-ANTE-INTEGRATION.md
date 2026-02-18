# R2 Session 2: Ante Handler Integration Testing

## Context
Round 1 Session 6 discovered that ALL ante handler checks (frozen accounts, session key capabilities, DID validation) were silently bypassed for standard Cosmos SDK messages due to a type assertion failure. The fix switched to tx-level signer extraction. But we need to **verify the fix actually works end-to-end** with real transaction types.

## Scope
- `app/ante.go`
- `app/ante_legible.go`
- `app/ante_test.go`
- `app/sybil_funding_decorator.go`

## Tasks

### 1. Write Integration Tests for the Ante Fix
Create `app/ante_integration_test.go` with tests that construct REAL SDK message types and verify they go through the full ante handler chain:

**Frozen Account Tests:**
- `TestFrozenAccount_MsgSend_Rejected` — Create a frozen account, submit `MsgSend` → should be rejected
- `TestFrozenAccount_MsgDelegate_Rejected` — Frozen account submits `MsgDelegate` → rejected
- `TestFrozenAccount_CustomMsg_Rejected` — Frozen account submits `MsgCreateHome` → rejected
- `TestNonFrozenAccount_MsgSend_Passes` — Normal account submits `MsgSend` → passes

**Session Key Capability Tests:**
- `TestSessionKey_TransferOnly_MsgSend_Passes` — Session key with CanTransfer sends MsgSend → passes
- `TestSessionKey_TransferOnly_MsgDelegate_Rejected` — Session key with CanTransfer only sends MsgDelegate → rejected
- `TestSessionKey_VerifyOnly_MsgSend_Rejected` — Verify-only session key sends MsgSend → rejected

**Emergency Halt Tests:**
- `TestEmergencyHalt_RegularTx_Rejected` — During halt, regular tx rejected
- `TestEmergencyHalt_EmergencyMsg_Passes` — During halt, emergency msg passes

### 2. Gas Estimation Edge Cases
Session 6 noted a uint64 overflow fix in gas calculation. Add tests:
- `TestGasAccumulation_ManyMessages` — Transaction with 100+ messages → gas doesn't overflow
- `TestGasAccumulation_MaxGas` — Transaction with messages at MaxGas boundary → handled correctly

### 3. Bootstrap Sybil Decorator
Review `sybil_funding_decorator.go` — the bootstrap gas waiver. Add tests:
- `TestBootstrap_FreeGas_OnlyDuringBootstrap` — After bootstrap ends, no free gas
- `TestBootstrap_FreeGas_OnlyForWhitelistedTypes` — During bootstrap, only allowed msg types get free gas

### 4. Full Ante Chain Test
Create one comprehensive test that constructs a real `DeliverTx` with a multi-message transaction (MsgSend + MsgDelegate) and traces it through the ENTIRE ante handler chain, verifying each decorator runs.

## Report
Write to `reports/R2-SESSION-2-ANTE-INTEGRATION.md`
