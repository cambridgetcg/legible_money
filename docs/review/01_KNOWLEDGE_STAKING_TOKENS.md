# Core PoT Cluster — Knowledge + Staking + Tokens

> These three modules form the heart of Legible Money's Proof of Truth consensus. Knowledge drives verification, Staking provides economic security and validator management, and Tokens implements the expirable balance system that prevents hoarding.
>
> **Related**: [00 — Module Overview](./00_MODULE_OVERVIEW.md) for dependency graph and app wiring. [05 — Ontology](./05_ONTOLOGY.md) for stratum-based confidence ceilings used by knowledge. [02 — Disputes](./02_DISPUTES_EVIDENCE.md) for post-verification fact challenges. [03 — Auth](./03_AUTH_IDENTITY.md) for the ante decorators that gate these transactions.

---

## x/knowledge — Proof of Truth Engine

**Store key**: `knowledge`
**Store prefixes**: `0x01` facts, `0x02` claims, `0x03` rounds, `0x04` references, `0x05` domain-fact index, `0x06` params

### What It Does

The knowledge module is where "truth verification IS the useful work." Users submit knowledge claims, validators verify them through a commit-reveal process, and accepted claims become facts in the on-chain knowledge graph.

### Key Types (`x/knowledge/types/types.go`)

**Fact** — Verified knowledge stored permanently:
- `Id`, `ClaimText`, `Domain`, `Category` (9 epistemic categories)
- `Confidence` (0–1,000,000 representing 0–100%)
- `Status`: verified, contested, falsified, expired
- `References[]` — links to other facts (knowledge graph)
- `VerifiedAtHeight`, `VerifierCount`, `BaseConfidence`

**Claim** — Pending knowledge awaiting verification:
- `Id`, `ClaimText`, `Domain`, `Category`, `Submitter`, `StakeAmount`
- `Status`: pending, in_verification, accepted, rejected, challenged, expired
- `ContentHash` — SHA-256 of (text + domain + category) for deduplication

**VerificationRound** — The commit-reveal process:
- `Phase`: commit → reveal → aggregation → complete
- `CommitDeadlineHeight`, `RevealDeadlineHeight`
- `Commitments` map — validator address → hash
- `Reveals` map — validator address → verdict + confidence + salt
- `SelectedValidators[]` — VRF-selected participants
- `FinalVerdict`, `FinalConfidence`

**Params** (defaults):
- `MinClaimTextLength`: 20 chars
- `MaxClaimRewardMultiplier`: 2x stake
- `CommitPhaseDuration`: 4 blocks (~10s)
- `RevealPhaseDuration`: 4 blocks (~10s)
- `AggregationPhaseDuration`: 3 blocks (~7.5s)
- `AcceptanceThreshold`: 77%
- `QuorumThreshold`: 66%
- `MaxValidatorsPerRound`: 22

### Claim-to-Fact Lifecycle

```
1. SUBMIT CLAIM (MsgSubmitClaim)
   ├─ Validate text length, domain exists, category valid
   ├─ Compute content hash → reject duplicates
   ├─ Deduct stake (BankKeeper + TokensKeeper period debit)
   ├─ Create Claim with deterministic ID = SHA256(submitter || hash || height)
   └─ Create VerificationRound (phase = "commit")

2. COMMIT PHASE (MsgSubmitCommitment) — 4 blocks
   ├─ Validator generates VRF → checks if selected
   ├─ Computes hash = SHA256(roundID || verdict || confidence || salt)
   ├─ Submits commitment hash + VRF proof
   └─ Keeper stores in round's Commitments map

3. PHASE TRANSITION (BeginBlocker)
   └─ AdvanceRoundPhases() detects commit deadline passed → transitions to reveal

4. REVEAL PHASE (MsgSubmitReveal) — 4 blocks
   ├─ Validator submits: verdict, confidence, salt
   ├─ Keeper verifies hash(roundID||verdict||conf||salt) == stored commitment
   ├─ Stores reveal in round's Reveals map
   └─ IF reveals >= MinVerifiers (3): early aggregation + completion

5. AGGREGATION (BeginBlocker or early trigger)
   ├─ Gather effective stake per revealing validator
   ├─ Weight verdicts: weightedConf = confidence * stake / 1M
   ├─ Sum: acceptWeight, rejectWeight, abstainWeight
   ├─ Check quorum: reveals / selected >= 66%
   ├─ Apply thresholds: >=77% → verdict holds, >=55% → contested, <55% → inconclusive
   └─ Calculate rewards (+100K ulgm correct) / slashes (-1% wrong, -2% missed reveal)

6. FACT CREATION (if accepted)
   ├─ Fact.Id = SHA256(claimID || roundID)
   ├─ Copy claim fields, set confidence from aggregation
   ├─ Fact.Status = "verified"
   └─ Store in KV (prefix 0x01)

7. CHALLENGE (MsgChallengeFact — anytime after verification)
   ├─ Deduct challenger's stake
   ├─ Mark fact as "contested"
   └─ Create NEW verification round for reverification
```

