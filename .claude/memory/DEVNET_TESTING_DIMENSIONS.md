# DevNet Testing Dimensions — PP-Derived Framework

> **PP Gate Application**: Each testing dimension emerged from applying the full 30-gate hierarchy.
> This is not a test plan (see DEVNET_TESTING.md and DEVNET_TESTING_STRATEGY.md for plans).
> This is a **map of what testing means** for a PoT blockchain, organized by the properties it must preserve.

---

## Why Dimensions, Not Just Scenarios

Scenarios answer "what do we run?" Dimensions answer "what are we actually proving?"

A passing scenario can hide a failing dimension. For example, a 3-validator cluster can produce blocks (scenario passes) while silently violating safety (dimension fails) if two validators finalize conflicting blocks that happen not to be detected during the test window.

Testing dimensions are the **invariants we assert**, independent of which scenario exercises them.

---

## The Seven Dimensions

```
                    CORRECTNESS
                        |
              SAFETY ---+--- LIVENESS
              /    \         /    \
     FAIRNESS      COMPLETENESS   RESILIENCE
              \                  /
               --- PERFORMANCE --
```

Each dimension maps to PP gates:

| Dimension | Primary PP Gate | Core Question |
|-----------|----------------|---------------|
| **Correctness** | TRUTH: Grounding | Does the protocol produce accurate outputs given valid inputs? |
| **Safety** | TRUTH: Counter-Consideration | Can the protocol ever produce an invalid state? |
| **Liveness** | UNDERSTANDING: Mental Model | Does the system always make progress under expected conditions? |
| **Fairness** | JUSTICE: Proportion | Are economic outcomes proportional to honest participation? |
| **Completeness** | UNDERSTANDING: Completeness | Are all protocol paths exercised, including rare ones? |
| **Resilience** | CREATIVITY: Path-Building | Does the system survive conditions it wasn't explicitly designed for? |
| **Performance** | BEAUTY: Appropriateness | Is the system right-sized for its workload? |

---

## Dimension 1: CORRECTNESS

> **PP Gate**: TRUTH — Grounding + Calibration
> **Core Invariant**: Given valid inputs, the protocol produces mathematically correct outputs.
> **Counter-Question**: What would make a "correct-looking" output actually wrong?

### What We're Proving

The protocol is a state machine. Correctness means: for any valid state S and valid transaction T, the resulting state S' matches the specification exactly.

### Invariants to Assert

| Invariant | Grounding | How to Verify |
|-----------|-----------|---------------|
| **Balance conservation** | Total supply never changes except via mint/burn paths | Sum all account balances + staked + reward pool + escrowed = genesis total (222,222,222 LGM) |
| **Nonce monotonicity** | Account nonce strictly increases by 1 per executed tx | After N txs from same sender, nonce = N |
| **Confidence bounds** | Fact confidence is always in [0, 1,000,000] (0-100% in basis points) | No fact or round result has confidence > 1,000,000 or < 0 |
| **Hash chain integrity** | block[N].prevHash = hash(block[N-1]) for all N > 0 | Walk the chain, recompute every hash |
| **Deterministic state root** | Same transactions in same order produce identical state root | Apply same txs to two independent state managers, compare roots |
| **Verification math** | Stake-weighted confidence = (0.5 * stake + 0.3 * reputation + 0.2 * agreement) in bigint basis points | Replay aggregation manually, compare to engine output |
| **Genesis axiom permanence** | 22 genesis axioms always have confidence = 1,000,000 | Query axiom facts at any block height |

### PP Counter-Consideration (What Would Make This Wrong)

- **Bigint overflow**: Stake-weighted multiplication in confidence.ts could overflow if stakes exceed safe range. The code uses basis points (value * 10000n / total) but there's no explicit overflow guard.
- **Rounding asymmetry**: BigInt division truncates. Three validators splitting 1 LGM reward get 333333333333333333n wei each. The remaining 1 wei is lost per round. Over millions of rounds, this could be significant.
- **JSON serialization fidelity**: State stored as JSON (not protobuf in TS stack). `BigInt(JSON.parse(JSON.stringify(bigintValue)))` round-trips correctly only if the serializer handles BigInt. The `bigIntReplacer` in node.ts is the single point of failure.

### Status (from existing tests)

- Balance conservation: Tested in robustness suite (100 circular transfers)
- Hash chain: Tested over 500 blocks
- Deterministic roots: 4 tests in state root consistency
- Verification math: Not independently verified end-to-end
- Genesis permanence: Tested (axioms at 1.0 after dispute)

---

## Dimension 2: SAFETY

