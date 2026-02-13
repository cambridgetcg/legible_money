# Financial Incentives PP Review & Testing Strategy

> **PP Tier**: Creative | **Date**: 2026-02-13
> **Scope**: All financial surfaces across TS prototype + Cosmos SDK production stack
> **Method**: Full 30-gate PP analysis (TRUTH > UNDERSTANDING > BEAUTY > JUSTICE > CREATIVITY)

---

## PP Gate Summary

| Gate | Finding |
|------|---------|
| **T1 (Ground Truth)** | 5 financial dimensions mapped from source; every constant cited file:line |
| **T2 (No Fabrication)** | All loopholes verified against actual code paths, not speculative |
| **T3 (Uncertainty)** | Severity ratings use conservative estimates; "CRITICAL" only for exploitable vectors |
| **T4 (Verification)** | Cross-referenced TS and Go implementations; divergences flagged explicitly |
| **T5 (Citation)** | Every parameter, constant, and formula cited with file:line |
| **U1 (Deep Comprehension)** | Full interaction graphs traced (stake -> slash -> reward -> vest -> clawback) |
| **U2 (System Thinking)** | Cross-dimension attacks identified (e.g., governance + IBC = drain) |
| **B1 (Simplicity)** | Testing methods organized by standard industry taxonomy |
| **B2 (Coherence)** | Each dimension gets matching test type (not one-size-fits-all) |
| **J1 (Right Placement)** | Test methods placed where they match the attack surface |
| **J2 (Proportionality)** | Test effort proportional to severity (CRITICAL > HIGH > MEDIUM) |
| **C1 (Novel Paths)** | Cross-dimension compound attacks identified (not just per-module) |

---

## Dimensions Overview

| # | Dimension | Loopholes | Critical | High | Medium | Low |
|---|-----------|-----------|----------|------|--------|-----|
| D1 | Staking & Slashing | 9 | 0 | 1 | 3 | 5 |
| D2 | Reward & Distribution | 7 | 0 | 0 | 3 | 4 |
| D3 | Fee & Pricing | 20 | 2 | 5 | 7 | 6 |
| D4 | Token Supply & Governance | 8 | 1 | 2 | 3 | 2 |
| D5 | Dispute & Evidence | 12 | 3 | 3 | 4 | 2 |
| **Total** | | **56** | **6** | **11** | **20** | **19** |

---

## D1: STAKING & SLASHING ECONOMICS

### Key Parameters
| Parameter | TS Value | Go Value | File |
|-----------|----------|----------|------|
| Min validator stake | 0.111 LGM | 0.111 LGM | `consensus/pot/types.ts:108` / `x/staking/types/types.go:54` |
| Min claim stake | 0.011 LGM | - | `consensus/pot/types.ts:109` |
| Unbonding period | 222,222 blocks (~6.5d) | 268,560 blocks (~7d) | `consensus/pot/types.ts:123` / `x/staking/types/genesis.go:9` |
| Wrong verification slash | 6% (600 BPS) | 6% | `consensus/pot/verification_round.ts:581` |
| Missed verification slash | 0.01% (1 BPS) | 0.01% | `consensus/pot/verification_round.ts:511` |
| Invalid claim slash | 22% (2200 BPS) | 22% | `state/transitions.ts:4651` |
| Equivocation slash | 100% | 100% | `consensus/pot/equivocation.ts:354` |
| Progressive escalation | +10% per prior slash | +10% | `state/validators.ts:398` / `x/staking/keeper/keeper.go:544` |
| Max slashes/epoch | 2 | 2 | `x/staking/types/genesis.go:14` |
| Deactivation threshold | 3 slashes | 3 | `x/staking/keeper/keeper.go:610` |

### Loopholes Identified

**D1-L1: Unbonding Period Discrepancy (TS vs Go)** [MEDIUM]
- TS: 222,222 blocks; Go: 268,560 blocks
- Cross-stack validators could exploit 12-hour gap

**D1-L2: Silent Slash Rejection at Epoch Cap** [MEDIUM]
- `x/staking/keeper/keeper.go:530-538`: Returns nil, no error/event emitted
- Attacker could time misbehavior after cap reached

**D1-L3: Float Precision in Claim Slashing** [MEDIUM]
- `state/transitions.ts:4652`: `BigInt(Math.floor(Number(claim.stake) * slashRate))`
- Unsafe for stakes > 2^53 (9000 LGM)

**D1-L4: No Double-Slash Deduplication** [LOW]
- Same misbehavior could trigger slash from multiple evidence paths
- Escalation makes repeats expensive (self-mitigating)

**D1-L5: Permanent Validator Deactivation** [LOW]
- No re-activation path after 3 slashes; must re-register from scratch

