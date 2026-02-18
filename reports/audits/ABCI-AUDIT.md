# ABCI-AUDIT: ABCI++ and Proof of Truth Consensus Integration

**Date**: 2026-02-18
**Scope**: `app/abci.go`, `app/vote_extensions.go`, `app/local_state.go`, `app/sybil_funding_decorator.go`, `app/ante.go`, `app/ante_legible.go`, `x/knowledge/crypto/vrf.go`, `x/knowledge/evaluation/`, `x/knowledge/keeper/rounds.go`, `x/knowledge/keeper/phases.go`, `x/knowledge/keeper/confidence.go`, `x/knowledge/keeper/validator_selection.go`

---

## ABCI++ Data Flow (Per-Block Lifecycle)

```
Block N-1 finalized
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  ExtendVote (each validator, block N)                   │
│  ─ For each active round in commit phase:               │
│    1. Generate VRF(privKey, claimID, startHeight, prevHash)│
│    2. Check stake-weighted selection                     │
│    3. If selected: evaluate claim → verdict + confidence │
│    4. Generate random salt, compute commitment hash      │
│    5. Store locally (LocalCommitmentStore)               │
│    6. Attach VoteCommitment to vote extension            │
│  ─ For each active round in reveal phase:               │
│    1. Retrieve local commitment (salt, verdict, confidence)│
│    2. Attach VoteReveal to vote extension                │
└─────────────────────────────────────────────────────────┘
        │ (CometBFT broadcasts vote with extension)
        ▼
┌─────────────────────────────────────────────────────────┐
│  VerifyVoteExtension (each validator, for OTHER validators)│
│  ─ Structural: valid JSON, max 50 commitments/reveals   │
│  ─ Commitments: non-empty roundID + 64-char hash        │
│  ─ VRF proof verification (if present)                  │
│  ─ Reveals: valid verdict enum, confidence ≤ 1M         │
│  ─ Reveal-matches-commitment hash check                 │
└─────────────────────────────────────────────────────────┘
        │ (CometBFT commits block N with verified extensions)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PrepareProposal (proposer of block N+1)                │
│  ─ Emergency halt gate: filter non-emergency txs        │
│  ─ Collect vote extensions from LocalLastCommit         │
│  ─ Extract commitments + reveals into VoteExtInjection  │
│  ─ Sort by (roundID, validator) for determinism         │
│  ─ Encode with magic prefix 0x00VEX → inject at pos 0  │
│  ─ Add mempool txs (gas-limited by BlockGasLimit)       │
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  ProcessProposal (all validators, block N+1)            │
│  ─ Emergency halt gate: reject non-emergency txs        │
│  ─ VEX injection: decode check (valid JSON)             │
│  ─ Regular txs: decode + ValidateBasic                  │
│  ─ Accept or reject proposal                            │
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  FinalizeBlock → PreBlocker (all validators, block N+1) │
│  ─ Iterate txs, find VEX injection pseudo-tx            │
│  ─ ProcessVoteExtInjection:                             │
│    • Store each commitment in keeper state               │
│      (StoreCommitmentInRound: phase check, equivocation)│
│    • Store each reveal in keeper state                   │
│      (StoreRevealInRound: phase check, commitment check) │
│  ─ Count real txs for vesting_rewards                   │
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  BeginBlocker (knowledge module — LAST in ordering)     │
│  ─ AdvanceRoundPhases: commit→reveal→aggregation→expired│
│  ─ Adversarial pipeline (if enabled):                   │
│    1. EvaluatePendingClaims (structural scoring)        │
│    2. ResolvePendingDeterministicChallenges              │
│    3. AdvanceChallengeRoundPhases                        │
│    4. ProcessChallengeWindowExpiry                        │
│    5. Confidence growth + maturity (at epoch boundaries) │
│  ─ performAggregation when round enters aggregation     │
│  ─ CompleteRound: create fact, distribute rewards/slashes│
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  DeliverTx (regular user transactions)                  │
│  ─ Ante handler chain (16 decorators)                   │
│  ─ Message handlers                                      │
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  EndBlocker (knowledge module)                          │
│  ─ CleanupExpiredRounds (>100 blocks old)               │
│  ─ ExpireStaleFactsInEndBlocker (max 50/block)          │
│  ─ TriggerReverificationScan (max 10/block)             │
│  ─ ApplyCyclePenalties (every 1000 blocks)              │
│  ─ PruneUnusedFactsInEndBlocker (if enabled)            │
└─────────────────────────────────────────────────────────┘
```

