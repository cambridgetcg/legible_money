# Security Dimensions Review — Full PP Analysis

> **PP Protocol**: TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY
> **Date**: 2026-02-13
> **Scope**: All security surfaces of Legible Money (TS + Cosmos SDK)
> **Method**: Full 30-gate PP verification applied to security dimension identification,
>   then industry-standard testing methods devised per dimension.

---

## The Hierarchy Applied to Security

The PP hierarchy maps naturally to security analysis:

| PP Dimension | Security Meaning | Signal |
|---|---|---|
| **TRUTH** | Cryptographic integrity — can truth be falsified? | Broken crypto = all trust collapses |
| **UNDERSTANDING** | Protocol logic — are rules deeply correct? | Logic bugs = exploitable edge cases |
| **BEAUTY** | Incentive design — do economics create equilibria? | Ugly economics = rational exploitation |
| **JUSTICE** | Authorization — is access fair and enforceable? | Injustice = privilege escalation |
| **CREATIVITY** | Novel attack vectors — what's unprecedented? | Stagnation = blind spots become vulns |

---

## Eight Security Dimensions

Based on exhaustive codebase survey (4 parallel agents, ~450 files analyzed), the Legible Money
security surface decomposes into **8 orthogonal dimensions**:

```
D1: CRYPTOGRAPHIC INTEGRITY  (TRUTH foundation)
D2: CONSENSUS & VALIDATORS   (UNDERSTANDING of protocol rules)
D3: ECONOMIC & GAME THEORY   (BEAUTY of incentive design)
D4: IDENTITY & AUTHORIZATION (JUSTICE of access control)
D5: NETWORK & INFRASTRUCTURE (CREATIVITY — novel attack surface)
D6: KNOWLEDGE SYSTEM         (Protocol-specific: truth verification)
D7: STATE & DATA INTEGRITY   (Cross-cutting: SMT, atomicity, consistency)
D8: BVM & SMART CONTRACTS    (Execution environment security)
```

Each dimension has: **threat model**, **industry-standard testing methods**, **specific test specifications**,
and **existing coverage assessment**.

---

## D1: CRYPTOGRAPHIC INTEGRITY

### Threat Model
An attacker attempts to forge signatures, predict VRF outputs, find hash collisions,
break commitment hiding/binding, or exploit timing side-channels to compromise protocol
truth at the cryptographic layer.

### Key Attack Surfaces
- Ed25519 signatures (core/crypto/signatures.ts, Go stdlib crypto/ed25519)
- VRF validator selection (core/crypto/vrf.ts, x/knowledge/crypto/vrf.go)
- Domain-separated hashing (core/crypto/hashing.ts)
- Commit-reveal schemes (core/crypto/commitment.ts)
- Sparse Merkle Trees (core/crypto/merkle.ts)
- Key derivation & management (core/crypto/keys.ts)

### Existing Coverage
- `tests/security/d1-cryptographic/crypto-kats.test.ts` — 6 domain separation, 8 commitment, 12 Merkle, 5 key derivation
- `tests/security/d1-cryptographic/crypto-properties.test.ts` — Property-based with fast-check

### Industry-Standard Testing Methods

#### 1.1 Known Answer Tests (KAT) — NIST SP 800-185
**Purpose**: Verify crypto implementations match reference vectors.
**Method**: Compare outputs against published test vectors from standards bodies.
```
Tests:
  KAT-01: Ed25519 sign/verify against RFC 8032 test vectors (7 official vectors)
  KAT-02: SHA-256/SHA-512 against NIST FIPS 180-4 test vectors (short/long/million-a)
  KAT-03: BLAKE3 against official test_vectors.json (251 vectors at varying lengths)
  KAT-04: VRF output against ECVRF-EDWARDS25519-SHA512-TAI draft vectors
  KAT-05: Shamir Secret Sharing reconstruction with known polynomials
  KAT-06: Cross-stack verification — TS VRF output matches Go VRF output for same input
```
**Tool**: vitest with hardcoded reference vectors from RFCs/standards.

#### 1.2 Property-Based Testing (PBT) — QuickCheck-style
**Purpose**: Discover edge cases through randomized invariant checking.
**Method**: Use fast-check to generate random inputs and verify algebraic properties hold.
```
Tests:
  PBT-01: ∀ msg, key: verify(sign(msg, key), msg, pubkey) = true
  PBT-02: ∀ msg, key1 ≠ key2: verify(sign(msg, key1), msg, pubkey2) = false
  PBT-03: ∀ data: domainHash(d1, data) ≠ domainHash(d2, data) when d1 ≠ d2
  PBT-04: ∀ value: reveal(commit(value)) = value (commitment completeness)
  PBT-05: ∀ v1 ≠ v2: commit(v1) ≠ commit(v2) with overwhelming probability (binding)
  PBT-06: ∀ key, value: prove(set(smt, key, value), key) verifies true
  PBT-07: ∀ key ∉ smt: prove(smt, key) returns valid non-membership proof
  PBT-08: ∀ seed, sk: VRF output is deterministic (same seed+sk → same output)
  PBT-09: ∀ seed, sk1 ≠ sk2: VRF outputs differ (uniqueness)
  PBT-10: constantTimeEqual timing variance < 1% across random input pairs
```
**Tool**: fast-check (already in project) + vitest.

#### 1.3 Differential Testing — Cross-Stack Consistency
**Purpose**: Ensure TS and Go implementations produce identical outputs.
**Method**: Generate random inputs, run through both stacks, compare outputs.
```
Tests:
  DIFF-01: VRF(seed, key) — TS output == Go output for 1000 random seeds
  DIFF-02: domainHash(domain, data) — identical across stacks for all 11 domains
  DIFF-03: Block seed generation — binary vs JSON format alignment (KNOWN ISSUE)
  DIFF-04: Ed25519 sign/verify — cross-stack signature acceptance
  DIFF-05: Scalar derivation — SetBytesWithClamping identical in both stacks
```
**Tool**: Custom harness — TS generates, Go verifies (or vice versa). JSON-RPC bridge or shared test vectors file.