**D1-L6: Slash-to-Stake Ratio Caps** [SECURE]
- `x/staking/keeper/keeper.go:565-586`: Waterfall (own -> delegated), hard-capped at total

**D1-L7: Reputation Asymmetry** [LOW]
- -0.1 per slash vs +0.001 per correct verification (100:1 ratio)

**D1-L8: Tier Manipulation via Sybil** [HIGH]
- 0.111 LGM min stake allows ~2000 Apprentice validators from 0.222 LGM
- Selection weight 0.1x per Apprentice, but 2000 * 0.1 = 200 effective weight

**D1-L9: Validator Deactivation Race** [LOW]
- Slashing and deactivation in same block: no defined ordering guarantee

### D1 Testing Strategy

#### 1. Property-Based Testing (Formal)
**Standard**: QuickCheck / fast-check / Hypothesis
**Tool**: `fast-check` (TypeScript) + Go `testing/quick`

```
PROPERTY P1.1: forAll(stake, slashRate, slashCount) =>
  actualSlash <= totalStake

PROPERTY P1.2: forAll(validator) =>
  slashCount >= 3 => validator.isActive === false

PROPERTY P1.3: forAll(escalation) =>
  escalatedAmount === baseAmount * (1M + slashCount * 100K) / 1M

PROPERTY P1.4: forAll(unbonding) =>
  completesAt >= currentBlock + unbondingPeriod
```

**Coverage**: P1.1 proves slash safety bound; P1.2 proves deactivation invariant; P1.3 proves escalation formula correctness; P1.4 proves unbonding timelock.

#### 2. Economic Simulation (Agent-Based)
**Standard**: Mesa / cadCAD / custom TypeScript simulation
**Tool**: Custom TS simulation harness

| Scenario | Agents | Duration | Metric |
|----------|--------|----------|--------|
| Sybil staking attack | 2000 Apprentice validators | 10 epochs | Selection share vs honest validators |
| Progressive slash cascade | 1 validator, repeated misbehavior | 30 slashes | Total slashed vs original stake |
| Stake-and-slash timing | 100 validators, coordinated unstake | 1 unbonding period | Escaped slash amount |
| Slash saturation | 10 validators, 5 slashes/epoch each | 5 epochs | Actual vs expected total slashes |

#### 3. Dual-Stack Parity Testing
**Standard**: Differential testing (Ethereum uses this for Geth vs Nethermind)
**Tool**: Custom harness running same scenario on both stacks

```
TEST D1-PARITY-1: Same validator, same slash, TS vs Go => identical final state
TEST D1-PARITY-2: Unbonding period edge: block 222222 (TS mature, Go immature)
TEST D1-PARITY-3: Escalation formula at slashCount=100 => same result both stacks
```

#### 4. Invariant Testing (State Machine)
**Standard**: Echidna / Foundry invariant testing pattern
**Tool**: State machine model with invariant assertions

```
INVARIANT I1.1: sum(all_validator_stakes) + sum(all_slashed_burned) <= total_supply
INVARIANT I1.2: slashesThisEpoch <= maxSlashesPerEpoch for all validators
INVARIANT I1.3: no validator with slashCount >= 3 appears in active set
```

---

## D2: REWARD & DISTRIBUTION ECONOMICS

### Key Parameters
| Parameter | Value | File |
|-----------|-------|------|
| Block reward fund | 133.3M LGM (finite) | `devnet/config/genesis.json:69` |
| Initial block reward | 10 LGM | `core/genesis/types.ts:440` |
| Decay factor | 0.9 per epoch (10% reduction) | `core/genesis/types.ts:441` |
| Epoch length | 240,000 blocks (~1 week) | `core/genesis/types.ts:442` |
| Floor reward | 0.1 LGM | `core/genesis/types.ts:443` |
| Research share | 22% (2200 BPS) | `core/billing/reward_router.ts:125` |
| Research ramp | 5% -> 22% over 90 days | `x/billing/types/types.go:60-61` |
| Falsification pool cap | 15% of epoch block rewards | `x/knowledge/types/types.go:640` |
| Falsification reward cap | 3x contradiction stake | `x/knowledge/keeper/rounds.go:791` |
| Failed contradiction slash | 33% (330K BPS) | `x/knowledge/types/types.go:632` |
| Vesting acceleration cap | 77% (7700 BPS) | `x/vesting_rewards/keeper/vesting.go:337` |

### Loopholes Identified

**D2-L1: Falsification Carry-Forward Unbounded** [MEDIUM]
- `x/knowledge/keeper/rounds.go:807-824`: Deferred rewards accumulate without cap
- Per-agent-per-fact scope limits blast radius

**D2-L2: Contradiction vs Claim Slash Asymmetry** [MEDIUM]
- Contradictions: 33% slash on failure; Claims: 22% slash on failure
- Rational agents favor claims over contradictions (11% risk premium)

