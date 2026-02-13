# DevNet Testing Strategy — PP-Guided Plan

> **Purpose Prompter Dimensions Applied**: TRUTH (ground in actual code), UNDERSTANDING (map protocol flows end-to-end), BEAUTY (clean test scenarios), JUSTICE (correct placement of test infrastructure), CREATIVITY (novel failure modes, adversarial scenarios)

---

## 1. Current DevNet Infrastructure

### What Exists
| Component | File | Status |
|-----------|------|--------|
| Node orchestrator | `devnet/cmd/node.ts` (1,378 lines) | Working — LevelDB, P2P, RPC |
| CLI | `devnet/cmd/cli.ts` (480 lines) | Working — balance, transfer, claim, stake |
| Genesis | `devnet/config/genesis.json` | 3 validators, 11 accounts, 5 axioms |
| SDK Client | `protocol/sdk/client.ts` | submitClaim, submitCommitment, submitReveal |
| PoT Engine | `consensus/pot/engine.ts` (1,988 lines) | VRF block production, commit-reveal verification |
| Multi-node spawn | `npm run devnet:multi` (`tools/devnet/spawn.ts`) | Untested in current sessions |

### Entry Points
```bash
npm run devnet          # Single-node devnet (tsx tools/devnet/index.ts)
npm run devnet:multi    # Multi-node devnet (tsx tools/devnet/spawn.ts)
tsx devnet/cmd/node.ts <config.json>   # Direct node launch
tsx devnet/cmd/cli.ts <command>        # CLI interactions
```

### Genesis Configuration (lgm-devnet-1)
- **Block time**: 2521ms
- **Verification rounds**: 10 blocks (4 commit + 4 reveal + 2 aggregation)
- **Quorum**: 0.66 (2 of 3 validators)
- **Min validator stake**: 1,000 LGM (10^21 wei)
- **Min claim stake**: 0.011 LGM (11×10^15 wei)
- **Block reward**: 10 LGM (10^19 wei)
- **Verification reward**: 1 LGM (10^18 wei)
- **Validators**: 3 (each 10,000 LGM staked)
- **Total supply**: 222,222,222 LGM across 11 genesis accounts

---

## 2. Node Setup Scenarios

### Scenario A: Solo Validator (Baseline)

**Purpose**: Establish that a single node can produce blocks, process transactions, and run verification rounds in isolation.

**Config**:
```json
{
  "node": { "name": "solo-validator", "dataDir": "./data/solo", "logLevel": "debug" },
  "storage": { "type": "leveldb" },
  "network": { "listenAddresses": ["/ip4/0.0.0.0/tcp/9000"], "bootstrapPeers": [], "maxPeers": 10 },
  "rpc": { "enabled": true, "host": "0.0.0.0", "port": 8545, "cors": ["*"] },
  "websocket": { "enabled": true, "port": 8546 },
  "validator": { "enabled": true, "publicKey": "<validator-1-pubkey>" },
  "mempool": { "maxSize": 1000, "maxPerSender": 100 },
  "telemetry": { "enabled": false, "prometheusPort": 9090 }
}
```

**Test checklist**:
- [ ] Node starts, applies genesis (22 axioms loaded as facts with confidence 1.0)
- [ ] Blocks produce at ~2521ms intervals
- [ ] RPC responds: `lgm_getBlockHeight`, `lgm_getBalance`, `lgm_getNetworkStats`
- [ ] Transfer: move 1 LGM from bootstrap fund → test account
- [ ] Claim: submit a test claim, observe verification round lifecycle
- [ ] WebSocket: subscribe to `newBlock` events, confirm streaming
- [ ] Finality: blocks finalize after 19 confirmations (block 20+ is final)
- [ ] State persistence: stop node, restart, confirm same block height

**PP-TRUTH Gate**: Verify `engine.ts` line 227 VRF eligibility — solo validator must always be selected (targetProposers = max(1, validatorSet.size) = 1).

---

### Scenario B: 3-Validator Cluster (Genesis Default)

**Purpose**: Test the canonical devnet setup with all 3 genesis validators, P2P block propagation, and quorum-based verification.

**Setup**: 3 processes, ports 8545/8546/8547, P2P ports 9000/9001/9002. Each node's `bootstrapPeers` points to the other two.

