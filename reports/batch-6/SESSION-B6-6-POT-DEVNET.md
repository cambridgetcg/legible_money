# Session B6-6: Proof of Truth — Live DevNet Verification

**Date**: 2026-02-18
**Status**: PASS (with P0 fix applied)
**Network**: 3-validator Docker devnet (`lgm-devnet-1`)

---

## Executive Summary

Verified the Proof of Truth (PoT) consensus mechanism end-to-end on a live 3-validator devnet running in adversarial verification mode. Successfully demonstrated:

1. **Claim submission** → structural evaluation → provisional fact creation
2. **Challenge submission** → deterministic resolution → challenger bond slashing
3. **Multi-validator consensus** — all 3 nodes at same height/hash throughout
4. **Confidence tracking** — survived_challenges counter incremented correctly

**P0 bug found and fixed**: Knowledge module lacked `authtypes.Burner` permission, causing `CONSENSUS FAILURE` panic when BeginBlocker tried to burn slashed challenger bonds.

---

## Test Environment

| Parameter | Value |
|-----------|-------|
| Image | `legbled:latest` (Docker, alpine-based) |
| Chain ID | `lgm-devnet-1` |
| Validators | 3 (equal stake, 10,000 LGM each) |
| Block time | ~2.5s (2521ms target) |
| Adversarial mode | `enabled` |
| Ports | RPC: 26657/26667/26677, REST: 1317/1318/1319 |

### Genesis Parameters (Knowledge Module)

```
min_verifiers: 1
commit_phase_duration: 4 blocks
reveal_phase_duration: 4 blocks
aggregation_phase_duration: 3 blocks
adversarial_verification_enabled: true
provisional_threshold: 500000 (50%)
min_challenge_stake: 11000000 ulgm
base_challenge_duration: 34272 blocks (~1 day)
confidence_growth_per_epoch: 11000
failed_challenge_slash_bps: 220000 (22%)
```

---

## Block-by-Block Narrative

### Phase 1: Network Boot (Blocks 1-10)

```
Block 1:  Genesis — 3 validators bonded, VRF vote extensions enabled
Block 2:  First consensus round — all validators participating
Block 3:  Autopoiesis: SSI=300000, VCF=4, DVI=4 (healthy)
          Alignment: FI=700000, HI=909725, DCI=200000
```

All 3 validators in sync from block 1. Autopoiesis and alignment modules active.

### Phase 2: Account Setup (Block 50)

```
Block 50: Bank send test1→test2 (1000 ulgm)
          Purpose: Set test1 pubkey on chain (required for subsequent tx signing)
          Result: Code 0, gas used 91964
```

**Discovery**: Cosmos SDK v0.50 requires an account's public key to be set on-chain before signature verification works. Genesis-funded accounts have `null` pubkey until their first outbound transaction. This caused "signature verification failed" errors on knowledge tx attempts before the bank send.

### Phase 3: Formal Claim Submission (Block 75)

```
Block 75: MsgSubmitClaim (formal, mathematics, propositional zone)
          Claim: "The Pythagorean theorem states that a^2 + b^2 = c^2 for right triangles"
          Stake: 10,000 ulgm
          Claim ID: f215e5ec3f0bc68f131ca9b625eb8a4eae3e54b8631316d0f167fbc47fe6d259
          Mode: adversarial
```

### Phase 4: Structural Evaluation (Block 76)

```
Block 76: BeginBlocker → EvaluatePendingClaims
          10-stage structural evaluation pipeline:
            Stage 1 (category_base): formal → 700000
            Stages 2-10: calibration adjustments (no evidence/participant context)
          Final structural_score: 700000 (70%)
          Status: provisional (above 500000 threshold)
          Provisional fact created: d99c4615fd9040da36bc9a4ea39e782adc35d45e58c779f021b9d3b0a57d51d0
          Challenge window: blocks 76 → 102892 (~3 days for formal claims)
```

**Key insight**: The 10-stage evaluation pipeline works correctly. Formal claims get base 700000, which is preserved through calibration since there's no evidence network to adjust against.

### Phase 5: Challenge Submission (Block 109)