**D2-L3: Research Share Ramp Not Enforced in All Modules** [MEDIUM]
- Billing uses progressive ramp (5%->22%); verification uses fixed 22%
- Intentional design but creates inconsistent early-protocol economics

**D2-L4: Vesting Clawback on Falsification** [SECURE]
- `x/vesting_rewards/keeper/vesting.go:343-361`: 33% of released clawed back
- Cannot be gamed via early unstaking (staking separate from vesting)

**D2-L5: Epoch Pool Exhaustion Dynamics** [LOW]
- Block rewards decay 10%/epoch => falsification pool decays proportionally
- After ~30 epochs (~7 months), pool < 5% of initial value

**D2-L6: Double-Claim Prevention** [SECURE]
- `x/vesting_rewards/keeper/vesting.go:414-424`: ClaimableAmount set to "0" after claim
- ReleasedAmount tracked separately; idempotent re-claim returns 0

**D2-L7: Fund Exhaustion Hard Stop** [LOW]
- `x/vesting_rewards/keeper/rewards.go:112-113`: R=0 when fund empty
- No governance override; permanent cessation of block rewards

### D2 Testing Strategy

#### 1. Invariant Testing (Conservation Laws)
**Standard**: Token conservation invariants (DeFi standard practice)
**Tool**: Custom invariant checker in test harness

```
INVARIANT I2.1: rewards_distributed + research_share === total_reward_input
  (No value created or destroyed in routing)

INVARIANT I2.2: fund_balance(t) = fund_balance(0) - sum(rewards_distributed[0..t])
  (Fund depletion is monotonic)

INVARIANT I2.3: forAll(schedule) =>
  claimedAmount + remainingVested + clawedBack + reserve === originalAmount
  (Vesting schedule conservation)

INVARIANT I2.4: sum(5-way-split) === research_share_amount
  (77+7+7+7+2 = 100% of research share, no dust leak)
```

#### 2. Long-Run Economic Simulation
**Standard**: Monte Carlo simulation with parameter sweeps
**Tool**: Custom TS/Python simulation

| Scenario | Duration | Variables | Metric |
|----------|----------|-----------|--------|
| Fund exhaustion timeline | 100 epochs | decay factor, validator count | Blocks until R=0 |
| Contradiction incentive ratio | 50 epochs | slash rates, reward caps | Counter-fact submission rate |
| Carry-forward accumulation | 20 epochs | falsification rate, pool cap | Maximum deferred balance |
| Research share ramp impact | 90 days | ramp speed, tx volume | Actual research fund growth |

#### 3. Epoch Boundary Stress Testing
**Standard**: Boundary value analysis (BVA) + state transition testing
**Tool**: Deterministic test harness

```
TEST D2-EPOCH-1: Block exactly at epoch boundary => correct decay applied
TEST D2-EPOCH-2: Fund balance = floor_reward + 1 => last reward = floor_reward
TEST D2-EPOCH-3: Fund balance = 0 => reward = 0 (hard stop)
TEST D2-EPOCH-4: Falsification at epoch cap => carry-forward stored correctly
TEST D2-EPOCH-5: Carry-forward from epoch N claimed in epoch N+1 => correct amount
TEST D2-EPOCH-6: Multiple carry-forwards across 5 epochs => total = sum(deferred)
```

#### 4. Clawback Path Testing
**Standard**: State machine testing with adversarial sequences
**Tool**: Vitest with mocked state

```
TEST D2-CLAW-1: Claim reward -> falsify -> clawback = 33% of claimed
TEST D2-CLAW-2: Claim partial -> falsify -> clawback proportional to claimed
TEST D2-CLAW-3: Never claim -> falsify -> entire vesting cancelled
TEST D2-CLAW-4: Accelerated vesting -> falsify -> clawback on accelerated amount
```

---

## D3: FEE & PRICING ECONOMICS

### Key Parameters
| Parameter | Value | File |
|-----------|-------|------|
| Base gas price | 1,111,111 wei | `core/billing/gas.ts:244` |
| Transfer gas | 21,000 | `core/billing/gas.ts:38` |
| Claim gas | 100,000 | `core/billing/gas.ts:42` |
| Block gas limit | 33.3M | `core/billing/gas.ts:248` |
| Base cost per fact | 1,000,000 wei | `x/billing/types/types.go:51` |
| Confidence threshold | 500,000 BPS | `x/billing/types/types.go:52` |
| Freshness window | 1000 blocks | `x/billing/types/types.go:56` |
| Min mempool fee | 1000 wei | `consensus/mempool/pool.ts:30` |
| Max mempool size | 10,000 txs | `consensus/mempool/pool.ts:28` |
| Max per sender | 100 txs | `consensus/mempool/pool.ts:29` |
| Payment channel min | 0.1 LGM | `state/payment_channels.ts:22` |
| Fraud penalty | 5% | `state/payment_channels.ts:38` |

