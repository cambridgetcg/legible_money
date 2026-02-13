# DevNet Dimensional Testing Strategy — PP-Guided

> **Framework**: Each testing dimension maps to a PP virtue.
> TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY
> Tests inherit the gate structure of the virtue they serve.

---

## Overview

| Dimension | PP Virtue | Focus | Tests |
|-----------|-----------|-------|-------|
| **D1: Correctness** | TRUTH | Does the protocol compute correctly? | 28 tests |
| **D2: Completeness** | UNDERSTANDING | Does the protocol cover all flows? | 24 tests |
| **D3: Interface** | BEAUTY | Is the API surface clean and complete? | 18 tests |
| **D4: Economics** | JUSTICE | Are rewards/penalties distributed fairly? | 22 tests |
| **D5: Resilience** | CREATIVITY | What novel failures can break the protocol? | 20 tests |

**Total**: 112 test scenarios across 5 dimensions.

**Existing Coverage**: 56/56 PASS (Phases 1-6) cover ~35% of these scenarios.
**Gap**: ~77 new test scenarios needed for comprehensive dimensional coverage.

---

## D1: TRUTH — Correctness Dimension

> *"Are my claims grounded in something I can identify?"* (Grounding Gate)
> *"Does my confidence match my evidence?"* (Calibration Gate)

### Principle

Every computation must produce the exact byte sequence it claims to produce. Every state transition must be deterministic. Every signature must verify. Truth testing is about **bit-perfect correctness** — the protocol either computes right or wrong, with no room for "close enough."

### PP Gates Applied

| Gate | Testing Interpretation |
|------|----------------------|
| **Grounding** | Every test assertion is grounded in a specific line of source code |
| **Calibration** | Test confidence matches implementation maturity |
| **Counter-Consideration** | Tests include negative cases (what SHOULD fail) |
| **Completeness** | All code paths exercised, not just happy paths |
| **Representation** | Test names accurately describe what they verify |
| **Citation** | Each test cites the canonical source it validates |

### T1: Cryptographic Primitives (7 tests)

| ID | Test | Source | Insight |
|----|------|--------|---------|
| T1.1 | `domainHash` uses SHA-256 (not SHA-512) | `core/crypto/hashing.ts:120` | **THE** critical truth — `sha512_256` is misnamed. Every reimplementation WILL diverge unless developers read line 42-44. |
| T1.2 | `sha512_256Hex` returns hex of SHA-256 hash | `core/crypto/hashing.ts:47` | Functional alias — must match `sha256` not `sha512.slice(0,32)` |
| T1.3 | `signDomainSeparated` → `verifyDomainSeparated` roundtrip | `core/crypto/signatures.ts:57-64,128-136` | Domain-separated signatures must verify with same domain string |
| T1.4 | Ed25519 signature rejects wrong message | `core/crypto/signatures.ts:128` | Negative case — altered message must return false |
| T1.5 | VRF output deterministic for same input | `core/crypto/vrf.ts` | Same (key, message) → same (output, proof), always |
| T1.6 | VRF proof verifies with public key | `core/crypto/vrf.ts` | Proof generated with private key validates with public key |
| T1.7 | Merkle proof validates inclusion | `core/crypto/merkle.ts` | Insert value → get proof → verify proof → true |

**PP Insight (Truth/Representation)**: Test T1.1 is the most important single test in this dimension. The `sha512_256` naming creates a **representation failure** that caused 3 independent test scripts to diverge. Any test harness MUST validate this canonical behavior first.

### T2: State Transition Determinism (8 tests)

| ID | Test | Source | Insight |
|----|------|--------|---------|
| T2.1 | Transfer: exact balance changes | `state/transitions.ts` | sender -= (amount + fee), recipient += amount, fee → routing |
| T2.2 | Transfer: nonce increments atomically | `state/transitions.ts` | sender.nonce += 1 after successful execution |
| T2.3 | Stake: balance decreases, validator stake increases | `state/transitions.ts` | Exact bigint math — no precision loss |
| T2.4 | Unstake: enters unbonding with correct period | `state/transitions.ts` | Unbonding duration = config.unbondingPeriod blocks |
| T2.5 | Delegate: stake attributed to correct validator | `state/transitions.ts` | Delegator retains ownership, validator gains weight |
| T2.6 | State root deterministic after same tx sequence | `state/state_manager.ts` | Two nodes processing identical blocks produce identical roots |
| T2.7 | Rollback restores exact prior state | `state/state_manager.ts` | beginTransaction → mutations → rollback → original state |
| T2.8 | Fee deduction uses `deductBalance` (no nonce bump) | `state/transitions.ts` | Fee routing must NOT increment nonce (fixed in Session 9) |

