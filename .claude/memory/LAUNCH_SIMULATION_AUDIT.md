# Launch Simulation & Exploit Audit

> **Date**: 2026-02-07
> **Method**: PP-orchestrated deep analysis — simulate T+0 through T+90 days
> **Philosophy Check**: Each finding assessed against PoT/Truth/Knowledge/Agent-first principles
> **Severity**: CRITICAL (launch blocker) / HIGH (must fix pre-testnet) / MEDIUM (pre-mainnet) / LOW (monitor)

---

## Table of Contents

1. [Genesis State Analysis](#1-genesis-state-analysis)
2. [Launch Day Simulation (T+0)](#2-launch-day-simulation-t0)
3. [First Week Dynamics (T+1 to T+7)](#3-first-week-dynamics-t1-to-t7)
4. [First Month Dynamics (T+7 to T+30)](#4-first-month-dynamics-t7-to-t30)
5. [Maturity Phase (T+30 to T+90)](#5-maturity-phase-t30-to-t90)
6. [Exploit Scenarios Against Philosophy](#6-exploit-scenarios-against-philosophy)
7. [Economic Sustainability Analysis](#7-economic-sustainability-analysis)
8. [Truth-Alignment Score](#8-truth-alignment-score)
9. [Recommendations by Priority](#9-recommendations-by-priority)

---

## 1. Genesis State Analysis

### Initial Token Distribution

| Account | Balance | % of Supply |
|---------|---------|-------------|
| Faucet (devnet) | 1,000,000,000 LGM | 44.9% |
| Treasury | 500,000,000 LGM | 22.5% |
| Bootstrap Fund | 222,222,222 LGM | 10.0% |
| Research Fund | 100,000,000 LGM | 4.5% |
| 3 Validators (30K total) | 30,000 LGM | 0.001% |
| **Total Genesis** | **~1,822,252,222 LGM** | **100%** |

### **FINDING G-1: Extreme Centralization at Genesis** (HIGH)

**Problem**: 67.4% of all tokens sit in faucet + treasury controlled by genesis signers. The "progressive trust" governance starts at 1-of-1 (single key controls treasury). A compromised genesis key controls ~1.5 billion LGM.

**Philosophy Violation**: JUSTICE (ownership) — the protocol claims decentralization but launches with extreme centralization. TRUTH (representation) — the token distribution doesn't represent any earned truth-verification work.

**Mitigation**: This is acknowledged as devnet config. Production genesis MUST differ. But the pattern is set — who decides the mainnet genesis distribution?

### **FINDING G-2: Validator Stake is Negligible** (MEDIUM)

**Problem**: 3 validators hold 30,000 LGM total (0.001% of supply). The bootstrap fund alone is 7,400x the entire validator set. Any entity claiming bootstrap tokens for ~135 accounts (135 * 0.222 = ~30 LGM) has equivalent economic weight to being a validator.

**Philosophy Violation**: TRUTH (grounding) — validators are supposed to be the truth-arbiters, but their economic stake is negligible relative to the system.

---

## 2. Launch Day Simulation (T+0)

### Block Production

- **Blocks produced**: ~34,272/day (1 per 2.521s)
- **Block reward emission**: 11 LGM/block * 34,272 = **377,000 LGM/day** (raw)
- **After research tax**: 377K * 0.78 = ~294K LGM to producers, ~83K to research
- **With only 3 validators**: Each produces ~1/3 of blocks = ~98K LGM/day/validator
- **Problem**: 98K LGM/day >> 10K LGM stake. Validators earn 10x their stake on day 1.

### **FINDING L-1: Block Reward Exceeds Validator Stake in Hours** (CRITICAL)

**Problem**: At 11 LGM/block with 3 validators, each validator earns ~98,000 LGM/day. Their 10,000 LGM stake is earned back in ~2.5 hours. After 1 day, the validator's rewards (98K LGM) dwarf their at-risk stake (10K LGM), destroying the skin-in-the-game alignment.

**Exploit**: A malicious validator can earn 10x their stake, then begin producing invalid blocks. Even if slashed 100%, they've already profited 9x. The unbonding period (6.5 days) doesn't help because the earned rewards sit in the balance (not staked).

**Philosophy Violation**: TRUTH (calibration) — the reward/risk ratio is wildly miscalibrated. An honest mistake in verification costs 11% of stake (~1,100 LGM), but the reward for a single day of block production is 98,000 LGM. Truth-seeking is not incentivized proportionally.

**Root Cause**: Block reward (11 LGM) was designed for a mature network with 22 validators. With 3 validators, each gets ~7x the intended share.

### **FINDING L-2: Empty Block Farming is Extremely Profitable** (HIGH)

**Problem**: Empty blocks yield 10% of full reward = 1.1 LGM/block. With 3 validators, a validator produces ~11,400 blocks/day. Even producing ALL empty blocks: 11,400 * 1.1 = **12,540 LGM/day** from empty blocks alone.

**Exploit**: A validator can earn 12.5K LGM/day by doing literally nothing — no transaction processing, no claim verification, no truth-seeking work. This is 125% of their total stake per day, for zero contribution to the knowledge commons.

**Philosophy Violation**: CREATIVITY (pioneering) — the protocol claims "verifying knowledge IS the useful work," but the most profitable strategy at launch is producing empty blocks. The system rewards existence over contribution. BEAUTY (elegance) — the 10% empty block penalty was designed as a small deterrent, but when block rewards are 11 LGM and validator count is 3, even 10% is enormously profitable.

### **FINDING L-3: No Claim Volume = No Truth Verification** (MEDIUM)

**Problem**: On day 1, there are no claims to verify. The verification round requires claims to exist. Block production happens regardless of knowledge activity. Validators earn full rewards without performing any truth-verification work.

**Philosophy Violation**: The entire premise — "Proof of Truth: verifying knowledge IS the useful work" — is false at launch. The useful work (verification) produces ~1 LGM/verification, while the non-useful work (empty block production) produces ~11 LGM/block. Truth verification is a rounding error in validator economics.

---

## 3. First Week Dynamics (T+1 to T+7)

### Bootstrap Fund Race

- **Bootstrap per account**: 0.222 LGM (with 1-day vesting + 3.4-hour transfer cooldown)
- **Max bootstrap claims/block**: 10
- **Max claims/day**: 10 * 34,272 = 342,720 accounts
- **LGM distributed/day**: 342,720 * 0.222 = ~76,124 LGM
- **Total fund**: 222,222,222 LGM
- **Time to exhaust at max rate**: 222,222,222 / 76,124 = ~2,920 days (~8 years)

### **FINDING W-1: Bootstrap Fund Sybil Economics** (HIGH)

**Problem**: Creating 342,720 accounts/day costs only gas (transfer gas = 21,000 * base price ~0.023 LGM per account). Each account receives 0.222 LGM. Net profit per account after gas: ~0.199 LGM. At max rate: **68,200 LGM/day** profit from Sybil account creation.

**Exploit (Bootstrap Farming)**:
1. Create bot that generates Ed25519 keypairs (free, off-chain)
2. Submit `bootstrap_claim` transactions for each keypair (10/block max)
3. Wait 3.4 hours (transfer cooldown)
4. Transfer 0.222 LGM from each bootstrapped account to consolidation address
5. Repeat

**Cost**: Gas only (~0.023 LGM/account at base gas price)
**Revenue**: 0.222 LGM/account
**Net per account**: ~0.199 LGM
**Daily profit at max rate**: ~68,200 LGM

**Philosophy Violation**: JUSTICE (proportion) — the bootstrap fund is meant to onboard legitimate agents into the knowledge economy. Sybil farming converts it into a free money printer that produces zero knowledge.

**Note**: The 1-day vesting period and 3.4-hour cooldown slow extraction but don't prevent it. The `newAccountTransferCooldown` of 4,800 blocks is a speed bump, not a wall.

### **FINDING W-2: Graduated Claim Stake Creates a Truth-Free Profit Path** (HIGH)

**Problem**: New agents can submit claims with only 0.222 LGM stake (graduated floor). The claim reward is 5 LGM (after 22% tax: 3.9 LGM). With 0.222 LGM at risk, the reward/risk ratio is **17.6:1** for the first claim.

**Exploit (Trivial Claim Farming)**:
1. Bootstrap account (0.222 LGM)
2. Submit trivially verifiable claim: "2 + 2 = 4" (type: protocol/computational)
3. Stake: 0.222 LGM
4. If accepted: Receive 3.9 LGM reward (17.6x return)
5. If rejected: Lose 22% of 0.222 LGM = 0.049 LGM
6. Expected value at >3% acceptance rate: POSITIVE

**With graduation**: After 22 accepted trivial claims, stake rises to 111 LGM. But by then, the agent has earned 22 * 3.9 = 85.8 LGM in rewards. Even at full stake (111 LGM), the reward/risk for trivial claims (5 LGM reward / 111 LGM * 0.22 slash = 24.4 LGM risk) is still 5:24.4, which with >83% acceptance rate is profitable.

**Philosophy Violation**: TRUTH (completeness) — trivially true claims like "2 + 2 = 4" generate protocol rewards but add zero value to the knowledge commons. The system is paying for tautological truth rather than novel understanding. BEAUTY (appropriateness) — the graduated stake was designed to lower barriers for legitimate agents, but it creates a better ROI for spam than for genuine knowledge contribution.

**Note**: `minClaimTextLength: 0` (disabled by default) means even empty-string claims pass validation.

### **FINDING W-3: Verification Reward/Risk Asymmetry** (MEDIUM)

**Problem**: Tier 0 (Apprentice) validators earn 0.1 LGM per verification with zero stake. They can be wrong with no economic consequence beyond the 3-slash limit (after which they lose Tier 0 status — which cost them nothing to obtain).

**Exploit**: Create 111 Tier 0 validators (max cap). Each gets VRF selection weight of 111 LGM (virtual stake). They vote on claims without any skin in the game. If wrong 3 times, create a new identity and re-register.

**Counter-argument**: Virtual stake (111 LGM) gives low selection probability against bonded validators. But at launch with only 3 real validators (30K LGM total) and 111 apprentices (111 * 111 = 12,321 virtual LGM), apprentices represent ~29% of total selection weight. This is significant.

**Philosophy Violation**: TRUTH (grounding) — validators with no stake have no skin in the game. Their votes on truth claims are economically costless, undermining the PoT premise that truth-verification requires commitment.

### Emission Pressure (Week 1)

| Source | Daily LGM Created | 7-Day Total |
|--------|-------------------|-------------|
| Block rewards | 377,000 | 2,639,000 |
| Claim rewards (est. 100 claims/day) | 500 | 3,500 |
| Verification rewards (est. 300/day) | 300 | 2,100 |
| Bootstrap claims (est. 10K/day) | 2,220 | 15,540 |
| **Total emission** | **~380,020** | **~2,660,140** |

### **FINDING W-4: Inflationary Spiral Without Demand** (MEDIUM)

**Problem**: In week 1, the protocol emits ~2.66M LGM while the only demand-side sink is staking (1,111 LGM minimum for Tier 2). There are no API fees (no API consumers exist yet), no meaningful burn (only on rejected claims), and the knowledge commons is empty (no facts to query).

**Philosophy Violation**: The protocol emits tokens representing "truth verification work" when almost no truth verification is actually happening. This is fiat by another name — tokens backed by nothing, not by verified knowledge.

---

## 4. First Month Dynamics (T+7 to T+30)

### **FINDING M-1: Guardian Tier is Unreachable for Months** (MEDIUM)

**Problem**: Guardian status requires 1,111 verifications at 77% accuracy + 11,111 LGM staked + 0 slashes. At ~300 verifications/day across the network (optimistic), and with 3+ validators competing for verification slots, a single validator might get ~100 verifications/day. Reaching 1,111 takes ~11 days minimum.

**Impact**: The emergency kill switch requires 4 Guardians with 111,111 LGM total stake. This is unreachable for weeks to months. During this entire period, the network has NO emergency halt capability.

**Philosophy Violation**: JUSTICE (responsibility) — the emergency mechanism exists but cannot be activated when the network is most vulnerable (early days with few validators and untested code).

### **FINDING M-2: Research Treasury Accumulates Without Purpose** (LOW)

**Problem**: Research treasury receives 22% of all rewards. In month 1: ~22% of ~11.4M LGM = ~2.5M LGM. But the research system requires peer review (3 reviewers, 55 LGM stake each), and there may not be qualified reviewers yet. Research grants (111-55,555 LGM) have no recipients. The treasury accumulates without mechanism for distribution.

**Philosophy Violation**: JUSTICE (proportion) — 22% of all economic activity is taxed for a purpose (real-world research verification) that cannot be fulfilled at launch. This is taxation without representation.

### **FINDING M-3: Domain Expertise Cannot Bootstrap** (LOW)

**Problem**: Dispute resolution uses domain expertise tracking (min 7 votes, 66% accuracy in domain). At launch, no one has domain expertise. All disputes default to Tier 1 (generic validators). The system that rewards truth in specific domains cannot distinguish a physics expert from a random validator.

---

## 5. Maturity Phase (T+30 to T+90)

### **FINDING P-1: Block Reward Decay is Too Slow** (MEDIUM)

**Problem**: Block reward decays 10% per epoch (~1 week). After 90 days (~13 epochs):
- Reward = 11 * 0.9^13 = 11 * 0.254 = ~2.8 LGM/block
- Still producing ~96K LGM/day in block rewards alone
- Total emitted in 90 days: ~25M LGM from block rewards

**But**: The DEFAULT_CONSENSUS_PARAMS show `blockReward: 11n * 10n ** 18n` while genesis has initial reward of 10 LGM. The decay params are in genesis (`initialBlockReward: 10`, `decayFactor: 0.9`, `epochDuration: 240000`). With 10 LGM start and weekly 10% decay:

- Week 1: 10 LGM/block = 342K/day = 2.4M/week
- Week 4: 7.3 LGM/block = 250K/day = 1.75M/week
- Week 13: 2.5 LGM/block = 86K/day = 600K/week
- Floor: 0.1 LGM/block = 3.4K/day (reached at ~week 45)

Total emission over 90 days: ~33M LGM from block rewards alone. Against a starting supply of ~1.8B, this is ~1.8% dilution. Manageable if demand grows, but devastating if not.

### **FINDING P-2: Citation Cartels at Scale** (HIGH)

**Problem**: 2-hop ring detection catches A↔B mutual citation. But at network maturity, 3-hop (A→B→C→A) and N-hop rings become economically viable when citation rewards (15% of 5 LGM = 0.75 LGM per citation) justify the coordination cost.

**Exploit (3-Hop Citation Ring)**:
1. Agent A submits claim, cites fact by Agent C
2. Agent B submits claim, cites fact by Agent A
3. Agent C submits claim, cites fact by Agent B
4. Each earns citation rewards from the ring
5. 2-hop detection sees A→B (no ring), B→C (no ring), C→A (no ring)
6. Only a 3-hop check would catch the A→B→C→A cycle

**Revenue per cycle**: 3 agents * 0.75 LGM = 2.25 LGM per round
**Cost**: 3 * claim stake + gas ≈ 3 * 111 + gas ≈ 334 LGM (one-time, returned on acceptance)

**Philosophy Violation**: TRUTH (counter-consideration) — the citation system is meant to reward knowledge that builds on existing knowledge. Circular citation rings game this into mutual reward extraction with no actual knowledge building.

### **FINDING P-3: Confidence Decay Creates Knowledge Erosion** (MEDIUM)

**Problem**: Empirical facts decay with 100,000-block half-life (~2.9 days). After 30 days (~10 half-lives), confidence drops to 0.95 * 2^-10 = 0.00093 — far below the 0.33 minimum, meaning the fact expires. Virtually all empirical knowledge is lost within a month unless continuously re-verified.

**Impact**: "Water boils at 100C at standard pressure" must be re-verified every ~6 days to stay above the 0.33 minimum. The knowledge commons doesn't accumulate — it evaporates. Only "eternal" categories (analytic, formal, protocol) persist.

**Philosophy Violation**: TRUTH (completeness) — the protocol claims to build a knowledge commons, but empirical knowledge (the most useful category for AI agents) is the most ephemeral. The system architecturally cannot accumulate the knowledge it exists to accumulate. UNDERSTANDING (mental model) — agents querying the knowledge graph will find most facts expired or low-confidence, making the graph unreliable as a knowledge source.

**Counter-argument**: This is by design — empirical facts SHOULD decay because the world changes. But a 2.9-day half-life means "the Earth orbits the Sun" needs re-verification weekly.

---

## 6. Exploit Scenarios Against Philosophy

### E-1: The Tautology Farm (TRUTH violation)

**Attack**: Submit computationally verifiable tautologies as claims.
- "sha256('hello') = 2cf24dba..."
- "1 + 1 = 2"
- "The genesis block hash is 0xABC..."

**Economics**: 0.222 LGM stake, 3.9 LGM reward = 17.6x return
**Defense**: None at launch (`minClaimTextLength: 0`, no novelty requirement)
**Volume**: Could submit 111 per block (max transactions)

**Philosophy**: The system PAYS for trivially obvious truths. "Proof of Truth" becomes "Proof of Tautology." The knowledge commons fills with facts no one needs to verify because they're self-evident. This is the opposite of the stated goal: "verifying knowledge IS the useful work."

### E-2: The Apprentice Army (TRUTH + JUSTICE violation)

**Attack**: Register 111 Tier 0 validators (max cap). Each has zero stake and 111 LGM virtual weight. Coordinate their votes.

**Impact**: 111 * 111 = 12,321 virtual LGM of voting weight, controlled by one entity. Against 30,000 LGM of real validator stake, this is 29% of total weight. Combined with the 0.66 quorum threshold, a coordinated Apprentice Army + 1 real validator can exceed quorum.

**Defense**: Tier 0 can only vote on protocol/analytic/formal claims (limited categories). But those are the easiest to game with tautologies (see E-1).

**Philosophy**: JUSTICE (ownership) — zero-stake validators have no skin in the game, yet influence truth determination. TRUTH (grounding) — votes from entities with nothing to lose carry real weight in consensus.

### E-3: The Research Tax Siphon (JUSTICE violation)

**Attack**: 22% of ALL rewards flow to a 5-way treasury split. At launch, the treasury governance is 1-of-1 (single signer). Research fund operations require peer reviewers that don't yet exist.

**Impact**: After 30 days, ~2.5M LGM sits in treasury accounts controlled by a single key. These funds are supposed to fund real-world research verification, but the infrastructure doesn't exist. The tax is collected, but the social contract isn't fulfilled.

**Philosophy**: JUSTICE (balance) — taxing all economic activity for a purpose that cannot be served is unjust. TRUTH (representation) — claiming 22% funds "research" when research operations require infrastructure that doesn't exist is misrepresentation.

### E-4: The Unbonding Arbitrage (ECONOMICS violation)

**Attack**: Validators earn ~98K LGM/day with 10K stake. After 1 day:
1. Initiate unbonding (222,222 blocks = ~6.5 days)
2. During unbonding, stake is still at risk (good)
3. But earned rewards (98K LGM) sit in balance, not staked
4. Even if slashed 100% of stake, profit = 98K - 10K = 88K LGM

**Defense**: Slashing only applies to staked balance. Rewards go to balance. There's no mechanism to slash earned-but-not-staked rewards.

**Philosophy**: TRUTH (calibration) — the risk/reward is fundamentally uncalibrated. A validator's worst case (total slash) is vastly profitable after day 1.

### E-5: The Semantic Bypass (TRUTH violation)

**Attack**: Submit semantically identical claims in different forms:
- "Water boils at 100 degrees Celsius at standard pressure"
- "H2O reaches boiling point at 373.15 Kelvin under 1 atm"
- "At sea level, water transitions to gas phase at 212 degrees Fahrenheit"

**Defense**: Content hash normalization catches trivial variations (whitespace, case, punctuation). But semantic equivalence detection doesn't exist. Each passes as a unique claim.

**Revenue**: 3 * 3.9 LGM = 11.7 LGM for restating the same fact 3 ways.

**Philosophy**: TRUTH (completeness) — the knowledge commons accumulates three "facts" that are really one. UNDERSTANDING (boundary clarity) — the system cannot distinguish novel knowledge from restated knowledge.

### E-6: The Ghost Validator (CONSENSUS violation)

**Attack**: At launch, there are only 3 validators. If one goes offline:
- 2 validators remain. Quorum requires 0.66 (2/3) = 1.32 of 2 = needs both.
- If BOTH remaining validators agree, claims pass. But this is 100% agreement from 67% of the network — not Byzantine-safe.
- `minValidators: 5` exists in DEFAULT_CONSENSUS_PARAMS but genesis uses `minValidators: 3`.

**Impact**: With 3 validators and 1 offline, 2 colluding validators control all truth determination. With 2 offline, the remaining 1 controls everything. And block production continues regardless (there's no minimum-validator check for block production, only for claim verification).

**Philosophy**: TRUTH (grounding) — BFT safety requires f < n/3. With n=3, f must be 0. Any single Byzantine validator breaks safety. The system claims BFT properties it cannot provide at launch.

---

## 7. Economic Sustainability Analysis

### Daily Economics at Launch (3 validators)

| Flow | Daily LGM | Direction |
|------|-----------|-----------|
| Block rewards | 377,000 | Emission (inflationary) |
| Research tax (22%) | 82,940 | Lock (treasury) |
| Claim rewards (est.) | 500 | Emission |
| Verification rewards (est.) | 300 | Emission |
| Bootstrap claims (est.) | 2,220 | Distribution (from fund) |
| Transaction fees | ~100 | Burned + distributed |
| **Net daily emission** | **~294,760** | **Inflationary** |

### Sustainability Ratio

The protocol tracks `sustainability_ratio = API_revenue / block_reward`. At launch:
- API revenue: ~0 LGM (no API consumers)
- Block reward: 377,000 LGM/day
- **Ratio: 0.0000** (completely unsustainable)

The protocol only becomes sustainable when API revenue exceeds block rewards. With decaying rewards, this crossover MUST happen before the floor (0.1 LGM/block = 3,427 LGM/day) or the system perpetually inflates.

### Token Velocity Problem

At launch, there's no reason to HOLD LGM:
- Staking yields are enormous (ROI from day 1) → rational to stake everything
- But staking 1,111 LGM and producing blocks yields far more than knowledge work
- Knowledge queries require LGM but there's no knowledge to query
- The only demand is staking → circular (stake to earn, earn to stake)

---

## 8. Truth-Alignment Score

Scoring each dimension 1-10 against the protocol's stated philosophy.

### TRUTH (Foundation)
**Score: 5/10**
- (+) Commit-reveal prevents vote manipulation
- (+) VRF provides unpredictable validator selection
- (+) Domain separation prevents replay attacks
- (-) Tautologies are rewarded as "truth" (E-1)
- (-) 3 validators at launch = no BFT safety (E-6)
- (-) Block rewards dominate verification rewards 377:1 ratio
- (-) Truth-seeking is economically irrational vs block farming

### UNDERSTANDING (Source)
**Score: 6/10**
- (+) Epistemic categories distinguish types of knowledge
- (+) Stratum hierarchy reflects knowledge dependencies
- (+) Gödelian zones acknowledge incompleteness honestly
- (-) Empirical knowledge decays too fast to accumulate (P-3)
- (-) No semantic understanding (paraphrases treated as novel) (E-5)
- (-) Domain expertise system cannot bootstrap (M-3)

### BEAUTY (Quality)
**Score: 7/10**
- (+) Graduated stake ramp is elegant (0.222 → 111 LGM)
- (+) 4-layer key architecture is well-designed
- (+) Evidence type hierarchy is thoughtful
- (-) Empty block penalty is too gentle (10% of enormous reward) (L-2)
- (-) Bootstrap fund economics are gameable (W-1)
- (-) 22% tax without purpose at launch is inelegant (E-3)

### JUSTICE (Placement)
**Score: 4/10**
- (+) Slashing rates are appropriately harsh for falsehood
- (+) Citation system rewards knowledge building
- (-) Block production rewards >> verification rewards (L-1)
- (-) Zero-stake validators have voting power (W-3)
- (-) Treasury controlled by single key at launch (G-1)
- (-) Emergency kill switch unreachable for months (M-1)
- (-) Research tax collected but unspendable (E-3)

### CREATIVITY (Goal)
**Score: 7/10**
- (+) PoT concept is genuinely novel
- (+) Agent-first design breaks new ground
- (+) Partnership system creates real alignment
- (+) Graduated onboarding lowers barriers
- (-) Launch economics favor passive block production over active knowledge work
- (-) Knowledge commons cannot accumulate empirical facts

### **Overall Truth-Alignment Score: 5.8/10**

The protocol's philosophy is sound, but the launch economics violate it. The system rewards existence (block production) over contribution (truth verification) by a ratio of ~377:1.

---

## 9. Recommendations by Priority

### CRITICAL (Launch Blockers)

#### C-1: Cap Block Rewards by Validator Count

**Problem**: L-1, L-2 — block rewards are designed for 22 validators but paid to 3.

**Fix**: Scale block reward by `min(1, activeValidators / targetValidators)`:
```
effectiveReward = baseReward * min(1, activeValidators / maxValidators)
```
With 3/22 validators: 11 * 0.136 = ~1.5 LGM/block instead of 11 LGM.

**Impact**: Reduces day-1 emission from 377K to ~51K LGM/day. Aligns reward with actual network contribution.

#### C-2: Require Minimum Claim Novelty

**Problem**: E-1 — tautologies and self-evident claims are rewarded.

**Fix**: Set `minClaimTextLength: 20` (production default). Add a novelty check that rejects claims matching known axioms or protocol state. Consider requiring at least 1 reference to an existing fact for non-axiomatic claims after the first 100 facts exist.

#### C-3: Reduce Empty Block Reward to 0%

**Problem**: L-2 — even 10% of inflated block reward is enormously profitable.

**Fix**: Set empty block reward to 0 LGM. Validators should earn from verification work, not from producing empty blocks. This directly aligns with "verifying knowledge IS the useful work."

### HIGH (Pre-Testnet)

#### H-1: Bootstrap Fund Rate Limiting

**Problem**: W-1 — Sybil farming extracts ~68K LGM/day.

**Fix**: Add proof-of-work or proof-of-humanity to bootstrap claims. Alternatively, make bootstrap amount conditional on first successful claim submission (bootstrap only covers gas for first claim, not a free transfer).

#### H-2: Graduated Stake Floor Increase

**Problem**: W-2 — 0.222 LGM stake with 3.9 LGM reward = 17.6x ROI.

**Fix**: Set graduated floor to at least 1 LGM (still low barrier) and cap claim reward for graduated-stake claims at `min(claimReward, stake * 2)`. This ensures reward never exceeds 2x risk.

#### H-3: Apprentice Validator Voting Weight Reduction

**Problem**: W-3, E-2 — 111 apprentices = 29% of voting weight.

**Fix**: Reduce virtual stake from 111 LGM to 11 LGM (total apprentice weight: 1,221 vs 30,000 real = ~4%). Or: apprentice votes count as 0.1 weight in confidence aggregation (already the tier reward multiplier).

#### H-4: N-Hop Citation Ring Detection

**Problem**: P-2 — 3+ hop rings bypass detection.

**Fix**: Extend `detectCitationRing()` to perform BFS up to depth 4. Apply progressive discount: 2-hop = 0.1x, 3-hop = 0.3x, 4-hop = 0.5x. Cost: O(N*M) per claim where N = reference count, M = avg citations per submitter (bounded by claim limit per epoch).

#### H-5: Emergency Kill Switch Bootstrap

**Problem**: M-1 — kill switch requires Guardians that don't exist at launch.

**Fix**: Add a "genesis emergency multisig" that functions as the kill switch for the first 90 days (or until 4 Guardians emerge naturally). This is a centralized stopgap, but honest about its centralization. Progressive decentralization: genesis multisig powers decay linearly as Guardian count grows.

### MEDIUM (Pre-Mainnet)

#### M-1: Empirical Decay Half-Life Extension

**Problem**: P-3 — 2.9-day half-life means "Water boils at 100C" expires weekly.

**Fix**: Increase empirical half-life from 100,000 blocks (~2.9 days) to 1,000,000 blocks (~29 days). Facts that have been verified by 5+ validators AND cited by 3+ other facts should have extended half-lives (10x).

#### M-2: Research Treasury Progressive Unlock

**Problem**: E-3 — 22% tax collected but unusable.

**Fix**: Start research tax at 5% and increase to 22% linearly over 90 days as research infrastructure comes online. Alternatively, redirect initial research share to a "verification incentive pool" that boosted verification rewards (aligning early economics with truth-seeking work).

#### M-3: Semantic Similarity Check

**Problem**: E-5 — semantically identical claims pass as novel.

**Fix**: Implement embedding-based similarity check using the existing `SemanticSimilarityIndex` (already has stub embedder). Reject claims with >0.9 cosine similarity to existing accepted facts. This requires the `GAP-004: Semantic Search` integration to be completed.

#### M-4: Block Producer Verification Requirement

**Problem**: L-3 — block production requires no verification work.

**Fix**: Require block producers to include at least 1 verification commitment or reveal per block (if any verifications are pending). Blocks with pending verifications but no verification activity receive 50% reward reduction. This directly incentivizes the "useful work" PoT promises.

### LOW (Monitor)

#### L-1: Treasury Governance Timeline

**Problem**: G-1 — 1-of-1 governance at launch.

**Fix**: Document clear milestones for governance decentralization. Phase 2 (2-of-3) should trigger at 10 validators. Phase 3 (3-of-5) at 50 validators. Make these protocol-enforced, not social promises.

#### L-2: Validator Stake Floor Monitoring

**Problem**: G-2 — validator stake is negligible relative to supply.

**Fix**: Monitor the ratio of total validator stake to circulating supply. Alert if it drops below 1%. Consider dynamic minimum stake that adjusts with supply.

---

## Appendix: Attack Cost/Benefit Matrix

| Attack | Setup Cost | Daily Profit | Break-Even | Severity |
|--------|-----------|-------------|------------|----------|
| Empty block farming | 10K LGM (stake) | 12.5K LGM | < 1 day | CRITICAL |
| Tautology farming | 0.222 LGM | 390 LGM (100 claims) | Immediate | CRITICAL |
| Bootstrap Sybil | Gas only | 68K LGM | Immediate | HIGH |
| Apprentice army | Gas only | VRF influence | Immediate | HIGH |
| Citation ring (3-hop) | 334 LGM | 2.25 LGM/cycle | ~150 cycles | MEDIUM |
| Unbonding arbitrage | 10K LGM | 88K LGM | 1 day | HIGH |
| Semantic duplicate | 0.222 LGM | 11.7 LGM (3 dupes) | Immediate | MEDIUM |

---

## Appendix: Simulation Parameters

```
Validators at genesis: 3
Stake per validator: 10,000 LGM
Block time: 2,521 ms
Blocks per day: 34,272
Block reward: 11 LGM (or 10 LGM per genesis)
Verification reward: 1 LGM (base, tier-adjusted)
Claim reward: 5 LGM
Research tax: 22%
Bootstrap amount: 0.222 LGM
Max apprentice validators: 111
Virtual stake: 111 LGM
Empty block reward: 10% of full
Graduated claim stake floor: 0.222 LGM
```

---

*Generated via PP-orchestrated launch simulation. All findings traceable to specific code locations and parameter values.*