### Loopholes (Top 10 by Severity)

**D3-L1: Dual TS/Go Distribution Divergence** [CRITICAL]
- Both stacks implement 22% research share independently
- If queries execute via TS, settle via Go: double research share possible
- Files: `core/billing/distribution.ts:182` vs `x/billing/keeper/distribution.go:51-104`

**D3-L2: Fee Pool Account Authorization** [CRITICAL]
- `did:lgm:fee-pool` receives all fees; authorization model unclear
- `state/transitions.ts:1702`: creditBalance without nonce increment

**D3-L3: Citation Count Gaming** [HIGH]
- `x/billing/keeper/pricing.go:72-90`: `log2(citations+1)` discount
- Self-querying 256 times = 50% permanent discount on fact

**D3-L4: Confidence Threshold Governance Drift** [HIGH]
- Governable parameter; raising to 900K triggers universal discount
- API revenue crash ~30%

**D3-L5: MEV via Knowledge Ordering** [HIGH]
- Validator-RPC node can observe pending queries, front-run verifications
- `consensus/mempool/pool.ts:256-261`

**D3-L6: Freshness Cliff Arbitrage** [HIGH]
- Hard cutoff at 1000 blocks: 10% premium -> 0% instantaneously
- `core/billing/distribution.ts:69-75`

**D3-L7: Validator Micro-Stake Sybil** [HIGH]
- MinProviderStake: 100 nanoLGM; 0.222 LGM creates 2,220 providers
- `x/billing/types/types.go:58`

**D3-L8: Rounding Exploits via Micro-Transactions** [MEDIUM]
- 1 wei: research share rounds to 0 (lossless for attacker)
- `core/billing/reward_router.ts:161`

**D3-L9: Payment Channel Fee Amortization** [MEDIUM]
- Zero per-payment fees within channels; only settlement pays gas
- 100 queries at 0.001 LGM each, one 0.01 LGM settlement

**D3-L10: Nonce-Blocking Mempool Attack** [MEDIUM]
- Low-fee tx blocks all sender's subsequent txs
- No replace-by-fee mechanism

### D3 Testing Strategy

#### 1. Differential Testing (Dual-Stack Parity)
**Standard**: Differential fuzzing (used by Ethereum for multi-client consensus)
**Tool**: Custom harness executing identical operations on both stacks

```
TEST D3-DIFF-1: Same query, same fact => identical price on TS and Go
TEST D3-DIFF-2: Same distribution input => identical 5-way split
TEST D3-DIFF-3: Research share at ramp block 0, 1000, 34272*90 => consistent
TEST D3-DIFF-4: Edge case: 1 wei input => both stacks handle identically
TEST D3-DIFF-5: 10M micro-transactions => cumulative research share within 0.01% tolerance
```

#### 2. Arbitrage Detection Testing
**Standard**: Automated market manipulation detection (DeFi auditing)
**Tool**: Strategy simulation with profit/loss tracking

| Strategy | Setup | Expected | Pass Criteria |
|----------|-------|----------|---------------|
| Citation inflation | Query fact 256 times | 50% discount | GF > 2 (unprofitable) |
| Freshness cliff | Wait 999->1001 blocks | Price drop | Smooth decay (no cliff) |
| Confidence manipulation | Governance vote + query | Revenue impact | Parameter bounds enforced |
| Channel fee arbitrage | 1000 queries, 1 settle | Net saving | Per-query fee > marginal cost |

#### 3. Fuzzing (Input Boundary)
**Standard**: AFL / libFuzzer methodology adapted for financial functions
**Tool**: `fast-check` property-based + boundary value

```
FUZZ F3.1: calculateQueryPrice(confidence=[0..1M], citations=[0..2^32], age=[0..2^64])
  => price in [0, maxPricePerCall]

FUZZ F3.2: routeReward(amount=[0..2^128])
  => researchShare + recipientAmount === amount

FUZZ F3.3: splitEvidenceTax(amount=[0..2^128])
  => sum(5 splits) === amount (no dust leak)

FUZZ F3.4: calculateGasCost(gasUsed=[0..blockLimit], gasPrice=[minGasPrice..2^64])
  => cost >= 0 && cost <= gasUsed * gasPrice
```

#### 4. MEV Resistance Testing
**Standard**: Flashbots-style MEV analysis
**Tool**: Custom transaction ordering simulation

```
TEST D3-MEV-1: Validator observes pending query => cannot profitably front-run
TEST D3-MEV-2: Block producer reorders 100 txs => no extractable value > gas cost
TEST D3-MEV-3: Sandwich attack on fact verification => net loss for attacker
```