---

## Vote Extension Format

### VoteExtension (JSON, per validator per block)
```json
{
  "validator_address": "lgmvaloper1...",
  "commitments": [
    {
      "round_id": "round_abc123",
      "commitment_hash": "sha256(domain || roundID || verdict || confidence || salt)",
      "vrf_output": "hex-encoded VRF output (32 bytes)",
      "vrf_proof": "hex-encoded VRF proof (96 bytes)",
      "height": 12345
    }
  ],
  "reveals": [
    {
      "round_id": "round_abc123",
      "verdict": "accept|reject|abstain",
      "confidence": 850000,
      "salt": "hex-encoded 16 random bytes"
    }
  ]
}
```

### Commitment Hash Construction (domain-separated, length-prefixed)
```
hash = SHA-256("LGM.commitment.v1:" || u32BE(len(roundID)) || roundID
              || u32BE(len(verdict)) || verdict || u64BE(confidence)
              || u32BE(len(salt)) || salt)
```
(`app/abci.go:461-482`)

### VRF Selection (ECVRF with Ed25519)
```
seed = SHA-256("LGM.vrf.seed.v1:" || u32BE(len(claimID)) || claimID
              || u64BE(round.StartHeight) || prevBlockHash)
selected = (vrfOutput * totalStake) < (stake * targetCount * 2^64)
```
All arithmetic uses `math/big.Int`. Selection probability ≈ `(stake/totalStake) * targetCount`.
(`x/knowledge/crypto/vrf.go:164-200`)

---

## Verification Round Lifecycle (Multi-Block)

```
                                                   Block heights
                                                   ─────────────►
        ┌──────────────┬──────────────┬────────────────┬──────────────┐
        │   COMMIT     │   REVEAL     │  AGGREGATION   │  COMPLETE    │
        │ (CommitDur)  │ (RevealDur)  │ (AggDur)       │  (final)     │
        └──────────────┴──────────────┴────────────────┴──────────────┘
        ^              ^              ^                 ^
  StartHeight   CommitDeadline  RevealDeadline   RevealDeadline+AggDur

COMMIT phase:
  - VRF-selected validators generate commitment hashes
  - Commitment = hash(verdict || confidence || salt)
  - Salt stored locally for reveal phase
  - Equivocation: different hash for same validator → slash

REVEAL phase:
  - Validators reveal verdict, confidence, salt
  - Reveal verified against stored commitment hash
  - No commitment → ErrNoCommitment (reject)
  - Equivocation: different verdict/confidence → slash

AGGREGATION phase:
  - Stake-weighted confidence aggregation
  - Agreement ratio = majority_weight / total_weight
  - Verdict thresholds: accepted (≥AcceptanceThreshold), contested (≥55%), else inconclusive
  - Stratum ceiling + category ceiling applied
  - Evidence bonus (max 15%) applied

COMPLETE:
  - accepted → create fact, return stake, vesting schedule
  - rejected → calibration negative, create counter-fact, slash submitter
  - contested → provisional fact, lower confidence
  - insufficient/inconclusive → return stake (no penalty)
```

(`x/knowledge/keeper/rounds.go:79-98`, `x/knowledge/keeper/phases.go`)

### Phase Determination (pure function)
```go
func GetRoundPhase(round, height, params) string:
  if round.Phase == "complete" → "complete"
  if height <= CommitDeadlineHeight → "commit"
  if height <= RevealDeadlineHeight → "reveal"
  if height <= RevealDeadlineHeight + AggregationPhaseDuration → "aggregation"
  else → "expired"
```

---

## PoT Consensus Rules → ABCI Methods

