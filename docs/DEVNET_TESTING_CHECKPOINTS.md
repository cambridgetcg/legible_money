# DevNet Testing Checkpoints

> **Version**: 1.0.0
> **Generated**: 2026-02-06
> **Protocol**: Legible Money (LGM)
> **DevNet**: 3 validators, Docker Compose, LevelDB persistence
> **Prerequisite**: `.claude/memory/DEVNET_TESTING.md` (54 functional tests, 44 robustness tests)

---

## Purpose

This document defines **structured go/no-go checkpoints** for validating DevNet readiness before launch. Each checkpoint is a discrete, verifiable gate that must pass before proceeding to the next tier.

Unlike the functional test plan (DEVNET_TESTING.md) which validates individual features, checkpoints validate **system-level readiness** — the difference between "transfers work" and "the network is safe to operate."

---

## Checkpoint Tiers

```
TIER 0: Infrastructure    "Can the network exist?"
   │
   ▼
TIER 1: Consensus          "Can the network agree?"
   │
   ▼
TIER 2: Economics           "Can value flow?"
   │
   ▼
TIER 3: Knowledge           "Can truth emerge?"
   │
   ▼
TIER 4: Resilience          "Can the network survive?"
   │
   ▼
TIER 5: Operations          "Can operators run it?"
   │
   ▼
LAUNCH GATE                 "Go / No-Go decision"
```

Each tier depends on all previous tiers. A failure in Tier 0 blocks all subsequent tiers.

---

## Checkpoint Format

Each checkpoint follows this structure:

```
CP-{TIER}.{NUMBER}: {Name}
  Validates: {What system property this proves}
  Method:    {How to verify — command, RPC call, or observation}
  Pass:      {Exact pass criteria}
  Fail:      {What constitutes failure}
  Blocks:    {What cannot proceed if this fails}
  Priority:  P0 (launch blocker) | P1 (should fix) | P2 (nice to have)
```

---

## Tier 0: Infrastructure (6 checkpoints)

> **Gate Question**: Can the network physically exist and be reachable?
> **Entry**: Docker Compose file and genesis config exist
> **Signal**: All services respond to health probes

### CP-0.1: Container Initialization

- **Validates**: All Docker containers start without crash loops
- **Method**: `docker-compose up -d && sleep 10 && docker-compose ps`
- **Pass**: All 3 validators show `Up` status (not `Restarting` or `Exit`)
- **Fail**: Any container in `Restarting` or `Exit` state after 30s
- **Blocks**: All subsequent checkpoints
- **Priority**: P0

### CP-0.2: Health Endpoint Availability

- **Validates**: RPC servers are accepting HTTP connections
- **Method**:
  ```bash
  for port in 8545 8547 8549; do
    curl -sf http://localhost:$port/health | jq .status
  done
  ```
- **Pass**: All 3 return `"healthy"` with HTTP 200
- **Fail**: Any endpoint unreachable or non-200 after 30s
- **Blocks**: All RPC-dependent checkpoints (CP-0.3+)
- **Priority**: P0

### CP-0.3: Genesis State Loaded

- **Validates**: All nodes loaded identical genesis state
- **Method**:
  ```bash
  # Check faucet balance on all 3 validators
  for port in 8545 8547 8549; do
    curl -s -X POST http://localhost:$port \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"account_getBalance","params":[{"did":"did:lgm-devnet:faucet"}],"id":1}' \
      | jq .result.balance
  done
  ```
- **Pass**: All 3 return `"1000000000000000000000000000"` (1B * 10^18)
- **Fail**: Any node returns different balance or error
- **Blocks**: CP-2.x (economic checkpoints)
- **Priority**: P0

### CP-0.4: Genesis Axioms Present

- **Validates**: Knowledge tree initialized with foundational axioms
- **Method**:
  ```bash
  curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"knowledge_getFacts","params":[{"domain":"logic.foundational"}],"id":1}' \
    | jq '.result.facts | length'
  ```