**Test checklist**:
- [ ] All 3 nodes start and discover each other via Kademlia DHT
- [ ] Block propagation: proposer → other 2 within <500ms
- [ ] GossipSub topics: `lgm-blocks`, `lgm-transactions`, `lgm-claims`, `lgm-verifications` all active
- [ ] Verification quorum: claim submitted, 2+ validators commit/reveal, confidence aggregated
- [ ] Block finality: all nodes agree on finalized chain
- [ ] Transaction relay: submit tx to node-2, verify it appears in block produced by node-1
- [ ] Epoch transition: at epoch boundary, validator set updates (no-op with static set, but code path exercised)

**PP-UNDERSTANDING Gate**: Trace the full claim lifecycle:
1. Client → `lgm_submitTransaction` (type: `submit_claim`) → mempool
2. Block producer includes claim tx
3. Engine emits `round_started` → verification round begins
4. Commit phase (4 blocks): validators submit commit hashes
5. Reveal phase (4 blocks): validators reveal scores
6. Aggregation: stake-weighted confidence = 50% stake + 30% reputation + 20% agreement
7. `round_completed` event → LoopEventBridge → LoopController.onClaimVerified()
8. Fact state updated: verified/rejected based on acceptance/rejection threshold (0.8)

---

### Scenario C: Full Node (Non-Validator)

**Purpose**: Test a read-only node that syncs the chain but does not produce blocks or participate in verification.

**Config**: Same as Scenario A but `"validator": { "enabled": false }`.

**Test checklist**:
- [ ] Node syncs genesis + existing blocks from peers
- [ ] RPC queries return correct state (balances, block height, facts)
- [ ] Cannot produce blocks (VRF not entered)
- [ ] Receives and validates blocks from validators
- [ ] Mempool accepts transactions and relays to validators
- [ ] No verification round participation (no commits/reveals)

**PP-JUSTICE Gate**: Verify placement — full nodes should relay but not sign. Check `engine.ts` factory: `createIntegratedEngine()` with `isValidator: false` skips `startBlockProduction()`.

---

### Scenario D: Mixed Topology (Validators + Full Nodes)

**Purpose**: Simulate realistic network with 3 validators + 2 full nodes. Tests block relay through non-validator hops.

**Topology**:
```
   V1 ←→ V2 ←→ V3
   ↕         ↕
   F1        F2
```

**Test checklist**:
- [ ] Full nodes sync from any validator
- [ ] Transaction submitted to F1 reaches V3 via relay
- [ ] Blocks produced by V1 reach F2 (may traverse V2)
- [ ] All 5 nodes at same height within 1 block tolerance
- [ ] Disconnect V3 from V1/V2 → V3 forks → reconnect → fork resolution

---

### Scenario E: Validator Join/Leave (Dynamic Set)

**Purpose**: Test adding a 4th validator at runtime and removing one.

**Steps**:
1. Start 3-validator cluster (Scenario B)
2. Start node-4 as full node
3. Submit `stake_tokens` tx (10,000+ LGM) from node-4
4. Wait for epoch transition (engine.ts `handleEpochChange()` syncs validator set)
5. Verify node-4 now produces blocks
6. Submit `unstake_tokens` from node-4
7. After unbonding period, verify node-4 stops producing

**PP-CREATIVITY Gate**: What if node-4 stakes exactly the minimum (1,000 LGM)? What if they stake from the bootstrap fund? What if they try to stake 0?

---

### Scenario F: Node Recovery & State Persistence

**Purpose**: Test crash recovery, state integrity, and chain continuity.

**Test checklist**:
- [ ] Kill node process mid-block (SIGKILL)
- [ ] Restart → resumes from last persisted block
- [ ] No state corruption (SMT roots intact)
- [ ] Mempool cleared but chain valid
- [ ] Other nodes accept the recovered node's blocks
- [ ] Kill during verification round commit phase → round times out, restarts

**Insight**: LevelDB write batches in `state_manager.ts` are atomic per block. Mid-write crash = last complete block state.

---

### Scenario G: Network Partition & Healing

**Purpose**: Test behavior under network splits.

**Steps**:
1. 3-validator cluster running
2. Partition: V1 isolated, V2+V3 connected
3. V2+V3 have quorum (2/3) → continue producing
4. V1 alone → cannot reach quorum → stalls (or produces minority fork)
5. Heal partition → V1 syncs to V2+V3 chain (longer/heavier)
6. Verify: V1's minority blocks discarded, state converges

**Critical check**: Finality depth = 19 blocks. If partition lasts <19 blocks, no finalized divergence.