| PoT Concept | ABCI Method | Implementation |
|---|---|---|
| Claim submission | DeliverTx | `MsgSubmitClaim` → creates VerificationRound |
| Validator selection | ExtendVote | VRF with stake-weighted threshold |
| Commitment | ExtendVote → PreBlocker | Hash stored in round state |
| Reveal | ExtendVote → PreBlocker | Verdict + confidence stored |
| Aggregation | BeginBlocker | `AggregateVerificationResult` in confidence.go |
| Fact creation | BeginBlocker | `CompleteRound` → stores verified fact |
| Rewards/Slashes | BeginBlocker | `ApplyRewardsAndSlashes` |
| Equivocation | PreBlocker | `handleEquivocation` → slash + evidence |
| Sybil resistance | AnteHandler | `SybilFundingDecorator` → vote-weight decay |
| Emergency halt | PrepareProposal + ProcessProposal + AnteHandler | Triple-gated |

---

## Security Assessment

### Liveness

**Can a malicious proposer censor vote extensions?**
Yes. The proposer creates the injection tx from `LocalLastCommit.Votes` in PrepareProposal. They can:
- Omit specific validator's commitments/reveals (selective censorship)
- Omit the entire injection tx (drop all extensions)

**Mitigation**: Proposer rotation (CometBFT round-robin); commit/reveal phases span multiple blocks; censored validators can re-submit in the next block via ExtendVote.

**What happens if extensions are missing?**
- No injection tx → no commitments/reveals stored for that block
- Validators re-generate extensions in the next block
- But see P0-1: re-generated commitments use a new salt → equivocation risk

**Emergency halt**: Triple-gated at PrepareProposal (filter txs), ProcessProposal (reject proposals), and AnteHandler (block txs). Emergency module messages always pass.

### Safety

**Can a validator forge a commitment/reveal?**
- Commitments: The hash commits to verdict+confidence+salt. Forgery requires knowing the salt.
- Reveals: VerifyVoteExtension checks reveal matches commitment hash. But see P0-2: StoreRevealInRound does NOT re-verify this.
- VRF proofs: Verified cryptographically in VerifyVoteExtension. Forging requires the private key.

**What prevents vote extension tampering between ExtendVote and PrepareProposal?**
Nothing. The proposer extracts data from extensions and re-encodes as JSON. There's no signature or MAC on the injection data. ProcessProposal validators cannot verify the injection matches the original extensions. See P1-1.

### Determinism

| Component | Deterministic? | Mechanism |
|---|---|---|
| processVoteExtensions (sort) | ✓ | `sort.Slice` by (roundID, validator) |
| StoreCommitmentInRound | ✓ | Keyed by validator address |
| AggregateVerificationResult | ✓ | `sortedMapKeys()` for map iteration |
| Evaluation engine (10 stages) | ✓ | Integer BPS arithmetic, no floats |
| VRF generation | ✓ | Deterministic from seed (no randomness in VRF itself) |
| Reward/slash distribution | ✓ | `sortedMapKeys()` for deterministic order |
| Commitment salt | Non-deterministic (intentional) | `crypto/rand.Read` |

**Concern**: Salt generation uses `crypto/rand.Read`. This is intentional — salts must be unpredictable to prevent front-running. But it means the commitment hash is non-deterministic across validators (each produces a unique commitment), which is the design intent.

### Equivocation

**Double-commit detection**: `StoreCommitmentInRound` checks if a commitment exists for the validator. If the hash differs → equivocation detected, validator slashed, evidence stored. Same hash → idempotent duplicate (ErrDuplicateCommitment).

**Double-reveal detection**: `StoreRevealInRound` checks if a reveal exists. Different verdict/confidence → equivocation. Same → idempotent duplicate.

**Cross-validator equivocation**: A validator sending different extensions to different peers is prevented by CometBFT's vote signing — extensions are part of the signed precommit vote. Inconsistent votes are detectable by CometBFT's evidence mechanism.

### Timing (Multi-Block State Consistency)

