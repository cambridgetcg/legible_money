# DevNet PoT Testing Strategy — Full PP Analysis

> **PP Protocol Applied**: All 30 gates across 5 dimensions
> **Date**: 2026-02-12
> **Scope**: Node setup scenarios + PoT protocol participation across TS and Cosmos SDK stacks

---

## PP Dimensional Framework

### TRUTH: What do we actually know works?

| Claim | Evidence | Confidence |
|-------|----------|------------|
| TS engine produces blocks at ~2521ms | DevNet test results (56/56 PASS) | HIGH — directly observed |
| Cosmos chain produces blocks via CometBFT | DevNet testing (blocks observed) | HIGH — directly observed |
| TS commit-reveal rounds complete | engine.ts:938-985, verification_round.ts:179-462 | HIGH — code + tests |
| Cosmos ABCI++ vote extensions wired | app/abci.go VoteExtInjection struct | MEDIUM — code exists, not E2E tested |
| VRF selection works for solo validator | `targetProposers = max(1, validatorSet.size)` | HIGH — tested in devnet |
| Multi-node P2P discovery works | GossipSub + Kademlia in network/node.ts | MEDIUM — 3-node tested in prior sessions |
| Knowledge gRPC queries work on Cosmos | gRPC list shows 13 methods, Params returns | HIGH — tested this session |
| Custom Msg types decode on Cosmos | MsgCreateToken in block 34 | HIGH — observed |

