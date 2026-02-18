# SESSION-B7-4: ARBITER-POT — Vote Extensions via Arbiter-Path Challenge

**Date**: 2026-02-18
**Status**: BLOCKED — Arbiter path NOT WIRED (design exists, round creation missing)

---

## Summary

The arbiter verification path (Path B) — the core ABCI++ vote extension integration for non-deterministic claims — **cannot be triggered on the live network**. The resolution path logic correctly identifies non-deterministic categories and sets `ResolutionPath: "arbiter"` on `ProvisionalChallenge`, but **no code creates a `VerificationRound` with `Type: "challenge"`**. Vote extensions filter exclusively on `round.Type == RoundTypeChallenge`, so they silently skip all existing rounds.

This is a **design gap, not a bug** — the scaffolding is complete and consistent, but the final wiring step was never implemented.

---

## Step 1: Resolution Path Selection Analysis

### How the path is chosen

**File**: `x/knowledge/keeper/challenge.go:122-126`
```go
resolutionPath := "arbiter"
if types.IsDeterministicCategory(claim.EpistemicCategory) {
    resolutionPath = "deterministic"
}
```

**File**: `x/knowledge/types/types.go:137-143`
```go
func IsDeterministicCategory(category string) bool {
    switch category {
    case "protocol", "computational", "formal", "analytic":
        return true
    }
    return false
}
```

| Resolution Path | Epistemic Categories |
|----------------|---------------------|
| **Deterministic** (Path A) | `protocol`, `computational`, `formal`, `analytic` |
| **Arbiter** (Path B) | `empirical`, `social`, `predictive`, `oracle`, `attestation`, everything else |

### Vote extension filter

**File**: `app/vote_extensions.go:74-78`
```go
for _, round := range activeRounds {
    if adversarialEnabled && round.Type != knowledgetypes.RoundTypeChallenge {
        continue
    }
```

Vote extensions **only** process rounds where `Type == "challenge"`.

### Round type constants

**File**: `x/knowledge/types/types.go:158-160,780-781`
```go
RoundTypeVerification   = "verification"   // legacy commit-reveal
RoundTypeChallenge      = "challenge"      // arbiter round for Path B challenge
RoundTypeReverification = "reverification" // auto-triggered by confidence decay
RoundTypeContradiction  = "contradiction"  // contradiction verification
```

### Phase advancement filter

**File**: `x/knowledge/keeper/phases.go:61-69`
```go
func (k Keeper) AdvanceChallengeRoundPhases(ctx sdk.Context, currentHeight uint64) error {
    // ...
    for _, round := range activeRounds {
        if round.Type != types.RoundTypeChallenge {
            continue  // ← skips ALL existing rounds
        }
```

Both vote extensions AND phase advancement filter on `RoundTypeChallenge`.

---

## Step 2: Complete Audit of VerificationRound Creation Sites

Exhaustive grep of `VerificationRound{` in all production Go files:

| # | File:Line | Context | `Type` Field Set? |
|---|-----------|---------|-------------------|
| 1 | `keeper/rounds.go:23` | `CreateVerificationRound()` — initial claim verification | **NO** (empty string) |
| 2 | `keeper/msg_server.go:391` | `ChallengeFact` handler | **NO** (empty string) |
| 3 | `keeper/msg_server.go:796` | `ReportContradiction` handler | `RoundTypeContradiction` |
| 4 | `keeper/challenge.go:236` | `ResolvePendingDeterministicChallenges` synthetic round | **NO** (empty string) |
| 5 | `keeper/bvm_bridge.go:261` | BVM-triggered challenge round | **NO** (empty string) |
| 6 | `keeper/provisional.go:46` | Adversarial provisional fact synthetic round | **NO** (empty string) |

**Finding**: Out of 6 production `VerificationRound` creation sites, **ZERO** set `Type: types.RoundTypeChallenge`.