Phase transitions happen in BeginBlocker based on block height vs round deadlines. The ordering guarantee (knowledge module runs LAST in BeginBlocker) ensures:
- Validator set is current (staking runs before knowledge)
- Qualifications are current (qualification runs before knowledge)
- Emergency halt status is known (emergency runs before knowledge)

Machine-checked at startup: `ValidateBlockerOrder` panics if dependency edges are violated (`app/blocker_order_validator.go`).

---

## Sybil Resistance Analysis

### SybilFundingDecorator (`app/sybil_funding_decorator.go`)

**Position in ante chain**: After `IncrementSequenceDecorator` (signer authenticated), before Legible post-auth decorators.

**Mechanism**:
1. Intercepts `MsgSend` and `MsgMultiSend` transactions
2. Records `sender → recipient` funding relationships with LGM amount and block height
3. Data stored via `govKeeper.RecordFunding(ctx, sender, recipient, amount, blockHeight)`
4. Used by governance module for vote-weight decay (voters funded by the same source get reduced weight)

**Scope**: Only tracks ulgm-denominated transfers. Non-LGM tokens are ignored.

**MsgMultiSend handling**: Each input sender is recorded as funding each output recipient. This creates N×M relationships for N inputs and M outputs — proportional tracking.

**Nil-safe**: Skips if govKeeper is nil. Skips simulation (state mutation).

**Limitation**: Only tracks on-chain MsgSend transfers. Off-chain funding (OTC, CEX) is invisible. IBC transfers are not tracked (only cosmos bank transfers). This is a known limitation — the decorator catches the simplest sybil pattern (on-chain splitting).

---

## Findings

### P0 — Critical

#### P0-1: Multi-block commitment generates different hashes → false equivocation slashing

**File**: `app/vote_extensions.go:116-201`

**Problem**: `handleCommitPhase` generates a **new random salt** on every invocation. Since `ExtendVote` runs every block, a validator selected for a round in two consecutive blocks produces two commitments with different hashes. When both are injected by (honest) proposers, `StoreCommitmentInRound` detects the second as equivocation and slashes the validator.

**Trace**:
1. Block N: V selected via VRF → generates C1 (salt S1), stored locally, attached to vote extension
2. Block N+1 (honest proposer): Injects C1 → `StoreCommitmentInRound` stores it
3. Block N+1: V's `ExtendVote` runs → generates C2 (salt S2) — **overwrites** S1 in `LocalCommitmentStore`
4. Block N+2 (honest proposer): Injects C2 → `StoreCommitmentInRound` finds C1 with different hash → **EQUIVOCATION** → V is slashed
5. Reveal phase: V's local store has S2 (from overwrite), but stored commitment uses S1 → reveal won't match → V also gets missed-reveal slash

**Root cause**: `handleCommitPhase` never checks the local store for an existing commitment before generating a new one.

**Impact**: Honest validators get slashed for equivocation (most severe PoT penalty) whenever they're VRF-selected in two consecutive blocks for the same round.

**Fix**:
```go
// In handleCommitPhase, after VRF selection check:
if existing, found := config.LocalStore.Get(round.Id); found {
    // Re-use existing commitment to prevent equivocation
    commitmentHash := ComputeCommitmentHash(round.Id, existing.Verdict, existing.Confidence, existing.Salt)
    return &VoteCommitment{
        RoundID:        round.Id,
        CommitmentHash: commitmentHash,
        VRFOutput:      hex.EncodeToString(vrfOutput),
        VRFProof:       hex.EncodeToString(vrfProof),
        Height:         uint64(ctx.BlockHeight()),
    }, nil
}
```

---

#### P0-2: StoreRevealInRound does not verify commitment hash

**File**: `x/knowledge/keeper/rounds.go:160-196`

**Problem**: When a reveal is stored, the keeper checks that the validator has a prior commitment (`round.Commitments[reveal.Validator]`) but does NOT verify that the reveal data (verdict, confidence, salt) produces the correct commitment hash.

**Current code** (rounds.go:175-177):
```go
// Only checks existence, not hash match
if _, hasCommitment := round.Commitments[reveal.Validator]; !hasCommitment {
    return ErrNoCommitment
}
```