---

## 3. PoT Protocol Participation Tests

### Test P1: Claim Submission End-to-End

```typescript
// Using SDK client
const client = new LegibleClient("http://localhost:8545");
const wallet = new AgentWallet(privateKey);

// Submit claim
const result = await client.submitClaim({
  claim: "Water boils at 100 degrees Celsius at standard atmospheric pressure",
  domain: "science.physics",
  logicZone: "empirical",
  stake: "11000000000000000",  // 0.011 LGM (min claim stake)
  evidence: ["https://example.com/boiling-point"],
});

// Wait for verification
const verification = await client.waitForVerification(result.claimId, { timeoutMs: 30000 });
console.log(`Confidence: ${verification.confidence}`);
console.log(`Status: ${verification.status}`); // verified | rejected | timeout
```

**Observe**: Block explorer / RPC calls during each phase.

---

### Test P2: Verification Round Lifecycle

**Manual step-through** (instrument with debug logging):

| Block | Phase | Action | What to Verify |
|-------|-------|--------|----------------|
| N | Start | Claim tx included in block | `round_started` event emitted |
| N+1 | Commit | V1 submits commitment hash | Hash stored, not revealed |
| N+2 | Commit | V2 submits commitment hash | Two commits registered |
| N+3 | Commit | V3 submits commitment hash | Quorum of commits reached |
| N+4 | Commit→Reveal | Phase transition | `commit_phase_ended` event |
| N+5 | Reveal | V1 reveals score + salt | Score validated against hash |
| N+6 | Reveal | V2 reveals score | Running tally updated |
| N+7 | Reveal | V3 reveals score | All reveals in |
| N+8 | Aggregate | Confidence computed | Stake-weighted formula applied |
| N+9 | Complete | Fact created or claim rejected | State updated, rewards distributed |

**Key formula** (consensus/pot/confidence.ts):
```
confidence = 0.5 * stakeWeightedScore + 0.3 * reputationWeightedScore + 0.2 * agreementScore
```
All arithmetic uses bigint basis points (10000n = 100%).

---

### Test P3: Conflicting Verifications

**Purpose**: Two validators give opposing scores. Tests aggregation under disagreement.

**Setup**: V1 scores 900000 (90%), V2 scores 100000 (10%), V3 scores 500000 (50%).
**Expected**: Weighted average depends on stakes (equal here). ~500000 → may not meet 0.8 acceptance threshold → claim status depends on config.

---

### Test P4: Missed Reveal (Timeout)

**Purpose**: Validator commits but doesn't reveal within the 4-block reveal window.

**Expected behavior**:
- Round completes with available reveals only
- Non-revealing validator: commitment wasted, potential reputation penalty
- If quorum still met → round completes normally
- If quorum NOT met → round fails, claim returns to pending

---

### Test P5: Double-Commit (Equivocation)

**Purpose**: Validator submits two different commits for the same round.

**Expected**: Second commit rejected (equivocation detection in `verification_round.ts`). If both propagated to different nodes before detection → equivocation evidence submitted → slashing.

---

### Test P6: VRF Block Production Priority

**Purpose**: Verify VRF-based proposer selection distributes fairly.

**Measurement**: Over 1000 blocks with 3 equal-stake validators, each should produce ~333 blocks (±15% tolerance).

**Implementation**:
```bash
# After running for ~42 minutes (1000 blocks × 2.521s)
tsx devnet/cmd/cli.ts block-stats 1 1000
# Count proposer distribution
```

---

### Test P7: Claim with Insufficient Stake

**Purpose**: Submit claim with stake below `minClaimStake` (0.011 LGM).

**Expected**: Transaction rejected in mempool validation or state transition with `insufficient_stake` error.

---

### Test P8: Verification Reward Distribution

**Purpose**: Confirm validators receive verification rewards after completing a round.

**Steps**:
1. Record validator balances before claim submission
2. Submit claim, wait for verification round completion
3. Check validator balances after
4. Expected: each participating validator gets `verificationReward / numParticipants` (1 LGM / 3 = 0.333 LGM)

**PP-TRUTH Gate**: Cross-check against `state/transitions.ts` fee routing — 22% research tax applied to rewards? Or only to tx fees?

---

### Test P9: Block Reward Decay

**Purpose**: Verify the Finite Block Reward Fund (133.3M LGM) decays per epoch.