The only non-empty `Type` is `RoundTypeContradiction` (site #3, `ReportContradiction`).

---

## Step 3: The Gap — What's Missing

### What exists (correctly implemented):

1. **Path selection** (`challenge.go:122-126`): `OpenChallenge()` correctly identifies arbiter vs deterministic based on `IsDeterministicCategory()`
2. **ProvisionalChallenge struct** (`challenge.go:130-142`): Stores `ResolutionPath: "arbiter"` correctly
3. **Vote extension handlers** (`vote_extensions.go`): `handleCommitPhase` and `handleRevealPhase` fully implement VRF selection, claim evaluation, commitment hash generation, salt storage, and reveal
4. **Phase advancement** (`phases.go:59-96`): `AdvanceChallengeRoundPhases` correctly filters for `RoundTypeChallenge` and handles transitions
5. **Aggregation** (`phases.go`): `performAggregation` correctly handles challenge-type round outcomes
6. **Round type constant** (`types.go:159`): `RoundTypeChallenge = "challenge"` is defined

### What's missing:

**When `ChallengeProvisionalFact` is called and `ResolutionPath == "arbiter"`, no `VerificationRound` with `Type: RoundTypeChallenge` is created.**

The handler at `msg_server.go:422-467` calls `OpenChallenge()` which creates a `ProvisionalChallenge` and returns it, but the handler only sets `resp.DisputeId` (which is always empty — `DisputeId` is never populated by `OpenChallenge`).

Contrast with `ChallengeFact` at `msg_server.go:386-420` which **does** create a `VerificationRound` — but without setting `Type`, so it defaults to `""` and is still skipped by vote extensions.

---

## Step 4: Docker Devnet (Not Attempted)

Since the arbiter path cannot trigger without code changes, booting the Docker devnet would not produce any arbiter-path activity. The deterministic path (Path A) was already verified in B6-6.

---

## Step 5: Full Arbiter Cycle (Not Achievable)

The expected arbiter lifecycle is correctly designed:
```
Block N:     Challenge submitted → VerificationRound{Type: "challenge"} created
Block N+1:   ExtendVote: VRF selection → commit vote extension
             VerifyVoteExtension: validate commitment
             PrepareProposal: collect extensions → injection TX
             PreBlocker: process injection → store commitments
Block N+C:   CommitDeadlineHeight → transition to reveal
Block N+C+1: ExtendVote: reveal vote extension (salt + verdict)
Block N+C+R: RevealDeadlineHeight → aggregation
             performAggregation → verdict → ApplyChallengeOutcome
```

But this cycle **cannot execute** because no round with `Type: "challenge"` is ever created.

---

## Step 6: Commit/Reveal Integrity

The P0-1 batch-1 fixes in `vote_extensions.go` are correctly implemented:

1. **No false equivocation** (line 133-147): If commitment already exists for this validator+round, reuse it instead of generating a new one with different salt
2. **Reveal hash verification** (line 174-189): Reveal includes commitment hash for cross-check
3. **Proposer tampering protection** (line 264-300): `VerifyVoteExtension` validates commitment format and VRF proof

These are all correct but untestable on the live network until the wiring gap is fixed.

---

## Required Code Changes

### Fix 1: Set `Type: RoundTypeChallenge` in `ChallengeFact` (MINIMAL)

**File**: `x/knowledge/keeper/msg_server.go:391`

```go
// BEFORE:
round := types.VerificationRound{
    Id:                   roundID,
    ClaimId:              msg.FactId,
    Phase:                "commit",
    StartHeight:          height,
    CommitDeadlineHeight: height + params.CommitPhaseDuration,
    RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
    SelectedValidators:   []string{},
    Commitments:          make(map[string]*types.VerificationCommitment),
    Reveals:              make(map[string]*types.VerificationReveal),
}

// AFTER:
round := types.VerificationRound{
    Id:                   roundID,
    ClaimId:              msg.FactId,
    Phase:                "commit",
    Type:                 types.RoundTypeChallenge,  // ← ADD THIS
    StartHeight:          height,
    CommitDeadlineHeight: height + params.CommitPhaseDuration,
    RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
    SelectedValidators:   []string{},
    Commitments:          make(map[string]*types.VerificationCommitment),
    Reveals:              make(map[string]*types.VerificationReveal),
}
```

### Fix 2: Create `VerificationRound` in `ChallengeProvisionalFact` for arbiter path (ESSENTIAL)

**File**: `x/knowledge/keeper/msg_server.go:444-454`

```go
// BEFORE:
// For Path B (arbiter resolution), open a dispute via disputes module
if challenge.ResolutionPath == "arbiter" && challenge.FactId != "" {
    // The dispute will be handled by the existing disputes module
    resp.DisputeId = challenge.DisputeId
}

// AFTER:
// For Path B (arbiter resolution), create a challenge verification round
// that will be processed by vote extensions (ABCI++ arbiter path)
if challenge.ResolutionPath == "arbiter" {
    height := uint64(ctx.BlockHeight())
    roundID := GenerateRoundID(challenge.ClaimId+":arbiter-challenge", height)
    round := types.VerificationRound{
        Id:                   roundID,
        ClaimId:              challenge.ClaimId,
        Phase:                "commit",
        Type:                 types.RoundTypeChallenge,
        StartHeight:          height,
        CommitDeadlineHeight: height + params.CommitPhaseDuration,
        RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
        SelectedValidators:   []string{},
        Commitments:          make(map[string]*types.VerificationCommitment),
        Reveals:              make(map[string]*types.VerificationReveal),
    }
    k.SetVerificationRound(ctx, &round)
    resp.RoundId = roundID
}
```

### Fix 3 (Optional): Set `Type` in `bvm_bridge.go` and `rounds.go`

For completeness, consider setting `Type: RoundTypeVerification` explicitly on legacy rounds (sites #1, #5) to distinguish them from unset. This is not blocking but improves clarity.

---

## Findings Summary

| ID | Severity | Finding |
|----|----------|---------|
| **F1** | **P0** | `ChallengeProvisionalFact` creates `ProvisionalChallenge{ResolutionPath: "arbiter"}` but NO `VerificationRound{Type: "challenge"}` — arbiter path dead code |
| **F2** | **P0** | `ChallengeFact` creates `VerificationRound` without `Type` field — always skipped by vote extensions and `AdvanceChallengeRoundPhases` |
| **F3** | **P1** | `ChallengeProvisionalFact` references `challenge.DisputeId` but `OpenChallenge` never sets it — always empty string |
| **F4** | **P2** | Legacy round creation sites (rounds.go, bvm_bridge.go, provisional.go) don't set `Type` — implicit empty string works but explicit `RoundTypeVerification` would be clearer |

---

## Conclusion

The arbiter path (Path B) is **fully designed and scaffolded** but has a critical wiring gap: no code creates the `VerificationRound{Type: "challenge"}` that vote extensions and phase advancement filter on. Fix 1 + Fix 2 above (adding `Type: RoundTypeChallenge` and creating a round in the arbiter path) would wire the full ABCI++ arbiter lifecycle end-to-end.

Once fixed, the arbiter path should be re-tested on the 3-validator Docker devnet following the original B7-4 test plan (Steps 3-6).