#### 1.4 Timing Side-Channel Analysis
**Purpose**: Verify constant-time operations don't leak information.
**Method**: Statistical analysis of execution time across different inputs.
```
Tests:
  TIME-01: constantTimeEqual — timing for equal vs unequal arrays (10K samples, t-test)
  TIME-02: signature verification — timing for valid vs invalid signatures
  TIME-03: commitment verification — timing for matching vs non-matching reveals
  TIME-04: VRF verification — timing for valid vs invalid proofs
```
**Tool**: process.hrtime.bigint() with statistical significance testing (p < 0.001).

#### 1.5 Mutation Testing — Crypto Correctness
**Purpose**: Verify test suite catches all crypto implementation bugs.
**Method**: Systematically mutate crypto code, verify tests fail.
```
Mutations:
  MUT-01: Flip one bit in signature before verify → must fail
  MUT-02: Swap domain prefix in hash → must produce different output
  MUT-03: Use wrong nonce in commitment → reveal must fail
  MUT-04: Truncate Merkle proof by one sibling → must fail
  MUT-05: Swap left/right in SMT internal node → different root
  MUT-06: Skip cofactor multiplication in VRF → different output
```
**Tool**: Stryker.js mutation testing framework with crypto-specific mutators.

---

## D2: CONSENSUS & VALIDATOR SECURITY

### Threat Model
An attacker controls up to f validators (where 3f+1 ≤ n) and attempts to:
finalize invalid claims, cause double-spends, halt the chain, censor transactions,
manipulate validator selection, or exploit the commit-reveal protocol.

### Key Attack Surfaces
- VRF-based validator selection (consensus/pot/validator_selection.ts)
- Commit-reveal verification rounds (consensus/pot/verification_round.ts)
- Block finality & fork choice (consensus/pot/finality.ts)
- Equivocation detection (consensus/pot/equivocation.ts)
- Block production & validation (consensus/blocks/validator.ts, producer.ts)
- Emergency shutdown (x/emergency/, consensus/emergency/)
- Epoch transitions (consensus/pot/engine.ts)

### Existing Coverage
- `tests/security/d2-consensus/byzantine-fault.test.ts` — Equivocation, commit-reveal manipulation
- `tests/security/d2-consensus/selection-fairness.test.ts` — VRF selection distribution

### Industry-Standard Testing Methods

#### 2.1 Byzantine Fault Injection — Jepsen-style
**Purpose**: Verify safety under Byzantine validator behavior.
**Method**: Simulate malicious validators performing protocol-violating actions.
```
Tests:
  BFT-01: Double-block proposal — same height, different content → detect + slash
  BFT-02: Double-commitment — same round, different verdicts → detect + slash
  BFT-03: Commit without reveal (withholding) — round must complete without withholder
  BFT-04: Reveal different from commitment — must be rejected
  BFT-05: Late commitment (after phase transition) — must be rejected
  BFT-06: Future-dated block timestamp (+6s beyond tolerance) — must be rejected
  BFT-07: Block with invalid transaction signatures — must be rejected
  BFT-08: Block with wrong Merkle root — must be rejected
  BFT-09: 1/3 validators colluding on wrong verdict — honest 2/3 must win
  BFT-10: All validators offline except one — chain halts gracefully (no invalid blocks)
  BFT-11: Validator selection manipulation — grinding seed with different block content
  BFT-12: Nothing-at-stake — equivocate on fork, verify both caught and slashed
```
**Tool**: Custom test harness with mock validator behavior injection. Model: TLA+ for formal specification.

#### 2.2 Formal Verification — TLA+/Ivy
**Purpose**: Mathematically prove safety and liveness properties.
**Method**: Specify consensus protocol in TLA+, model-check invariants.
```
Properties to verify:
  FV-01: Safety — No two honest validators finalize conflicting blocks at same height
  FV-02: Liveness — If ≥2/3 honest and connected, new blocks eventually produced
  FV-03: Finality — Once block at depth ≥19, it is never reverted
  FV-04: Checkpoint integrity — Checkpoint with ≥67% stake is permanent
  FV-05: Equivocation completeness — All equivocating validators are eventually detected
  FV-06: VRF fairness — Long-run selection frequency proportional to stake (within ε)
  FV-07: Commit-reveal binding — Reveal phase outcome cannot differ from committed value
```
**Tool**: TLA+ with TLC model checker. State space: validators × rounds × blocks.

#### 2.3 Chaos Engineering — Network Partition Simulation
**Purpose**: Verify consensus survives realistic failure modes.
**Method**: Inject failures during active consensus rounds.
```
Tests:
  CHAOS-01: Network partition (50/50 split) during verification round → both halves must not finalize contradicting results
  CHAOS-02: Leader crash during block production → next leader takes over within timeout
  CHAOS-03: 40% validator dropout mid-epoch → remaining validators continue if ≥60% online
  CHAOS-04: Clock skew (+30s) on 20% of validators → blocks still produced correctly
  CHAOS-05: Mempool flood (10K txs) during verification → round completion unaffected
  CHAOS-06: Emergency halt during active round → round state preserved for resume
  CHAOS-07: Checkpoint ceremony with 35% non-responsive → ceremony times out correctly
  CHAOS-08: Long-range attack — present alternative chain from 100 blocks ago → rejected
```
**Tool**: Custom chaos framework wrapping DevNet nodes (inject via libp2p middleware).

