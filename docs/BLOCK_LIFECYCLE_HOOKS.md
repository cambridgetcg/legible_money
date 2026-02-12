# Block Lifecycle Hooks — Implementation Insights

> **Legible Money Cosmos SDK Chain** | 24 custom modules
> Complete ABCI++ lifecycle analysis: PreBlocker → BeginBlock → DeliverTx → EndBlock → Commit

---

## 1. Full Block Lifecycle

```
                     ┌─────────────────────────────┐
                     │     PREVIOUS BLOCK VOTE      │
                     │                              │
                     │  ExtendVote()                │
                     │  ├─ VRF sortition per round  │
                     │  ├─ Generate commitment hash │
                     │  └─ Reveal prior commitments │
                     │                              │
                     │  VerifyVoteExtension()       │
                     │  ├─ Validate VRF proofs      │
                     │  ├─ Check hash format        │
                     │  └─ Verify reveal↔commitment │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │      BLOCK PROPOSAL          │
                     │                              │
                     │  PrepareProposal()           │
                     │  ├─ Emergency halt filter    │
                     │  ├─ Collect vote extensions  │
                     │  ├─ Sort deterministically   │
                     │  ├─ Inject at tx[0] (0xVEX)  │
                     │  └─ Fill mempool txs to 33M  │
                     │                              │
                     │  ProcessProposal()           │
                     │  ├─ Validate injection tx    │
                     │  ├─ ValidateBasic() all msgs │
                     │  └─ Emergency halt check     │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │        PreBlocker            │
                     │  (app.go:1043-1051)          │
                     │                              │
                     │  Process vote extension      │
                     │  injection tx:               │
                     │  ├─ Detect 0x00VEX prefix    │
                     │  ├─ Decode commitments       │
                     │  ├─ Store in knowledge keeper│
                     │  └─ Decode & store reveals   │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │     BeginBlock (27 phases)   │
                     │     ──────────────────────   │
                     │  7 Cosmos SDK standard       │
                     │  20 Legible custom modules   │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │        DeliverTx             │
                     │  ──────────────              │
                     │  AnteHandler (16 decorators) │
                     │  → Msg Handler execution     │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │      EndBlock (25 phases)    │
                     │      ────────────────────    │
                     │  5 Cosmos SDK standard       │
                     │  20 Legible custom modules   │
                     └──────────────┬──────────────┘
                                    │
                     ┌──────────────▼──────────────┐
                     │          Commit              │
                     │  State committed to disk     │
                     └─────────────────────────────┘
```

---

## 2. ABCI++ Vote Extensions (Proof of Truth Engine)

### ExtendVote (app/vote_extensions.go:40-101)

Validators attach PoT verification data to their CometBFT votes.

**Flow per active round**:

```
For each active verification round at current height:
  │
  ├─ COMMIT phase:
  │   ├─ Generate VRF: knowledgekeeper.GenerateValidatorVRF(privkey, claimID, height, prevHash)
  │   ├─ Check selection: vrfcrypto.IsValidatorSelected(vrfOutput, stake, totalStake, minVerifiers)
  │   ├─ If NOT selected → skip (VRF sortition failed)
  │   ├─ Evaluate claim → verdict="accept", confidence=0.8 (Phase 2 stub)
  │   ├─ Generate salt: 16 random bytes (hex)
  │   ├─ Commitment hash: SHA-256(roundID || verdict || confidence || salt)
  │   └─ Store locally in memory (LocalCommitmentStore)
  │
  └─ REVEAL phase:
      ├─ Lookup prior commitment from LocalCommitmentStore
      ├─ If not found → skip (validator restarted or didn't commit)
      └─ Return: verdict, confidence, salt
```

**Critical detail**: LocalCommitmentStore is **in-memory only**. Validator restart between commit and reveal forfeits that round's rewards. Phase 3 will add persistent storage.

### VerifyVoteExtension (app/vote_extensions.go:219-309)

Validates other validators' vote extensions before accepting votes.