**Attack**: A malicious proposer controls the injection tx content. They can modify a reveal's verdict (e.g., change "accept" to "reject") before injecting it. `StoreRevealInRound` stores the tampered reveal without verifying it matches the commitment hash. When aggregation runs, the wrong verdict is counted, altering the verification outcome.

**Defense gap**: `VerifyVoteExtension` (vote_extensions.go:311-324) DOES verify the commitment hash, but this only validates the original extension — not the injection tx which the proposer controls.

**Impact**: A malicious proposer (any validator during their proposer turn) can alter verification verdicts by tampering with reveals in the injection tx.

**Fix**: Add commitment hash verification in `StoreRevealInRound`:
```go
// After checking commitment exists:
commitment := round.Commitments[reveal.Validator]
if !VerifyCommitmentHash(commitment.CommitmentHash, roundID, reveal.Verdict, reveal.Confidence, reveal.Salt) {
    return fmt.Errorf("reveal does not match commitment hash for %s in round %s", reveal.Validator, roundID)
}
```

Note: `VerifyCommitmentHash` already exists in `app/abci.go:486-489` but is only used in `VerifyVoteExtension`. It needs to be called from the keeper storage path as well. Since it's in `app/` and the keeper is in `x/knowledge/keeper/`, either move the function to a shared package or duplicate it in the keeper.

---

### P1 — Important

#### P1-1: Vote extension injection not cryptographically bound to commit

**Files**: `app/abci.go:213-296` (ProcessProposal), `app/abci.go:132-211` (PrepareProposal)

**Problem**: `ProcessProposal` validates the injection tx by decoding JSON (line 252). It does NOT verify that the injection content matches the actual committed vote extensions from the previous block. Validators in `ProcessProposal` don't have access to `LocalLastCommit` (CometBFT v0.38 limitation — only `PrepareProposal` receives it).

**Consequence**: The proposer is trusted to faithfully translate extensions into the injection. A malicious proposer can:
- Selectively omit commitments/reveals (censorship)
- Modify reveals before injection (see P0-2)
- Inject commitments with altered hashes (see P0-1 context)

**Mitigation already present**: Proposer rotation; multi-block phases allow re-submission; VRF proofs can't be forged; commitment hashes bind verdict+salt.

**Standard fix** (used by other ABCI++ chains): Include the extended commit data (with validator signatures over their vote extensions) in the injection, and verify those signatures in `ProcessProposal`. This is a larger architectural change.

---

#### P1-2: No gas/size limit on VEX injection pseudo-tx

**File**: `app/abci.go:192-201`

**Problem**: PrepareProposal adds the injection tx at position 0 **before** the gas-limited tx loop. The injection bypasses gas accounting entirely:
```go
txs = append(txs, injBytes)  // no gas check
for _, tx := range req.Txs {
    estimatedGas := uint64(len(tx)) * 10
    if totalGas+estimatedGas > BlockGasLimit { break }
    ...
}
```

**Impact**: A large injection (many commitments/reveals) could consume significant block space, crowding out regular transactions.

**Mitigation**: `VerifyVoteExtension` limits each validator to 50 commitments + 50 reveals. With 100 validators, worst case ≈ 10,000 entries. At ~200 bytes/entry in JSON, this is ~2MB — significant but within typical block size limits.

**Fix**: Add an explicit size limit on the injection tx, or account for its size in the gas budget.

---

#### P1-3: VRF seed uses per-block prevBlockHash → unstable selection

**Files**: `app/vote_extensions.go:123-129`, `x/knowledge/crypto/vrf.go:203-214`

**Problem**: The VRF seed includes `prevBlockHash` which changes every block:
```go
seed = GenerateVRFSeed(claimID, round.StartHeight, prevBlockHash)
//                                                 ↑ changes per block!
```

This means the VRF output changes every block, so a validator may be selected in block N but not N+1, or selected in both with different outputs. This is the root enabler of P0-1.