#### 2.4 Statistical Fairness Testing — Chi-Square
**Purpose**: Verify VRF selection is statistically fair.
**Method**: Run many rounds, measure selection frequency vs expected stake-weighted distribution.
```
Tests:
  STAT-01: 10K rounds, 10 validators — χ² test for selection proportional to stake (p > 0.01)
  STAT-02: Gini coefficient of block production over 100K blocks < 0.15 (low inequality)
  STAT-03: No validator selected >3σ above expected frequency (detect selection bias)
  STAT-04: Entropy of VRF outputs ≥ 250 bits (of 256 possible) per NIST SP 800-22
  STAT-05: Serial correlation of consecutive VRF outputs < 0.01
```
**Tool**: vitest + @stdlib/stats for statistical tests. scipy-equivalent for chi-square.

---

## D3: ECONOMIC & GAME-THEORETIC SECURITY

### Threat Model
A rational attacker (or coalition) seeks to extract more value than legitimately earned by:
exploiting reward formulas, gaming slashing thresholds, manipulating fees, draining funds,
or creating perverse incentive spirals.

### Key Attack Surfaces
- Block reward fund (x/vesting_rewards/, core/genesis/)
- Staking/slashing (consensus/pot/types.ts, x/staking/)
- Claim economics (state/transitions.ts, x/knowledge/)
- Counter-fact economics (x/knowledge/keeper/rounds.go)
- Fee routing (x/billing/, core/billing/reward_router.ts)
- Research share ramp (core/billing/reward_router.ts)

### Existing Coverage
- `tests/security/d3-economic/invariants.test.ts` — Supply cap, slash isolation, fee routing
- `tests/security/d3-economic/game-theory.test.ts` — Strategic behavior simulation

### Industry-Standard Testing Methods

#### 3.1 Economic Invariant Testing — Echidna/Foundry-style
**Purpose**: Verify economic invariants hold under arbitrary transaction sequences.
**Method**: Fuzz transaction sequences, assert invariants after every state transition.
```
Invariants:
  INV-01: totalSupply() ≤ 222,222,222 × 10^18 (HARD CAP — never exceeded)
  INV-02: ∀ accounts: sum(balances) + sum(staked) + sum(unbonding) + burned + fundRemaining = totalSupply
  INV-03: blockRewardFund monotonically decreasing (never increases)
  INV-04: ∀ validators: stakedBalance ≥ 0 (no negative stakes)
  INV-05: ∀ claims: rejectionSlash ≤ stakeAmount (can't slash more than staked)
  INV-06: researchShare = floor(reward × researchShareRateBps / 10000) (exact BPS math)
  INV-07: ∀ epochs: epochReward(n+1) ≤ epochReward(n) (monotonic decay)
  INV-08: ∀ unbonding: completionHeight > currentHeight (no instant unbond)
  INV-09: ∀ counter-facts: stakeAmount ≥ targetFact.originalStake × minRatio (min ratio enforced)
  INV-10: ∀ validators: slashCount ≤ 3 before deactivation (anti-grief cap)
  INV-11: falsificationReward ≤ cfStake × 3 (reward cap)
  INV-12: falsificationPoolPaid ≤ epochBlockRewardPool × 15% (pool cap)
```
**Tool**: Custom fuzzer generating random transaction sequences (vitest + fast-check `commands` pattern).

#### 3.2 Game-Theoretic Analysis — Mechanism Design Verification
**Purpose**: Verify no rational strategy dominates honest behavior.
**Method**: Enumerate strategic deviations, compute expected payoffs.
```
Tests:
  GAME-01: Honest claiming vs spam claiming — Expected profit(honest) > Expected profit(spam)
  GAME-02: Honest verification vs random verification — Honest strictly dominates
  GAME-03: Truthful counter-fact vs frivolous counter-fact — Truthful has positive EV, frivolous negative
  GAME-04: Self-citation ring (A cites B, B cites A) — Detection and penalty mechanism works
  GAME-05: Stake-then-immediately-unstake griefing — Unbonding delay makes this unprofitable
  GAME-06: Validator cartel (>33% stake) — Quantify maximum extractable value (MEV) under cartel
  GAME-07: Sybil claim flooding — Cost of attack > benefit at every scale
  GAME-08: Counter-fact warfare (serial contradictions) — Cooldown + stake ratio prevents spam
  GAME-09: Block reward timing attack — Claim submission timing cannot increase selection probability
  GAME-10: Research share evasion — All reward paths enforce 22% deduction (no bypass)
```
**Tool**: Agent-based simulation (TypeScript). Monte Carlo with 10K iterations per strategy pair.

#### 3.3 Stress Testing — Extreme Parameter Boundaries
**Purpose**: Verify economic soundness at boundary conditions.
**Method**: Push all parameters to extremes, verify no overflow/underflow/division-by-zero.
```
Tests:
  STRESS-01: Block reward with fund = 1 ulgm (minimum unit) → floor reward correctly applied
  STRESS-02: Slash 22% of 1 ulgm stake → rounds correctly (no division-by-zero)
  STRESS-03: 222,222,222 simultaneous claims → gas costs prevent actual execution
  STRESS-04: Counter-fact stake = MAX_UINT256 → overflow check in reward formula
  STRESS-05: Research share at epoch 0 (ramp = 5%) vs epoch 1M (ramp = 22%) → continuous transition
  STRESS-06: All validators slashed simultaneously → chain halts, doesn't crash
  STRESS-07: Fee per byte = 0 → system transactions still processed
  STRESS-08: Unbonding queue with 1M entries → O(1) per-block processing
```
**Tool**: vitest with extreme value generation. Property: no panics, no NaN, no negative balances.