| Check | What | Rejection? |
|-------|------|-----------|
| Empty extension | Always valid (non-validators) | No |
| Commitment hash | 64 hex chars (SHA-256) | Yes if malformed |
| VRF proof | `KnowledgeKeeper.VerifyValidatorVRFSelection()` | Yes if invalid |
| Reveal verdict | Must be "accept", "reject", or "abstain" | Yes if other |
| Reveal confidence | Must be <= 1,000,000 | Yes if exceeded |
| Reveal↔commitment match | Recompute SHA-256 and compare to stored hash | Yes if mismatch |

### PrepareProposal (app/abci.go:134-204)

Block proposer builds the block:

1. **Emergency halt check** — if `EmergencyKeeper.IsHalted()`, filter to emergency-only txs
2. **Collect vote extensions** from `req.LocalLastCommit.Votes`
3. **Sort deterministically** by RoundID then Validator address (consensus-critical)
4. **Encode with magic prefix** `0x00VEX` and inject at position `tx[0]`
5. **Fill remaining space** with mempool txs up to `BlockGasLimit` (33,333,333)

### ProcessProposal (app/abci.go:212-290)

Validators verify proposed block:
- Skip vote extension injection tx (magic prefix)
- Reject if injection is malformed
- `ValidateBasic()` on all other transactions
- Emergency halt enforcement

---

## 3. BeginBlock — Module-by-Module

### Execution Order & Logic

| # | Module | Logic | Cross-Module? | Batch Limit |
|---|--------|-------|---------------|-------------|
| 1 | upgrade | SDK standard | — | — |
| 2 | distribution | SDK standard | — | — |
| 3 | slashing | SDK standard | — | — |
| 4 | evidence | SDK standard | — | — |
| 5 | staking (SDK) | SDK standard | — | — |
| 6 | ibc | SDK standard | — | — |
| 7 | ibc-transfer | SDK standard | — | — |
| 8 | **emergency** | Check ceremony progress, finalize/fail | No | — |
| 9 | **vesting_rewards** | **Mint block rewards**, 78/22 split | Yes (staking, bank) | — |
| 10 | **tokens** | Sweep expired tokens, distribute RAI | No | SweepBatchSize |
| 11 | **staking** (legible) | Process mature unbondings | Yes (bank, tokens) | — |
| 12 | **gov** (legible) | Tally expired LIP/disbursement votes | No | — |
| 13 | **research** | Expire bounties past deadline | No | — |
| 14 | **ontology** | NO-OP | — | — |
| 15 | **disputes** | NO-OP | — | — |
| 16 | **evidence_mgmt** | NO-OP | — | — |
| 17 | **claiming_pot** | Expire old pots | No | — |
| 18 | **channels** | Resolve expired channel disputes | No | — |
| 19 | **schedule** | NO-OP | — | — |
| 20 | **partnership** | Expire formations, lift freezes, expire coercion | No | — |
| 21 | **tree** | Expire opportunity seeds | No | — |
| 22 | **bvm** | Execute pending contract schedules | Potentially all | — |
| 23 | **home** | Deadman switches, session cleanup | Potentially (auth/bank) | — |
| 24 | **qualification** | Promote probationary, expire stale | No | — |
| 25 | **capture_defense** | Expire qualifications, decay reputations | No | MaxExpirationsPerBlock |
| 26 | **capture_challenge** | Advance challenge phases | No | — |
| 27 | **knowledge** | **Advance verification round phases** | No | — |

### Detailed Implementations

#### emergency (x/emergency/module.go:128-154)
```
GetActiveCeremony() → if found:
  CheckCeremonyProgress(ceremony.Id)
    → if finalized: HandleCeremonyFinalization()
    → if failed: HandleCeremonyFailure()
Error: logged, returns nil
```

#### vesting_rewards (x/vesting_rewards/module.go:130-162) — CRITICAL
```
Skip if height <= 1 (genesis)
Get block proposer from header
Get activeValidatorCount from staking keeper
Check hasTransactions (PoT: 0% for empty blocks)

DistributeBlockReward():
  baseReward = 11 LGM (11000000000000000000 ulgm)
  scale = min(1, activeValidators / 22)  ← 22 target validators
  Split: 78% producer, 22% research fund
  bank.MintCoins()
  bank.SendCoinsFromModuleToAccount() → producer
  bank.SendCoinsFromModuleToModule() → research fund
  Save distribution record
```

**Only minting operation in entire BeginBlock.** Must run before any module that transfers funds.