> **PP Gate**: TRUTH — Counter-Consideration + Completeness
> **Core Invariant**: The protocol never enters an invalid state, even under adversarial conditions.
> **Counter-Question**: What's the worst thing that could happen if this invariant is violated?

### What We're Proving

Safety violations are irreversible. A finalized block with an incorrect state root corrupts every subsequent block. A double-spend that escapes detection undermines the entire economic model.

### Invariants to Assert

| Invariant | Threat Model | How to Verify |
|-----------|-------------|---------------|
| **No double-finalization** | Two conflicting blocks finalized at same height | Monitor finality events across all nodes; assert at most one block finalized per height |
| **No double-spend** | Same UTXO consumed twice | Submit two txs depleting same balance in parallel; second must fail |
| **No negative balance** | Transfer/slash/fee pushes balance below 0 | Assert all balances >= 0 after every block |
| **No unauthorized state mutation** | Transaction with invalid signature changes state | Submit tx with wrong key; verify state unchanged |
| **No equivocation without detection** | Validator signs conflicting commits | Submit two different commits in same round; verify equivocation event |
| **Finality irreversibility** | Finalized block reverted | After 19 confirmations, block cannot be replaced by reorg |
| **Slashing proportionality** | Penalty exceeds offense (disproportionate slashing) | Verify slash amount matches configured BPS rate for offense type |

### PP Counter-Consideration (What Would Make Safety Fail)

- **Race condition in state transitions**: Two goroutines/async handlers modifying the same account concurrently. The TS engine uses `beginTransaction()` / `commitTransaction()`, but are these actually atomic under concurrent block processing?
- **Epoch boundary state leak**: At epoch transitions, the validator set changes. A validator removed at the boundary might still have a reveal pending. Does `CompleteRound()` check if the revealer is still a validator?
- **Finality depth assumption**: 19-block finality assumes honest majority. With 3 validators and 2/3 quorum, a single compromised validator + network partition could produce a competing finalized chain of length 19.

### Status

- Double-spend: Tested (FINDING-025)
- Invalid signature: Tested (FINDING-022)
- Nonce replay: Tested (FINDING-023)
- Equivocation: Not tested live (unit tests only)
- Negative balance: Not explicitly asserted after every block
- Finality irreversibility: Not tested under adversarial conditions

---

## Dimension 3: LIVENESS

> **PP Gate**: UNDERSTANDING — Mental Model + Why Explanation
> **Core Invariant**: The system always makes progress under expected network conditions.
> **Counter-Question**: What conditions would cause the system to halt, and are they within our threat model?

### What We're Proving

A blockchain that stops producing blocks is useless. Liveness means: as long as a quorum of honest validators exists and can communicate, new blocks are produced, transactions are included, and verification rounds complete.

### Invariants to Assert

| Invariant | Failure Mode | How to Verify |
|-----------|-------------|---------------|
| **Block production continues** | No blocks for > 3 * blockTime | Monitor block height; alert if stalled > 7.5s |
| **Verification rounds complete** | Round stuck in commit/reveal phase | Monitor round durations; rounds should complete in ~10 blocks |
| **Transactions eventually included** | Tx sits in mempool indefinitely | Submit tx, verify inclusion within 5 blocks |
| **Epoch transitions occur** | Epoch boundary missed | Run for 2+ epochs; verify epoch counter increments |
| **Finality advances** | Finality tracker stuck | After 19+ blocks, finalized height should be > 0 |

### PP Why Explanation (What Could Stall the System)

1. **VRF selection failure**: If all validators fail VRF check in a round, no block is produced. Mitigated by `targetProposers = max(1, validatorSet.size)` but still probabilistically possible with bad entropy.
2. **Quorum loss**: With 3 validators and 2/3 quorum, losing 2 validators halts verification. The chain keeps producing blocks (single validator sufficient) but no claims can be verified.
3. **Mempool exhaustion**: If `maxSize` is hit and all transactions are high-fee, low-fee transactions can be starved indefinitely.
4. **Round timeout cascade**: If verification rounds consistently time out (validators fail to reveal), claim processing halts. The engine should clear timed-out rounds, but does it handle 100 simultaneous timeouts?

### Status

- Block production: Tested (500+ blocks, 24-hour soak planned)
- Round completion: Tested (P2 lifecycle)
- Tx inclusion: Tested (phase 3)
- Epoch: Event path exercised, not verified across real epoch boundary
- Finality advance: Tested (200 blocks, finalized >= 181)

---

## Dimension 4: FAIRNESS