**PP Insight (Truth/Grounding)**: T2.6 is the *Grounding Gate* for consensus — if two nodes compute different state roots from the same input, they cannot reach consensus. This test grounds every other consensus test.

### T3: Commit-Reveal Correctness (7 tests)

| ID | Test | Source | Insight |
|----|------|--------|---------|
| T3.1 | Commitment hash matches reveal data | `consensus/pot/verification_round.ts:414-424` | `JSON.stringify({...salt})` → `sha512_256Hex(domainHash(...))` must match |
| T3.2 | Reveal with wrong salt rejected | `consensus/pot/verification_round.ts:430` | hash(data+wrongSalt) ≠ commitment |
| T3.3 | Reveal with wrong score rejected | `consensus/pot/verification_round.ts:430` | hash(wrongScore+salt) ≠ commitment |
| T3.4 | JSON key must be `salt` not `nonce` | `consensus/pot/verification_round.ts:420` | The salt/nonce semantic split — RPC accepts `nonce`, core uses `salt` |
| T3.5 | Domain-separated commit signature verifies | `consensus/pot/engine.ts:1001-1064` | `signDomainSeparated('LGM.pot.commit.v1', ...)` must verify |
| T3.6 | Evidence commit/reveal hash matches | `protocol/rpc/handlers/disputes.ts:210-215` | `sha512_256Hex()` (SHA-256) for evidence hashing |
| T3.7 | Double-commit same round rejected | `consensus/pot/verification_round.ts` | Equivocation detection must prevent second commit |

**PP Insight (Truth/Counter-Consideration)**: T3.4 is the Counter-Consideration gate in action. The test actively checks for the WRONG key name because history shows this exact error occurs when developers don't read the reveal verification code. Every commit-reveal test should include a "wrong key" negative case.

### T4: Block Validation (6 tests)

| ID | Test | Source | Insight |
|----|------|--------|---------|
| T4.1 | Block hash matches header contents | `consensus/blocks/producer.ts` | `domainHash('LGM.block.hash.v1', serializedHeader)` |
| T4.2 | Previous block hash links chain | `consensus/blocks/validator.ts` | `block.header.previousHash === previousBlock.hash` |
| T4.3 | Transaction Merkle root matches tx list | `consensus/blocks/validator.ts` | Recomputed root must equal header `txRoot` |
| T4.4 | State root matches after applying all txs | `consensus/blocks/validator.ts` | Apply block → compute root → compare header `stateRoot` |
| T4.5 | Block timestamp within acceptable range | `consensus/blocks/validator.ts` | Not in future, not too far in past |
| T4.6 | VRF proof validates proposer eligibility | `consensus/pot/validator_selection.ts` | `verifyBlockProposer()` with same `targetProposers` |

**PP Insight (Truth/Completeness)**: T4.3 and T4.4 are the most expensive tests but the most critical. A block whose transaction root doesn't match its contents is a malicious block. A block whose state root doesn't match post-application is an invalid state transition.

---

## D2: UNDERSTANDING — Completeness Dimension

> *"Does this build the right mental model?"* (Mental Model Gate)
> *"Does this explain WHY, not just WHAT?"* (Why Explanation Gate)
> *"Does the user know when this DOESN'T apply?"* (Boundary Clarity Gate)

### Principle

Every protocol flow must be tested end-to-end, from initiation through completion AND through failure. Understanding testing maps the **entire lifecycle** of each subsystem, identifying which transitions are tested and which are dark (untested paths that could harbor bugs).

### PP Gates Applied