- **Pass**: Returns `3` (Identity, Non-Contradiction, Excluded Middle) with confidence `1.0`
- **Fail**: Missing axioms or confidence != 1.0
- **Blocks**: CP-3.x (knowledge checkpoints)
- **Priority**: P0

### CP-0.5: P2P Peer Discovery

- **Validates**: Validators can discover each other via configured dial peers
- **Method**:
  ```bash
  # Wait 30s for mDNS/bootstrap, then check
  curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"network_health","params":[],"id":1}' \
    | jq .result.peerCount
  ```
- **Pass**: At least 1 peer connected on each validator (ideally 2)
- **Fail**: `peerCount: 0` on all validators after 60s
- **Blocks**: CP-1.x (consensus requires peer communication)
- **Priority**: P0
- **Note**: mDNS can be slow on localhost; bootstrap peers via DIAL_PEERS env var are the primary mechanism

### CP-0.6: LevelDB Persistence Initialized

- **Validates**: State storage backend is operational
- **Method**: `docker exec lgm-validator-1 ls /data/state/`
- **Pass**: LevelDB files exist (`CURRENT`, `LOCK`, `*.ldb` or `*.log`)
- **Fail**: Empty directory or missing LevelDB markers
- **Blocks**: CP-4.4 (crash recovery)
- **Priority**: P0

---

## Tier 1: Consensus (7 checkpoints)

> **Gate Question**: Can validators agree on the state of the world?
> **Entry**: Tier 0 passes (infrastructure healthy)
> **Signal**: Block height incrementing consistently across all nodes

### CP-1.1: Block Production Active

- **Validates**: At least one validator is producing blocks
- **Method**:
  ```bash
  # Sample block height at t=0 and t=10s
  H0=$(curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"network_health","params":[],"id":1}' | jq .result.latestBlock)
  sleep 10
  H1=$(curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"network_health","params":[],"id":1}' | jq .result.latestBlock)
  echo "Blocks produced: $((H1 - H0))"
  ```
- **Pass**: `H1 > H0` (blocks advancing)
- **Fail**: `H1 == H0` after 10s (chain stalled)
- **Blocks**: All Tier 1+ checkpoints
- **Priority**: P0

### CP-1.2: Block Height Consistency

- **Validates**: All validators converge to the same chain tip
- **Method**: Compare `latestBlock` across all 3 validators
- **Pass**: Heights within 2 blocks of each other
- **Fail**: Height difference > 5 blocks (indicates fork or sync failure)
- **Blocks**: CP-1.5 (state consistency)
- **Priority**: P0

### CP-1.3: Block Timing

- **Validates**: Block production rate matches protocol parameters
- **Method**: Measure 20 consecutive block intervals
- **Pass**: Average interval between 500ms and 5000ms (with 3 validators, expected ~840ms)
- **Fail**: Average > 10s or < 100ms (runaway production)
- **Blocks**: None (informational but critical for economics)
- **Priority**: P1
- **Reference**: FINDING-006 — 3 validators produce ~1070ms average, within tolerance

### CP-1.4: Finality Tracking

- **Validates**: Blocks become finalized after 19 confirmations
- **Method**: Wait for height > 25, then query finality
- **Pass**: `lastFinalizedBlock >= currentHeight - 19`
- **Fail**: No finality after 30+ blocks
- **Blocks**: CP-2.x (transfers need finality guarantees)
- **Priority**: P0

### CP-1.5: State Root Convergence

- **Validates**: All validators compute identical state roots
- **Method**: Compare state root hashes at same block height across validators
- **Pass**: All 3 validators report identical state root for height N
- **Fail**: Divergent state roots (consensus failure)
- **Blocks**: All state-dependent checkpoints
- **Priority**: P0

### CP-1.6: Validator Set Stable