```
Block 109: MsgChallengeProvisionalFact
           Challenger: test2 (lgm182j5cq6v95ukf4w97g0h3aedl64g5cvjqd70xu)
           Stake: 11,000,000 ulgm (min_challenge_stake)
           Reason: "Requires specification of Euclidean geometry context"
           Resolution path: deterministic
           Challenge ID: 1692779754cf7504f5e225e75b74d719
```

**Note**: First attempt with stake=5000 failed (code 1) — correctly enforced `min_challenge_stake: 11000000`.

### Phase 6: Challenge Resolution (Block 110) — P0 FIX VALIDATED

```
Block 110: BeginBlocker → ResolvePendingDeterministicChallenges
           Outcome: challenge upheld (claim defended)
           Challenger bond slashed: 2,420,000 ulgm (22% of 11M)
           Burn: 2,420,000 ulgm burned from knowledge module account ✓
           Returned: 8,580,000 ulgm to challenger

           Events emitted:
             challenger_bond_slashed (challenge_id, challenger, slashed, returned)
             burn (burner=knowledge_module, amount=2420000ulgm)
             challenge_upheld (challenge_id, claim_id)
```

**P0 FIX VALIDATED**: Before the fix, this block would have caused:
```
CONSENSUS FAILURE!!! module account knowledge does not have permissions to burn tokens: unauthorized
```
After adding `{authtypes.Burner}` to the knowledge module's `maccPerms`, the burn succeeds.

### Phase 7: Post-Challenge State

```
Claim status: provisional (unchanged — challenge was deterministic, claim defended)
Fact confidence: 810000 (up from 700000 — growth from surviving challenge)
Survived challenges: 1
Active challenges: 0
```

### Phase 8: Empirical Claim (Block 148-149)

```
Block 148: MsgSubmitClaim (empirical, physics, empirical zone)
           Claim: "Water boils at 100°C at standard atmospheric pressure"
           Stake: 10,000 ulgm

Block 149: Structural evaluation
           Stage 1 (category_base): empirical → 500000
           Calibration stages reduce to: 300000
           Status: evaluated (below 500000 provisional threshold)
           No provisional fact created
```

**Key insight**: Empirical claims without supporting evidence score lower. The 500000 base is reduced to 300000 by calibration stages that account for lack of evidence network and participant history.

### Phase 9: Sustained Consensus (Blocks 1-161+)

```
All validators at height 161:
  validator-1: hash=CB4761128E1ED18E
  validator-2: hash=CB4761128E1ED18E
  validator-3: hash=CB4761128E1ED18E
```

No consensus failures, no fork, all nodes agree on every block.

---

## P0 Bug: Knowledge Module Burner Permission

### Root Cause

In `app/app.go`, the module account permissions map (`maccPerms`) had:
```go
knowledgetypes.ModuleName: nil, // ← NO permissions
```

The knowledge module's `challenge.go` calls `BurnCoins()` in two places:
- `slashChallengerBond()` (line 426) — deterministic challenge resolution
- `slashChallengerBond()` (line 539) — arbiter challenge resolution

Without `Burner` permission, `BurnCoins` panics with `CONSENSUS FAILURE`.

### Fix Applied

```go
knowledgetypes.ModuleName: {authtypes.Burner}, // ← Burner for challenge bond slashing
```

The knowledge module does NOT need `Minter` — it only receives coins via `SendCoinsFromAccountToModule` and slashes via `BurnCoins`.

### Additional `BurnCoins` Call Sites (All Now Protected)

| File | Line | Context |
|------|------|---------|
| `challenge.go:426` | `slashChallengerBond` | Deterministic challenge |
| `challenge.go:539` | `slashChallengerBond` | Arbiter challenge |
| `rounds.go:608` | Slash wrong verifiers | Legacy PoT path |
| `rounds.go:769` | Slash equivocators | Legacy PoT path |
| `rounds.go:1303` | Slash failed challenge | Legacy PoT path |
| `rounds.go:1366` | Slash challenge bond | Legacy PoT path |

---

## Vote Extensions

In adversarial mode, vote extensions only activate for **arbiter-path challenges** (Path B). The challenges tested in this session used the **deterministic path** (Path A), so no vote extensions were triggered.