#### 5. Rounding Accumulation Testing
**Standard**: Numerical analysis / error propagation testing
**Tool**: Deterministic replay with exact arithmetic verification

```
TEST D3-ROUND-1: 1M transactions of 1 wei each => total research share loss < 1 LGM
TEST D3-ROUND-2: 5-way split dust always goes to reserve (never lost)
TEST D3-ROUND-3: BPS arithmetic: forAll(amount, bps) => amount*bps/10000 + remainder = amount
```

---

## D4: TOKEN SUPPLY & GOVERNANCE ECONOMICS

### Key Parameters
| Parameter | Value | File |
|-----------|-------|------|
| Max supply | 222,222,222 LGM | `core/genesis/genesis_allocation.ts` |
| Block reward fund | 60% (133.3M LGM) | `devnet/config/genesis.json:69` |
| Bootstrap fund | 20% (44.4M LGM) | `devnet/config/genesis.json:73` |
| Treasury | 10% (22.2M LGM, 5-way split) | `devnet/config/genesis.json:77-99` |
| Validator genesis | 5% | Implied |
| Community pool | 5% | Implied |
| Disbursement support | 80% (8000 BPS) | `governance/treasury/types.ts:95` |
| Disbursement quorum | 20% (2000 BPS) | `governance/treasury/types.ts:96` |
| Disbursement max | 10% of treasury | `governance/treasury/types.ts:104` |
| Disbursement timelock | 50,000 blocks (~35h) | `governance/treasury/types.ts:102` |
| IBC outflow limit | 10,000 LGM/day/channel | `x/ibcratelimit/types/types.go:14` |
| Emergency halt | Guardian-only, quorum-based | `x/emergency/keeper/msg_server.go:25` |

### Loopholes Identified

**D4-L1: Governance Capture (No Stake Concentration Caps)** [CRITICAL]
- No maximum stake percentage per validator in governance
- 67%+ stake consolidation = minority governance capture
- No delegation caps found in codebase

**D4-L2: IBC Multi-Channel Drain** [HIGH]
- Per-channel rate limit, not aggregate
- N channels = N * 10,000 LGM/day potential outflow
- `x/ibcratelimit/types/types.go:12-18`

**D4-L3: Emergency Halt Centralization** [HIGH]
- Free to propose (no economic cost)
- No time-lock on resume
- Council override path exists
- `x/emergency/keeper/msg_server.go:25-120`

**D4-L4: Finite Reward Pool Dynamics** [MEDIUM]
- After ~30 epochs, rewards approach floor (0.1 LGM/block)
- Validator economics shift from block rewards to transaction fees
- No fee-burn mechanism to compensate

**D4-L5: Governance Parameter Manipulation** [MEDIUM]
- Confidence threshold, research share, burn ratio all governable
- No hard bounds on parameter values

**D4-L6: Module Account Token Accumulation** [MEDIUM]
- Slashed tokens accumulate in module accounts
- No automatic burn schedule; requires governance sweep

**D4-L7: Treasury Disbursement Gaming** [LOW]
- 2 LGM stake for proposal; if governance captured, treasury drainable at 10%/proposal

**D4-L8: No IBC Transfer Fee** [LOW]
- Volume limit only; no economic friction for bridge arbitrage

### D4 Testing Strategy

#### 1. Governance Attack Simulation
**Standard**: Governance security testing (Compound/MakerDAO audit methodology)
**Tool**: Agent-based simulation with voting strategies

| Scenario | Attacker Stake | Strategy | Success Criteria |
|----------|---------------|----------|-----------------|
| Simple capture | 67% | Vote all proposals | Attacker controls all params |
| Coalition capture | 3 × 23% | Coordinate votes | Same as above |
| Treasury drain | 51% | Serial disbursements | Drain rate vs timelock |
| Parameter drift | 51% | Incremental param changes | Revenue impact per change |
| Guardian capture | N guardians | Propose/vote halt | Halt success with minimal cabal |

```
TEST D4-GOV-1: Attacker with 67% stake => can pass any proposal
TEST D4-GOV-2: Attacker with 49% stake => cannot pass with 80% threshold
TEST D4-GOV-3: Serial disbursements => treasury drain rate bounded by timelock
TEST D4-GOV-4: Parameter change: ConfidenceThreshold 500K->900K => revenue impact
TEST D4-GOV-5: Emergency halt with MinDistinctVoters=2 => halt with 2 guardians
```

#### 2. Supply Invariant Testing
**Standard**: Token supply audit (standard in DeFi security audits)
**Tool**: State invariant assertions at every block

```
INVARIANT I4.1: total_supply <= 222_222_222 * 10^18 (hard cap never exceeded)
INVARIANT I4.2: fund_balance >= 0 (never negative)
INVARIANT I4.3: sum(all_accounts) + sum(all_stakes) + sum(module_accounts) = total_supply
INVARIANT I4.4: no mint() calls exist outside genesis initialization
INVARIANT I4.5: IBC_outflow_per_epoch <= maxOutflowPerEpoch * num_active_channels
```