- **Validates**: The active validator set matches genesis configuration
- **Method**: Query validator set from each node
- **Pass**: 3 validators with expected stakes (10,000 LGM each)
- **Fail**: Missing validators or unexpected stake amounts
- **Blocks**: CP-1.7 (VRF requires stable validator set)
- **Priority**: P0

### CP-1.7: VRF Validator Selection

- **Validates**: Different validators are selected to produce blocks over time
- **Method**: Observe 50 consecutive blocks, count unique producers
- **Pass**: At least 2 distinct block producers (ideally all 3)
- **Fail**: Single producer for 50+ blocks (VRF broken or only 1 active)
- **Blocks**: None (network functions but centralized)
- **Priority**: P1

---

## Tier 2: Economics (8 checkpoints)

> **Gate Question**: Can value move correctly through the system?
> **Entry**: Tier 1 passes (consensus operational)
> **Signal**: Balances change atomically with correct accounting

### CP-2.1: Token Transfer

- **Validates**: LGM transfers execute and settle correctly
- **Method**: Transfer 1 LGM from faucet to test account, verify both balances
- **Pass**: Sender balance decreased by (amount + fee), recipient balance increased by amount
- **Fail**: Balance mismatch, missing deduction, or transaction rejection
- **Blocks**: All economic checkpoints
- **Priority**: P0

### CP-2.2: Nonce Enforcement

- **Validates**: Replay protection via sequential nonces
- **Method**: Send two transfers with identical nonce
- **Pass**: Second transaction rejected with error code `-32004` (INVALID_NONCE)
- **Fail**: Both accepted (replay vulnerability)
- **Blocks**: None (critical security property)
- **Priority**: P0

### CP-2.3: Insufficient Balance Rejection

- **Validates**: Overdraft protection
- **Method**: Attempt transfer exceeding account balance
- **Pass**: Transaction rejected or fails with insufficient balance error
- **Fail**: Transaction succeeds (money created from nothing)
- **Blocks**: None (critical security property)
- **Priority**: P0

### CP-2.4: Staking Operations

- **Validates**: Stake deposit and withdrawal work correctly
- **Method**: Stake 1000 LGM, verify staked balance, then unstake
- **Pass**: `stakedBalance` increases on stake, decreases on unstake
- **Fail**: Stake/unstake fails or balance accounting incorrect
- **Blocks**: CP-3.x (verification rewards require staking)
- **Priority**: P0

### CP-2.5: Delegation Operations

- **Validates**: Token delegation to validators works
- **Method**: Delegate 100 LGM to validator-1, then undelegate
- **Pass**: Delegation recorded, unbonding period active after undelegate
- **Fail**: Delegation fails or immediate return (no unbonding)
- **Blocks**: None
- **Priority**: P1

### CP-2.6: Fee Routing

- **Validates**: Transaction fees split correctly (78% fee pool / 22% research)
- **Method**: Send transaction with explicit fee, check fee pool and research treasury
- **Pass**: Fee deducted from sender, split reaches designated treasuries
- **Fail**: Fee lost or incorrect split
- **Blocks**: None
- **Priority**: P1
- **Known Issue**: Fee routing increments nonce 3x per tx (FINDING in state/transitions.ts:306-338)

### CP-2.7: Double-Spend Prevention

- **Validates**: Concurrent transfers cannot overdraw an account
- **Method**: Send two transfers each requesting 60% of balance
- **Pass**: First succeeds, second rejected (insufficient balance)
- **Fail**: Both succeed (double-spend)
- **Blocks**: None (critical security property)
- **Priority**: P0

### CP-2.8: Faucet Service

- **Validates**: Faucet distributes tokens correctly with rate limiting
- **Method**:
  ```bash
  curl -s -X POST http://localhost:8080/faucet \
    -H "Content-Type: application/json" \
    -d '{"did":"did:lgm-devnet:test-user"}'
  ```
- **Pass**: Returns success with tx hash, cooldown enforced on repeat request
- **Fail**: Distribution fails or cooldown not enforced
- **Blocks**: None (quality of life)
- **Priority**: P1