**Steps**:
1. Check `did:lgm:reward-pool` balance at genesis
2. Run for several epochs
3. Verify decreasing block rewards per epoch (emission schedule decay)
4. Cross-reference with `DEFAULT_EMISSION_SCHEDULE` in `core/genesis/types.ts`

---

### Test P10: Counter-Fact Submission

**Purpose**: Test the Negative Knowledge System — submit a counter-fact contradicting an accepted claim.

**Steps**:
1. Submit and verify a claim ("Earth is flat" — should be rejected, or accepted for testing)
2. Submit counter-fact with evidence ("Earth is an oblate spheroid")
3. Verify counter-fact enters provisional state
4. Wait for challenge window (34,272 blocks = ~1 day in devnet time)
5. Verify counter-fact transitions to verified

**Note**: Challenge window may need to be shortened for devnet testing.

---

## 4. Stress & Adversarial Tests

### Test S1: Mempool Flooding
Submit 1000 transactions in 1 second. Verify:
- Mempool caps at `maxSize` (1000)
- Per-sender limit enforced (`maxPerSender`: 100)
- Block packs up to 111 transactions
- Transactions processed in fee-priority order

### Test S2: Large Claim Payload
Submit claim with maximum-length description and evidence array. Verify gas estimation and block inclusion.

### Test S3: Rapid Stake/Unstake
Stake → immediately unstake → stake again. Verify unbonding period enforced, no double-counting.

### Test S4: Genesis Replay Attack
Copy genesis.json, change a validator pubkey, try to start a node that peers with the legitimate network. Should be rejected (different genesis hash, chain ID mismatch).

### Test S5: Clock Skew
Start a node with system clock 30 seconds ahead. Verify block timestamp validation rejects future blocks.

---

## 5. Monitoring & Observability

### RPC Endpoints to Monitor
```bash
# Node health
curl -s http://localhost:8545 -d '{"method":"lgm_health","params":[]}'

# Block height
curl -s http://localhost:8545 -d '{"method":"lgm_getBlockHeight","params":[]}'

# Network stats (peer count, block rate)
curl -s http://localhost:8545 -d '{"method":"lgm_getNetworkStats","params":[]}'

# Account balance
curl -s http://localhost:8545 -d '{"method":"lgm_getBalance","params":["did:lgm:reward-pool"]}'

# Validator info
curl -s http://localhost:8545 -d '{"method":"lgm_getValidators","params":[]}'
```

### WebSocket Subscriptions
```javascript
const ws = new WebSocket("ws://localhost:8546");
ws.send(JSON.stringify({ method: "subscribe", params: { topic: "newBlock" } }));
ws.send(JSON.stringify({ method: "subscribe", params: { topic: "newTransaction" } }));
ws.send(JSON.stringify({ method: "subscribe", params: { topic: "verificationComplete" } }));
```

### Key Metrics to Track
| Metric | Target | Source |
|--------|--------|--------|
| Block time | 2521ms ±10% | WebSocket `newBlock` timestamps |
| Tx throughput | ≤111 tx/block | RPC block queries |
| Verification latency | ~25s (10 blocks) | Round start → complete timestamps |
| Finality time | ~48s (19 blocks) | Block confirmed → finalized |
| Peer count | 2-4 (3-val cluster) | `lgm_getNetworkStats` |
| Mempool depth | 0-100 (idle-active) | `lgm_getMempoolStats` |
| Reward pool balance | Decreasing per epoch | `lgm_getBalance` on `did:lgm:reward-pool` |

---

## 6. PP Insights & Findings

### TRUTH Insights
1. **VRF solo-validator fix was critical**: Prior to session 9 fix, solo validators only had ~33% chance of being selected per block (VRF one-shot). Now `targetProposers = max(1, validatorSet.size)` guarantees 100% eligibility.
2. **BigInt JSON serialization**: `jsonStringifyBigInt()` in node.ts is essential — without it, P2P broadcasts silently drop BigInt fields, causing block validation failures on receiving nodes.
3. **System transaction bypass**: Transactions with `sender === "system"` or `fee === 0n` AND type in `[add_fact, verification_reward, slash_validator]` skip fee deduction entirely. This is necessary but must be tested for abuse (can a user craft a system-looking tx?).