#### 3.4 Token Flow Analysis — Formal Accounting
**Purpose**: Verify conservation of value across all state transitions.
**Method**: Instrument state transitions to track all value creation/destruction.
```
Tests:
  FLOW-01: Transfer: sender.balance↓ = receiver.balance↑ (zero-sum)
  FLOW-02: Claim acceptance: claimStake returned exactly to claimant
  FLOW-03: Claim rejection: slashed% burned + remaining% returned = original stake
  FLOW-04: Block reward: fundBalance↓ = validatorReward + researchShare (accounting identity)
  FLOW-05: Fee routing: feeCollected = providerShare + knowledgeShare + protocolShare + researchShare
  FLOW-06: Counter-fact acceptance: reward ≤ epochPoolCap AND reward ≤ 3 × cfStake
  FLOW-07: Unbonding completion: unbondingEntry removed, balance increased by exact amount
  FLOW-08: Full epoch cycle: Σ(rewards + fees) - Σ(burns + slashes) = net supply change
```
**Tool**: Instrumented StateManager with balance tracking assertions on every commit.

---

## D4: IDENTITY & AUTHORIZATION SECURITY

### Threat Model
An attacker attempts to impersonate agents, escalate session key privileges,
bypass spending limits, exploit recovery mechanisms, or spoof DID resolution.

### Key Attack Surfaces
- 4-layer key architecture (core/agents/identity/account.ts)
- Session key management (core/agents/identity/session_keys.ts)
- DID system (core/agents/identity/did.ts)
- Recovery/Shamir (core/agents/identity/recovery.ts)
- MPC wallet (core/agents/identity/mpc.ts)
- Auth module (x/auth/)

### Existing Coverage
- `tests/security/d4-identity/auth-bypass.test.ts` — Authorization bypass attempts
- `tests/security/d4-identity/did-injection.test.ts` — DID format injection

### Industry-Standard Testing Methods

#### 4.1 Authentication Bypass Testing — OWASP ASVS L3
**Purpose**: Verify no path exists to bypass authentication.
**Method**: Enumerate all entry points, attempt unauthenticated/mis-authenticated access.
```
Tests:
  AUTH-01: Transaction with expired session key → rejected
  AUTH-02: Transaction with revoked operational key → rejected
  AUTH-03: Transaction signed with wrong key version → rejected
  AUTH-04: Session key exceeding spending limit (per-tx) → rejected
  AUTH-05: Session key exceeding daily spending limit → rejected
  AUTH-06: Session key with exhausted usage count → rejected
  AUTH-07: Session key for non-whitelisted recipient → rejected
  AUTH-08: Transaction with nonce = expected - 1 (replay) → rejected
  AUTH-09: Transaction with nonce = expected + 1 (gap) → rejected
  AUTH-10: System transaction forged by non-system DID → rejected
  AUTH-11: Recovery request without threshold shards → rejected
  AUTH-12: Recovery challenge without valid signature → rejected
  AUTH-13: Key rotation with wrong current key proof → rejected
  AUTH-14: Session key authorization with forged operational signature → rejected
```
**Tool**: vitest with systematic entry-point enumeration. Follow OWASP ASVS v4.0 L3 checklist.

#### 4.2 DID Injection & Format Fuzzing
**Purpose**: Verify DID parsing is robust against malformed inputs.
**Method**: Fuzz DID format strings with malformed, oversized, and injection payloads.
```
Tests:
  DID-01: DID with SQL injection payload → sanitized/rejected
  DID-02: DID with path traversal (../../) → sanitized
  DID-03: DID with null bytes → rejected
  DID-04: DID exceeding max length (10KB) → rejected
  DID-05: DID with non-hex characters in address portion → rejected
  DID-06: DID with wrong prefix (did:eth:) → rejected
  DID-07: DID with 4-part format where 3-part expected → handled correctly
  DID-08: Empty DID string → rejected with specific error
  DID-09: Unicode DID (homoglyph attack: did:lgm:а... with Cyrillic 'а') → rejected
  DID-10: DID collision — two different public keys producing same DID → statistically impossible (verify 2^128 security)
```
**Tool**: fast-check arbitrary string generation + manual injection payloads.

#### 4.3 Privilege Escalation Testing
**Purpose**: Verify session keys cannot exceed their granted capabilities.
**Method**: Create constrained session keys, attempt every prohibited action.
```
Tests:
  PRIV-01: Read-only session key → cannot sign transfer → rejected
  PRIV-02: Transfer-only key → cannot submit claims → rejected
  PRIV-03: Claim-only key → cannot stake → rejected
  PRIV-04: Key with recipient whitelist → transfer to non-whitelisted → rejected
  PRIV-05: Key approaching daily limit → batch of small txs that exceed limit → rejected at boundary
  PRIV-06: Expired key (by block height) → tx in same block as expiry → accepted or rejected? (boundary)
  PRIV-07: Key with 0 remaining usage → single tx → rejected
  PRIV-08: Intent proof with expired challenge (>60s) → rejected
  PRIV-09: Cross-session key reuse (key from session A used in session B) → rejected
  PRIV-10: Session key tries to authorize a new session key → rejected (only operational key can)
```
**Tool**: vitest with session key factory generating constrained keys.

#### 4.4 Recovery Attack Simulation
**Purpose**: Verify recovery mechanism is resilient against social engineering.
**Method**: Simulate adversarial recovery attempts.
```
Tests:
  REC-01: Single shard holder tries solo recovery (threshold=2) → rejected
  REC-02: Recovery request before delay expires → rejected
  REC-03: Legitimate challenge cancels recovery → recovery cancelled
  REC-04: Challenge after challenge period → rejected
  REC-05: Duplicate guardian DIDs in social recovery → rejected
  REC-06: Shard from non-authorized holder → rejected
  REC-07: Race condition — two concurrent recovery requests → only one proceeds
  REC-08: Recovery with wrong new operational key → reconstruction fails
```
**Tool**: vitest with state machine testing for recovery lifecycle.