---

## Tier 3: Knowledge (7 checkpoints)

> **Gate Question**: Can the Proof of Truth mechanism produce verified knowledge?
> **Entry**: Tier 2 passes (economics operational)
> **Signal**: Claims become facts with confidence > 0

### CP-3.1: Claim Submission

- **Validates**: Claims enter the verification pipeline
- **Method**:
  ```bash
  curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"knowledge_submitClaim","params":[{
      "statement":"Water boils at 100C at sea level",
      "domain":"science.physics",
      "stake":"100000000000000000000",
      "references":[{"factId":"AXIOM_ID","type":"supports"}],
      "sender":"VALIDATOR_PUBKEY",
      "signature":"VALID_SIG"
    }],"id":1}'
  ```
- **Pass**: Returns `claimId` and `verificationRoundId`
- **Fail**: Claim rejected or no round created
- **Blocks**: CP-3.2+
- **Priority**: P0

### CP-3.2: Verification Round Lifecycle

- **Validates**: Full commit-reveal cycle completes
- **Method**: Submit commitments from all 3 validators, then reveals
- **Pass**: Round status transitions: `commit_phase` -> `reveal_phase` -> `completed`
- **Fail**: Round stalls in any phase
- **Blocks**: CP-3.3
- **Priority**: P0

### CP-3.3: Fact Creation

- **Validates**: Accepted claims become facts in the knowledge tree
- **Method**: After verification completes, query for the fact
- **Pass**: Fact exists with `confidence > 0` and `status: "active"`
- **Fail**: Fact not created or confidence == 0
- **Blocks**: CP-3.4+
- **Priority**: P0

### CP-3.4: Confidence Decay

- **Validates**: Fact confidence decreases over time per epistemic category
- **Method**: Query fact at creation, then after 100+ blocks
- **Pass**: `effectiveConfidence < storedConfidence` (logarithmic decay applied)
- **Fail**: No decay observed or confidence increases without reverification
- **Blocks**: None
- **Priority**: P1
- **Note**: Decay is lazy (computed on read), not background

### CP-3.5: Dispute Resolution

- **Validates**: Facts can be challenged through the dispute mechanism
- **Method**: Open dispute on existing fact, submit evidence, resolve
- **Pass**: Dispute lifecycle completes, fact confidence adjusted
- **Fail**: Dispute rejected or no confidence change
- **Blocks**: None
- **Priority**: P1

### CP-3.6: Knowledge Session Economics

- **Validates**: Knowledge queries have correct LGM cost model
- **Method**: Start session, make queries, end session, check total cost
- **Pass**: Session tracks costs, free tier queries available, costs accumulate
- **Fail**: Costs not tracked or free tier not enforced
- **Blocks**: None
- **Priority**: P1

### CP-3.7: Payment Channel Knowledge Access

- **Validates**: Payment channels can fund streaming knowledge queries
- **Method**: Open channel with 200 LGM, make queries, close with settlement
- **Pass**: Channel opens, queries succeed, close distributes remaining balance
- **Fail**: Channel operations fail
- **Blocks**: None
- **Priority**: P2

---

## Tier 4: Resilience (8 checkpoints)

> **Gate Question**: Can the network survive adverse conditions?
> **Entry**: Tier 3 passes (full protocol functional)
> **Signal**: Network recovers from disruptions without data loss

### CP-4.1: Single Validator Crash Recovery

- **Validates**: Network continues with 2 of 3 validators
- **Method**: `docker stop lgm-validator-3`, observe block production continues
- **Pass**: Blocks still produced by remaining 2 validators
- **Fail**: Chain stalls (quorum lost)
- **Blocks**: CP-4.2
- **Priority**: P0

### CP-4.2: Validator Rejoin

- **Validates**: Crashed validator can rejoin and sync
- **Method**: `docker start lgm-validator-3`, wait 30s, check block height
- **Pass**: Validator-3 catches up to within 2 blocks of others
- **Fail**: Validator stuck at old height or crashes on rejoin
- **Blocks**: None
- **Priority**: P0