**Truth gaps** (things we DON'T know):
- Whether TS commit-reveal works across multiple nodes (only tested solo)
- Whether Cosmos vote extensions actually carry PoT data through CometBFT
- Whether VRF selection distributes fairly across 3+ equal-stake validators
- Whether claim → fact lifecycle completes end-to-end on either stack
- Whether slashing actually executes (code exists, never triggered in test)

### UNDERSTANDING: Mental model of PoT flow

```
                        TS PROTOTYPE                              COSMOS SDK
                        ──────────                                ──────────
Claim Submit    →  engine.submitClaim()                    →  MsgSubmitClaim → keeper
                   ↓ Ontology validation                      ↓ Domain + category validation
                   ↓ Gödelian paradox check                   ↓ Content hash dedup
                   ↓ Stake deduction                          ↓ Bank escrow
                   ↓ Round creation                           ↓ CreateVerificationRound()

VRF Selection   →  Each validator computes VRF locally     →  ExtendVote: validator computes VRF
                   ↓ stake-weighted probability                ↓ Attaches to CometBFT vote
                   ↓ Selected if vrfOutput < threshold         ↓ Injected as tx #0 in next block

Commit Phase    →  submitCommitment() with VRF proof       →  Vote extension carries commitment
(4 blocks)         ↓ Hash = SHA512(verdict:conf:refs:salt)    ↓ ProcessVoteExtInjection stores
                   ↓ Stored in round.commitments               ↓ StoreCommitmentInRound()

Reveal Phase    →  submitReveal() with plaintext           →  Vote extension carries reveal
(4 blocks)         ↓ Verify hash matches commitment           ↓ StoreRevealInRound()
                   ↓ Accept or reject reveal                   ↓ Verify reveal matches commitment

Aggregation     →  aggregateRound() in confidence.ts       →  BeginBlocker → AdvanceRoundPhases
(3 blocks)         ↓ C = 0.5×stake + 0.3×rep + 0.2×agree    ↓ AggregateVerificationResult()
                   ↓ Apply stratum ceiling                     ↓ Stake-weighted with rep adjustment
                   ↓ Determine outcome                         ↓ Apply thresholds (77% accept)

Result          →  accepted/rejected/contested/insufficient →  CompleteRound()
                   ↓ Create fact or slash stake                ↓ Fact creation or stake slash
                   ↓ Distribute rewards                        ↓ Vesting schedule creation
                   ↓ Emit events                               ↓ Emit events
```

**Key divergences between stacks**:
1. TS uses libp2p GossipSub for commit/reveal propagation; Cosmos uses ABCI++ vote extensions
2. TS confidence: 50% stake + 30% reputation + 20% agreement; Cosmos: stake-weighted with rep adjustment
3. TS acceptance threshold: 0.77; Cosmos: configurable via params (default 770000/1M = 77%)
4. TS VRF: Ed25519-based; Cosmos: secp256k1 (different key type)
5. TS fee deduction: AFTER handler; Cosmos: BEFORE handler (ante decorator)

### BEAUTY: Clean test scenario design

Each scenario tests exactly **one dimension**. No mixed concerns.

| Scenario | Dimension | What It Proves |
|----------|-----------|---------------|
| A: Solo Validator | Block production baseline | Single node can produce + verify |
| B: 3-Node Cluster | P2P consensus | Multi-node agreement |
| C: Full Node | Read-only sync | Non-validator correctness |
| D: Claim Lifecycle | PoT end-to-end | Claim → Fact pipeline |
| E: Verification Round | Commit-reveal | Phase transitions + aggregation |
| F: Validator Join | Dynamic set | Runtime validator addition |
| G: Challenge | Negative knowledge | Counter-fact pipeline |
| H: Slashing | Economic security | Penalty execution |
| I: Equivocation | Byzantine detection | Double-sign punishment |
| J: Epoch Transition | Set rotation | Validator set changes |

### JUSTICE: Correct test placement

| Test Type | Where It Belongs | Why |
|-----------|-----------------|-----|
| Block production | TS DevNet (npm run devnet) | TS engine owns block production |
| CometBFT consensus | Cosmos DevNet (cosmos-devnet.sh) | CometBFT owns consensus |
| Claim submission E2E | Both stacks | Critical path must work on both |
| Verification rounds | TS DevNet primarily | Cosmos vote extensions less mature |
| gRPC/REST queries | Cosmos DevNet | Cosmos owns the API layer |
| VRF fairness | TS DevNet (1000+ blocks) | TS has direct VRF measurement |
| Slashing execution | Both stacks | Must verify independently |
| State persistence | TS DevNet (kill + restart) | TS LevelDB persistence tested |

### CREATIVITY: Novel failure modes to explore

1. **Epoch boundary race**: Claim submitted in last block before epoch → verification round spans two epochs → validator set changes mid-round
2. **Gödelian edge**: Submit self-referential claim that passes syntactic checks but is semantically paradoxical
3. **VRF grinding**: Validator restarts with new key to get better VRF output (should be prevented by deterministic seed)
4. **Counter-fact timing**: Submit counter-fact before original claim's verification completes
5. **Cascading revocation**: Fact A cites Fact B cites Fact C → revoke C → BFS propagation to A
6. **Stake exhaustion**: Validator receives so many slashes they go below minStake mid-epoch
7. **Commitment withholding**: Validator commits but strategically withholds reveal to influence outcome
8. **Split-brain verification**: Network partition during commit phase → different validators see different rounds

---

## Phase 1: Solo Validator Baseline (TS Stack)

### Setup
```bash
npm run devnet   # Single-node devnet with all 3 genesis validators simulated
```

### Tests

#### T1.1: Block Production Sanity
**What**: Verify blocks produce at ~2521ms with correct structure
**How**:
```bash
# RPC query for block height over 30 seconds
curl -s http://localhost:8545 -d '{"method":"lgm_getBlockHeight","params":[]}'
# Expect: height increases by ~12 blocks per 30 seconds
```
**Pass criteria**: Block time within 2521ms ±20%
**PP gate**: TRUTH.Grounding — directly measure, don't assume

#### T1.2: Genesis Axiom Verification
**What**: Verify 5 genesis axioms loaded with confidence 1.0
**How**:
```bash
npm run lgm -- facts
# Expect: 5 axioms (identity, non-contradiction, excluded middle, zero, successor)
# All confidence: 1.0
```
**Pass criteria**: All 5 axioms present, confidence = 1.0
**PP gate**: TRUTH.Completeness — verify ALL axioms, not just one

#### T1.3: Account Balance Check
**What**: Verify genesis accounts have correct balances
**How**:
```bash
npm run lgm -- balance   # Check faucet, reward pool, validators
```
**Pass criteria**: Faucet = 1B LGM, Reward pool = 133.3M LGM, each validator = 10K LGM staked
**PP gate**: TRUTH.Calibration — exact values, not "looks right"

#### T1.4: Transfer Transaction
**What**: Move 1 LGM from test account to another
**How**:
```bash
npm run lgm -- transfer <from> <to> 1000000000000000000
# Check balances before and after
```
**Pass criteria**: Sender balance decremented by 1 LGM + fee, receiver incremented by 1 LGM
**PP gate**: UNDERSTANDING.Practical — can we actually transact?

---

## Phase 2: Claim Lifecycle E2E (TS Stack)

### T2.1: Submit Claim
**What**: Submit a simple empirical claim and observe round creation
**How**:
```bash
npm run lgm -- claim science "Water boils at 100 degrees Celsius at standard pressure"
# Observe: claimId returned, round started
```
**Pass criteria**: claimId returned, `round_started` event emitted
**PP gate**: TRUTH.Grounding — observe the event, don't infer

#### T2.2: Verification Round Commit Phase
**What**: Observe validator(s) submitting commitments
**How**: Watch node logs for commitment submissions during 4-block window
```
Expected log pattern:
  [COMMIT] Round <roundId> | Validator <key> | Height N+1
  [COMMIT] Round <roundId> | Validator <key> | Height N+2
```
**Pass criteria**: At least 2 of 3 validators commit within 4 blocks
**PP gate**: UNDERSTANDING.Mental Model — confirm commit → reveal → aggregate flow

#### T2.3: Verification Round Reveal Phase
**What**: Observe validators revealing verdicts after commit deadline
**How**: Watch logs for reveal submissions during blocks N+5 through N+8
```
Expected log pattern:
  [REVEAL] Round <roundId> | Validator <key> | Verdict: confirmed | Confidence: 0.85
```
**Pass criteria**: Reveals match commitments (hash verification passes)
**PP gate**: TRUTH.Counter-Consideration — what if a reveal DOESN'T match?

#### T2.4: Aggregation & Fact Creation
**What**: Observe confidence aggregation and fact creation
**How**: Watch for `round_completed` event, then query fact
```bash
npm run lgm -- facts   # Should show new fact with confidence > 0
```
**Pass criteria**:
- Fact created with status "verified"
- Confidence = weighted average per formula (50% stake + 30% rep + 20% agreement)
- Stake returned to submitter
**PP gate**: UNDERSTANDING.Why — WHY is the confidence this value? Verify formula.

#### T2.5: Claim Rejection
**What**: Submit a claim that validators should reject
**How**:
```bash
npm run lgm -- claim science "The sun orbits the Earth"
# Expect: validators reject with low confidence
```
**Pass criteria**: Claim status = "rejected", submitter stake partially slashed (22%)
**PP gate**: BEAUTY.Elegance — the rejection path should be as clean as the acceptance path

#### T2.6: Insufficient Quorum
**What**: Submit claim when validators can't reach quorum
**How**: Reduce active validators to 1, submit claim requiring 3 verifiers (quorum = 0.66)
**Pass criteria**: Round outcome = "insufficient", stake returned to submitter (not their fault)
**PP gate**: JUSTICE.Proportion — submitter shouldn't be punished for validator failure

---

## Phase 3: Multi-Node Topology (TS Stack)

### T3.1: 3-Validator Cluster
**Setup**:
```bash
npm run devnet:multi -- --nodes 3
# Or manually: 3x tsx devnet/cmd/node.ts with separate configs
```
**Port mapping**:
- Node 1: RPC 8545, P2P 26656
- Node 2: RPC 8547, P2P 26657
- Node 3: RPC 8549, P2P 26658

**What**: Verify all 3 nodes discover each other and agree on chain state
**Pass criteria**:
- All 3 nodes at same height (±1 block)
- Blocks propagate within <500ms
- GossipSub topics active: `lgm-blocks`, `lgm-transactions`, `lgm-claims`, `lgm-verifications`
**PP gate**: TRUTH.Completeness — check ALL 4 topics, not just blocks

#### T3.2: Cross-Node Transaction Relay
**What**: Submit tx to node-2, verify it appears in block produced by node-1
**How**:
```bash
# Submit to node-2
curl -s http://localhost:8547 -d '{"method":"lgm_sendTransaction","params":[...]}'
# Check block on node-1
curl -s http://localhost:8545 -d '{"method":"lgm_getBlock","params":["latest"]}'
```
**Pass criteria**: Tx submitted to node-2 appears in block on node-1
**PP gate**: UNDERSTANDING.Practical — can we transact across nodes?

#### T3.3: Distributed Verification Round
**What**: Submit claim to node-1, verify all 3 validators participate in round
**How**: Submit claim via node-1 RPC, monitor all 3 node logs for commit/reveal activity
**Pass criteria**:
- All 3 validators commit (may commit to different nodes)
- All 3 validators reveal
- Confidence aggregation uses all 3 scores
- Same fact created on all 3 nodes
**PP gate**: BEAUTY.Coherence — the verification result must be identical across all nodes

#### T3.4: Full Node (Non-Validator)
**Setup**: Start 4th node with `validator.enabled: false`
**What**: Verify read-only node syncs chain without producing blocks
**Pass criteria**:
- Full node syncs to same height as validators
- RPC queries return correct state
- No block production from full node
- Mempool accepts and relays transactions
**PP gate**: JUSTICE.Responsibility — full nodes relay but don't sign

---

## Phase 4: Cosmos SDK PoT Participation

### Prerequisites
```bash
# Start Cosmos devnet
cd devnet/scripts && ./cosmos-devnet.sh start

# Verify running
curl -s localhost:26657/status | jq .result.sync_info.latest_block_height
```

#### T4.1: Register Validator (Cosmos)
**What**: Register a new validator via MsgRegisterValidator
**How**:
```bash
# Create and fund a new account
build/legbled keys add new-val --keyring-backend test --home ~/.legbled-devnet
build/legbled tx bank send validator $(build/legbled keys show new-val -a --keyring-backend test --home ~/.legbled-devnet) 1000000ulgm \
  --chain-id lgm-devnet-1 --keyring-backend test --home ~/.legbled-devnet -y

# Register as validator
# Note: Requires MsgRegisterValidator type to be registered (should work after our InterfaceRegistry fix)
```
**Pass criteria**: Validator appears in staking query, tier = Apprentice
**PP gate**: TRUTH.Grounding — verify the validator is actually registered, not just tx accepted

#### T4.2: Submit Knowledge Claim (Cosmos)
**What**: Submit a claim via Cosmos gRPC
**How**:
```bash
# Using grpcurl
grpcurl -plaintext -d '{
  "submitter": "lgm1...",
  "claim_text": "The Earth orbits the Sun at an average distance of 149.6 million km",
  "domain": "general",
  "epistemic_category": "empirical",
  "stake_amount": "111000"
}' localhost:9090 legible.knowledge.v1.Msg/SubmitClaim
```
**Pass criteria**: ClaimId returned, round created (or claim in pending state)
**PP gate**: UNDERSTANDING.Mental Model — trace full path through keeper → bank → state

#### T4.3: Vote Extension PoT Round (Cosmos)
**What**: Verify that ABCI++ vote extensions carry PoT commitments/reveals
**How**: Monitor CometBFT logs for vote extension activity after claim submission
```
Expected: VoteExtension with Commitments[] populated
Expected: Next block's tx[0] = VoteExtInjection with injected data
```
**Pass criteria**: Vote extensions contain PoT data, injected into blocks
**Known risk**: Vote extensions may not be functional yet — this is the **primary unknown**
**PP gate**: TRUTH.Counter-Consideration — what if vote extensions DON'T work? Fallback: direct MsgSubmitCommitment/MsgSubmitReveal

#### T4.4: Knowledge Params Query (Cosmos)
**What**: Verify knowledge module params are queryable and correct
**How**:
```bash
grpcurl -plaintext localhost:9090 legible.knowledge.v1.Query/Params
```
**Pass criteria**: Returns params with correct defaults:
- CommitPhaseDuration: 4
- RevealPhaseDuration: 4
- MinVerifiers: 1 (devnet override for single-node)
- AcceptanceThreshold: 770000
**PP gate**: TRUTH.Calibration — exact values must match DefaultParams()

#### T4.5: Direct Commit/Reveal via Msg (Cosmos Fallback)
**What**: If vote extensions don't work, test direct MsgSubmitCommitment/MsgSubmitReveal
**How**:
```bash
# After claim creates a round:
# 1. Compute commitment hash
# hash = SHA-256("LGM.commitment.v1:" || LP(roundId) || LP("accept") || LE64(850000) || LP(salt))

# 2. Submit commitment
grpcurl -plaintext -d '{
  "round_id": "<round-id>",
  "validator": "lgm1...",
  "commitment_hash": "<hex-hash>"
}' localhost:9090 legible.knowledge.v1.Msg/SubmitCommitment

# 3. Wait for reveal phase (4 blocks ≈ 10 seconds)

# 4. Submit reveal
grpcurl -plaintext -d '{
  "round_id": "<round-id>",
  "validator": "lgm1...",
  "verdict": "accept",
  "confidence": 850000,
  "salt": "<hex-salt>"
}' localhost:9090 legible.knowledge.v1.Msg/SubmitReveal
```
**Pass criteria**: Commitment stored, reveal verified, round completes
**PP gate**: CREATIVITY.Path-Building — if vote extensions fail, this path must work

---

## Phase 5: Economic Security Tests

#### T5.1: Wrong Verification Slash
**What**: Validator submits wrong verdict, verify 6% slash applied
**How**: Manually construct a reveal with "reject" verdict for a claim that gets accepted
**Pass criteria**: Validator stake reduced by 6%
**PP gate**: JUSTICE.Proportion — 6% is proportional to offense severity

#### T5.2: Missed Reveal Slash
**What**: Validator commits but fails to reveal
**How**: Submit commitment, then don't submit reveal within 4-block window
**Pass criteria**: Validator slashed 0.01% (minimal penalty — not their fault if network issue)
**PP gate**: JUSTICE.Balance — minimal penalty preserves fairness

#### T5.3: Equivocation Detection
**What**: Validator submits two different commitments for same round
**How**: Submit commitment with hash A, then submit with hash B
**Pass criteria**: Second commitment rejected, 100% stake slashed, evidence recorded
**PP gate**: TRUTH.Counter-Consideration — is the detection cryptographically sound?

#### T5.4: Block Reward Distribution
**What**: Verify block rewards decrease per epoch from finite fund
**How**: Track reward pool balance across multiple epochs (1111 blocks each)
**Pass criteria**: Balance decreases per block, rate decreases at epoch boundaries
**PP gate**: UNDERSTANDING.Boundary — what happens when fund is exhausted?

#### T5.5: Claim Stake Economics
**What**: Verify stake flow: lock → verify → return (accepted) or slash (rejected)
**How**:
1. Record submitter balance
2. Submit claim (stake locked)
3. Wait for verification
4. Check balance: if accepted, stake returned; if rejected, 22% slashed
**Pass criteria**: Balances match expected values exactly
**PP gate**: TRUTH.Calibration — exact numbers, not approximations

---

## Phase 6: Adversarial Scenarios

#### T6.1: Claim with Insufficient Stake
**What**: Submit claim with stake below minClaimStake (0.011 LGM)
**Pass criteria**: Transaction rejected with `insufficient_stake` error
**Why**: Economic spam prevention

#### T6.2: Claim in Nonexistent Domain
**What**: Submit claim targeting domain "unicorn_science"
**Pass criteria**: Transaction rejected with `domain_not_found` error
**Why**: Ontological integrity

#### T6.3: Duplicate Claim Submission
**What**: Submit identical claim text + domain twice
**Pass criteria**: Second submission rejected with `duplicate_claim` (content hash match)
**Why**: Knowledge graph deduplication

#### T6.4: Self-Referential Paradox
**What**: Submit claim that references itself: "This claim is false"
**Pass criteria**: Rejected by Gödelian analysis (TS) or accepted but flagged (Cosmos — if no Gödelian check)
**PP gate**: CREATIVITY — this is genuinely novel territory for blockchain consensus

#### T6.5: Counter-Fact Against Unverified Claim
**What**: Submit counter-fact before original claim completes verification
**Pass criteria**: Counter-fact rejected (can only contradict verified/contested facts)
**Why**: Counter-fact timing attack prevention (PP Insight #14 from DEVNET_TESTING_STRATEGY.md)

#### T6.6: Validator Stake Below Minimum After Slash
**What**: Slash a validator until stake < minValidatorStake
**Pass criteria**: Validator deactivated (`isActive = false`), stops producing blocks
**Why**: Economic threshold enforcement

---

## Phase 7: Cross-Stack Verification

#### T7.1: Same Claim, Both Stacks
**What**: Submit identical claim on TS devnet and Cosmos devnet, compare outcomes
**How**: Same claim text, domain, category, stake amount on both
**Pass criteria**: Same verification outcome (accept/reject) — confidence may differ slightly due to formula differences
**PP gate**: BEAUTY.Coherence — the two stacks should agree on truth

#### T7.2: Parameter Parity Check
**What**: Compare consensus parameters between TS genesis and Cosmos genesis
**How**: Read genesis.json (TS) and query params (Cosmos)
**Pass criteria**: Block time, quorum, thresholds, phase durations match
**PP gate**: TRUTH.Completeness — every parameter must be checked

#### T7.3: Stake Amount Parity
**What**: Verify stake denominations are equivalent
**How**:
- TS: 0.111 LGM = 111 × 10^15 wei
- Cosmos: 111000 ulgm (1 LGM = 10^6 ulgm)
**Pass criteria**: Documented conversion verified
**PP gate**: TRUTH.Representation — are we saying "same stake" when we mean it?

---

## Insights & Findings

### Insight 1: Vote Extension Uncertainty (CRITICAL)
The Cosmos PoT integration via ABCI++ vote extensions (`app/abci.go`) is architecturally complete but **never tested end-to-end**. The `VoteExtInjection` struct is defined, `PrepareProposal/ProcessProposal/ExtendVote/VerifyVoteExtension` handlers exist, but we don't know if CometBFT actually calls `ExtendVote` with our custom data. This is the **#1 risk** for Cosmos devnet PoT participation.

**Fallback**: Direct `MsgSubmitCommitment`/`MsgSubmitReveal` transactions bypass vote extensions entirely and use the keeper directly. This path is fully implemented and tested (keeper tests pass).

**Action**: Test T4.3 first. If vote extensions work, great. If not, fall back to T4.5 and document vote extensions as Phase 2.

### Insight 2: Key Type Mismatch
TS prototype uses Ed25519 keys throughout (signatures, VRF, DID). Cosmos SDK uses secp256k1 by default. This means:
- VRF proofs generated on TS can't be verified on Cosmos (different curve)
- Validator identities differ between stacks
- Cross-stack claim verification isn't possible without key bridging

**Implication**: The two stacks test PoT **independently**. There's no cross-stack verification of the same claim by the same validators.

### Insight 3: Confidence Formula Divergence
TS uses a 3-component formula: `0.5 × stake + 0.3 × reputation + 0.2 × agreement`
Cosmos uses stake-weighted with optional reputation adjustment from `captureDefenseKeeper`.

The formulas will produce **different confidence values** for the same inputs. This is acceptable for prototype vs production, but must be documented. The Cosmos formula is simpler but less nuanced.

### Insight 4: Solo vs Multi-Node Commit-Reveal
In solo-validator mode, the commit-reveal protocol is **performative** — the single validator commits and reveals to itself, so there's no information hiding. True commit-reveal security only manifests with 3+ validators where each independently evaluates without seeing others' verdicts.

**Testing implication**: Phase 2 (claim lifecycle) should ideally run on 3-node cluster, not solo. But solo is useful for basic smoke testing.

### Insight 5: Cosmos MinVerifiers Override
The Cosmos devnet genesis patches `min_verifiers: 1` to allow single-node operation. This bypasses quorum checks entirely. For meaningful PoT testing on Cosmos, we need either:
- Multi-validator Cosmos devnet (requires `persistent_peers` config)
- Or test quorum logic via Go unit tests (already done: knowledge keeper tests pass)

### Insight 6: Anti-Grief Measures Are Testable
The anti-grief system (max 2 slashes per epoch, slash count decay, deactivation at 3 cumulative slashes) can be tested by submitting multiple wrong verifications in rapid succession. This is a good candidate for automated testing.

### Insight 7: Challenge Window Impracticality
The counter-fact challenge window is 34,272 blocks (~24 hours at 2521ms). In devnet, this makes challenge-based tests impractical without:
- Overriding the genesis parameter to a shorter window (e.g., 100 blocks = ~4 minutes)
- Or a time-warp mechanism (not implemented)

**Recommendation**: Add devnet-specific `challenge_window_blocks: 100` to Cosmos genesis params.

### Insight 8: TS Engine Has Richer PoT Logic
The TS engine implements features not yet in Cosmos:
- Gödelian paradox detection
- Stratum confidence ceilings (from ontology)
- Adversarial verification mode (survive-to-accept)
- Anti-hallucination measures (AH-1 through AH-5)
- Marginal value pricing
- Checkpoint ceremony (BFT 2/3 signing)

The Cosmos implementation has: basic commit-reveal, stake-weighted aggregation, equivocation detection, and tier-based rewards. The gap is intentional (TS = prototype, Cosmos = production foundation) but affects which tests are possible on which stack.

### Insight 9: BeginBlocker Phase Advancement
On Cosmos, verification round phase transitions happen in `BeginBlocker` via `AdvanceRoundPhases()`. This means:
- Phase transitions are **block-triggered**, not time-triggered
- A round that should transition at block N+4 will transition when block N+5 is processed
- This introduces a 1-block latency compared to the TS engine (which transitions at exact block)

This is a subtle behavioral difference that could affect test expectations.

### Insight 10: Equivocation Is the Most Severe PoT Violation
Both stacks implement 100% stake slashing for equivocation (submitting conflicting commitments/reveals). The Cosmos implementation in `handleEquivocation()` additionally stores on-chain `EquivocationEvidence` for audit. This is the strongest economic deterrent in the protocol.

**Testing value**: Equivocation tests (T5.3) are the highest-value economic security tests because they validate the protocol's fundamental Byzantine assumption.

---

## Execution Order

| Phase | Tests | Stack | Duration | Prerequisites |
|-------|-------|-------|----------|---------------|
| **1: Smoke** | T1.1-T1.4 | TS | 15 min | `npm run devnet` works |
| **2: Claim E2E** | T2.1-T2.6 | TS | 45 min | Phase 1 passing |
| **3: Multi-Node** | T3.1-T3.4 | TS | 1 hour | Phase 1 passing |
| **4: Cosmos PoT** | T4.1-T4.5 | Cosmos | 1 hour | `cosmos-devnet.sh start` works |
| **5: Economics** | T5.1-T5.5 | Both | 1 hour | Phase 2 + Phase 4 passing |
| **6: Adversarial** | T6.1-T6.6 | Both | 1 hour | Phase 5 passing |
| **7: Cross-Stack** | T7.1-T7.3 | Both | 30 min | Phase 2 + Phase 4 passing |

**Total estimated hands-on time**: ~5.5 hours

---

## PP Gate Verification Summary

| Dimension | Gates Applied | Key Insight |
|-----------|--------------|-------------|
| **TRUTH** | Grounding, Calibration, Counter-Consideration, Completeness, Representation | Vote extension uncertainty is the #1 truth gap |
| **UNDERSTANDING** | Mental Model, Why, Practical, Boundary | Two stacks have different PoT formulas — understand the divergence |
| **BEAUTY** | Emergence, Fit, Elegance, Coherence, Appropriateness | Each test scenario tests exactly one dimension |
| **JUSTICE** | Placement, Proportion, Balance, Integrity | TS tests PoT logic, Cosmos tests infrastructure — right placement |
| **CREATIVITY** | Territory, Pioneering, Vision, Foundation | Equivocation, paradox, timing attacks are genuinely novel test territory |

---

*Generated: 2026-02-12 | PP Protocol: Full 30 gates applied | Supersedes: DEVNET_TESTING_STRATEGY.md (supplements, doesn't replace)*