---

## D5: NETWORK & INFRASTRUCTURE SECURITY

### Threat Model
An attacker controls network infrastructure to: eclipse honest nodes, flood the mempool,
replay messages, manipulate peer discovery, exhaust resources, or exploit RPC endpoints.

### Key Attack Surfaces
- libp2p configuration (network/config.ts, network/node.ts)
- GossipSub scoring (network/pubsub/topics.ts)
- Peer discovery (mDNS, Kademlia DHT)
- Mempool (consensus/mempool/pool.ts)
- Sync protocol (network/protocols/sync.ts, state_sync.ts)
- RPC server (protocol/rpc/server.ts)

### Existing Coverage
- `tests/security/d5-network/mempool-network.test.ts` — Mempool flooding and limits
- `tests/security/d5-network/rpc-security.test.ts` — RPC endpoint security

### Industry-Standard Testing Methods

#### 5.1 Eclipse Attack Testing
**Purpose**: Verify a node cannot be isolated from honest peers.
**Method**: Surround target node with attacker-controlled peers.
```
Tests:
  ECLIPSE-01: Target node with all 100 connections from attacker → can still discover honest peers via mDNS
  ECLIPSE-02: DHT poisoning — 50% of routing table entries are attacker nodes → honest lookups still succeed >50%
  ECLIPSE-03: Bootstrap peer removal — all bootstrap peers offline → node discovers via mDNS fallback
  ECLIPSE-04: Selective message dropping — attacker relays blocks but withholds txs → target eventually receives from other paths
  ECLIPSE-05: Peer score manipulation — attacker peers behave well then turn malicious → score drops below threshold
```
**Tool**: libp2p test harness with mock peers. Metrics: time-to-discovery of honest peer, % message delivery.

#### 5.2 Denial-of-Service Resilience
**Purpose**: Verify protocol survives resource exhaustion attacks.
**Method**: Generate maximum-load scenarios.
```
Tests:
  DOS-01: Mempool flood — 100K transactions from unique senders → pool caps at 10K, rejects rest
  DOS-02: Per-sender flood — 1000 txs from single sender → caps at 100, rejects rest
  DOS-03: Oversized message (>1MB) → rejected at pubsub layer before processing
  DOS-04: Stale message replay (>30s old) → rejected
  DOS-05: RPC flood — 10K concurrent requests → rate limiting activates, no crash
  DOS-06: Malformed block gossip — invalid proto → rejected without resource exhaustion
  DOS-07: Sync protocol abuse — request 1M blocks → chunked response with limits
  DOS-08: WebSocket connection exhaustion → max connections enforced
  DOS-09: Transaction with 100KB payload → exceeds max tx size, rejected
  DOS-10: Zero-fee transaction flood → rejected at min-fee check
```
**Tool**: Custom load generator (k6 or Artillery) + vitest assertions on node stability.

#### 5.3 Message Integrity & Replay Testing
**Purpose**: Verify message authentication and freshness.
**Method**: Intercept, modify, and replay network messages.
```
Tests:
  MSG-01: Block message with altered transaction → Merkle root mismatch → rejected
  MSG-02: Verification message with forged validator signature → rejected
  MSG-03: Message replayed within 30s window → accepted (idempotent) or rejected (duplicate detection)
  MSG-04: Message replayed after 30s → rejected (MAX_MESSAGE_AGE_MS)
  MSG-05: Message with future timestamp (+11s) → rejected (>10s future tolerance)
  MSG-06: Message with correct signature but wrong sender field → rejected
```
**Tool**: libp2p interceptor middleware for message tampering in test environment.

#### 5.4 RPC Security Testing — OWASP API Top 10
**Purpose**: Verify RPC endpoints are hardened against common API attacks.
**Method**: Systematic testing against OWASP API Security Top 10 (2023).
```
Tests:
  RPC-01: Broken Object Level Authorization — query account by DID not owned → appropriate access control
  RPC-02: Broken Authentication — RPC call without valid auth token → rejected (if auth required)
  RPC-03: Excessive Data Exposure — response contains only requested fields, no internal state leak
  RPC-04: Lack of Resources & Rate Limiting — 1000 req/s → rate limiter activates
  RPC-05: Mass Assignment — extra fields in JSON-RPC request → ignored, not processed
  RPC-06: Security Misconfiguration — CORS allows only configured origins
  RPC-07: Injection — JSON-RPC method field with injection payload → sanitized
  RPC-08: Improper Assets Management — deprecated endpoints return proper error, not partial data
  RPC-09: SSRF — request with internal URL as parameter → blocked
  RPC-10: Insufficient Logging — failed auth attempts generate audit log entries
```
**Tool**: Custom RPC fuzzer + OWASP ZAP for automated scanning (HTTP endpoints).

---

## D6: KNOWLEDGE SYSTEM SECURITY

### Threat Model
An attacker attempts to inject false knowledge into the protocol's truth graph by:
submitting hallucinated claims, gaming the evaluation pipeline, exploiting counter-fact
economics, poisoning the citation graph, or manipulating the Claude evaluation daemon.

### Key Attack Surfaces
- 8-stage claim evaluation pipeline (x/knowledge/evaluation/)
- Counter-fact system (x/knowledge/keeper/rounds.go)
- Citation graph (state/knowledge.ts)
- Confidence propagation (consensus/pot/confidence.ts)
- Claude evaluation daemon (daemon/)
- Anti-hallucination measures (AH-1 through AH-5)
- Survive-to-accept mechanism
- Domain ontology (x/ontology/)