### CP-4.3: Rapid Restart

- **Validates**: Full network restart preserves state
- **Method**: `docker-compose restart`, wait 30s, check state
- **Pass**: All balances, facts, and validator stakes intact after restart
- **Fail**: Any state lost or genesis re-applied
- **Blocks**: None
- **Priority**: P0

### CP-4.4: State Persistence After Crash

- **Validates**: LevelDB data survives unclean shutdown
- **Method**: `docker kill lgm-validator-1`, then `docker start lgm-validator-1`
- **Pass**: Node recovers to last finalized block, no data corruption
- **Fail**: Node fails to start or loses post-genesis state
- **Blocks**: None
- **Priority**: P0

### CP-4.5: Invalid Transaction Rejection

- **Validates**: Malformed or malicious transactions are rejected
- **Method**: Send transactions with bad signatures, wrong nonces, invalid amounts
- **Pass**: All rejected with appropriate error codes (-32003, -32004, -32002)
- **Fail**: Any invalid transaction accepted
- **Blocks**: None
- **Priority**: P0
- **Reference**: Phase 6 tests (DEVNET_TESTING.md)

### CP-4.6: Rate Limiting Under Load

- **Validates**: RPC rate limits protect validators from abuse
- **Method**: Send 1000 rapid requests to single validator
- **Pass**: Requests throttled after limit (600/min for free tier), 429 returned
- **Fail**: All requests served without throttling
- **Blocks**: None
- **Priority**: P1

### CP-4.7: Block Production Under Transaction Load

- **Validates**: Block production continues during high transaction volume
- **Method**: Submit 500 transactions, verify blocks still produced on schedule
- **Pass**: Blocks continue at expected rate, transactions ordered by fee
- **Fail**: Block production stalls or mempool overflow
- **Blocks**: None
- **Priority**: P1

### CP-4.8: State Sync for New Node

- **Validates**: A new fullnode can sync from existing validators
- **Method**: Start fullnode profile after validators have produced 100+ blocks
  ```bash
  docker-compose --profile fullnode up -d fullnode
  ```
- **Pass**: Fullnode syncs to current height within 60s
- **Fail**: Sync fails or fullnode stuck at genesis
- **Blocks**: None (required for testnet but acceptable for devnet)
- **Priority**: P1

---

## Tier 5: Operations (6 checkpoints)

> **Gate Question**: Can an operator effectively manage this network?
> **Entry**: Tier 4 passes (resilient network)
> **Signal**: All operational tools and monitoring work

### CP-5.1: CLI Operations

- **Validates**: CLI tool can perform all common operations
- **Method**:
  ```bash
  npx ts-node cmd/cli.ts health
  npx ts-node cmd/cli.ts balance
  npx ts-node cmd/cli.ts facts logic.foundational
  ```
- **Pass**: All commands return meaningful output without errors
- **Fail**: CLI crashes or returns unintelligible output
- **Blocks**: None
- **Priority**: P1

### CP-5.2: Explorer Service

- **Validates**: Block explorer shows current network state
- **Method**:
  ```bash
  curl -s http://localhost:3000/api/overview | jq .
  curl -s http://localhost:3000/api/blocks | jq '.blocks | length'
  curl -s http://localhost:3000/api/validators | jq '.validators | length'
  ```
- **Pass**: Overview returns current data, blocks listed, 3 validators shown
- **Fail**: Explorer returns stale data or errors
- **Blocks**: None
- **Priority**: P2

### CP-5.3: Prometheus Metrics

- **Validates**: Metrics scraped from all validators
- **Method**: `curl -s http://localhost:9093/api/v1/targets | jq '.data.activeTargets | length'`
- **Pass**: 3 active targets (one per validator), all "up"
- **Fail**: Missing targets or scrape errors
- **Blocks**: CP-5.4
- **Priority**: P1