**Design tension**: Using per-block hash prevents VRF grinding (attackers can't precompute which round they'll be selected for). Using a fixed hash (e.g., block hash at round start) would make selection stable but grindable.

**Recommendation**: Use block hash at `round.StartHeight` as the VRF seed for all blocks during that round. Grinding resistance comes from the VRF's cryptographic properties — the attacker can't influence the VRF output without the private key, regardless of the seed.

---

### P2 — Minor

#### P2-1: ProcessProposal halt check skips VEX injection

**File**: `app/abci.go:224-245`

```go
if isVoteExtInjectionTx(txBytes) {
    continue  // VEX injection passes even during halt
}
```

PoT data continues to be injected during emergency halt. If intentional, this allows verification rounds to complete during halt recovery. If unintentional, it's a leak in the halt gate. **Recommendation**: Document the intent. If PoT should pause during halt, don't skip VEX injection.

---

#### P2-2: PrepareProposal gas estimation uses tx-size heuristic

**File**: `app/abci.go:195`

```go
estimatedGas := uint64(len(tx)) * 10
```

This is a rough estimate for mempool tx gas. Actual gas depends on message type and execution cost. A small tx with expensive execution (e.g., `MsgSubmitClaim` with many references) could be under-estimated, causing blocks to exceed gas limits during execution.

---

#### P2-3: Comment inconsistencies

- `app/abci.go:218` says "Verify VRF proofs for proposer eligibility" but proposer selection is CometBFT's responsibility, not the application's.
- `app/abci.go:137` says "position 0" for injection tx, but the code doesn't enforce this — it just appends first. ProcessProposal processes VEX injection at any position.
- `app/abci.go:7-18` has a duplicated comment block (lines 7-12 and 13-18 are identical).

---

#### P2-4: LocalCommitmentStore CleanupExpired hardcoded to 100 blocks

**File**: `app/vote_extensions.go:95`

```go
config.LocalStore.CleanupExpired(currentHeight, 100)
```

The max age of 100 blocks (~252 seconds at 2521ms/block) might be too short if commit+reveal phases span more than 100 blocks. In that case, the local commitment would be cleaned up before the reveal phase, and the validator couldn't reveal.

**Check**: Commit duration + reveal duration from params. If they exceed 100 blocks, this is a bug. If they're shorter (likely, given PoT's ~19-block finality), this is fine.

---

## Evaluation Engine Determinism

The 10-stage evaluation pipeline (`x/knowledge/evaluation/`) produces identical results on all validators:

| Stage | Input | Determinism Guarantee |
|---|---|---|
| 1. Category base | Claim.EpistemicCategory | Static lookup table |
| 2. Calibration | Submitter's historical rate | Integer BPS arithmetic |
| 3. Participant context | Evidence type flags | Static adjustment table |
| 4. Evidence quality | Evidence type × reliability | Integer multiplication / division |
| 5. Fact graph | Referenced facts, contradictions | Sorted iteration, BFS depth-limited |
| 6. Novelty | Domain sparsity, bridge score | Integer fractions, capped |
| 7. Logic zone | Ontology zone + axioms | Zone ordering table |
| 8. Ceiling | Min(zone, stratum, category) | Simple min |
| 9. Edge cases | Claim properties | Boolean checks |
| 10. Counter-facts | Counter-fact confidence | Tiered integer penalties |

**No floats. No Date.now(). No randomness. Sorted map iteration everywhere.**

---

## Aggregation Algorithm

`AggregateVerificationResult` (confidence.go:45-220):

1. For each revealing validator (sorted order):
   - Get effective stake from staking keeper
   - Apply domain reputation weight: `stake = safeMulDiv(stake, repScore, 1M)`
   - Weighted confidence: `safeMulDiv(reveal.Confidence, stake, 1M)`
   - Accumulate into acceptWeight / rejectWeight / abstainWeight

2. Determine majority verdict (highest weight wins)

3. Agreement ratio: `safeMulDiv(majorWeight, 1M, totalWeight)`

4. Threshold logic:
   - `≥ AcceptanceThreshold` → use majority verdict
   - `≥ 550000` (55%) → "contested"
   - Below → "inconclusive"

