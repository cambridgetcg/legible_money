# PP Review: Validator Cycling & Micro-Slash Grief Attacks

**Date**: 2026-02-10
**Scope**: Both TS prototype and Cosmos SDK stacks
**PP Dimensions**: All 5 (TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY)
**Severity Scale**: CRITICAL / HIGH / MEDIUM / LOW / INFO

---

## Executive Summary

Both stacks share a fundamental vulnerability: **3 slashes = permanent deactivation with no recovery path**. An attacker who can induce 3 wrong verdicts on a target validator — by submitting carefully crafted controversial claims — can permanently remove that validator from the network. Combined with the absence of a validator set churn limit and the lack of slash count decay, this creates a viable grief attack against honest validators.

**Score**: 16/30 — CRITICAL remediation required before mainnet.

---

## Attack Surface 1: Micro-Slash Grief Attack

### The Core Vulnerability

**Both stacks**: A validator is permanently deactivated after `slashCount >= 3`, regardless of:
- How small each slash was (no minimum floor)
- How much time elapsed between slashes (no decay)
- Whether the validator was operating honestly (wrong verdicts in contested claims count)

#### TS Prototype Evidence

```
File: consensus/pot/validator_selection.ts:404
  if (updated.slashCount >= 3) {
    updated.isActive = false;
  }
```

Slashing rates:
- Wrong verification: 9% (`types.ts:99`)
- Missed verification: 7% (`types.ts:100`)
- Invalid claim: 22% (`types.ts:101`)

#### Cosmos SDK Evidence

```
File: x/staking/keeper/keeper.go:548
  if validator.SlashCount >= 3 {
    validator.IsActive = false
  }
```

Tier-adjusted slash multipliers (`x/staking/keeper/tiers.go:147-158`):
- Apprentice: 1.5× base slash
- Verified: 1.2× base slash
- Bonded/Guardian: 1.0× base slash

### Finding V1: No Slash Floor (MEDIUM)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | Slash amount can round to 0 for tiny stakes, yet slashCount still increments |
| JUSTICE | FAIL | A validator with 1 wei stake gets deactivated after 3 × 0-amount slashes |

**TS**: `(stake * BigInt(Math.floor(0.09 * 10000))) / 10000n` — a stake of `100n` yields slash of `0n`, but `slashCount` still increments.

**SDK**: Same issue — `slashInt` can be 0 after tier adjustment, but `SlashCount++` happens unconditionally at `keeper.go:523`.

**Impact**: An Apprentice with 0 stake can be slashed 3 times for `0 ulgm` and permanently deactivated. The attacker pays nothing; the victim loses their validator forever.

### Finding V2: No Slash Count Decay (HIGH)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | SlashCount monotonically increases, never decrements |
| UNDERSTANDING | FAIL | A validator slashed once at genesis can never fully recover |
| JUSTICE | FAIL | No distinction between a validator slashed 3 times in 1 epoch vs 3 times across 3 years |

**Both stacks**: No mechanism exists to reduce slashCount over time. A validator slashed at block 100, block 1,000,000, and block 10,000,000 is treated identically to one slashed 3 times in 3 consecutive blocks.

**Search evidence**: Grepped for `slashCount--`, `SlashCount--`, `decay`, `recover.*slash`, `reset.*slash` — 0 results in both stacks.

### Finding V3: Double Slash in Same Round (MEDIUM)

**TS only**: `verification_round.ts:583`:
```typescript
slashes.set(validator, (slashes.get(validator) || 0n) + slashAmount);
```

A validator who commits but reveals incorrectly gets slashed for **both** missed verification AND wrong verification in the same round. This adds to slashCount twice per round (once per `slashValidatorByAmount` call in the engine).