### CP-5.4: Grafana Dashboards

- **Validates**: Monitoring dashboards render with live data
- **Method**: Open `http://localhost:3001`, login (admin/devnet), check "Legible Money" dashboard
- **Pass**: Panels show non-zero data for block height, peer count, mempool
- **Fail**: Empty panels or connection errors
- **Blocks**: None
- **Priority**: P2

### CP-5.5: Log Accessibility

- **Validates**: Validator logs are structured and queryable
- **Method**: `docker-compose logs validator-1 | head -50`
- **Pass**: Logs contain structured output with timestamps, block heights, peer events
- **Fail**: No logs or unstructured noise
- **Blocks**: None
- **Priority**: P1

### CP-5.6: Network Reset Procedure

- **Validates**: DevNet can be fully reset to clean state
- **Method**:
  ```bash
  docker-compose down -v
  npx ts-node devnet/scripts/init-devnet.ts
  docker-compose up -d
  ```
- **Pass**: Fresh network starts with genesis state, no residual data
- **Fail**: Old state persists or initialization fails
- **Blocks**: None
- **Priority**: P1

---

## Launch Gate: Go/No-Go Decision

### Mandatory Pass (P0 Checkpoints)

All P0 checkpoints MUST pass for launch approval:

| ID | Checkpoint | Tier |
|----|-----------|------|
| CP-0.1 | Container Initialization | Infrastructure |
| CP-0.2 | Health Endpoint Availability | Infrastructure |
| CP-0.3 | Genesis State Loaded | Infrastructure |
| CP-0.4 | Genesis Axioms Present | Infrastructure |
| CP-0.5 | P2P Peer Discovery | Infrastructure |
| CP-0.6 | LevelDB Persistence Initialized | Infrastructure |
| CP-1.1 | Block Production Active | Consensus |
| CP-1.2 | Block Height Consistency | Consensus |
| CP-1.4 | Finality Tracking | Consensus |
| CP-1.5 | State Root Convergence | Consensus |
| CP-1.6 | Validator Set Stable | Consensus |
| CP-2.1 | Token Transfer | Economics |
| CP-2.2 | Nonce Enforcement | Economics |
| CP-2.3 | Insufficient Balance Rejection | Economics |
| CP-2.4 | Staking Operations | Economics |
| CP-2.7 | Double-Spend Prevention | Economics |
| CP-3.1 | Claim Submission | Knowledge |
| CP-3.2 | Verification Round Lifecycle | Knowledge |
| CP-3.3 | Fact Creation | Knowledge |
| CP-4.1 | Single Validator Crash Recovery | Resilience |
| CP-4.2 | Validator Rejoin | Resilience |
| CP-4.3 | Rapid Restart | Resilience |
| CP-4.4 | State Persistence After Crash | Resilience |
| CP-4.5 | Invalid Transaction Rejection | Resilience |

**Total P0**: 24 checkpoints (all must PASS)

### Should Pass (P1 Checkpoints)

P1 checkpoints should pass but are not launch-blocking:

| ID | Checkpoint | Tier |
|----|-----------|------|
| CP-1.3 | Block Timing | Consensus |
| CP-1.7 | VRF Validator Selection | Consensus |
| CP-2.5 | Delegation Operations | Economics |
| CP-2.6 | Fee Routing | Economics |
| CP-2.8 | Faucet Service | Economics |
| CP-3.4 | Confidence Decay | Knowledge |
| CP-3.5 | Dispute Resolution | Knowledge |
| CP-3.6 | Knowledge Session Economics | Knowledge |
| CP-4.6 | Rate Limiting Under Load | Resilience |
| CP-4.7 | Block Production Under Load | Resilience |
| CP-4.8 | State Sync for New Node | Resilience |
| CP-5.1 | CLI Operations | Operations |
| CP-5.3 | Prometheus Metrics | Operations |
| CP-5.5 | Log Accessibility | Operations |
| CP-5.6 | Network Reset Procedure | Operations |