### VRF Validator Selection (`x/knowledge/crypto/vrf.go`)

ECVRF implementation using Ed25519:

1. **Seed**: `SHA256("LGM.vrf.seed.v1:verification:" || claimID || blockHeight || prevBlockHash)`
2. **Hash-to-curve**: Repeatedly SHA512 until valid Ed25519 curve point
3. **Gamma**: `sk * H` (secret key times hashed point)
4. **Proof**: Schnorr-style `(Gamma, c, s)` — 96 bytes
5. **Output**: `SHA256(Gamma)` — 32 bytes, deterministic
6. **Selection**: `outputNum * totalStake < stake * targetCount * 2^64`

Selection probability is proportional to stake. Tier 0/1 validators use 111 LGM virtual stake (gives apprentice validators non-zero selection probability without real stake).

### Round Phase Automation (`x/knowledge/keeper/phases.go`)

`AdvanceRoundPhases()` is called every `BeginBlock`. For each active round:

- If current height > commit deadline → transition to reveal
- If current height > reveal deadline → transition to aggregation (check quorum)
- If current height > reveal + aggregation duration → perform aggregation and complete
- Expired rounds: slash non-revealers (-1% stake), aggregate available reveals

`CleanupExpiredRounds()` runs every `EndBlock` — deletes completed rounds older than 100 blocks.

### Message Handlers (`x/knowledge/keeper/msg_server.go`)

| Handler | Gas | Action |
|---------|-----|--------|
| `SubmitClaim` | 100K | Validate, dedup, stake, create claim + round |
| `SubmitCommitment` | 30K | Verify commit phase, validate hash format, store |
| `SubmitReveal` | 40K | Verify reveal phase, hash-match commitment, store, maybe aggregate |
| `ChallengeFact` | 80K | Deduct stake, mark contested, new round |
| `AddFact` | 120K | Governance-only direct fact creation |
| `ProposeDomain` | 80K | Create proposed domain, deduct stake |
| `EndorseDomainProposal` | 30K | Increment endorsements, activate if threshold met |
| `ChallengeDomainProposal` | 50K | Increment challenges |
| `RegisterStratum` | 60K | Phase 2 placeholder |
| `UpdateParams` | — | Governance-only parameter update |

### Queries (`x/knowledge/keeper/grpc_query.go`)

| Query | Returns |
|-------|---------|
| `Params` | Module configuration |
| `Fact(factId)` | Single fact |
| `Facts(limit, offset)` | Paginated facts (max 100) |
| `FactsByDomain(domain)` | All facts in domain |
| `Claim(claimId)` | Single claim |
| `VerificationRound(roundId)` | Round details |
| `ActiveRounds` | All non-complete rounds |
| `Domain(name)` | Domain by name |

---

## x/staking — Graduated Validator Tiers

**Store key**: `legible_staking`
**Store prefixes**: `0x01` validators, `0x02` delegations, `0x03` unbondings, `0x04` validator-by-DID index, `0x05` tier config, `0x06` params

### What It Does

Manages a 4-tier validator system where advancement depends on both stake AND verification performance. Integrates deeply with the tokens module for period-preserving unbonding.

### The Four Tiers