#### tokens (x/tokens/module.go:131-161)
```
SweepExpiredTokens(maxAccounts=SweepBatchSize):
  Iterate expired periods (current - period > PeriodsUntilExpiry)
  For each expired child account:
    Transfer balance → RAI holding pool
    Delete child account
  Return total swept

ShouldDistributeRAI() → if yes:
  GetEligibleRAIRecipients()
  DistributeRAI(recipients)
```

**Must run before staking** — prevents re-crediting already-expired tokens during unbonding.

#### staking/legible (x/staking/keeper/keeper.go:712-761)
```
GetMatureUnbondings(currentHeight)
For each mature unbonding:
  bank.SendCoinsFromModuleToAccount() → delegator
  if OriginPeriod > 0:
    tokens.CreditBalanceToSpecificPeriod() ← preserves original period
  else:
    tokens.CreditBalance()
  Mark unbonding status = "completed"
```

**Period preservation**: Unbonded tokens retain their original period, preventing expiration gaming.

#### gov/legible (x/gov/module.go:124-147)
```
For each LIP in StageVoting with expired period:
  Tally votes (support * 10000 / total)
  If support >= threshold → accepted
  Else → rejected
  Emit "lip_resolved" event

For each disbursement in StageVoting with expired period:
  Tally votes
  If approved → set TimelockExpiry
  Else → rejected
  Emit "disbursement_resolved" event
```

#### research (x/research/module.go:117-137)
```
Iterate all bounties:
  If status == Open AND currentBlock > deadline:
    Set status = Expired
    Emit "bounty_expired" event
```

#### claiming_pot (x/claiming_pot/module.go:130-143)
```
Iterate all pots:
  If Active AND currentBlock > ExpiresAtBlock:
    Set status = Expired
```

#### channels (x/channels/module.go:122-148)
```
GetExpiredDisputes(currentBlock)
For each expired dispute:
  Accept disputed state as final (ch.Spent = d.DisputedSpent)
  Revert channel status = Open
  Mark dispute resolved with reason = "timeout"
```

#### partnerships (x/partnerships/module.go:134-147)
```
ExpireFormations()    ← Forming → Dissolved (past window)
LiftExpiredFreezes()  ← Remove safety freezes past 222 blocks
ExpireCoercionSignals() ← Expire unresolved coercion past review window
```

#### tree (x/tree/module.go:122-134)
```
GetExpiredSeeds(currentBlock)
For each seed: Set status = SeedExpired
```

#### bvm (x/bvm/module.go:122-126)
```
ExecutePendingSchedules() ← runs all contracts due at current block
```
**Warning**: Contract execution can call ANY module via host functions (knowledge, auth, schedule keepers).

#### home (x/home/module.go:130-135)
```
CheckDeadmanSwitches()   ← inactivity triggers emergency recovery
CleanupExpiredSessions() ← delete expired session records
```

#### qualification (x/qualification/module.go:113-117)
```
CheckAndExpireQualifications():
  Probationary past period → promote to Active
  Active past expiry → mark Expired (weight=0)
```

#### capture_defense (x/capture_defense/module.go:128-156)
```
Iterate qualifications (max MaxExpirationsPerBlock):
  If Active/Probationary past expiry: status=Expired, weight=0

If currentBlock % DecayIntervalBlocks == 0:
  ApplyPeriodicDecay() ← reduce reputation scores
```

#### capture_challenge (x/capture_challenge/module.go:128-135)
```
AdvanceChallengePhases(currentBlock):
  Open → Evidence → Voting → Resolved (by block height)
```

#### knowledge (x/knowledge/module.go:131-141) — CRITICAL
```
AdvanceRoundPhases(currentHeight):
  Commit → Reveal → Aggregate → Completed
```

**Runs last in BeginBlock.** Depends on PreBlocker having already stored commitments/reveals from vote extensions.

---

## 4. EndBlock — Module-by-Module

### Execution Order & Logic