**SDK**: The knowledge module's `calculateRewardsAndSlashes` (`confidence.go:211-247`) iterates reveals and commitments separately, so a validator who commits but doesn't reveal gets the missed-reveal slash but not the wrong-verdict slash (correctly — they didn't reveal a verdict). **No double-slash in same round.**

### Finding V4: No Per-Epoch Slash Cap (HIGH)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | No limit on slashes per epoch |
| BEAUTY | FAIL | A validator can be deactivated in 3 consecutive blocks |

**Both stacks**: A validator selected for 3 verification rounds in the same epoch — where the consensus goes against them each time — accumulates 3 slashes and gets permanently deactivated.

**TS**: Verification rounds can occur every `verificationRoundBlocks = 11` blocks. An epoch of ~34,272 blocks contains ~3,115 potential rounds. A single validator can theoretically be slashed hundreds of times per epoch.

**SDK**: The knowledge module creates rounds per claim submitted. If many claims arrive in one epoch, a validator selected for multiple rounds can be slashed in each.

---

## Attack Surface 2: Validator Cycling

### Finding V5: Zero-Stake Apprentice Registration (HIGH — SDK only)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | Apprentice tier requires 0 stake |
| JUSTICE | FAIL | Virtual stake (111 LGM) grants VRF selection weight for free |

**SDK** (`x/staking/keeper/msg_server.go:33-38`): `RegisterValidator` allows `InitialStake = "0"`. The validator enters as Tier 0 (Apprentice) and gets `VirtualStake = 111 LGM` for VRF selection (`tiers.go:95-97`).

**Attack**: Register 111 zero-stake Apprentices (cap: `MaxApprenticeValidators = 111`). Each participates in verification with 111 LGM virtual stake. They vote wrong deliberately to grief target validators into wrong-verdict situations.

**TS**: Minimum stake is `minValidatorStake: 1111 LGM` (types.ts:92) — Apprentice tier allows `minStake: 0n` but the VRF selection filters by `params.minValidatorStake`. The TS prototype has a **higher effective bar** for participation than the SDK.

### Finding V6: No Validator Set Churn Limit (HIGH — Both)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | No max % of validator set can change per epoch |
| BEAUTY | FAIL | Entire validator set can be replaced in one block |

**Neither stack** has a `MaxValidatorChanges` or churn rate parameter. An attacker can register (and deactivate via 3 slashes) dozens of validators per epoch.

**SDK specific**: `MaxValidators = 100` is a hard global cap but no per-epoch delta limit.

### Finding V7: No Deregistration Handler — Zombie Validators (MEDIUM — SDK)

**SDK**: No `MsgDeregisterValidator` exists. Deactivated validators (`IsActive = false`) remain in the KV store permanently. They can never re-activate (no `MsgReactivateValidator` either).

**TS**: Validators can be re-registered by calling registration again, but deactivated validators (`isActive = false`) remain in the `ValidatorSet` map.

**Impact**: Over time, storage bloats with zombie validators that serve no purpose. If the validator count approaches `MaxValidators = 100`, inactive validators consume slots that new honest validators cannot fill.

### Finding V8: Unbonding Period Difference (INFO)

| Stack | Unbonding Period | Blocks |
|-------|-----------------|--------|
| TS | ~6.5 days | 222,222 blocks (`types.ts:107`) |
| SDK | ~7.0 days | 268,560 blocks (`genesis.go:13`) |

Minor divergence. Both are long enough to prevent instant-exit attacks. However, neither prevents a validator from registering and immediately beginning to participate in verification (no minimum "bonding warmup" period).

### Finding V9: No Registration Cooldown (MEDIUM — Both)

**Neither stack** has a cooldown between deactivation and re-registration with a new identity. An attacker can:
1. Register validator A
2. Get deactivated (3 slashes)
3. Immediately register validator B with a fresh DID/address
4. Repeat

The only cost is the registration gas fee + any stake they choose to put up (which can be 0 in SDK).

---

## Attack Surface 3: Weaponized Verification Selection

### Finding V10: Mandatory Participation, No Opt-Out (HIGH — Both)

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH | FAIL | VRF-selected validators must participate or face slash |
| JUSTICE | FAIL | Honest validators can be forced into contested claims |

Once a validator is VRF-selected for a verification round, they have two choices:
1. **Participate**: Risk 9% slash if their verdict disagrees with consensus
2. **Skip**: Guaranteed 7% slash for missed verification (TS) / 5% slash (`missedRevealSlashBps = 50000` in SDK)

There is no mechanism to decline selection, delegate participation, or abstain without penalty. A validator voting "abstain" avoids the wrong-verdict slash but the code only considers `accept` and `reject` verdicts (TS `verification_round.ts:583` — "uncertain" is not slashed but neither rewarded).

**SDK**: "abstain" verdict is explicitly not slashed (`confidence.go:223`: `else if reveal.Verdict != "abstain"`). This is a partial mitigation — validators can abstain safely. **But**: abstaining doesn't increment slashCount, so this only matters for reward loss, not grief.

### Finding V11: Claim Submission as Attack Vector (HIGH)

An attacker can submit claims designed to be **maximally controversial** — claims where honest validators will disagree with each other. If 51% of validators see the claim differently than the target validator, the target gets slashed.

**TS**: Any user can submit claims via `submit_claim` transaction type. The only cost is `minClaimStake = 100 LGM` (genesis.json:18). If the claim is rejected, the submitter loses 22% of their stake.

**SDK**: Claims submitted via `x/knowledge/keeper/msg_server.go`. The `InvalidClaimSlashBps = 220000` (22%) penalizes rejected claims.

**Attack economics**: Submitter stakes 100 LGM, loses 22 LGM if claim rejected. But if the controversial claim causes 3 target validators to get wrong-verdict slashes and deactivated, the attacker removes 3 validators from the network for 22 LGM.

---

## Cross-Layer Coherence Analysis

### Finding V12: TS/SDK Deactivation Threshold Aligned (INFO — Positive)

Both stacks use `slashCount >= 3` for deactivation. This is consistent but arguably too aggressive for both.

### Finding V13: Slash Rates Divergence (MEDIUM)

| Slash Type | TS Rate | SDK Rate | Delta |
|------------|---------|----------|-------|
| Wrong verification | 9% | 11% (110000 bps) | SDK harsher |
| Missed verification | 7% | 5% (50000 bps) | TS harsher |
| Invalid claim | 22% | 22% (220000 bps) | Aligned |

The wrong-verification rate divergence means SDK validators are slashed 22% more per wrong verdict than TS validators.

### Finding V14: Guardian MaxSlashCount Logic Divergence (MEDIUM)

**TS** (`types.ts:632`): `maxSlashCount: 0` — Guardian tier check is `state.slashCount < config.maxSlashCount` (validator_tiers.ts:49), meaning 0 slashes allowed. One slash → lose Guardian tier.

**SDK** (`tiers.go:41`): `guardianConfig.MaxSlashCount >= 0 && slashCount <= uint64(guardianConfig.MaxSlashCount)` — this means `slashCount <= 0`, which means 0 slashes allowed. Aligned with TS.

But the **DefaultTierConfigs** in SDK have `MaxSlashCount: -1` for tiers 0-2 (`types.go:56-87`), meaning "unlimited" — checking `MaxSlashCount >= 0` would be false, so the check is skipped. This is correct behavior but fragile logic.

---

## PP Gate Assessment

### TRUTH (Grounding)
- **FAIL**: Slash count increments even on 0-amount slashes (V1)
- **FAIL**: No empirical data on how often honest validators get wrong verdicts in contested domains (calibration gap)

### UNDERSTANDING (Mental Model)
- **FAIL**: The system presents itself as "harsh but fair" (settlement.ts comment at line 22) but honest validators can be permanently removed via 3 unfortunate selections
- **PASS**: Slashing rates are well-documented in both stacks

### BEAUTY (Elegance)
- **FAIL**: Double-slash in same round (TS) is inelegant — same event counted twice (V3)
- **FAIL**: Permanent deactivation with no recovery is a dead-end design (V2)
- **PASS**: Tier system is clean and well-structured

### JUSTICE (Right Placement)
- **FAIL**: No distinction between malicious and unfortunate slashing (V2)
- **FAIL**: Victims of grief attacks have no recourse — no appeal, no recovery (V2)
- **FAIL**: Zero-stake Apprentices can participate risk-free but cause damage (V5)

### CREATIVITY (New Territory)
- **FAIL**: The lack of anti-cycling and anti-grief mechanisms is a gap, not a design choice
- Recovery paths, cooldown mechanisms, and churn limits are well-understood in PoS literature

---

## Prioritized Findings Summary

| # | Finding | Severity | TS | SDK | Effort |
|---|---------|----------|----|----|--------|
| V2 | No slash count decay | **CRITICAL** | Yes | Yes | Medium |
| V4 | No per-epoch slash cap | **HIGH** | Yes | Yes | Low |
| V5 | Zero-stake Apprentice registration | **HIGH** | No* | Yes | Low |
| V10 | Mandatory participation, no safe exit | **HIGH** | Partial | Yes | Medium |
| V11 | Claim submission as grief weapon | **HIGH** | Yes | Yes | Medium |
| V6 | No validator set churn limit | **HIGH** | Yes | Yes | Low |
| V1 | No slash floor (0-amount slashes) | **MEDIUM** | Yes | Yes | Low |
| V3 | Double slash in same round | **MEDIUM** | Yes | No | Low |
| V9 | No registration cooldown | **MEDIUM** | Yes | Yes | Low |
| V7 | Zombie validators, no deregistration | **MEDIUM** | Yes | Yes | Medium |
| V13 | Slash rate divergence | **MEDIUM** | — | — | Low |
| V14 | Guardian MaxSlashCount fragile logic | **MEDIUM** | — | — | Low |
| V8 | Unbonding period divergence | **INFO** | — | — | Trivial |
| V12 | Deactivation threshold aligned | **INFO** | — | — | N/A |

*TS requires `minValidatorStake` for VRF eligibility, which blocks zero-stake participation.

---

## Recommended Remediations

### R1: Slash Count Decay (addresses V2) — CRITICAL
Add time-based decay: after N blocks with no new slashes, decrement `slashCount` by 1.
- **Suggested**: Decay 1 slash per epoch (~34,272 blocks / ~1 day)
- **Cap**: Never decay below 0
- **Implementation**: In BeginBlocker, check `currentBlock - lastSlashedBlock >= decayPeriod`

### R2: Per-Epoch Slash Cap (addresses V4) — HIGH
Add parameter `MaxSlashesPerEpoch` (suggested: 2).
- Track `slashesThisEpoch` per validator
- Reset at epoch boundary
- If cap reached, skip slash but still record wrong verdict (for reputation)

### R3: Minimum Stake for Participation (addresses V5) — HIGH
**SDK only**: Require `MinStakeForVerification` (e.g., 100 ulgm) for VRF eligibility, separate from tier minimum. Apprentices can register with 0 but cannot participate in verification until they stake.

### R4: Slash Floor (addresses V1) — MEDIUM
Skip `slashCount++` if `actualSlashAmount == 0`. A slash that takes nothing shouldn't count.

### R5: Claim Submission Rate Limit (addresses V11) — HIGH
Add `MaxClaimsPerAddressPerEpoch` parameter (suggested: 5). Prevents an attacker from flooding the network with controversial claims to force-select target validators.

### R6: Validator Set Churn Limit (addresses V6) — HIGH
Add `MaxNewValidatorsPerEpoch` (suggested: 10% of MaxValidators = 10). Prevents mass registration attacks.

### R7: Registration Cooldown (addresses V9) — MEDIUM
After deactivation, the same address cannot re-register for N blocks (suggested: 1 epoch).

### R8: Deregistration & Pruning (addresses V7) — MEDIUM
Add `MsgDeregisterValidator` for voluntary exit. Auto-prune validators inactive for >30 epochs from the store, freeing their slot.

### R9: Fix Double-Slash (addresses V3) — MEDIUM
**TS only**: In `finalizeVerificationRound`, ensure a validator is only slashed once per round (the higher of missed vs wrong, not both).

### R10: Bonding Warmup Period — LOW
New validators must wait N blocks (suggested: 1 epoch) before being eligible for VRF selection. Prevents instant cycling.

---

## Game Theory Analysis

### Current Nash Equilibrium (Broken)

Under current rules, a well-funded attacker can remove honest validators at cost:
- **Attacker cost per target removed**: ~66 LGM (3 claims × 22 LGM loss on rejection)
- **Victim cost**: Permanent deactivation + loss of staked capital (frozen forever)
- **Ratio**: 1:167× for a 11,111 LGM Guardian — attacker spends 66 LGM, victim loses 11,111 LGM

This is **not** a stable equilibrium. Honest validators face existential risk from well-funded attackers.

### Post-Remediation Equilibrium

With remediations R1-R6:
- **Attacker cost per target**: 5× higher (rate-limited claims, churn limit)
- **Victim recovery**: Slash count decays, per-epoch cap limits damage
- **Break-even**: Attacker needs sustained multi-epoch campaign (expensive) vs temporary setback (recoverable)

This restores cooperation as the dominant strategy.