5. Apply ceilings: stratum, category

6. Apply evidence bonus (max 15%)

7. AH-4 method diversity check (≥ MinDiversityParticipants)

8. Calculate rewards (correct verdict) and slashes (wrong verdict, missed reveal)

All arithmetic uses `safeMulDiv(a, b, c)` via `math/big.Int` to prevent overflow.

---

## BeginBlocker Ordering (Machine-Checked)

```
upgrade → vesting_rewards → distribution → slashing → SDK_staking → IBC
    → auth → emergency → tokens → LGM_staking → gov → research
    → ontology → disputes → evidence_mgmt → schedule → tree → bvm → home
    → qualification → capture_defense → capture_challenge
    → autopoiesis → alignment → billing → compute_pool → discovery → partnerships
    → KNOWLEDGE (LAST)
```

Critical edges verified at startup (`app/blocker_order_validator.go`):
- `upgrade → ALL custom modules` (halt before state changes)
- `auth → LGM_staking` (session cleanup before stake ops)
- `vesting_rewards → LGM_staking` (rewards available for unbonding)
- `tokens → LGM_staking` (period sweep before OriginPeriod)
- `emergency → capture_defense, knowledge` (halt status known)
- `ontology → qualification → knowledge` (domain → eligibility → rounds)
- `LGM_staking → knowledge` (validator set current)
- `capture_defense → knowledge` (reputation current)

---

## Files Read

| File | Lines | Purpose |
|---|---|---|
| `app/abci.go` | 490 | PrepareProposal, ProcessProposal, VoteExtInjection, CommitmentHash |
| `app/vote_extensions.go` | 397 | ExtendVote, VerifyVoteExtension, handleCommitPhase, handleRevealPhase |
| `app/local_state.go` | 144 | LocalCommitmentStore (disk-persisted, thread-safe, atomic write) |
| `app/sybil_funding_decorator.go` | 76 | MsgSend/MultiSend funding tracker for vote-weight decay |
| `app/ante.go` | 506 | 16-decorator ante chain, LGMGasDecorator, FeeRouterDecorator |
| `app/ante_legible.go` | 398 | BootstrapGasFree, EmergencyHalt, Account, Capability, DID decorators |
| `app/gas.go` | 60+ | Block/Tx gas limits, gas cost table |
| `app/blocker_order_validator.go` | 93 | Machine-checked BeginBlocker dependency graph |
| `app/app.go` | 1500+ | Module wiring, PreBlocker, BeginBlocker/EndBlocker ordering |
| `x/knowledge/crypto/vrf.go` | 372 | ECVRF Ed25519, GenerateVRF, VerifyVRF, IsValidatorSelected |
| `x/knowledge/evaluation/*.go` | ~1500 | 10-stage deterministic claim evaluation pipeline |
| `x/knowledge/keeper/phases.go` | 262 | BeginBlockerAdversarial, AdvanceRoundPhases, phase transitions |
| `x/knowledge/keeper/rounds.go` | 1435 | StoreCommitment/Reveal, equivocation, CompleteRound |
| `x/knowledge/keeper/confidence.go` | 400+ | AggregateVerificationResult, rewards/slashes, safeMulDiv |
| `x/knowledge/keeper/validator_selection.go` | 168 | VRF selection, eligible validators |
| `x/knowledge/module.go` | 200+ | BeginBlock/EndBlock entry points |

---

## Summary

| Severity | Count | Description |
|---|---|---|
| **P0** | **2** | Multi-block equivocation (false slashing), reveal hash not verified in storage |
| **P1** | **3** | Injection not crypto-bound, no gas limit on injection, unstable VRF seed |
| **P2** | **4** | Halt gate skip, gas heuristic, comment errors, cleanup hardcoded |

The ABCI++ integration is architecturally sound — the mapping from TS prototype to CometBFT lifecycle is correct, determinism is enforced via sorted iteration and integer arithmetic, and the BeginBlocker ordering has machine-checked invariants. The two P0 bugs are in the commit/reveal storage path and are fixable with targeted changes.