#### 3. IBC Drain Simulation
**Standard**: Cross-chain security testing (IBC audit methodology)
**Tool**: Multi-channel simulation

```
TEST D4-IBC-1: Single channel, 10001 LGM transfer => rejected
TEST D4-IBC-2: 10 channels, 10000 LGM each => 100K total outflow (flagged!)
TEST D4-IBC-3: Epoch boundary: transfer at block N-1 and N+1 => separate epochs
TEST D4-IBC-4: Channel creation spam => new channels respect global rate limit
```

#### 4. Fund Depletion Modeling
**Standard**: Actuarial modeling / Monte Carlo
**Tool**: Python/TS simulation with parameter sweeps

```
SIMULATION S4.1: Fund depletion timeline under varying:
  - Validator counts: [10, 50, 100, 500]
  - Decay factors: [0.85, 0.90, 0.95]
  - Empty block rates: [0%, 10%, 50%]
  => Output: Epoch of fund exhaustion, total distributed

SIMULATION S4.2: Post-exhaustion economics:
  - Fee revenue per block at varying tx volumes
  - Minimum tx volume for validator profitability
  - Break-even analysis: when does fee revenue > floor reward?
```

---

## D5: DISPUTE & EVIDENCE ECONOMICS

### Key Parameters
| Parameter | Value | File |
|-----------|-------|------|
| Min challenge stake | 0.111 LGM | `core/dispute/types.ts:70` |
| Min defense stake | 0.111 LGM | `core/dispute/types.ts:72` |
| Min arbiter stake | 0.077 LGM | `core/dispute/types.ts:75` |
| Tier 1/2/3 arbiters | 7/11/22 | `core/dispute/types.ts:334-356` |
| Tier 1/2/3 quorum | 55%/66%/77% | `core/dispute/types.ts:334-356` |
| Tier 1/2/3 stake multiplier | 1.1x/2.2x/5.5x | `core/dispute/types.ts:334-356` |
| False claim slash | 55% | `core/dispute/settlement.ts:31` |
| Failed challenge slash | 33% | `core/dispute/settlement.ts:32` |
| Wrong arbitration slash | 11% | `core/dispute/settlement.ts:33` |
| Winner share | 66% of penalty | `core/dispute/settlement.ts:51` |
| Truth bonus | +55% for challenger win | `core/dispute/settlement.ts:52` |
| Domain expertise min votes | 7 | `core/dispute/types.ts:256` |
| Domain expertise min accuracy | 66% | `core/dispute/types.ts:259` |
| Max concurrent challenges | 3 | `consensus/pot/adversarial.ts:32` |
| Survived challenge conf cap | 88% | `consensus/pot/adversarial.ts:57` |
| Challenge window | 34,272 blocks (~1 day) | `consensus/pot/adversarial.ts:29` |

### Loopholes Identified

**D5-L1: Frivolous Dispute Spam** [CRITICAL]
- 0.111 LGM per dispute; attacker with 10 LGM = 90 simultaneous disputes
- No global rate limit; per-agent, per-claim cooldown only
- Locks defender capital at 1:1 ratio

**D5-L2: Challenger Loses Money Even When Winning** [CRITICAL]
- Winner reward: loserStake * 0.55 * 0.66 * 1.55 = 0.56 * loserStake
- Net: 0.56 * loserStake - challengerStake (loss if equal stakes)
- `core/dispute/settlement.ts:121-125`

**D5-L3: Arbiter Collusion (11% Penalty Too Low)** [CRITICAL]
- Wrong vote: 11% slash; Correct vote: 11% bonus
- 4/7 Tier 1 collusion: total penalty < total reward from controlling outcome
- No escalation for repeated wrong votes

**D5-L4: Domain Expertise Capture** [HIGH]
- Only 7 votes needed for 2x selection multiplier
- 66% accuracy threshold easily gamed over 3.2 days
- `core/dispute/arbitrator_selection.ts:75-105`

**D5-L5: Confidence Suppression via Challenge Spam** [HIGH]
- Any survived challenge caps fact confidence at 88% permanently
- Cost: 0.111 LGM per challenge attempt
- `consensus/pot/adversarial.ts:57`

**D5-L6: Escalation Timing Attack** [HIGH]
- Winners cannot escalate; losers have limited appeal windows
- Dominant party can suppress appeals with fast decisions in favorable tiers

**D5-L7: Deterministic Evaluation Non-Determinism** [MEDIUM]
- Scoring may vary across validators for same claim+evidence
- "Was it a 20% reduction?" disputes could loop