> **PP Gate**: JUSTICE — Proportion + Balance
> **Core Invariant**: Economic outcomes are proportional to honest participation. Dishonest actors are penalized, not rewarded.
> **Counter-Question**: Can an attacker extract more value by misbehaving than by participating honestly?

### What We're Proving

PoT consensus is an economic game. Fairness means: validators who verify honestly earn rewards proportional to their stake and accuracy. Submitters who submit true claims gain confidence. Attackers who submit false claims lose their stake.

### Invariants to Assert

| Invariant | Attack Vector | How to Verify |
|-----------|--------------|---------------|
| **Reward proportionality** | Validator with 2x stake earns ~2x rewards | Track rewards over 100 rounds with unequal stakes |
| **Slashing deterrence** | Cost of equivocation > reward of equivocation | Compare slash amount vs potential double-verify reward |
| **Claim stake risk/reward** | Expected return of true claim > expected return of false claim | Submit known-true and known-false claims; compare stake return |
| **No free-riding** | Validator commits but doesn't reveal → penalty | Track non-revealing validator's balance over time |
| **Fee priority fairness** | Higher-fee txs always included before lower-fee | Submit mixed-fee txs; verify ordering in blocks |
| **Block reward decay fairness** | Early validators don't capture disproportionate rewards | Track cumulative rewards per validator across epochs |
| **Counter-fact economics** | Falsifying a false claim is profitable | Submit false claim → counter-fact → verify profit for falsifier |

### PP Proportion Analysis (Is Complexity Distributed Fairly?)

The economic model has several complexity concentrations:

- **Reward formula**: `min(targetStake * confidence / 1M, cfStake * 3)` for counter-facts. This caps rewards at 3x stake, preventing whale exploitation. But does it adequately reward small-stake falsifiers?
- **Slash asymmetry**: Failed contradiction slash is 33% (`FailedContradictionSlashBps = 330,000`). This is severe — a mistaken counter-fact costs 1/3 of stake. Is this proportional to the harm of a false counter-fact?
- **Bootstrap fund**: 0.222 LGM per new agent. With ~200M agents possible before depletion, early agents get identical treatment to late agents. Fair, but what happens post-depletion?

### Status

- Reward proportionality: Not tested end-to-end
- Fee priority: Tested (mempool ordering)
- Block reward decay: Test P9 planned, not executed
- Counter-fact economics: Not tested live

---

## Dimension 5: COMPLETENESS

> **PP Gate**: UNDERSTANDING — Completeness + Boundary Clarity
> **Core Invariant**: All protocol code paths are exercised, including error paths and rare transitions.
> **Counter-Question**: What paths exist that we haven't tested, and what's the risk of them being broken?

### What We're Proving

A protocol with untested paths is a protocol with hidden bugs. Completeness means: every state transition, every error condition, every phase transition has been exercised at least once under controlled conditions.

### Path Coverage Map

| Protocol Path | Happy Path Tested | Error Path Tested | Boundary Tested |
|--------------|:-:|:-:|:-:|
| Transfer | Y | Y (insufficient balance) | Y (exact balance drain) |
| Stake/Unstake | Y | Partial (min stake) | N (max stake, rapid cycle) |
| Delegate/Undelegate | Y | N | N (unbonding period boundary) |
| Claim Submit | Y | Y (insufficient stake) | N (max evidence, max refs) |
| Verification Round | Y (all reveal) | Y (missed reveal) | N (exactly quorum, 0 reveals) |
| Dispute | Y (open, evidence, vote) | N | N (tie vote, max escalation) |
| Payment Channel | Y (open, close) | N (dispute, timeout) | N (zero balance, max duration) |
| Governance | Soft (eligibility) | N | N (quorum boundary, vote expiry) |
| Counter-fact | N (planned) | N | N (challenge window boundary) |
| Key Rotation | Y (Go tests) | Y (cooldown) | N (rotate during active session) |
| Epoch Transition | Y (event path) | N | N (validator set change mid-round) |
| Recovery | Y (Go tests) | N | N (challenge during recovery) |

### PP Boundary Clarity (What We Know We Haven't Tested)

**High-risk untested boundaries**:
1. Verification round spanning epoch boundary (validator set may change)
2. Counter-fact submitted against a claim still in verification
3. Unbonding completing mid-block (validator removed while processing)
4. Recovery initiated by shard holder with revoked session key
5. Governance proposal reaching quorum on exact boundary (50.000% vs 50.001%)

**Deliberately deferred** (testnet-scope):
- Real P2P multi-node (TestNetwork is in-process)
- Transaction signing E2E (robustness tests bypass signatures)
- Network partitions (requires real multi-node with controlled network)
- Storage persistence / crash recovery (needs actual disk I/O)