| # | Module | Logic | Cross-Module? | Periodic? |
|---|--------|-------|---------------|-----------|
| 1 | gov (SDK) | SDK standard | — | — |
| 2 | staking (SDK) | SDK standard | — | — |
| 3-5 | ibc, transfer, feegrant | SDK standard | — | — |
| 6 | **staking** (legible) | Tier transitions | No | — |
| 7 | **knowledge** | Cleanup rounds, expire facts, reverify | No | 1000-block penalty |
| 8 | **tokens** | NO-OP | — | — |
| 9 | **emergency** | Reset epoch counters | No | EpochBlocks |
| 10 | **gov** (legible) | NO-OP | — | — |
| 11 | **research** | NO-OP | — | — |
| 12 | **vesting_rewards** | Release vested amounts | No | — |
| 13 | **ontology** | Expire domain proposals, delete proposed domains | No | — |
| 14 | **disputes** | Advance disputes (Evidence→Voting→Resolved) | No | — |
| 15 | **evidence_mgmt** | Expire stale evidence, prune oracles | No | — |
| 16 | **claiming_pot** | NO-OP | — | — |
| 17 | **channels** | NO-OP | — | — |
| 18 | **schedule** | Process due schedules, cleanup expired | No | 100-block cleanup |
| 19 | **partnership** | NO-OP | — | — |
| 20 | **tree** | NO-OP | — | — |
| 21 | **bvm** | NO-OP | — | — |
| 22 | **home** | Cleanup old alerts | No | — |
| 23 | **qualification** | NO-OP | — | — |
| 24 | **capture_defense** | Run capture detection | No | DetectionIntervalBlocks |
| 25 | **capture_challenge** | Auto-fund bounty pools | No | Every 777 blocks |
| 26 | **billing** | Expire old API quotes | No | — |
| 27 | **partnerships** | Settle cooling partnerships (Cooling→Dissolved) | No | — |

### Detailed Implementations

#### staking/legible (x/staking/keeper/keeper.go:777-808)
```
IterateValidators():
  For each validator: CheckTierTransition()
    If tier changed → update validator, log transition
  Persist all transitioned validators
```

#### knowledge (x/knowledge/module.go:148-179)
```
CleanupExpiredRounds(maxAge=100)     ← delete rounds >100 blocks old
ExpireStaleFactsInEndBlocker(max=50) ← decay below threshold → expired
TriggerReverificationScan(max=10)    ← emit reverification events

Every 1000 blocks:
  ApplyCyclePenalties() ← penalize validators with stale verification
```

#### emergency (x/emergency/module.go:157-172)
```
If currentBlock % EpochBlocks == 0:
  ResetEpochCounters() ← clear per-epoch proposal rate limits
```

#### vesting_rewards (x/vesting_rewards/module.go:164-181)
```
GetAllActiveVestingSchedules()
For each active schedule:
  UpdateClaimableAmount() ← release vested tokens based on elapsed time
```

#### ontology (x/ontology/keeper/proposals.go:117-137)
```
ProcessExpiredProposals():
  Iterate proposals with status="active" past ExpiresAt
  Set status = "expired"
  If proposed domain exists → delete domain from state
```

#### disputes (x/disputes/keeper/disputes.go:283-314)
```
ProcessExpiredDisputes():
  Evidence status past EvidenceDeadline → TransitionToVoting()
  Voting status past VotingDeadline → TallyVotes() → Resolve
```

#### evidence_mgmt (x/evidence_mgmt/keeper/evidence.go:193-210)
```
ExpireStaleEvidence():
  Submitted evidence past AuditWindow → ResolveEvidence()
PruneInactiveOracles():
  Remove oracles with no recent activity
```

#### schedule (x/schedule/keeper/process.go:14-71)
```
ProcessDueSchedules():
  Get processes with NextExecuteAt <= currentBlock
  For each: execute action, deduct fee, increment count
  If repeating → recalculate next time
  If done → mark complete

Every 100 blocks:
  CleanupExpiredProcesses() ← remove old completed/cancelled
```

#### capture_defense (x/capture_defense/module.go:159-170)
```
Every DetectionIntervalBlocks:
  RunCaptureDetection() ← analyze validator behavior for capture risk
```

#### capture_challenge (x/capture_challenge/module.go:138-148)
```
Every 777 blocks:
  AutoFundBountyPools() ← allocate funds to high-risk domain bounties
```

#### billing (x/billing/module.go:134-159)
```
IterateQuotes():
  If unexecuted AND past ValidUntil block → DeleteQuote()
```