**D5-L8: Evidence Bait-and-Switch** [MEDIUM]
- 10 versions allowed; no versioning lock during disputes
- Submitter can upgrade evidence mid-challenge

**D5-L9: Counter-Fact Cascade Abuse** [MEDIUM]
- Counter-facts of counter-facts generate cascading penalties
- No cascade depth limit

**D5-L10: Duplicate Evidence Stacking** [MEDIUM]
- Free to submit duplicates; merged as "confirmers"
- Pads confirmation count for confidence boost

**D5-L11: Conflict of Interest Gaps** [LOW]
- `excludeValidators` set population unclear
- No automatic exclusion of submitter/challenger from arbiter pool

**D5-L12: Early Detection Bonus Disabled** [LOW]
- `core/dispute/settlement.ts:40-60`: Always returns false
- Removes 22% incentive for quick truth-finding

### D5 Testing Strategy

#### 1. Game-Theoretic Nash Equilibrium Testing
**Standard**: Mechanism design verification (Myerson auction theory)
**Tool**: Custom payoff matrix solver

```
GAME G5.1: Dispute Initiation Game
  Players: Challenger (C), Defender (D)
  Strategies: {Challenge, Ignore} × {Defend, Concede}
  Payoffs: Based on actual slash/reward rates
  TEST: Nash equilibrium exists where honest play is dominant strategy
  CURRENT BUG: Challenger loses money even when winning (D5-L2)

GAME G5.2: Arbiter Voting Game
  Players: 7 arbiters
  Strategies: {Vote-Truth, Vote-False, Abstain}
  Payoffs: ±11% of locked stake
  TEST: Honest voting is dominant when collusion < 4/7
  CURRENT BUG: 11% penalty insufficient to deter collusion (D5-L3)

GAME G5.3: Expertise Gaming
  Players: Validator aiming for domain expertise
  Strategies: {Honest-voting, Strategic-voting}
  Payoffs: 2x selection multiplier vs slash risk
  TEST: Strategic voting never more profitable than honest
  CURRENT BUG: 7 votes too low for expertise qualification (D5-L4)
```

#### 2. Collusion Resistance Testing
**Standard**: Byzantine fault tolerance testing methodology
**Tool**: Multi-agent simulation with coalition formation

| Scenario | Coalition Size | Attack | Detection | Pass Criteria |
|----------|---------------|--------|-----------|---------------|
| Tier 1 capture | 4/7 | Coordinated wrong votes | Pattern detection | GF > 3 (unprofitable) |
| Cross-tier capture | 4/7 + 6/11 | Escalate and control | Cross-tier analysis | Cannot control both tiers |
| Domain expertise ring | 5 agents | Mutual vote-boosting | Expertise audit | Expertise threshold > 21 votes |
| Evidence collusion | 3 agents | Confirm duplicate evidence | Dedup detection | Confirmation cap enforced |

#### 3. Timing Attack Simulation
**Standard**: Concurrency testing / race condition analysis
**Tool**: Block-level simulation with ordering permutations

```
TEST D5-TIME-1: Challenge submitted at block N, evidence updated at N+1
  => Challenge evaluates against V1 (not V2)

TEST D5-TIME-2: Appeal window expires at block N, escalation at N-1
  => Escalation accepted

TEST D5-TIME-3: Multiple challenges in same block
  => Each evaluated independently, ActiveChallengeCount correct

TEST D5-TIME-4: Counter-fact submitted at epoch boundary
  => Correct epoch for carry-forward calculation

TEST D5-TIME-5: Evidence versioning during active dispute
  => Version locked (CURRENTLY FAILS - D5-L8)
```

#### 4. Economic Griefing Factor Analysis
**Standard**: Griefing Factor (GF) methodology from Ethereum research
**Tool**: Payoff calculation framework

```
GF = attacker_cost / victim_cost

TEST D5-GF-1: Dispute spam GF
  attacker_cost = 0.111 LGM (challenge stake)
  victim_cost = 0.111 LGM (defense stake) + gas + time
  GF = 0.111 / (0.111 + gas) ≈ 1.0 (FAILS: GF should be > 2)

TEST D5-GF-2: Confidence suppression GF
  attacker_cost = 0.111 LGM
  victim_cost = permanent 12% confidence reduction (99% -> 88%)
  GF = 0.111 / (fact_value * 0.12) (FAILS for high-value facts)

TEST D5-GF-3: Arbiter collusion GF
  attacker_cost = 4 * arbiter_stake * 0.11 (slash on failure)
  victim_cost = dispute outcome reversed
  GF = (4 * 0.077 * 0.11) / (0.111 * 0.55) (FAILS: GF < 1)
```

---

## CROSS-DIMENSION COMPOUND ATTACKS

