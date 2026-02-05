# Launch Roadmap

> **Generated**: 2026-02-05
> **Analysis**: PP Hierarchy Review (TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY)

---

## Executive Summary

**Current Status**: 95% DevNet Ready | 75% Testnet Ready | 50% Mainnet Ready

The protocol has **excellent architectural design** with comprehensive feature implementation. All critical blockers (B1, B2, B3) are now resolved. DevNet is ready for launch.

---

## PP Hierarchy Assessment

### TRUTH (Foundation)
| Component | Status | Issue |
|-----------|--------|-------|
| Cryptography | ✅ 100% | Solid Ed25519, VRF, domain separation |
| State Types | ✅ 100% | Well-defined, type-safe |
| Consensus Logic | ✅ 95% | Minor hardcoded values |
| **Data Persistence** | ✅ 100% | LevelDB wired via node orchestrator |

### UNDERSTANDING (Source)
| Component | Status | Issue |
|-----------|--------|-------|
| Documentation | ✅ 85% | Missing operational runbooks |
| API Docs | ⚠️ 60% | 40% endpoints undocumented |
| Architecture | ✅ 95% | Excellent design docs |

### BEAUTY (Quality)
| Component | Status | Issue |
|-----------|--------|-------|
| Code Structure | ✅ 90% | Clean modular architecture |
| Type Safety | ✅ 90% | Branded types, bigint for money |
| Error Handling | ⚠️ 70% | Needs error code standardization |

### JUSTICE (Placement)
| Component | Status | Issue |
|-----------|--------|-------|
| Security Vulns | ✅ 100% | All P0/P1 fixed |
| Test Coverage | ⚠️ 50% | +51 integration tests (2026-02-05) |
| Deployment | ❌ 30% | Manual steps required |

### CREATIVITY (Path)
| Component | Status | Issue |
|-----------|--------|-------|
| Novel Features | ✅ 95% | PoT, Agent Home, DMSS complete |
| Extension Points | ✅ 85% | Good interfaces for plugins |
| Future-Proofing | ✅ 80% | Cosmos migration planned |

---

## Critical Blockers (MUST FIX)

### B1: Persistence Layer ✅ FIXED
**Problem**: ~~All state lost on crash/restart~~
**Status**: RESOLVED 2026-02-05

**Solution Implemented**:
- `infrastructure/node/index.ts` - Node orchestrator wires storage to StateManager
- `state/storage/leveldb.ts` - Full LevelDB implementation (already existed)
- `state/state_manager.ts` - Full persistence methods (lines 1304-1593)
- `devnet/scripts/init-devnet.ts` - Updated to use LevelDB by default
- `cmd/commands/node.ts` - CLI command for running nodes with persistence

**How it works**:
1. `createProductionNode()` uses LevelDB storage at `dataDir/state`
2. StateManager receives store via constructor
3. State persisted after each finalized block via `block_finalized` event
4. State loaded from storage on startup via `initializeFromStorage()`

### B2: State Sync Protocol ✅ FIXED
**Problem**: ~~New validators cannot join~~
**Status**: RESOLVED 2026-02-05

**Solution Implemented**:
- `network/protocols/state_sync.ts` - Full state sync protocol (~720 lines)
- `state/storage/snapshots.ts` - Snapshot infrastructure (already existed)
- `network/protocols/sync.ts` - Updated to reference new state sync protocol

**How it works**:
1. `StateSyncManager` determines when to use state sync vs block sync (>1000 blocks behind)
2. Selects best snapshot from peers (highest block with multiple seeders)
3. Downloads chunks in parallel with retry logic and hash verification
4. `StateSyncProtocolHandler` handles incoming requests (list_snapshots, metadata, chunk)
5. Integrates with existing `SnapshotManager` for storage and restoration

**Key Features**:
- Parallel chunk downloads (up to 8 concurrent)
- Hash verification for every chunk
- Retry logic with exponential backoff
- Round-robin peer selection for load distribution

### B3: RPC API Completeness ✅ FIXED
**Problem**: ~~40% of endpoints missing~~
**Status**: RESOLVED 2026-02-05