#### partnerships (x/partnerships/keeper/exit.go:107-121)
```
SettleCoolingPartnerships():
  Cooling status past CooldownEnd → status = Dissolved
```

---

## 5. BeginBlock vs EndBlock Split Pattern

| Operation Type | Phase | Rationale |
|---------------|-------|-----------|
| **Minting** | BeginBlock | New tokens available for block's transactions |
| **Unbonding returns** | BeginBlock | Funds available before block processing |
| **Vote tallying** | BeginBlock | Governance resolved before dependent operations |
| **Expiration of active entities** | BeginBlock | Prevent interactions with expired state |
| **Tier transitions** | EndBlock | Reflect verification activity from current block |
| **Round cleanup** | EndBlock | After all block activity is finalized |
| **Fact expiration** | EndBlock | After any last-block citations/references |
| **Vesting release** | EndBlock | Include current block's time in calculation |
| **Capture detection** | EndBlock | Analyze full block's behavior pattern |
| **Schedule execution** | EndBlock | After all regular txs processed |

---

## 6. Ordering Dependencies (Critical Path)

### BeginBlock Critical Order

```
vesting_rewards BEFORE staking
  │  Reason: Mint rewards before unbonding returns (fresh supply)
  │
tokens BEFORE staking
  │  Reason: Sweep expired tokens before re-crediting unbonded tokens
  │          Prevents expiration gaming
  │
emergency BEFORE knowledge
  │  Reason: Halt state must be set before rounds advance
  │
ALL modules BEFORE knowledge
  │  Reason: Knowledge is the PoT engine — needs all state settled
  │          Depends on PreBlocker having stored vote extensions
```

### EndBlock Critical Order

```
staking BEFORE knowledge
  │  Reason: Tier transitions before round cleanup
  │
knowledge BEFORE disputes
  │  Reason: Fact state must be current before dispute resolution
  │
capture_defense BEFORE capture_challenge
  │  Reason: Detection results available for challenge processing
```

---

## 7. Error Handling Philosophy

**Universal pattern: Log and Continue**

No module panics in BeginBlock or EndBlock. Every module follows:
```go
if err != nil {
    am.keeper.Logger(ctx).Error("description", "error", err)
    // continue processing, return nil
}
```

This ensures **chain liveness** — a single module's transient failure cannot halt the chain. The tradeoff is that failed operations silently skip, requiring monitoring.

### Modules with NO-OP BeginBlock (5)
ontology, disputes, evidence_mgmt, schedule, gov(EndBlock)

### Modules with NO-OP EndBlock (11)
tokens, gov, research, claiming_pot, channels, tree, bvm, qualification, partnership

---

## 8. Periodic Operations

| Module | Interval | Operation |
|--------|----------|-----------|
| **knowledge** | Every 1000 blocks (~42 min) | Cycle penalties for stale validators |
| **emergency** | Every EpochBlocks | Reset proposal rate counters |
| **capture_defense** | Every DecayIntervalBlocks | Reputation score decay |
| **capture_defense** | Every DetectionIntervalBlocks | Run capture detection scan |
| **capture_challenge** | Every 777 blocks (~33 min) | Auto-fund bounty pools |
| **schedule** | Every 100 blocks (~4 min) | Cleanup expired processes |

---

## 9. Amortized/Batched Operations

| Module | Limit | Parameter | Purpose |
|--------|-------|-----------|---------|
| **tokens** | SweepBatchSize | Per block | Prevent block time blowup with many expired accounts |
| **capture_defense** | MaxExpirationsPerBlock | Per block | Bound qualification expiration processing |
| **knowledge** | 100 | CleanupExpiredRounds | Cap round deletion per block |
| **knowledge** | 50 | ExpireStaleFactsInEndBlocker | Cap fact expiration per block |
| **knowledge** | 10 | TriggerReverificationScan | Cap reverification events per block |

---

## 10. Cross-Module Dependencies in Block Hooks

### BeginBlock Cross-Module Calls

