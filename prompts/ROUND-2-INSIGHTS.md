# Round 2: Insights & Aha Moments from Sessions 1-7

## Build Status: ✅ CLEAN
- `go build ./...` — PASS
- `go test ./x/... -count=1` — ALL 37 packages PASS, 0 FAIL

---

## Key Discoveries (Aha Moments)

### 1. BVM Gas Was Completely Disconnected (S1+S2)
The BVM had its own gas meter that was entirely independent from the Cosmos SDK gas meter. A user could run arbitrary computation for a flat 21,000 SDK gas. Scheduled executions consumed ZERO gas. This is now bridged.

### 2. Proposer Could Tamper Reveal Data (S6 — CRITICAL)
The most dangerous finding: `StoreRevealInRound` never verified that reveal data (verdict + confidence + salt) actually hashed to the stored commitment. A malicious block proposer injecting the VEX transaction could silently flip verdicts from "accept" to "reject". This undermines the entire Proof of Truth mechanism. Fixed with canonical commitment hash verification.

### 3. All Ante Handler Checks Bypassed for SDK Messages (S6)
`LegibleAccountDecorator`, `LegibleCapabilityDecorator`, and `LegibleDIDDecorator` used per-message type assertions that SDK v0.50 proto types don't implement. So `MsgSend`, `MsgDelegate`, etc. silently bypassed frozen account checks, session key capabilities, and DID validation. Every standard Cosmos transaction was unguarded.

### 4. Guardian Tier Was Unreachable (S7)
The `ContestedVerificationsCorrect` counter was never incremented because the hooks interface lacked a `contested` parameter. Since Guardian tier requires 33 contested correct verifications, no validator could ever reach Guardian. The entire 4-tier hierarchy was capped at Bonded.

### 5. Payment Channel Signatures Were Decorative (S4)
All 4 channel mutation handlers accepted signature bytes but never verified them. Off-chain state was completely unverifiable on-chain. Anyone who knew a channel ID could submit fake balances.

### 6. ICA Completely Bypassed Rate Limits (S3)
The ICA controller was wired directly to IBC core, not through the rate limit middleware. An attacker could drain the bridge via ICA-initiated transfers with zero rate limiting.

### 7. Innocent Validators Were Being Slashed (S6)
Multi-block commit phases generated a new random salt each block, creating different commitment hashes for the same claim. The second hash was treated as equivocation → 100% slash. Honest validators participating normally would get slashed.

---

## Cross-Session Conflicts Identified

### S5 noted: `RecordVerification` / `AfterVerificationCompleted` signature mismatch
Session 5 reported a pre-existing build error between `x/staking/keeper/hooks.go` and `x/knowledge/keeper/confidence.go`. Session 7 fixed this by adding the `contested bool` parameter to the hooks interface.

### S6 noted: 3 knowledge tests needed updating for commitment hash verification
Session 6 added `VerifyCommitmentHash` to `StoreRevealInRound`, which broke 3 existing tests that used dummy commitment hashes. Fixed in S6.

**These conflicts resolved cleanly — no remaining issues.**

---

## What Still Needs Work

### From Session Reports — Remaining Items

| Priority | Item | Source |
|----------|------|--------|
| P1 | VRF seed uses per-block prevBlockHash (predictable) | S6 |
| P2 | BVM: No state rollback on failed scheduled execution | S2 |
| P2 | BVM: No gas schedule governance proposal type | S2 |
| P2 | Channels: No chain-ID domain separation in signatures | S4 |
| P2 | Channels: BeginBlock doesn't auto-resolve closing channels | S4 |
| P2 | Staking: `ValidateTransition` dead code (future use) | S7 |
| P2 | Staking: `CalculateTierSlash` not wired to `SlashValidator` | S7 |

### Original Audit Items Still Open

| Priority | Item | Source |
|----------|------|--------|
| P2 | SCHED-001: Hardcoded cleanup interval not governance-adjustable | CLEANUP.md |
| P2 | 13 modules still lack tx.proto (no gRPC/REST for 68 handlers) | PROGRESS-REPORT |
| P2 | DEFER-005: Gov QuorumThresholdBps — now implemented but needs integration tests | S5 |

---

## Round 2 Prompts — Focus Areas

| # | Prompt | Focus |
|---|--------|-------|
| R2-1 | PoT Integrity | End-to-end commit-reveal-finalize tamper-proofing |
| R2-2 | Ante Integration | Integration tests for the SDK message bypass fix |
| R2-3 | BVM Gas Bridge | Integration tests for gas bridge + scheduled execution budget |
| R2-4 | Channels Signatures | Chain-ID domain separation + cooperative close flow + BeginBlock auto-resolution |
| R2-5 | IBC Sliding Window | Adversarial stress tests for sliding window + multi-denom + ICA rate limits |
| R2-6 | Tier Lifecycle | Full Apprentice → Guardian progression + demotion + VRF selection weight |
| R2-7 | Gov Quorum + Home | Quorum with real staking keeper + agent home lifecycle integration |