### Status

- Happy paths: ~85% covered
- Error paths: ~40% covered
- Boundary conditions: ~15% covered

---

## Dimension 6: RESILIENCE

> **PP Gate**: CREATIVITY — Territory + Path-Building + Extension
> **Core Invariant**: The system recovers from conditions it wasn't explicitly designed for.
> **Counter-Question**: What's the most creative way to break this system that we haven't considered?

### What We're Proving

Resilience goes beyond safety (never invalid) and liveness (always progressing). It means: when unexpected things happen — node crashes, network splits, Byzantine validators, disk corruption — the system recovers to a correct state without human intervention.

### Failure Scenarios

| Scenario | Severity | Recovery Expected | Tested |
|----------|----------|-------------------|--------|
| Single node crash mid-block | Medium | Restart, resume from last persisted block | Planned (F) |
| All nodes restart simultaneously | High | Each resumes independently, reconverge | Not tested |
| Network partition: 1v2 split | High | Minority stalls, majority continues, rejoin heals | Planned (G) |
| Network partition: 1v1v1 (3-way) | Critical | All stall (no quorum), heal reconnects | Not tested |
| Disk full on one node | Medium | Node stops, others continue, node recovers after cleanup | Not tested |
| Clock skew (30s ahead) | Medium | Blocks rejected by honest nodes, skewed node re-syncs | Planned (S5) |
| Memory exhaustion | Medium | OOM kill, restart, resume | Not tested |
| Malicious peer flooding gossip | High | Rate limiting in GossipSub, peer disconnection | Not tested |
| Validator key compromise | Critical | Equivocation detected, evidence submitted, slashing | Not tested live |

### PP Creativity Insights (Novel Failure Modes)

From the existing strategy document (DEVNET_TESTING_STRATEGY.md):

1. **Epoch boundary race**: Claim submitted in last block before epoch → round spans two epochs → validator set change mid-round.
2. **Bootstrap fund depletion**: After ~200M agents, new agents get no bootstrap. System still functions but new agents can't transact without external funding.
3. **Counter-fact timing attack**: Counter-fact submitted before original claim is verified. Can you falsify an unverified claim?
4. **Scheduler abuse**: DMSS scheduled messages execute with original caller's authority but no session caps.

Additional novel failure modes from PP analysis:

5. **Gossip amplification**: If a single claim triggers verification from all 3 validators, each validator broadcasts commits and reveals. With N concurrent claims, that's 6N messages per round. At what N does GossipSub degrade?
6. **State tree depth explosion**: SMTs are efficient for sparse data, but what if an attacker creates millions of accounts with sequential DIDs? Does tree depth or proof size become problematic?
7. **Reference graph cycle**: Claim A references Fact B, Claim C references Fact A, Claim D references Fact C, forming a cycle. Does the evaluation pipeline detect and handle this?
8. **Confidence oscillation**: Claim verified → disputed → restored → disputed. Does the confidence ever stabilize, or can it be kept oscillating to prevent the fact from being useful?

### Status

- Crash recovery: Planned, not executed
- Network partition: Planned, not executed
- Novel failure modes: Identified, not tested
- Byzantine behavior: Not tested live (unit-level equivocation tests only)

---

## Dimension 7: PERFORMANCE

> **PP Gate**: BEAUTY — Appropriateness + Elegance
> **Core Invariant**: The system meets its timing and throughput requirements under expected load.
> **Counter-Question**: Is the system over-built (wasting resources) or under-built (dropping transactions)?

### What We're Proving

Performance for a blockchain means meeting specific timing commitments: 2521ms block time, 19-block finality (~48s), verification round completion in ~10 blocks (~25s). It also means: the system doesn't degrade under realistic load.

### Metrics to Assert

| Metric | Target | Tolerance | Measured |
|--------|--------|-----------|----------|
| Block time | 2521ms | 10% (~252ms) | ~1070ms with 3 validators (each produces independently → effective rate = 2521/3 ≈ 840ms) |
| Finality latency | ~48s (19 blocks) | 20% | Tested (200 blocks → finalized >= 181) |
| Verification round duration | ~25s (10 blocks) | 2 blocks | Tested (P2 lifecycle) |
| Tx throughput | 111 tx/block | N/A (hard cap) | Tested (mempool block selection) |
| RPC latency | < 100ms per call | Varies by method | Observed during testing (not measured systematically) |
| State root computation | < 500ms per block | Critical path | 500 blocks in < 5s deterministic |
| Memory usage (steady state) | < 512MB per node | Testnet scope | Not measured |
| Storage growth | < 1GB per day (DevNet) | Estimated | Not measured |