**Total P1**: 15 checkpoints (target: >80% pass)

### Nice to Have (P2 Checkpoints)

| ID | Checkpoint | Tier |
|----|-----------|------|
| CP-3.7 | Payment Channel Knowledge Access | Knowledge |
| CP-5.2 | Explorer Service | Operations |
| CP-5.4 | Grafana Dashboards | Operations |

**Total P2**: 3 checkpoints

---

## Checkpoint Status Tracker

| ID | Checkpoint | Priority | Status | Date | Notes |
|----|-----------|----------|--------|------|-------|
| CP-0.1 | Container Initialization | P0 | [ ] | | |
| CP-0.2 | Health Endpoint Availability | P0 | [ ] | | |
| CP-0.3 | Genesis State Loaded | P0 | [ ] | | |
| CP-0.4 | Genesis Axioms Present | P0 | [ ] | | |
| CP-0.5 | P2P Peer Discovery | P0 | [ ] | | |
| CP-0.6 | LevelDB Persistence Initialized | P0 | [ ] | | |
| CP-1.1 | Block Production Active | P0 | [ ] | | |
| CP-1.2 | Block Height Consistency | P0 | [ ] | | |
| CP-1.3 | Block Timing | P1 | [ ] | | |
| CP-1.4 | Finality Tracking | P0 | [ ] | | |
| CP-1.5 | State Root Convergence | P0 | [ ] | | |
| CP-1.6 | Validator Set Stable | P0 | [ ] | | |
| CP-1.7 | VRF Validator Selection | P1 | [ ] | | |
| CP-2.1 | Token Transfer | P0 | [ ] | | |
| CP-2.2 | Nonce Enforcement | P0 | [ ] | | |
| CP-2.3 | Insufficient Balance Rejection | P0 | [ ] | | |
| CP-2.4 | Staking Operations | P0 | [ ] | | |
| CP-2.5 | Delegation Operations | P1 | [ ] | | |
| CP-2.6 | Fee Routing | P1 | [ ] | | |
| CP-2.7 | Double-Spend Prevention | P0 | [ ] | | |
| CP-2.8 | Faucet Service | P1 | [ ] | | |
| CP-3.1 | Claim Submission | P0 | [ ] | | |
| CP-3.2 | Verification Round Lifecycle | P0 | [ ] | | |
| CP-3.3 | Fact Creation | P0 | [ ] | | |
| CP-3.4 | Confidence Decay | P1 | [ ] | | |
| CP-3.5 | Dispute Resolution | P1 | [ ] | | |
| CP-3.6 | Knowledge Session Economics | P1 | [ ] | | |
| CP-3.7 | Payment Channel Knowledge Access | P2 | [ ] | | |
| CP-4.1 | Single Validator Crash Recovery | P0 | [ ] | | |
| CP-4.2 | Validator Rejoin | P0 | [ ] | | |
| CP-4.3 | Rapid Restart | P0 | [ ] | | |
| CP-4.4 | State Persistence After Crash | P0 | [ ] | | |
| CP-4.5 | Invalid Transaction Rejection | P0 | [ ] | | |
| CP-4.6 | Rate Limiting Under Load | P1 | [ ] | | |
| CP-4.7 | Block Production Under Load | P1 | [ ] | | |
| CP-4.8 | State Sync for New Node | P1 | [ ] | | |
| CP-5.1 | CLI Operations | P1 | [ ] | | |
| CP-5.2 | Explorer Service | P2 | [ ] | | |
| CP-5.3 | Prometheus Metrics | P1 | [ ] | | |
| CP-5.4 | Grafana Dashboards | P2 | [ ] | | |
| CP-5.5 | Log Accessibility | P1 | [ ] | | |
| CP-5.6 | Network Reset Procedure | P1 | [ ] | | |

---

## Traceability Matrix

Maps checkpoints to existing functional tests from DEVNET_TESTING.md:

| Checkpoint | Functional Tests | Robustness Tests |
|------------|-----------------|------------------|
| CP-0.2 | 1.1 (Health endpoint) | — |
| CP-0.3 | 1.3, 1.4 (Genesis balances) | — |
| CP-0.4 | 1.5, 1.6 (Axioms) | — |
| CP-0.5 | 1.2 (Network health) | — |
| CP-1.1 | 2.1 (Block production) | Block Production Stress (5) |
| CP-1.2 | 2.2 (Height consistency) | Consensus State Integrity (4) |
| CP-1.3 | 2.3 (Block timing) | — |
| CP-1.4 | — | Long-Running Stability (2) |
| CP-1.5 | 2.2 (Height consistency) | State Root Consistency (4) |
| CP-2.1 | 3.2, 3.3 (Transfer) | State Transition Correctness (5) |
| CP-2.2 | 3.4 (Nonce) | Edge Cases (6) |
| CP-2.3 | — | State Transition Correctness (5) |
| CP-2.4 | 3.5, 3.6 (Stake/Unstake) | — |
| CP-2.5 | 3.7, 3.8 (Delegate) | — |
| CP-2.7 | 6.6 (Double-spend) | Edge Cases (6) |
| CP-3.1 | 4.4 (Submit claim) | Claim Verification Pipeline (4) |
| CP-3.2 | 4.5-4.8 (Commit/Reveal) | Validator Verification Rounds (3) |
| CP-3.3 | 4.9 (Fact created) | — |
| CP-3.5 | 5.1-5.5 (Disputes) | — |
| CP-4.5 | 6.1-6.3 (Invalid tx) | — |
| CP-4.6 | 6.7 (Rate limiting) | — |
| CP-4.7 | — | Mempool Robustness (8) |

---

## Protocol Constants Reference

| Constant | Value | Source |
|----------|-------|--------|
| Block time | 2521ms | consensus/pot/types.ts |
| Finality depth | 19 blocks | consensus/pot/finality.ts |
| Blocks per day | 34,272 | Derived (86400000 / 2521) |
| Min validator stake | 10,000 LGM | devnet/config/genesis.json |
| Min claim stake | 100 LGM | consensus/pot/types.ts |
| Commit phase | 4 blocks | consensus params |
| Reveal phase | 4 blocks | consensus params |
| Acceptance threshold | 80% | consensus params |
| Validator quorum | 66% (2/3) | consensus params |
| Rate limit (free) | 600 req/min | infrastructure/node/index.ts |
| Faucet amount | 1,000 LGM | devnet/docker-compose.yml |
| Faucet cooldown | 60 min | devnet/docker-compose.yml |
| Fee split | 78% pool / 22% research | state/transitions.ts |

---

## Execution Guide

### Prerequisites

```bash
# Ensure Docker and Node.js installed
docker --version    # >= 20.x
node --version      # >= 18.x

# Build the project
npm install && npm run build

# Initialize devnet keys and genesis
npx ts-node devnet/scripts/init-devnet.ts
```

### Running All Checkpoints

```bash
# Start infrastructure
docker-compose up -d

# Wait for services
sleep 15

# Run automated checkpoint validation
npx ts-node scripts/devnet-checkpoint.ts

# Or run specific tier
npx ts-node scripts/devnet-checkpoint.ts --tier 0
```

### Manual Checkpoint Execution

For each tier, execute checkpoints in order. Stop if a P0 checkpoint fails — fix before proceeding.

```
Tier 0 → Fix any P0 failures → Tier 1 → Fix any P0 failures → ... → Launch Gate
```

---

## History

| Date | Event | Result |
|------|-------|--------|
| 2026-02-06 | Framework created | 42 checkpoints defined |

---

_Last Updated: 2026-02-06_
_PP Verification: T (grounded in real code/configs) U (maps to existing tests) B (clean checkpoint format) J (placed in docs/ with project docs) C (new abstraction layer over existing tests)_
