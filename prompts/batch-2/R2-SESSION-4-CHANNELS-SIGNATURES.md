# R2 Session 4: Payment Channels — Signature Integration + Chain-ID Domain Separation

## Context
Round 1 Session 4 added Ed25519 signature verification to all 4 channel mutation handlers. The aha moment: **signatures were entirely decorative before**. Session 4 noted a remaining gap: no chain-ID domain separation in the signing payload.

## Scope
- `x/channels/types/signature.go`
- `x/channels/keeper/msg_server.go`
- `x/channels/keeper/keeper_test.go`

## Tasks

### 1. Add Chain-ID Domain Separation
The signing payload currently uses `SHA-256("legible.channels.v1:{operation}" | "|" | channelId | ...)`. This means a signature created on testnet is valid on mainnet (same channelId could exist on both chains). Fix:
- Add `chainId` to the canonical signing payload
- Source the chain-ID from `ctx.ChainID()` in each handler
- Update signature verification calls to include chain-ID
- Add test: signature from chain A rejected on chain B

### 2. Replay Protection Testing
Verify that existing signatures can't be replayed:
- `TestReplay_SameChannelDifferentNonce` — Sig for nonce 5 rejected when nonce is now 6
- `TestReplay_SameNonceDifferentChannel` — Sig for channel A rejected on channel B
- `TestReplay_SameOperationDifferentChain` — Sig from testnet rejected on mainnet (after chain-ID fix)
- `TestReplay_UpdateStateDualSig` — Both payer and provider sigs must match current state

### 3. Cooperative Close Flow Testing
Session 4 implemented the closing → closed state machine. Write end-to-end tests:
- `TestCooperativeClose_HappyPath` — Open → settle (closing) → wait deadline → settle (closed) → funds distributed
- `TestCooperativeClose_DisputeDuringClosing` — Open → settle (closing) → dispute → closed with disputed state
- `TestCooperativeClose_CounterpartyCanDispute` — Provider settles → Payer disputes during closing period
- `TestCooperativeClose_FundsDistribution` — Verify bank balances after close match the last agreed state

### 4. BeginBlock Auto-Resolution
Session 4 noted that closing channels require manual SettleChannel after deadline. Implement auto-resolution:
- In `BeginBlock`, iterate channels with status "closing" or "disputed" where `blockHeight > DisputeDeadline`
- Auto-settle them (distribute funds, set status to "closed")
- Add test: `TestBeginBlock_AutoResolvesExpiredClosing`
- Add test: `TestBeginBlock_AutoResolvesExpiredDispute`

## Report
Write to `reports/R2-SESSION-4-CHANNELS-SIGNATURES.md`