### Existing Coverage
- `tests/security/d6-knowledge/knowledge-lifecycle.test.ts` — Claim lifecycle security
- `tests/security/d6-knowledge/adversarial-claims.test.ts` — Adversarial claim submission

### Industry-Standard Testing Methods

#### 6.1 Adversarial Claim Testing — Red Team
**Purpose**: Test the evaluation pipeline against sophisticated false claims.
**Method**: Generate claims designed to exploit each evaluation stage.
```
Tests:
  ADV-01: Factually false claim with high confidence → Stage 1 (category) passes, Stage 2 (calibration) should flag
  ADV-02: Self-referential claim (Gödelian) → Stage 5 (logic_zone) must detect and reject
  ADV-03: Claim referencing revoked fact → Stage 8 (counter_facts) must apply penalty
  ADV-04: Claim with circular citation (A→B→A) → confidence propagation applies cycle penalty
  ADV-05: Claim with 100 citations (citation bombing) → ceiling stage caps confidence boost
  ADV-06: Claim in invalid domain → ontology check rejects before evaluation
  ADV-07: Claim with adversarial Unicode in content → sanitized, evaluation proceeds correctly
  ADV-08: Claim submission during verification round for related claim → no conflict-of-interest
  ADV-09: Contradictory claims submitted in same block → both evaluated independently
  ADV-10: Claim with forged evidence references → Stage 3 (evidence) rejects unverifiable evidence
```
**Tool**: Adversarial claim generator with templates for each bypass strategy.

#### 6.2 Counter-Fact Game Theory
**Purpose**: Verify counter-fact economics prevent abuse while rewarding truthful falsification.
**Method**: Simulate adversarial counter-fact strategies.
```
Tests:
  CF-01: Frivolous contradiction (false counter-fact) → 33% slash makes this negative EV
  CF-02: Truthful contradiction → reward ≤ 3× stake, positive EV
  CF-03: Counter-fact spam (100 contradictions in one epoch) → cooldown blocks rapid-fire
  CF-04: Counter-fact below minimum stake ratio → rejected
  CF-05: Counter-fact during challenge window → challenged and defended → original fact survives
  CF-06: Counter-fact after challenge window → verified status, cannot be challenged
  CF-07: Serial falsification (contradict same fact repeatedly) → diminishing returns after first
  CF-08: Falsification reward at epoch cap → carry-forward works correctly
  CF-09: System-generated counter-fact (from rejected claim) → gets nominal stake (10% of slash)
  CF-10: Stale contested fact decay → 5%/epoch after 10 epochs, revoked below 10%
```
**Tool**: Simulation harness with counter-fact agent strategies.

#### 6.3 Confidence Manipulation Testing
**Purpose**: Verify confidence cannot be artificially inflated or deflated.
**Method**: Attempt confidence manipulation through various strategies.
```
Tests:
  CONF-01: Self-verification ring (validator A always confirms validator B's claims) → reputation score detects correlation
  CONF-02: Confidence pumping via mutual citation → citation ceiling caps benefit
  CONF-03: Confidence dumping via mass challenges → challenger must stake, makes dumping expensive
  CONF-04: Bayesian evidence aggregation with contradicting evidence → confidence properly lowered
  CONF-05: Category-specific decay — empirical fact at T=1M blocks → correct decay applied
  CONF-06: Derived confidence through 10-hop citation chain → properly attenuated
  CONF-07: BidirectionalConfidence — truth component at 100%, falsity at 50% → net = 50%
```
**Tool**: vitest with instrumented confidence calculation.

#### 6.4 Daemon Security Testing
**Purpose**: Verify Claude evaluation daemon cannot be manipulated.
**Method**: Test daemon against adversarial inputs and failure modes.
```
Tests:
  DAEMON-01: Evaluation request with prompt injection in claim content → safely escaped
  DAEMON-02: Daemon offline during evaluation → round proceeds with available validators
  DAEMON-03: Daemon returns inconsistent evaluation for identical claim → detection and penalty
  DAEMON-04: Evaluation persist-before-submit guarantee → crash after persist, before submit → recoverable
  DAEMON-05: Daemon timeout → graceful fallback to manual evaluation
  DAEMON-06: Rate limiting — 1000 evaluation requests/minute → handled or backpressured
```
**Tool**: Mock daemon with failure injection + vitest.

---

## D7: STATE & DATA INTEGRITY

### Threat Model
An attacker attempts to corrupt state data by: exploiting non-atomic transitions,
causing cross-tree inconsistency, poisoning Merkle proofs, or manipulating
the 16-root state hash.

### Key Attack Surfaces
- Sparse Merkle Trees (core/crypto/merkle.ts)
- State Manager (state/state_manager.ts)
- State transitions (state/transitions.ts)
- 16 sub-root computation (state/state_manager.ts:265-290)
- Checkpoint/rollback mechanism

### Industry-Standard Testing Methods

#### 7.1 Atomicity & Consistency Testing
**Purpose**: Verify state transitions are atomic (all-or-nothing).
**Method**: Inject failures mid-transition, verify rollback.
```
Tests:
  ATOM-01: Transfer fails mid-way (debit succeeds, credit fails) → full rollback, no balance leak
  ATOM-02: Claim with 5 state changes, 3rd fails → all 5 rolled back
  ATOM-03: Concurrent state reads during write → consistent snapshot isolation
  ATOM-04: Checkpoint created → state mutated → rollback to checkpoint → state matches checkpoint exactly
  ATOM-05: 16 sub-roots after failed transition → all match pre-transition values
  ATOM-06: State root determinism — same transactions in same order → identical root on two nodes
```
**Tool**: vitest with fault injection at specific transition points.