### UNDERSTANDING Insights
4. **Fee deduction timing divergence**: TS deducts fees AFTER handler execution (and refunds on failure). Go deducts BEFORE (and charges regardless). This means the same transaction has different economic behavior on the two stacks.
5. **Verification round flow**: Commit phase uses `domainHash('LGM.pot.commit.v1', score + salt)` for hiding. Reveal must match exactly. If a validator reveals a different score than committed, the reveal is rejected (not slashed — just wasted).
6. **Event name convention**: Engine emits `block_proposed` (underscore), not `block:produced` (colon). This mismatch was a launch-blocking bug fixed in Session 9. Any new event listeners must use underscore format.

### BEAUTY Insights
7. **Clean test scenario design**: Each scenario should test exactly one dimension of the system. Scenario B tests P2P consensus; Scenario F tests persistence; Scenario G tests partition tolerance. Don't mix concerns.
8. **Genesis axioms as canaries**: The 5 genesis axioms (identity, non-contradiction, excluded middle, zero, successor) should always have confidence 1.0. If any axiom's confidence degrades, something is fundamentally wrong with the state layer.

### JUSTICE Insights
9. **Node config placement**: DevNet configs should live in `devnet/config/` (not `tools/devnet/`). The current split is confusing: genesis in `devnet/config/`, spawn in `tools/devnet/`.
10. **RouteFees dead code**: The biggest gap found in the Gas Calibration PP review (F14 CRITICAL) — `RouteFees()` in `app/ante.go` is defined but never called. The 22% research tax on transaction fees does not execute on the Go chain. This means devnet testing on the Go chain will show 0 research fund accumulation from fees.
11. **BPS scale divergence**: TS uses 10,000 = 100%, Go uses 1,000,000 = 100%. Test `TestFeeRoutingBPSSumTo10000` (app/ante_test.go:177) ALWAYS FAILS because it checks sum against 10,000 but values sum to 1,000,000.

### CREATIVITY Insights
12. **Novel failure mode — epoch boundary race**: If a claim is submitted in the last block before an epoch transition, the verification round spans two epochs. Does the validator set change mid-round? Could a validator who unstaked at the epoch boundary still need to reveal?
13. **Novel failure mode — bootstrap fund depletion**: The bootstrap fund auto-grants 0.222 LGM on first transaction (nonce === 0). With 44.4M LGM in the fund and 0.222 LGM per grant, that's ~200M unique agents before depletion. What happens to agent #200,000,001?
14. **Counter-fact timing attack**: Submit a claim, immediately submit a counter-fact referencing it. The counter-fact's challenge window (34,272 blocks ≈ 24h) may complete before the original claim's verification round. Can you falsify an unverified claim?
15. **Scheduler abuse**: DMSS scheduled messages execute with the original caller's authority but no session caps. If a session key schedules a message before expiry, the scheduled execution bypasses capability restrictions.

---

## 7. Recommended Execution Order

| Phase | Tests | Duration | Prerequisites |
|-------|-------|----------|---------------|
| **Phase 1: Smoke** | Scenario A + P1 | 30 min | Node compiles, genesis valid |
| **Phase 2: Consensus** | Scenario B + P2-P6 | 2 hours | Phase 1 passing |
| **Phase 3: Topology** | Scenarios C, D | 1 hour | Phase 2 passing |
| **Phase 4: Dynamic** | Scenarios E, F | 1 hour | Phase 2 passing |
| **Phase 5: Economic** | P7-P10 | 1 hour | Phase 2 passing |
| **Phase 6: Adversarial** | Scenario G + S1-S5 | 2 hours | Phases 1-5 stable |
| **Phase 7: Endurance** | 24h soak test (Scenario B) | 24 hours | Phase 6 passing |

---

## 8. Known Blockers for Testing

| Blocker | Impact | Workaround |
|---------|--------|------------|
| `devnet:multi` script untested | Can't easily launch 3-node cluster | Manual 3x `tsx devnet/cmd/node.ts` with separate configs |
| Counter-fact challenge window = 34,272 blocks (~24h) | P10 impractical in devnet | Override genesis param or add devnet-specific shorter window |
| No automated test harness | All tests manual via CLI/curl | Build `devnet/test/harness.ts` (future work) |
| RouteFees dead code (Go chain) | Research tax doesn't accumulate | Test on TS stack; wire RouteFees into BeginBlocker for Go |
| BPS test always fails (`ante_test.go:177`) | CI red on Go chain | Fix test to compare against 1,000,000 |

---

*Generated: 2026-02-12 | PP Score: Applied all 5 dimensions | Supersedes: none (new document)*