| Tier | Name | Stake Required | Performance Required | Reward Multiplier | Epistemic Access |
|------|------|---------------|---------------------|-------------------|-----------------|
| 0 | Apprentice | 0 LGM | None | 0.1x | protocol, analytic, formal |
| 1 | Verified | 0 LGM | 22 verifications @ 77% accuracy | 0.5x | + empirical, historical |
| 2 | Bonded | 1,111 LGM | None beyond stake | 1.0x | All categories |
| 3 | Guardian | 11,111 LGM | 1,111 verifications @ 77% accuracy, 0 slashes | 1.5x | All + governance |

- Max 111 apprentice validators (Sybil defense)
- Tier 0/1 get 111 LGM virtual stake for VRF selection probability
- Guardian tier is the strictest — zero slash tolerance

### Key Types (`x/staking/types/types.go`)

**LegibleValidator**:
- `OperatorAddress`, `Did`, `ConsensusPubkey`
- `Stake` (self-bonded), `DelegatedStake`, `TotalStake` = Stake + DelegatedStake
- `Commission` (basis points, 1000 = 10%)
- `Tier` (0–3), `IsActive`
- `VerificationsCompleted`, `VerificationsCorrect` — performance tracking
- `SlashCount` — deactivates at 3+
- `Reputation` (0–1,000,000)

**UnbondingEntry**:
- `CompletesAtHeight` = initiated + 268,560 blocks (~7 days)
- `OriginPeriod` — **captures token period at undelegation time** for expiration preservation

### Keeper Key Methods

**Tier Management**:
- `ComputeValidatorTier()` — compute tier from stake + performance
- `CheckTierTransition()` — detect when tier changes, auto-update
- `GetEffectiveSelectionStake()` — returns virtual stake (11 LGM) for Tier 0/1, actual for Tier 2/3
- `IsTierEligibleForCategory()` — enforce epistemic category restrictions

**Performance Tracking**:
- `RecordVerification(addr, correct)` — increment counters, trigger tier check
- `SlashValidator(addr, amount, reason)` — reduce stake, deactivate at 3+ slashes

**BeginBlocker**:
- Process mature unbondings → transfer from module to delegator
- **Critical**: Credits tokens module with `OriginPeriod` preservation (prevents expiration reset)

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `RegisterValidator` | 100K | Zero-stake registration allowed, auto-tier, apprentice cap check |
| `UpdateValidatorStake` | 60K | Increase → transfer + debit period; Decrease → unbonding entry |
| `Delegate` | 60K | External delegation, update delegated/total stake, tier check |
| `Undelegate` | 60K | Create UnbondingEntry with OriginPeriod, 7-day wait |
| `Redelegate` | 80K | Instant move between validators, both auto-tier check |

---

## x/tokens — Expirable Balances & RAI

**Store key**: `tokens`
**Store prefixes**: `0x01` accounts, `0x02` allowances, `0x03` RAI, `0x04` sweep index, `0x05` params

### What It Does

Every LGM token has an expiration date based on when it was minted/received. Tokens are tracked in "period balances" — segregated buckets by period. Expired tokens flow to the Regenerative Agent Income (RAI) pool for redistribution.

### Period System

```
period = floor(blockHeight / periodBlocks)    // periodBlocks = 34,272 (~1 day)
expiresAt = (period + periodsUntilExpiry + 1) * periodBlocks  // periodsUntilExpiry = 365 (~1 year)
isExpired = (currentPeriod - period) > periodsUntilExpiry
```

### Key Types (`x/tokens/types/types.go`)

**PeriodBalance** — tokens in one period:
- `Period` (index), `Balance` (ulgm string), `ExpiresAt` (block height), `Swept` (bool)

**ExpirableAccount** — per-address state:
- `PeriodBalances[]` — sorted by period, tracks age
- `TotalBalance` — sum of active (non-expired) period balances
- `TotalExpired` — cumulative swept amount
- `AccountType` — "standard", "validator", "foundation", "research"

**RAIHoldingAccount** — swept tokens pool:
- `Balance` — available for distribution
- `TotalReceived`, `TotalDistributed` — historical counters