| Module | Target | Method | Direction |
|--------|--------|--------|-----------|
| vesting_rewards | staking | GetActiveValidatorCount() | Read |
| vesting_rewards | bank | MintCoins() | Write |
| vesting_rewards | bank | SendCoinsFromModuleToAccount() | Write |
| vesting_rewards | bank | SendCoinsFromModuleToModule() | Write |
| staking | bank | SendCoinsFromModuleToAccount() | Write |
| staking | tokens | CreditBalanceToSpecificPeriod() | Write |
| staking | tokens | CreditBalance() | Write |
| bvm | * | ExecutePendingSchedules() | Write (any) |

### EndBlock Cross-Module Calls

| Module | Target | Method | Direction |
|--------|--------|--------|-----------|
| capture_challenge | bank | AutoFundBountyPools() | Write (likely) |

**Key insight**: EndBlock is almost entirely self-contained. Only 1 module makes cross-module calls. BeginBlock has 3 modules with cross-module writes.

---

## 11. Gas & Fee Constants

```
BlockGasLimit       = 33,333,333
TxGasLimit          = 11,111,111
SelfInvocationLimit =  5,555,555
MinGasLimit         =     22,222

Fee Split: 22% research fund, 78% validators
Burn Rate: 5% at penalty sites

System transactions (fee-exempt):
  verification_reward, slash_validator, epoch_transition,
  emergency_halt, emergency_revert, emergency_resume, block_reward
```

---

## 12. Findings & Recommendations

### FINDING-1: Vote Extension Persistence Gap (MEDIUM)

LocalCommitmentStore is **in-memory only**. A validator restart between commit and reveal phases forfeits that round entirely — no reveal means no reward and potential VRF slot waste.

**Impact**: Reduces effective verification participation during validator restarts/upgrades.

**Recommendation**: Add persistent storage (LevelDB) for local commitments before mainnet. Currently marked as Phase 3.

### FINDING-2: Claim Evaluation Stub (HIGH — for mainnet)

ExtendVote always returns `verdict="accept", confidence=0.8`. Real claim evaluation is not implemented.

**Impact**: All claims auto-accepted with 80% confidence. PoT consensus is non-functional for truth-finding until this is implemented.

**Recommendation**: Phase 3 priority — integrate ontology lookup, fact graph analysis, and external oracle queries.

### FINDING-3: BeginBlock Token-Staking Ordering is Critical (VERIFIED CORRECT)

```
tokens.SweepExpiredTokens() → THEN → staking.ProcessMatureUnbondings()
```

This prevents a scenario where expired tokens are re-credited during unbonding. Period preservation (`CreditBalanceToSpecificPeriod`) ensures unbonded tokens don't reset their expiration clock.

### FINDING-4: No BeginBlock/EndBlock Panics (STRENGTH)

All 24 custom modules use log-and-continue error handling. This guarantees chain liveness even with transient keeper failures.

**Tradeoff**: Silent failures require external monitoring. Consider adding Prometheus metrics for failed BeginBlock/EndBlock operations.

### FINDING-5: Knowledge Runs Last in BeginBlock (VERIFIED CORRECT)

`AdvanceRoundPhases()` must run after PreBlocker stores vote extension data. Running last ensures all validator commitments and reveals are available before phase transitions.

### FINDING-6: PoT Empty Block Penalty (DESIGN INSIGHT)

`vesting_rewards` checks `hasTransactions` — empty blocks get 0% reward when `EmptyBlockRewardRate == 0`. This incentivizes proposers to include verification transactions, aligning block production with truth-finding.

### FINDING-7: BVM BeginBlock is Unbounded (LOW RISK)

`ExecutePendingSchedules()` has no per-block batch limit. A burst of scheduled contracts could extend block time.

**Mitigation**: Gas limits constrain total execution. The schedule module limits per-account active schedules via `MaxSchedulesAllowed()`.

### FINDING-8: 5 NO-OP BeginBlock Modules (INFO)

ontology, disputes, evidence_mgmt, schedule have their logic entirely in EndBlock. They're registered in BeginBlock ordering purely for the module manager. Consider documenting this explicitly to prevent confusion.

### FINDING-9: 11 NO-OP EndBlock Modules (INFO)

tokens, gov, research, claiming_pot, channels, tree, bvm, qualification, partnership have their logic entirely in BeginBlock. Same registration pattern.

---

*Generated 2026-02-10 via PP block lifecycle review*