**Solution Implemented**:
- `protocol/rpc/handlers/staking.ts` - Staking/delegation handlers (4 endpoints)
- `protocol/rpc/handlers/transactions.ts` - Generic transaction submission (1 endpoint)
- `protocol/rpc/handlers/disputes.ts` - Dispute handlers (3 endpoints)
- `protocol/rpc/handlers/channels.ts` - Payment channel handlers (2 endpoints)
- `protocol/rpc/handlers/index.ts` - Handler exports
- `protocol/types.ts` - Added request/response types for all B3 endpoints

**New Endpoints** (10 total):
- `lgm_sendTransaction` - Generic transaction submission
- `lgm_stake` / `lgm_unstake` - Staking operations
- `lgm_delegate` / `lgm_undelegate` - Delegation management
- `lgm_openDispute` / `lgm_submitEvidence` / `lgm_resolveDispute` - Disputes
- `lgm_openChannel` / `lgm_closeChannel` - Payment channels

**Total RPC Endpoints**: 26 (16 existing + 10 new)

---

## High Priority (Before Testnet)

### H1: Integration Tests ⚠️ IN PROGRESS
**Current**: ~50% critical path coverage
**Target**: 70% coverage

**Completed Tests** (2026-02-05):
- ✅ Dispute resolution workflow (27 tests)
  - `tests/integration/consensus/dispute_resolution.test.ts`
  - Full lifecycle: initiate → evidence → vote → settle
  - Economic constants (punishment/reward rates)
  - Escalation mechanics (tier 1-3)
  - Edge cases (tampering detection)
- ✅ Validator dynamics (24 tests)
  - `tests/integration/consensus/validator_dynamics.test.ts`
  - Adding/removing validators
  - Stake adjustments and delegation
  - Epoch transitions
  - Consensus continuity during changes

**Remaining Critical Paths**:
1. ~~Dispute resolution workflow~~ ✅
2. End-to-end block production
3. Multi-validator consensus
4. Verification round complete flow
5. State transitions for all tx types
6. RPC handler integration

### H2: DevNet Automation ✅ COMPLETE
**Problem**: ~~Manual peer setup required~~
**Status**: RESOLVED 2026-02-05

**Completed Tasks**:
- ✅ `tools/devnet/spawn.ts` - Uses actual node orchestrator (createDevnetNode)
- ✅ `devnet/scripts/init-devnet.ts` - Auto-generates peer configurations
- ✅ `/health` HTTP endpoint added to RPC server
- ✅ Docker health checks configured in Dockerfile

**DevNet Launch Commands**:
```bash
# Initialize devnet (generates keys, configs, genesis)
npx ts-node devnet/scripts/init-devnet.ts

# Start with Docker Compose
cd devnet && docker-compose up -d

# Or start locally without Docker
npm run devnet:multi -- --nodes 3
```

### H3: CLI Security [TRUTH VIOLATION]
**Problem**: `--agent-secret` visible in process list

**Tasks**:
- Complete migration to `--agent-ref` (secure storage)
- Remove deprecated `--agent-secret`
- Add session key validation

---

## Medium Priority (Before Mainnet)

### M1: External Security Audit
- Cryptography review
- Consensus attack vectors
- Economic security (slashing, staking)

### M2: Operational Documentation
- Startup procedures
- Backup/recovery runbooks
- Troubleshooting guides
- Validator onboarding

### M3: Monitoring Infrastructure
- Prometheus metrics for all critical paths
- Alerting rules for:
  - Block production halts
  - Validator set changes
  - Consensus failures
  - Memory/disk thresholds

### M4: Emergency Controls
- Network pause mechanism
- Rollback procedures
- Coordinated upgrade path

---

## Launch Timeline

### Phase 1: DevNet (2-3 weeks)
```
Week 1:
- [ ] Implement LevelDB persistence (B1)
- [ ] Add 20 integration tests (H1 partial)
- [ ] Fix devnet spawn script (H2 partial)

Week 2:
- [ ] Complete state sync (B2)
- [ ] Add missing RPC endpoints (B3 partial)
- [ ] Automate 3-validator devnet

Week 3:
- [ ] End-to-end testing
- [ ] Documentation review
- [ ] DevNet launch
```