**Params** (defaults):
- `PeriodBlocks`: 34,272 (~1 day)
- `PeriodsUntilExpiry`: 365 (~1 year)
- `DustThreshold`: 1,000 ulgm
- `SweepBatchSize`: 100 accounts/block
- `FreshExpirationOnTransfer`: **false** (preserve sender's periods)

### Period-Based Operations

**CreditBalance(addr, amount)** — mint to current period (fresh expiration):
- Used for: rewards, system credits, fees
- Tokens expire ~1 year from now

**CreditBalanceToSpecificPeriod(addr, amount, period)** — credit at a specific period:
- Used for: unbonding completion (preserves original period)
- Tokens expire based on the specified period, not current

**DebitBalance(addr, amount)** — FIFO from oldest non-expired period:
- Used for: staking, fee payment, burns
- Deducts from earliest period first, moves to next when exhausted

**TransferPreservingPeriods(from, to, amount)** — peer-to-peer transfer:
- FIFO debit from sender → group by period → credit receiver at sender's original periods
- **Prevents ping-pong exploit**: receiver gets tokens that expire when sender's would have
- Example: sender has period 10 (100 tokens), period 50 (200 tokens), sends 150:
  - Receiver gets: 100 at period 10 + 50 at period 50

**GetBalance(addr)** — aggregate active balance:
- Sums all non-expired, non-swept period balances
- Lazy evaluation: expired periods excluded at query time

### RAI (Regenerative Agent Income)

Expired tokens don't burn — they flow to the RAI pool:

1. **BeginBlocker** calls `SweepExpiredTokens(batchSize)`:
   - Iterates sweep index for accounts with expired periods
   - Moves expired balance to RAI pool, marks period as `Swept = true`
   - Processes max 100 accounts per block (prevents DoS)

2. **Distribution** (governance-triggered):
   - `DistributeRAI()` — divides pool equally among eligible accounts
   - Eligible: "standard" or "validator" account type, has active balance
   - Ineligible: "foundation", "research" accounts

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `Transfer` | 21K | Period-preserving transfer + bank send |
| `ApproveAllowance` | — | Set spending allowance (0 = revoke) |
| `Burn` | — | FIFO debit + bank burn |
| `UpdateParams` | — | Governance-only parameter update |

### Queries

| Query | Returns |
|-------|---------|
| `Account(addr)` | Full ExpirableAccount with all period balances |
| `Balance(addr)` | Active balance + period breakdown |
| `Allowance(owner, spender)` | Current allowance |
| `RAI` | RAI pool balance and counters |
| `Params` | Module configuration |

---

## Cross-Module Integration

### The Staking ↔ Tokens Bridge

This is the most critical cross-module interaction:

```
DELEGATE (user stakes tokens):
  1. Staking: create/update delegation
  2. Tokens: DebitBalance(user, amount) — FIFO from oldest period
  3. Bank: SendCoinsFromAccountToModule

UNDELEGATE (user unstakes):
  1. Tokens: GetCurrentPeriod() → capture as OriginPeriod
  2. Staking: create UnbondingEntry { OriginPeriod, CompletesAt: now + 268560 }
  3. Wait ~7 days...

UNBONDING COMPLETE (in Staking BeginBlocker):
  1. Staking: find mature UnbondingEntries
  2. Tokens: CreditBalanceToSpecificPeriod(user, amount, entry.OriginPeriod)
     ← This preserves the original expiration! Not fresh.
  3. Bank: SendCoinsFromModuleToAccount
```

**Why this matters**: Without period preservation, a user could unstake and re-stake every 7 days to reset their token expiration indefinitely. The `OriginPeriod` capture prevents this arbitrage.

### Knowledge → Staking (via StakingKeeperAdapter)

```
Verification Round needs validator info:
  1. Knowledge: GetActiveValidatorInfos() → adapter converts Keeper data
  2. Knowledge: GetEffectiveStake(addr) → adapter calls keeper method
  3. Knowledge: RecordVerification(addr, correct) → adapter delegates
  4. Knowledge: SlashValidator(addr, amount) → adapter delegates
```

### Knowledge → Tokens

```
Claim Submission:
  1. Knowledge: calls TokensKeeper.DebitBalance(submitter, stakeAmount)
     ← FIFO debit from oldest period
```