| Gate | Testing Interpretation |
|------|----------------------|
| **Mental Model** | Tests create a complete map of each protocol flow |
| **Why Explanation** | Tests verify the reasoning behind each state transition |
| **Knowledge Connection** | Tests connect related subsystems (e.g., claims → verification → facts) |
| **Practical Application** | Tests demonstrate real usage patterns |
| **Boundary Clarity** | Tests verify boundaries (when features DON'T activate) |
| **Exhaustion** | Tests cover all reachable states, not just common paths |

### U1: Claim → Fact Lifecycle (6 tests)

| ID | Test | Flow | Status |
|----|------|------|--------|
| U1.1 | Submit claim → pending state | `claim_submit` → ClaimState.pending | Tested (Phase 4) |
| U1.2 | Pending → verification round created | Round assigned validators | Tested (Phase 4) |
| U1.3 | All validators commit+reveal → round complete | Full commit-reveal cycle | Tested (Phase 4) |
| U1.4 | Round result → fact created (accepted) | confidence ≥ 0.8 → FactState | Tested (Phase 4) |
| U1.5 | Round result → claim rejected (low confidence) | confidence < 0.8 → rejected | **UNTESTED** |
| U1.6 | Claim withdrawal before verification | `claim_withdraw` → stake returned | **UNTESTED** |

**Dark Path: U1.5** — No test currently submits a claim that gets rejected. We only test the happy path where validators give high scores. Need: validators score 0.2 → claim rejected → stake returned (minus penalty?).

**Dark Path: U1.6** — Claim withdrawal is a valid transaction type but never exercised in devnet tests.

### U2: Dispute Lifecycle (5 tests)

| ID | Test | Flow | Status |
|----|------|------|--------|
| U2.1 | Open dispute → set deadlines | disputeId, evidence deadline, voting deadline | Tested (Phase 5) |
| U2.2 | Evidence commit → evidence reveal | Hash verification | Tested (Phase 5) |
| U2.3 | Arbiter voting → resolution | 3/3 arbiter votes → outcome | Tested (Phase 5) |
| U2.4 | Dispute escalation → higher tier | `dispute_escalate` → new arbiters | **UNTESTED** |
| U2.5 | Dispute timeout → default resolution | Deadline passed, no resolution | **UNTESTED** |

**Dark Path: U2.4** — Escalation is a critical anti-gaming mechanism. If arbiters collude, escalation is the only recourse. This flow has never been tested on devnet.

### U3: Payment Channel Lifecycle (4 tests)

| ID | Test | Flow | Status |
|----|------|------|--------|
| U3.1 | Open channel → deposit locked | payer locks collateral | Tested (Phase 5) |
| U3.2 | Cooperative close → correct split | agreed split applied | Tested (Phase 5) |
| U3.3 | Unilateral close → challenge period | One party submits latest state | **UNTESTED** |
| U3.4 | Dispute channel → higher-nonce wins | Challenge with higher nonce | **UNTESTED** |

**Dark Path: U3.3/U3.4** — Only cooperative close is tested. Unilateral close and channel disputes are the security-critical paths (what happens when parties disagree).

### U4: Governance Lifecycle (5 tests)

| ID | Test | Flow | Status |
|----|------|------|--------|
| U4.1 | Submit LIP → eligibility check | reputation ≥ 0.70, capability ≥ 2 | Tested (Phase 5, correctly rejected) |
| U4.2 | LIP advance → review stage | Sufficient stake → transition | **UNTESTED** |
| U4.3 | Quadratic voting → tally | Vote weight = sqrt(stake) | **UNTESTED** |
| U4.4 | LIP voting → acceptance/rejection | Threshold met → advance | **UNTESTED** |
| U4.5 | LIP withdrawal by proposer | `lip_withdraw` → stake returned | **UNTESTED** |

**PP Insight (Understanding/Exhaustion)**: Governance is the most UNDER-tested lifecycle. Phase 5 only tests that governance endpoints are reachable and that eligibility is enforced. No LIP actually progresses through the full lifecycle (submit → stake → review → vote → accept/reject). This is a critical gap — governance bugs could lock the protocol's upgrade path.

### U5: Vesting Lifecycle (4 tests)

| ID | Test | Flow | Status |
|----|------|------|--------|
| U5.1 | Create vesting schedule | Verification reward → vesting | **UNTESTED** |
| U5.2 | Claim vested rewards | Cliff passed, linear release | **UNTESTED** |
| U5.3 | Falsification clawback | Counter-fact proven → clawback | **UNTESTED** |
| U5.4 | Vesting acceleration | High citation count → faster vesting | **UNTESTED** |

**PP Insight (Understanding/Boundary Clarity)**: The entire vesting subsystem is untested in devnet. This is the boundary where short-term incentives meet long-term alignment. Without vesting tests, we can't confirm that validators are actually receiving and claiming their rewards over time.

---

## D3: BEAUTY — Interface Dimension

> *"Is there anything extra? Anything missing?"* (Elegance Gate)
> *"Does this hang together as a unified whole?"* (Coherence Gate)
> *"Is this right-sized for the purpose?"* (Appropriateness Gate)

### Principle

The API surface must be complete, consistent, and well-proportioned. Every RPC method must return predictable structures. Error messages must be diagnostic. Beauty testing verifies the protocol's **external interface** — what developers and agents actually interact with.

### PP Gates Applied

| Gate | Testing Interpretation |
|------|----------------------|
| **Elegance** | API has no redundant endpoints and no missing ones |
| **Coherence** | Response formats are consistent across endpoints |
| **Clarity** | Error messages enable debugging |
| **Appropriateness** | Response payloads are right-sized |
| **Conformance** | API follows blockchain RPC conventions (cf. Ethereum JSON-RPC) |

### B1: RPC Surface Completeness (8 tests)

| ID | Test | Method | Status |
|----|------|--------|--------|
| B1.1 | Block height method exists | `lgm_getBlockHeight` | Tested (Phase 1) |
| B1.2 | Network stats method exists | `lgm_getNetworkStats` | Tested (Phase 1) |
| B1.3 | Account balance by DID | `lgm_getBalance` | Tested (Phase 1) |
| B1.4 | Validator set query | `lgm_getValidators` | **UNTESTED** — should return active set with stakes |
| B1.5 | Transaction by hash lookup | `lgm_getTransaction` | **UNTESTED** — essential for wallets |
| B1.6 | Block by height query | `lgm_getBlock` | **UNTESTED** — essential for explorers |
| B1.7 | Fact by ID query | `knowledge_getFact` | **UNTESTED** — core PoT product |
| B1.8 | Mempool stats | `lgm_getMempoolStats` | **UNTESTED** — operational visibility |

**PP Insight (Beauty/Elegance)**: The API has 31+ endpoints but tests only validate ~12 of them. The untested endpoints (B1.4-B1.8) represent the most commonly used methods in blockchain ecosystem tooling. An explorer, wallet, or indexer would hit these first.

**Conformance Check**: Compare against Ethereum JSON-RPC standard:

| Ethereum Method | LGM Equivalent | Status |
|-----------------|---------------|--------|
| `eth_blockNumber` | `lgm_getBlockHeight` | Exists |
| `eth_getBalance` | `lgm_getBalance` | Exists |
| `eth_getBlockByNumber` | `lgm_getBlock` | **Missing from tests** |
| `eth_getTransactionByHash` | `lgm_getTransaction` | **Missing from tests** |
| `eth_sendRawTransaction` | `lgm_sendTransaction` | Exists |
| `net_peerCount` | `lgm_getNetworkStats.peerCount` | Exists |

### B2: Error Response Quality (5 tests)

| ID | Test | Input | Expected |
|----|------|-------|----------|
| B2.1 | Invalid method name | `lgm_nonexistent` | `-32601: Method not found` |
| B2.2 | Missing required params | `lgm_getBalance` (no DID) | `-32602: Invalid params` with field name |
| B2.3 | Malformed DID format | `lgm_getBalance("not-a-did")` | Descriptive error, not stack trace |
| B2.4 | Integer overflow in amount | Transfer with `2^256` | Graceful rejection, not crash |
| B2.5 | Empty string params | `lgm_getBalance("")` | Descriptive error |

**PP Insight (Beauty/Clarity)**: Error messages are the API's diagnostic voice. A `-32000: Internal error` tells the developer nothing. `-32004: Invalid nonce: expected 12, got 11` (confirmed working in Phase 6) is excellent — it tells the developer exactly what's wrong and what's expected.

### B3: Response Format Consistency (5 tests)

| ID | Test | Check | Why |
|----|------|-------|-----|
| B3.1 | All bigint fields are strings | `"balance": "1000000"` not `1000000` | JSON has no bigint; Number loses precision |
| B3.2 | All hashes are hex strings | `"hash": "0xabc..."` | Consistent hex encoding |
| B3.3 | All block heights are strings | `"blockHeight": "42"` | BigInt safety |
| B3.4 | Timestamps are ISO-8601 or block numbers | Not Unix timestamps AND block numbers | Consistency |
| B3.5 | Error codes follow JSON-RPC 2.0 spec | `-32xxx` range | Standard compliance |

**PP Insight (Beauty/Coherence)**: Inconsistent response formats create invisible bugs. If `lgm_getBalance` returns balance as a number but `lgm_getBlockHeight` returns height as a string, every consumer must handle both formats. The protocol MUST pick one convention and enforce it everywhere.

---

## D4: JUSTICE — Economics Dimension

> *"Is complexity distributed fairly?"* (Proportion Gate)
> *"Does this maintain system equilibrium?"* (Balance Gate)
> *"Does this preserve architectural integrity?"* (Integrity Gate)

### Principle

Economic testing verifies that the protocol distributes rewards, penalties, and costs **fairly** — meaning according to its declared rules, with no exploitable asymmetries. Justice testing is about **mathematical fairness**: do the numbers add up? Are incentives aligned? Can anyone game the system?

### PP Gates Applied

| Gate | Testing Interpretation |
|------|----------------------|
| **Placement** | Rewards land in the right accounts |
| **Ownership** | The rightful validator receives the reward |
| **Proportion** | Rewards are proportional to contribution |
| **Balance** | Total system supply is conserved |
| **Integrity** | Economic invariants hold across all transactions |
| **Evidence** | Each reward/penalty can cite the triggering event |

### J1: Reward Distribution (7 tests)

| ID | Test | Mechanism | Status |
|----|------|-----------|--------|
| J1.1 | Block reward flows to proposer | 11 LGM per block (decaying) | **PARTIAL** — tested reward pool exists, not that proposer receives it |
| J1.2 | Verification reward splits among validators | 1 LGM / numParticipants | **UNTESTED** — Phase 4 confirms round completes but doesn't check validator balance changes |
| J1.3 | Citation reward flows to fact creators | 1 LGM per citation | **UNTESTED** |
| J1.4 | Research fund receives 22% of evidence tax | `splitEvidenceTax()` | **UNTESTED** — sustainability tests check fund exists but not inflow |
| J1.5 | 5-way treasury split is exact | 77% research, 7% each for foundation/community/developers, 2% reserve | **UNTESTED** |
| J1.6 | Block reward decay matches emission schedule | `DEFAULT_EMISSION_SCHEDULE` | **UNTESTED** |
| J1.7 | Total minted never exceeds supply cap | 222,222,222 LGM | **UNTESTED** — requires long-running test |

**PP Insight (Justice/Proportion)**: J1.2 is the most important untested economic test. Validators stake LGM to participate in verification. If they don't receive rewards proportional to their contribution, rational validators will leave the network. The Proportion Gate demands: does each validator receive `reward × (theirStake / totalStake)`?

### J2: Penalty & Slashing (5 tests)

| ID | Test | Mechanism | Status |
|----|------|-----------|--------|
| J2.1 | Invalid signature → transaction rejected (no penalty) | Mempool validation | Tested (Phase 6) |
| J2.2 | Nonce replay → transaction rejected (no penalty) | State transition check | Tested (Phase 6) |
| J2.3 | Equivocation → stake slashed | Double-commit detection | **UNTESTED** |
| J2.4 | Missed reveal → reputation penalty | Timeout detection | **UNTESTED** |
| J2.5 | Counter-fact proven → original staker slashed | `FailedContradictionSlashBps` = 33% | **UNTESTED** |

**PP Insight (Justice/Integrity)**: J2.3 (equivocation slashing) is the protocol's primary deterrent against Byzantine validators. Without testing this, we can't confirm that the economic game theory holds. A rational validator must know: if I equivocate, I lose X% of stake. If X = 0 (untested), the deterrent doesn't exist.

### J3: Fee Routing (5 tests)

| ID | Test | Mechanism | Status |
|----|------|-----------|--------|
| J3.1 | Transfer fee deducted from sender | `gas * gasPrice` | Tested (Phase 3 — 711 LGM spent, 611 staked) |
| J3.2 | Fee routes to protocol treasury | Not validator, not burned | **UNTESTED** — fee destination not verified |
| J3.3 | Gas estimation returns correct cost | `lgm_estimateGas` | **UNTESTED** |
| J3.4 | Complex tx types have higher gas | claim_submit > transfer | **UNTESTED** — gas map exists but not verified live |
| J3.5 | Free-tier queries don't charge | `freeTierQueries=10` | Tested (Phase 1, Phase 4) |

### J4: Supply Conservation (3 tests)

| ID | Test | Invariant | Status |
|----|------|-----------|--------|
| J4.1 | Total supply = sum of all balances + locked stakes | Conservation law | **UNTESTED** — critical accounting invariant |
| J4.2 | No transaction creates or destroys tokens (except block reward) | Only `add_fact` mints via reward pool | **UNTESTED** |
| J4.3 | Reward pool decreases by exactly the amount distributed | `pool -= blockReward` per block | **UNTESTED** |

**PP Insight (Justice/Balance)**: J4.1 is the **fundamental accounting equation** of the protocol. If total supply ≠ sum(balances) + sum(stakes) + sum(locked) at any point, tokens are being created or destroyed incorrectly. This should be checked after EVERY block in integration tests.

### J5: Bootstrap Fund & Faucet (2 tests)

| ID | Test | Mechanism | Status |
|----|------|-----------|--------|
| J5.1 | First-tx bootstrap grant = 0.222 LGM | `nonce === 0` → auto-grant | Tested (Phase 3 implicitly) |
| J5.2 | Second-tx no bootstrap grant | `nonce > 0` → no grant | **UNTESTED** — need explicit verification |

---

## D5: CREATIVITY — Resilience Dimension

> *"Is this genuinely new ground?"* (Pioneering Gate)
> *"Is there clarity about where this leads?"* (Vision Gate)

### Principle

Resilience testing explores the **uncharted territory** of failure modes — scenarios the developers didn't design for, edge cases at the intersection of subsystems, and adversarial strategies that exploit assumptions. Creativity testing asks: what happens when the protocol encounters something it was never designed to handle?

### PP Gates Applied

| Gate | Testing Interpretation |
|------|----------------------|
| **Territory** | Is this a genuinely novel failure mode or a known category? |
| **Pioneering** | Does this test explore territory no existing test covers? |
| **Path-Building** | Does this test create a reusable pattern for similar scenarios? |
| **Vision** | What systemic risk does this failure mode represent? |
| **Foundation** | Does fixing this class of bug prevent future variants? |
| **Integration** | Does this failure cross subsystem boundaries? |

### C1: Cross-Subsystem Attacks (5 tests)

| ID | Test | Attack Vector | Subsystems Crossed |
|----|------|--------------|-------------------|
| C1.1 | **Epoch boundary race**: Submit claim in last block before epoch, validator unstakes at epoch → who reveals? | Consensus × Staking | **NOVEL** — Timing exploits at epoch boundaries |
| C1.2 | **Counter-fact before verification**: Submit counter-fact referencing an unverified claim (still in verification round) | Knowledge × Negative Knowledge | **NOVEL** — Ordering assumption violation |
| C1.3 | **Channel fund double-spend**: Open payment channel with same stake used for validation | Payment Channels × Staking | **NOVEL** — Are funds truly locked? |
| C1.4 | **Governance grief**: Submit LIP, accumulate stake, withdraw LIP, re-submit → circumvent cooldowns? | Governance × Staking | **NOVEL** — Lifecycle reset attack |
| C1.5 | **Scheduler capability bypass**: Session key schedules execution → session expires → scheduled execution runs with full authority | DMSS × Identity × Auth | **NOVEL** — Authority escalation through time |

**PP Insight (Creativity/Integration)**: C1.5 is the most dangerous cross-subsystem attack vector identified. It crosses three subsystem boundaries (scheduling, identity, auth) and exploits a temporal assumption: that scheduled executions inherit the caller's authority at schedule-time, not at execution-time. This is documented in the existing strategy as Insight #15 but never tested.

### C2: State Corruption Scenarios (4 tests)

| ID | Test | Scenario | Expected Behavior |
|----|------|----------|-------------------|
| C2.1 | **SMT root divergence**: Two nodes process same block, one has corrupted storage → roots diverge → network partition | Graceful fork detection |
| C2.2 | **Genesis replay**: Connect with modified genesis (different validator pubkey) | Chain ID mismatch rejection |
| C2.3 | **Mid-block crash recovery**: SIGKILL during state transition → restart → clean rollback | Atomic write batches hold |
| C2.4 | **LevelDB corruption**: Truncate random bytes in DB file → restart → detect and recover or clean error | Not silent data loss |

### C3: Economic Game Theory (4 tests)

| ID | Test | Strategy | Expected Outcome |
|----|------|----------|-----------------|
| C3.1 | **Free-riding verifier**: Always submit 0.5 confidence → earn rewards without evaluating | Detection via agreement score penalty |
| C3.2 | **Sybil faucet**: Create 1000 accounts, claim 0.222 LGM each → extract 222 LGM | Rate limiting per IP + nonce/DID checks |
| C3.3 | **Stake-weight manipulation**: Stake 99% of supply → control all verification outcomes | Quorum + reputation weight dampens pure-stake control |
| C3.4 | **Claim spam**: Submit 1000 low-stake claims → overwhelm verification capacity | Gas pricing + mempool prioritization |

**PP Insight (Creativity/Vision)**: C3.1 (free-riding) represents the most realistic attack on PoT consensus. A lazy validator who always submits `confidence=0.5` collects verification rewards without doing any evaluation work. The protocol's defense is the `agreementScore` component in confidence aggregation (20% weight) — a free-rider's constant score will diverge from honest evaluators. Testing this requires multiple rounds with known honest evaluations.

### C4: Network-Level Attacks (4 tests)

| ID | Test | Attack | Expected Defense |
|----|------|--------|-----------------|
| C4.1 | **Eclipse attack**: All of a node's peers are malicious | Peer diversity requirements, honest peer discovery |
| C4.2 | **Block withholding**: Proposer mines block but delays broadcast | Timeout → next proposer selected |
| C4.3 | **Transaction censorship**: Validator refuses to include specific DID's txs | Other validators include from mempool |
| C4.4 | **GossipSub amplification**: Subscribe to all 12 topics, broadcast junk | Peer scoring → disconnect |

### C5: Boundary Conditions (3 tests)

| ID | Test | Boundary | Why It Matters |
|----|------|----------|---------------|
| C5.1 | **Minimum stake claim**: Submit with exactly `minClaimStake` (0.011 LGM) | Boundary acceptance/rejection |
| C5.2 | **Zero balance transfer**: Transfer 0 LGM between accounts | Should this succeed or fail? |
| C5.3 | **Max validators in round**: Create verification round requiring more validators than exist | Graceful degradation |

---

## Dimensional Interaction Map

> When dimensions overlap, the most important tests emerge.

```
         TRUTH
          /|\
         / | \
        /  |  \
       /   |   \
BEAUTY ----+---- JUSTICE
       \   |   /
        \  |  /
         \ | /
          \|/
     UNDERSTANDING
          |
      CREATIVITY
```

### Critical Intersections

| Intersection | What It Tests | Key Scenario |
|--------------|--------------|--------------|
| **Truth × Justice** | Rewards computed correctly | Verification reward = exactly 1 LGM / participants |
| **Truth × Understanding** | Lifecycle transitions deterministic | Claim → fact creates exactly 1 fact with correct confidence |
| **Beauty × Understanding** | API covers all lifecycle stages | Can query claim/fact/dispute at every stage |
| **Justice × Creativity** | Economic attacks fail | Free-riding detectable, Sybil unprofitable |
| **Understanding × Creativity** | Dark paths have defenses | Unilateral channel close has challenge period |
| **Truth × Beauty** | API returns accurate data | `lgm_getBalance` matches actual SMT value |

---

## Test Priority Matrix

| Priority | Tests | Rationale |
|----------|-------|-----------|
| **P0 — Must Run** | T1.1, T2.6, T3.1, T3.4, J4.1 | Foundational correctness — if these fail, nothing works |
| **P1 — Should Run** | U1.5, J1.2, J2.3, C3.1 | Critical dark paths and economic security |
| **P2 — Run Weekly** | B1.4-B1.8, U2.4, U3.3, U5.1-U5.4 | Interface completeness and lifecycle coverage |
| **P3 — Run Monthly** | C1.1-C1.5, C2.1-C2.4, C4.1-C4.4 | Cross-subsystem attacks and network resilience |

---

## Implementation Roadmap

### Phase A: Automated Test Harness (Foundation)

Build `devnet/test/harness.ts` — a reusable framework that:
1. Starts a DevNet node (or cluster) programmatically
2. Provides helper functions for all RPC calls
3. Includes canonical crypto functions (imported from `core/crypto/`, NOT reimplemented)
4. Tracks balances before/after for economic assertions
5. Supports both single-node and multi-node configurations

**PP Gate**: Justice/Integrity — the harness must import canonical crypto, never reimplement it (DRY Crypto Anti-Pattern from PP_DEVNET_INSIGHTS.md Insight #5).

### Phase B: P0 Tests (Truth Foundation)

Implement the 5 P0 tests:
- T1.1: `sha512_256` is SHA-256 (the representation canary)
- T2.6: State root determinism
- T3.1: Commit-reveal hash match
- T3.4: Salt not nonce key name
- J4.1: Supply conservation invariant

### Phase C: P1 Tests (Economic Security)

Implement the 4 P1 tests:
- U1.5: Claim rejection flow
- J1.2: Verification reward distribution
- J2.3: Equivocation slashing
- C3.1: Free-riding detection

### Phase D: P2 Tests (Lifecycle Completeness)

Fill in the lifecycle dark paths:
- Governance full lifecycle (U4.2-U4.5)
- Vesting lifecycle (U5.1-U5.4)
- Dispute escalation (U2.4)
- Unilateral channel close (U3.3)

### Phase E: P3 Tests (Adversarial Resilience)

Cross-subsystem and network-level attacks:
- Epoch boundary race (C1.1)
- Scheduler capability bypass (C1.5)
- Eclipse and censorship attacks (C4.1, C4.3)

---

## Meta-Insight: The PP Testing Hierarchy

Each PP virtue catches a different class of bug:

| Virtue | Bug Class | Example | Cost if Missed |
|--------|-----------|---------|---------------|
| **TRUTH** | Computation errors | SHA-256 vs SHA-512 | All signatures fail |
| **UNDERSTANDING** | Missing flows | No claim rejection test | Users lose stake silently |
| **BEAUTY** | Bad interfaces | Inconsistent response formats | Every integration breaks |
| **JUSTICE** | Economic imbalances | Rewards not distributed | Validators leave |
| **CREATIVITY** | Novel attacks | Cross-subsystem exploits | Protocol compromised |

**The hierarchy is correct**: Truth bugs are caught first (they break everything), Understanding bugs next (they miss real scenarios), Beauty bugs are caught by developers integrating, Justice bugs emerge in production economics, and Creativity bugs are exploited by adversaries.

**Testing should follow the same order.** A protocol with incorrect hashing (Truth failure) can't meaningfully test reward distribution (Justice). Fix the foundation before decorating the building.

---

## Current Coverage Analysis

### What the 56 Existing Tests Cover

| Dimension | Tests Covered | Gap |
|-----------|---------------|-----|
| **D1: Truth** | T1.1* (via test-phase4 fix), T2.1-T2.5 (Phase 3), T3.1-T3.6 (Phase 4) | T4.1-T4.6 (block validation) |
| **D2: Understanding** | U1.1-U1.4 (claim lifecycle), U2.1-U2.3 (disputes), U3.1-U3.2 (channels) | U1.5-U1.6, U2.4-U2.5, U3.3-U3.4, U4.*, U5.* |
| **D3: Beauty** | B1.1-B1.3 (RPC basics), B2.1 implicit | B1.4-B1.8, B2.*, B3.* |
| **D4: Justice** | J3.1, J3.5, J5.1 | J1.*, J2.3-J2.5, J3.2-J3.4, J4.* |
| **D5: Creativity** | Phase 6 adversarial (basic) | C1.*, C2.*, C3.*, C4.* |

### Coverage by Dimension

```
Truth:        ████████████░░░░░░░░  60%  (17/28)
Understanding: ██████░░░░░░░░░░░░░░  33%  (8/24)
Beauty:        ████░░░░░░░░░░░░░░░░  17%  (3/18)
Justice:       ████░░░░░░░░░░░░░░░░  14%  (3/22)
Creativity:    ████░░░░░░░░░░░░░░░░  25%  (5/20)
─────────────────────────────────────────────
Overall:       ██████░░░░░░░░░░░░░░  32%  (36/112)
```

**Biggest gaps**: Justice (economics) and Beauty (interface) are the most under-tested dimensions. The protocol's Truth foundation is reasonably solid but its economic security and API quality remain largely unverified.

---

*Generated: 2026-02-12 | PP Framework v1.0 | All 30 gates + 5 Claude-specific enhancements applied*
*Supersedes: DEVNET_TESTING_STRATEGY.md (operational plan) — this document adds dimensional analysis*
*Companion to: PP_DEVNET_INSIGHTS.md (debugging insights from actual test failures)*