From `app/vote_extensions.go:76`:
```go
if adversarialEnabled && round.Type != knowledgetypes.RoundTypeChallenge {
    continue // Skip non-challenge rounds in adversarial mode
}
```

To test vote extensions, an arbiter-path challenge would need to be triggered. This requires specific conditions (e.g., claim in a domain with arbiter resolution configured, or a dispute escalation).

---

## Category Base Confidence Scores (Verified)

| Category | Base Score | Test Result |
|----------|-----------|-------------|
| formal | 700000 | ✓ Pythagorean theorem → 700000 |
| empirical | 500000 | ✓ Water boiling → 300000 (reduced by calibration) |
| protocol | 800000 | Not tested |
| analytic | 700000 | Not tested |

---

## Discoveries and Observations

### 1. Account Pubkey Bootstrap Required

SDK v0.50 genesis-funded accounts have `null` pubkey on-chain. Any transaction attempt before pubkey is set results in:
```
signature verification failed: (unable to verify single signer signature): unauthorized
```

**Workaround**: Send a simple bank MsgSend first to bootstrap the pubkey.

**Recommendation**: The devnet init script should include a bootstrap transaction phase, or the faucet should pre-fund accounts via `gentx`-style transactions.

### 2. Logic Zone Validation

Formal claims MUST specify `--logic-zone propositional` (or `presburger`/`peano`). The default `empirical` zone is rejected:
```
category "formal" cannot use zone "empirical"
```

### 3. Minimum Gas and Fee Requirements

Knowledge transactions require ~236k gas. The minimum fee at 1 ulgm/gas means fees must be ≥236,000 ulgm. Using `--fees 500000ulgm` with `--gas 400000` works reliably.

### 4. Challenge Window Duration

The `base_challenge_duration` of 34272 blocks (~1 day) makes it impractical to observe `ProcessChallengeWindowExpiry` in a devnet test session. For testing, consider reducing to ~20 blocks.

### 5. Confidence Growth

After surviving a challenge, the fact's confidence jumped from 700000 to 810000 (a 110000 increase, consistent with the `confidence_growth_per_epoch` of 11000 × the challenge survival bonus).

---

## Summary

| Test | Result |
|------|--------|
| 3-validator devnet boot | ✅ PASS |
| Multi-validator consensus | ✅ PASS (161+ blocks, identical hashes) |
| Formal claim → provisional fact | ✅ PASS (score 700000) |
| Empirical claim → evaluated (no fact) | ✅ PASS (score 300000) |
| Challenge submission | ✅ PASS (deterministic path) |
| Challenge resolution (burn) | ✅ PASS (after P0 fix) |
| Challenger bond slashing | ✅ PASS (22% = 2.42M of 11M) |
| Confidence tracking | ✅ PASS (survived_challenges=1, conf=810000) |
| Vote extensions (arbiter path) | ⚠️ NOT TESTED (deterministic path only) |
| Challenge window expiry → verified | ⚠️ NOT TESTED (34272 block window) |

### Blockers for Full PoT Cycle

1. **Vote extensions** require arbiter-path challenge — needs domain-specific configuration
2. **Challenge window expiry** takes ~1 day — impractical for interactive testing
3. **Legacy commit/reveal path** disabled when `adversarial_verification_enabled=true`

### Code Changes

| File | Change |
|------|--------|
| `app/app.go:315` | `knowledgetypes.ModuleName: {authtypes.Burner}` (was `nil`) |
| `proto/legible/knowledge/v1/claim.proto` | Added `MsgChallengeProvisionalFact`, `MsgUpdateExtendedParams` proto messages |
| `proto/legible/knowledge/v1/tx.proto` | Added `ChallengeProvisionalFact`, `UpdateExtendedParams` RPCs |
| `x/knowledge/types/claim.pb.go` | Regenerated with new message types |
| `x/knowledge/types/tx_grpc.pb.go` | Regenerated with new RPCs |
| `x/knowledge/types/types.go` | Removed hand-written struct defs (replaced by proto-gen), kept method impls |
| `x/knowledge/keeper/msg_server.go` | Updated `UpdateExtendedParams` to use `ExtendedParamsJson []byte` |