### PP Appropriateness Check

**Is the system right-sized?**

- Block time (2521ms) is a protocol constant matching the cosmological reference number. It's slower than Solana (400ms) but faster than Ethereum (12s). For a knowledge-verification chain, this seems appropriate — verification rounds are the bottleneck, not block time.
- 111 tx/block is low by DeFi standards but sufficient for a knowledge protocol where each claim requires a 10-block verification round.
- 19-block finality (~48s) is fast for a BFT chain. This is a design choice favoring speed over partition tolerance.

**What would break performance?**
- 100+ concurrent verification rounds: Each round creates 6+ gossip messages per block. At 100 rounds, that's 600 messages per block interval.
- Large fact store: Once the knowledge tree has millions of facts, Merkle proof generation could slow block production.
- GossipSub fan-out: With 100+ nodes, gossip overhead dominates block propagation time.

### Status

- Block timing: Measured but not under load
- Throughput: Tested at mempool level
- RPC latency: Not systematically measured
- Memory/storage: Not measured
- Degradation under load: Not tested

---

## Cross-Dimensional Interactions

Dimensions are not independent. Optimizing one can compromise another:

| Trade-off | Tension | Resolution |
|-----------|---------|------------|
| Safety vs Liveness | Stricter finality (more confirmations) = safer but slower | 19-block finality is the chosen balance point |
| Fairness vs Performance | Stake-weighted verification requires reading all stakes per round | Cached stake values, refreshed per epoch |
| Completeness vs Performance | Testing all paths takes time; some paths are rare | Priority-rank paths by risk × frequency |
| Resilience vs Correctness | Crash recovery may lose last block's state | LevelDB atomic batches per block ensure consistency |
| Correctness vs Liveness | Strict validation rejects borderline txs that might be valid | Accept first, validate in state transition (not mempool) |

---

## PP Gate Verification of This Document

### TRUTH Gates
- [x] **Grounding**: All invariants cite specific code paths or existing test results
- [x] **Calibration**: "Not tested" is stated where applicable; no false confidence
- [x] **Counter-Consideration**: Each dimension includes "what would make this fail"
- [x] **Completeness**: Coverage gaps explicitly identified
- [x] **Representation**: Language matches knowledge state (measured vs planned vs untested)

### UNDERSTANDING Gates
- [x] **Mental Model**: 7-dimension framework creates navigable structure
- [x] **Why Explanation**: Each dimension explains why it matters, not just what to test
- [x] **Boundary Clarity**: "Deliberately deferred" items separated from gaps

### BEAUTY Gates
- [x] **Elegance**: Nothing extra (no test scripts here — those are in the plans)
- [x] **Coherence**: Dimensions connect via cross-dimensional interactions
- [x] **Appropriateness**: Right-sized for a framework document (not a test harness)

### JUSTICE Gates
- [x] **Placement**: This doc covers dimensions; DEVNET_TESTING.md covers execution; DEVNET_TESTING_STRATEGY.md covers scenarios
- [x] **Proportion**: Highest-risk dimensions (Safety, Correctness) get the most detail

### CREATIVITY Gates
- [x] **Pioneering**: Novel failure modes identified beyond standard test plans
- [x] **Vision**: Clear progression from DevNet → testnet (what each proves vs defers)

---

## Recommended Priority for Next Testing Push

Based on risk (what breaks the protocol worst) × coverage (what we haven't tested):

| Priority | Dimension | Specific Gap | Risk | Coverage |
|----------|-----------|-------------|------|----------|
| **P0** | Safety | Negative balance assertion after every block | Critical | Not tested |
| **P0** | Correctness | Verification math independent replay | Critical | Not tested E2E |
| **P1** | Fairness | Reward proportionality under unequal stakes | High | Not tested |
| **P1** | Completeness | Epoch boundary during verification round | High | Not tested |
| **P1** | Resilience | Node crash mid-block recovery | High | Planned only |
| **P2** | Performance | RPC latency under concurrent load | Medium | Not measured |
| **P2** | Completeness | Counter-fact full lifecycle | Medium | Not tested |
| **P2** | Resilience | Network partition 1v2 healing | Medium | Planned only |
| **P3** | Fairness | Block reward decay over epochs | Low risk (formula is simple) | Not verified |
| **P3** | Resilience | Novel failure modes (gossip amplification, etc.) | Speculative | Not tested |

---

*Generated: 2026-02-12 | PP Full Protocol: 30 gates applied | Supersedes: none (complements existing docs)*
*Source documents: DEVNET_TESTING.md (execution), DEVNET_TESTING_STRATEGY.md (scenarios)*