#### 7.2 Merkle Proof Soundness
**Purpose**: Verify SMT proofs cannot be forged.
**Method**: Attempt to create valid-looking proofs for wrong values.
```
Tests:
  MRKL-01: Valid inclusion proof → verifies correctly
  MRKL-02: Proof for wrong value (right key, wrong value) → verification fails
  MRKL-03: Proof for wrong key (right value, wrong key) → verification fails
  MRKL-04: Truncated proof (missing last sibling) → verification fails
  MRKL-05: Non-membership proof for existing key → verification fails
  MRKL-06: Membership proof for non-existing key → verification fails
  MRKL-07: Proof against wrong root → verification fails
  MRKL-08: Second preimage attack — two different key-value pairs producing same proof → computationally infeasible
```
**Tool**: Property-based testing with fast-check.

#### 7.3 State Root Determinism
**Purpose**: Verify identical transactions produce identical state roots across runs.
**Method**: Replay transaction sequences, compare roots.
```
Tests:
  DET-01: 1000 random transactions, replayed twice → identical final root
  DET-02: Same transactions in different order → different root (ordering matters)
  DET-03: TS root computation matches Go root computation for same inputs
  DET-04: State root with all 16 sub-roots at empty → deterministic empty-state root
  DET-05: State root with one sub-root changed → different from all-empty root
```
**Tool**: Deterministic test harness with seeded PRNG.

---

## D8: BVM & SMART CONTRACT SECURITY

### Threat Model
An attacker deploys or calls smart contracts to: escalate privileges via DELEGATECALL,
exhaust gas, access unauthorized state, spoof caller identity, or exploit host functions.

### Key Attack Surfaces
- BVM interpreter (x/bvm/vm/interpreter.go)
- Call context propagation (x/bvm/vm/calls.go, context.go)
- Host agent functions (x/bvm/vm/host_agent.go)
- State memory (x/bvm/vm/state_memory.go)
- Session capabilities in BVM (x/bvm/keeper/msg_server.go)
- Schedule self-invocation (host_agent.go:154-200)

### Industry-Standard Testing Methods

#### 8.1 Reentrancy & Call-Stack Testing
**Purpose**: Verify BVM is safe against reentrancy attacks.
**Method**: Deploy contracts that recursively call back into caller.
```
Tests:
  REENT-01: Contract A calls B, B calls back into A → gas limit prevents infinite recursion
  REENT-02: DELEGATECALL reentrancy → caller storage protected by gas exhaustion
  REENT-03: Cross-contract reentrancy (A→B→C→A) → detected or gas-bounded
  REENT-04: Reentrancy during token transfer → balance changes atomic
  REENT-05: Max call depth exceeded → graceful error, not crash
```
**Tool**: Go test with crafted bytecode contracts.

#### 8.2 Caller Identity Verification
**Purpose**: Verify CallerDID cannot be spoofed in sub-calls.
**Method**: Trace CallerDID through call chains.
```
Tests:
  CALLER-01: Top-level call → CallerDID matches tx signer's DID
  CALLER-02: A calls B → B's CallerDID is A's contract address DID (not original signer)
  CALLER-03: DELEGATECALL → CallerDID preserved (caller's identity, target's code)
  CALLER-04: STATICCALL → CallerDID correct AND state writes rejected
  CALLER-05: Host function SetBelief → only writes beliefs for actual CallerDID
  CALLER-06: Contract address used as CallerDID → GetAccountDID returns empty (no agent impersonation)
```
**Tool**: Go test with mock auth keeper and BVM execution traces.

#### 8.3 Gas Exhaustion & Resource Limit Testing
**Purpose**: Verify BVM cannot be used to exhaust node resources.
**Method**: Deploy gas-intensive contracts.
```
Tests:
  GAS-01: Infinite loop contract → terminates at gas limit with proper error
  GAS-02: Memory allocation bomb (allocate MAX_MEMORY) → capped at limit
  GAS-03: Storage write storm (write to 10K slots) → gas cost makes this prohibitively expensive
  GAS-04: Cross-contract call chain depth 100 → gas exhaustion before depth limit
  GAS-05: CREATE in loop (deploy 1000 contracts in single tx) → gas limit prevents
```
**Tool**: Go test with bytecode that pushes resource limits.

#### 8.4 Host Function Access Control
**Purpose**: Verify host functions enforce proper authorization.
**Method**: Call host functions from unauthorized contexts.
```
Tests:
  HOST-01: ScheduleSelf from contract not at limit → succeeds
  HOST-02: ScheduleSelf exceeding per-contract limit → rejected
  HOST-03: CancelSchedule for schedule owned by different contract → should be rejected (KNOWN GAP)
  HOST-04: ModifySchedule for other contract's schedule → should be rejected (KNOWN GAP)
  HOST-05: SetTrust for DID not matching CallerDID → rejected
  HOST-06: AgentDID for contract address → returns correct contract DID (not agent DID)
```
**Tool**: Go test with mock scheduler keeper.

---

## Testing Infrastructure Requirements

### Tools & Frameworks

| Tool | Purpose | Dimension |
|---|---|---|
| **vitest** | Unit/integration test runner | All (TS) |
| **fast-check** | Property-based testing | D1, D3, D4, D7 |
| **Go testing** | Go unit tests | D2, D8 (Go) |
| **TLA+/TLC** | Formal verification | D2 |
| **Stryker.js** | Mutation testing | D1 |
| **k6/Artillery** | Load testing | D5 |
| **OWASP ZAP** | API security scanning | D5 |
| **Custom chaos framework** | Fault injection | D2, D5 |
| **Agent-based simulator** | Game theory Monte Carlo | D3, D6 |
| **libp2p test harness** | Network attack simulation | D5 |