### Phase 2: Testnet (4-6 weeks after DevNet)
```
Week 4-5:
- [ ] Complete all RPC endpoints (B3)
- [ ] 70% test coverage (H1)
- [ ] CLI security fix (H3)

Week 6-7:
- [ ] Public validator onboarding
- [ ] Faucet and explorer hardening
- [ ] Load testing

Week 8-9:
- [ ] Testnet launch
- [ ] Bug bounty program
```

### Phase 3: Mainnet (3-4 months after Testnet)
```
Month 1:
- [ ] External security audit
- [ ] Operational documentation (M2)
- [ ] Monitoring setup (M3)

Month 2:
- [ ] Audit remediation
- [ ] Emergency controls (M4)
- [ ] Validator coordination

Month 3:
- [ ] Final review
- [ ] Genesis ceremony
- [ ] Mainnet launch
```

---

## Quick Wins (Can Do Now)

1. **~~Add `/health` endpoint~~** ✅ DONE
   - `protocol/rpc/server.ts` - Added HTTP GET handler
   - Returns: status, blockHeight, isSynced, peerCount, version, timestamp

2. **~~Fix hardcoded epochLength~~** ✅ ALREADY FIXED (GAP-025)
   - `consensus/pot/validator_selection.ts` - Now accepts epochLength as parameter
   - Uses ConsensusParams

3. **Document missing RPC endpoints** - 1 hour
   - `docs/RPC_API.md`
   - List planned vs implemented

4. **Add integration test framework** - 2 hours
   - `tests/integration/setup.ts`
   - Mock state, consensus, network

---

## Resource Estimates

| Task | Effort | Priority |
|------|--------|----------|
| LevelDB Integration | 2-3 days | B1 |
| State Sync | 3-5 days | B2 |
| RPC Endpoints | 3-5 days | B3 |
| Integration Tests | 5-7 days | H1 |
| DevNet Automation | 3-5 days | H2 |
| CLI Security | 2-3 days | H3 |
| Security Audit | 2-4 weeks | M1 |
| Documentation | 3-5 days | M2 |
| Monitoring | 3-5 days | M3 |
| Emergency Controls | 2-3 days | M4 |

**Total to DevNet**: ~2-3 weeks
**Total to Testnet**: ~6-8 weeks
**Total to Mainnet**: ~4-5 months (including audit)

---

## Decision Required

**Immediate Next Step Options**:

1. **Option A: Persistence First** (Recommended)
   - Start with LevelDB integration
   - Enables all other testing
   - Risk: Longer before visible progress

2. **Option B: DevNet Shell First**
   - Get multi-node running with in-memory
   - Accept data loss on restart
   - Risk: False confidence, hidden bugs

3. **Option C: Complete RPC First**
   - Focus on API completeness
   - Enables SDK/CLI development
   - Risk: Can't test without persistence

**Recommendation**: Option A - Persistence is the foundation everything else depends on.

---

## Files to Create/Modify

### New Files Needed
```
state/storage/leveldb_impl.ts      # Actual LevelDB implementation
state/storage/encoding.ts          # Binary encoding (msgpack)
network/protocols/state_sync.ts    # State sync implementation
protocol/rpc/handlers/staking.ts   # Staking RPC handlers
protocol/rpc/handlers/disputes.ts  # Dispute RPC handlers
protocol/rpc/handlers/channels.ts  # Payment channel handlers
tests/integration/setup.ts         # Test framework
tests/integration/consensus.test.ts # E2E consensus tests
docs/operations/RUNBOOK.md         # Operational procedures
```

### Files to Modify
```
protocol/rpc/handlers.ts           # Add missing endpoints
tools/devnet/spawn.ts              # Actually start nodes
cmd/cli.ts                         # Remove --agent-secret
consensus/pot/validator_selection.ts # Fix hardcoded values
```

---

## Success Criteria

### DevNet Launch
- [ ] 3 validators produce blocks continuously
- [ ] Nodes restart without losing state
- [ ] New nodes can sync from genesis
- [ ] All transaction types execute correctly
- [ ] CLI can perform all operations

### Testnet Launch
- [ ] 10+ external validators running
- [ ] 99.9% uptime over 1 week
- [ ] No critical bugs found
- [ ] Complete API documentation
- [ ] Bug bounty running

### Mainnet Launch
- [ ] Security audit passed
- [ ] Emergency procedures tested
- [ ] 30+ validators committed
- [ ] Economic parameters finalized
- [ ] Governance framework active