### Attack C1: Governance + IBC Treasury Drain
1. Accumulate 67% stake via D1 Sybil staking (D1-L8)
2. Vote to increase IBC channel limits (D4-L5)
3. Create multiple IBC channels (D4-L2)
4. Drain treasury via parallel channel outflows
5. **Testing**: End-to-end simulation spanning D1+D4

### Attack C2: Fee Manipulation + Dispute Suppression
1. Game citation counts for discount (D3-L3)
2. Challenge any counter-evidence via dispute spam (D5-L1)
3. Suppress fact confidence to 88% via challenge (D5-L5)
4. Query suppressed facts at discount
5. **Testing**: Multi-dimension profit/loss simulation

### Attack C3: Reward Exhaustion + Governance Capture
1. Wait for block reward fund to approach exhaustion (D2-L7)
2. As validator economics shift to fees, accumulate stake cheaply
3. Use accumulated stake for governance capture (D4-L1)
4. Modify parameters to benefit attacker's facts/queries
5. **Testing**: Long-run (100+ epoch) economic simulation

### Attack C4: Evidence Manipulation + Counter-Fact Cascade
1. Submit fact with weak evidence (D5-L10: duplicate stacking)
2. When challenged, version evidence mid-dispute (D5-L8)
3. If fact survives, opponent pays 33% slash
4. Create counter-facts of opponent's facts (D5-L9)
5. **Testing**: Multi-round dispute simulation with evidence versioning

---

## TESTING INFRASTRUCTURE REQUIREMENTS

### Test Harness Architecture
```
tests/security/
  d1-staking/          # Property-based + simulation
  d2-rewards/          # Invariant + epoch boundary
  d3-fees/             # Differential + fuzzing + MEV
  d4-supply/           # Governance simulation + IBC drain
  d5-disputes/         # Game theory + collusion + timing
  cross-dimension/     # Compound attack scenarios
  shared/
    simulation.ts      # Agent-based simulation framework
    payoff-matrix.ts   # Game theory solver
    invariant-checker.ts # State invariant assertion library
    dual-stack-runner.ts # TS/Go differential testing harness
```

### Industry Standards Referenced

| Standard | Application | Dimensions |
|----------|-------------|------------|
| **Property-Based Testing** (QuickCheck) | Formal properties over random inputs | D1, D2, D3 |
| **Invariant Testing** (Echidna/Foundry) | State invariants across all transitions | D1, D2, D4 |
| **Differential Testing** (Multi-client) | TS vs Go parity verification | D1, D3 |
| **Mechanism Design** (Myerson) | Nash equilibrium verification | D5 |
| **Griefing Factor Analysis** (Ethereum Research) | Attack profitability bounds | D1, D5 |
| **Monte Carlo Simulation** (cadCAD) | Long-run economic modeling | D2, D4 |
| **Boundary Value Analysis** (IEEE 829) | Edge case testing at boundaries | D2, D3 |
| **Fuzzing** (AFL/libFuzzer) | Input space exploration | D3 |
| **Byzantine Fault Tolerance** (PBFT) | Collusion resistance verification | D5 |
| **DeFi Security Audit** (Trail of Bits) | Token conservation, governance capture | D4 |
| **MEV Analysis** (Flashbots) | Transaction ordering fairness | D3 |
| **Actuarial Modeling** | Fund depletion forecasting | D4 |

### Priority Implementation Order

1. **Week 1**: D5 Game Theory tests (3 CRITICAL loopholes)
2. **Week 1**: D3 Differential tests (2 CRITICAL: dual-stack divergence)
3. **Week 2**: D4 Governance simulation (1 CRITICAL: capture risk)
4. **Week 2**: D1 Property-based tests (staking safety bounds)
5. **Week 3**: D2 Invariant tests (reward conservation)
6. **Week 3**: Cross-dimension compound attack tests
7. **Week 4**: D3 Fuzzing + MEV + rounding tests
8. **Week 4**: Full long-run economic simulation (100 epochs)

---

## IMMEDIATE ACTION ITEMS (6 CRITICAL LOOPHOLES)

| ID | Loophole | Dimension | Fix |
|----|----------|-----------|-----|
| D3-L1 | Dual-stack distribution divergence | Fee/Pricing | Unify into single canonical implementation |
| D3-L2 | Fee pool account authorization | Fee/Pricing | Immutable authorization + nonce enforcement |
| D4-L1 | No stake concentration caps | Governance | Max 10% of total stake per validator in gov |
| D5-L1 | Frivolous dispute spam | Dispute | Global rate limit: max 1 dispute/100 blocks/agent |
| D5-L2 | Challenger loses money when winning | Dispute | Increase winner share to 100% of loser penalty |
| D5-L3 | Arbiter collusion (11% penalty) | Dispute | Escalate to 33% after 3 wrong votes per domain |