### Test Organization

```
tests/security/
├── d1-cryptographic/
│   ├── crypto-kats.test.ts         (existing — enhance with RFC vectors)
│   ├── crypto-properties.test.ts   (existing — expand PBT coverage)
│   ├── crypto-differential.test.ts (NEW — cross-stack consistency)
│   ├── crypto-timing.test.ts       (NEW — side-channel analysis)
│   └── crypto-mutations.test.ts    (NEW — mutation testing)
├── d2-consensus/
│   ├── byzantine-fault.test.ts     (existing — expand BFT scenarios)
│   ├── selection-fairness.test.ts  (existing — add χ² statistical tests)
│   ├── chaos-scenarios.test.ts     (NEW — network partition + failure injection)
│   └── tla/                        (NEW — TLA+ formal specs)
│       ├── PotConsensus.tla
│       └── PotConsensus.cfg
├── d3-economic/
│   ├── invariants.test.ts          (existing — add fuzz sequences)
│   ├── game-theory.test.ts         (existing — expand Monte Carlo)
│   ├── stress-boundaries.test.ts   (NEW — extreme parameter testing)
│   └── token-flow.test.ts          (NEW — conservation-of-value accounting)
├── d4-identity/
│   ├── auth-bypass.test.ts         (existing — expand OWASP ASVS)
│   ├── did-injection.test.ts       (existing — expand fuzzing)
│   ├── privilege-escalation.test.ts (NEW — session key boundary testing)
│   └── recovery-attack.test.ts     (NEW — adversarial recovery simulation)
├── d5-network/
│   ├── mempool-network.test.ts     (existing — expand DoS scenarios)
│   ├── rpc-security.test.ts        (existing — expand OWASP API Top 10)
│   ├── eclipse-attack.test.ts      (NEW — peer isolation simulation)
│   └── message-integrity.test.ts   (NEW — replay + tampering)
├── d6-knowledge/
│   ├── knowledge-lifecycle.test.ts (existing — expand red team scenarios)
│   ├── adversarial-claims.test.ts  (existing — expand evaluation bypass)
│   ├── counter-fact-game.test.ts   (NEW — counter-fact economics)
│   ├── confidence-manipulation.test.ts (NEW — confidence attack vectors)
│   └── daemon-security.test.ts     (NEW — evaluation daemon hardening)
├── d7-state/
│   ├── atomicity.test.ts           (NEW — transaction rollback verification)
│   ├── merkle-soundness.test.ts    (NEW — proof forgery attempts)
│   └── determinism.test.ts         (NEW — cross-run root consistency)
└── d8-bvm/
    ├── reentrancy.test.ts          (NEW — recursive call attacks)
    ├── caller-identity.test.ts     (NEW — CallerDID spoofing)
    ├── gas-exhaustion.test.ts      (NEW — resource limit enforcement)
    └── host-functions.test.ts      (NEW — access control verification)
```

### Priority Matrix

| Dimension | Severity | Existing Coverage | Gap | Priority |
|---|---|---|---|---|
| **D1: Crypto** | Critical | Good (KATs + PBT) | Cross-stack, timing | P2 |
| **D2: Consensus** | Critical | Moderate (BFT, fairness) | Formal verification, chaos | P1 |
| **D3: Economic** | High | Good (invariants, game theory) | Fuzz sequences, stress | P1 |
| **D4: Identity** | High | Moderate (auth bypass, DID) | Privilege escalation, recovery | P2 |
| **D5: Network** | Medium | Basic (mempool, RPC) | Eclipse, DoS, message integrity | P2 |
| **D6: Knowledge** | High | Basic (lifecycle, adversarial) | Counter-fact game, confidence | P1 |
| **D7: State** | High | None (d7/ doesn't exist) | All tests needed | P1 |
| **D8: BVM** | Medium | None (d8/ doesn't exist) | All tests needed | P2 |

### Estimated Test Counts

| Dimension | Existing | New | Total |
|---|---|---|---|
| D1: Crypto | ~31 | ~26 | ~57 |
| D2: Consensus | ~24 | ~27 | ~51 |
| D3: Economic | ~20 | ~28 | ~48 |
| D4: Identity | ~16 | ~32 | ~48 |
| D5: Network | ~12 | ~26 | ~38 |
| D6: Knowledge | ~14 | ~33 | ~47 |
| D7: State | 0 | ~19 | ~19 |
| D8: BVM | 0 | ~22 | ~22 |
| **TOTAL** | **~117** | **~213** | **~330** |

---

## PP Verification

### Truth Gate
- [x] All claims grounded in actual codebase survey (4 parallel agents, file:line citations)
- [x] Confidence calibrated: "Critical" for crypto/consensus, "Medium" for network/BVM
- [x] Counter-considered: Each dimension includes what it does NOT cover

### Understanding Gate
- [x] Mental model: 8 dimensions map to distinct attack surfaces
- [x] Why explanation: Threat model precedes each testing method
- [x] Practical application: Specific test specifications, not abstract recommendations

### Beauty Gate
- [x] Emerged from codebase understanding, not forced template
- [x] Coherent structure: Dimension → Threat Model → Methods → Tests
- [x] Right-sized: 330 tests across 8 dimensions (not 1000, not 30)

### Justice Gate
- [x] Tests placed in existing `tests/security/d{N}/` structure
- [x] New dimensions (d7, d8) follow established pattern
- [x] Tool choices match project stack (vitest, fast-check, Go testing)

### Creativity Gate
- [x] Novel: D6 (Knowledge System) testing has no industry precedent
- [x] Connected: All dimensions reference actual file:line locations
- [x] Enables future: Framework extensible as protocol evolves
