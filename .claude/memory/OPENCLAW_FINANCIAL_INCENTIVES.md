# OpenClaw Agent Testing Session: Financial Incentive Exploits & Empty Work

> **Session**: OpenClaw x DevNet x BVM | **Date**: 2026-02-17
> **Test file**: `x/bvm/vm/empty_work_test.go` | **Result**: 11/11 PASS
> **Scope**: Token farming attacks against Proof of Truth incentive system
> **Goal**: Identify and close loopholes that enable empty work for token rewards

---

## Executive Summary

The Legible Money protocol's Proof of Truth consensus creates a rich incentive landscape where agents earn tokens by submitting and verifying knowledge claims. Through 10 BVM bytecode attack simulations, we discovered **8 exploitable loopholes** that allow agents to earn tokens through empty or minimal-effort work. Only 2 of the 10 attack vectors are adequately defended by the current evaluation pipeline.

**Critical finding**: The most dangerous exploit is the **Calibration Paradox** (FARM-2) — the calibration system designed to reward good track records actually incentivizes trivial claim spam, because a high acceptance rate earns a +5% bonus rather than raising suspicion.

**Second most dangerous**: **Self-citation loops** (FARM-3) allow agents to build permanent passive income streams from self-referencing knowledge graphs with no restriction.

---

## Attack Taxonomy

| ID | Attack | Verdict | Urgency |
|----|--------|---------|---------|
| FARM-1 | Rubber-stamp verifier | PARTIALLY DEFENDED | Medium |
| FARM-2 | Trivial claim spam | **LOOPHOLE** | **CRITICAL** |
| FARM-3 | Self-citation loop | **LOOPHOLE** | **CRITICAL** |
| FARM-4 | Confidence herding | **LOOPHOLE** | High |
| FARM-5 | Abstain farming | NO PROFIT (grief) | Low |
| FARM-6 | Vesting while misbehaving | **LOOPHOLE** | **CRITICAL** |
| FARM-7 | Domain squatting | **LOOPHOLE** | High |
| FARM-8 | Passive staking | **LOOPHOLE** | High |
| FARM-9 | Challenge spam | PARTIALLY DEFENDED | Medium |
| FARM-10 | Minimum effort evidence | DEFENDED | Low |

---

## Detailed Findings

### FARM-1: Rubber-Stamp Verifier
**Test**: `TestFarm_RubberStampVerifier`

**Attack**: Agent always votes "accept" on every claim without evaluating truth. Collects verification rewards when correct, pays 6% wrong-verdict slash when wrong.

**Economics**:
- Verification reward: 1.0 LGM x tier multiplier x difficulty bonus
- Wrong verdict slash: 6% of stake (WrongVerificationSlashBps = 60,000)
- Break-even: profitable when network truth rate >94%

**Analysis**: At a healthy 75% truth rate, rubber-stamping is unprofitable (25 wrong x 6% = 150% stake lost vs 75 correct x reward = 75 LGM earned). But in mature, high-truth domains where most claims are obviously correct, the truth rate approaches 95%+ and rubber-stamping becomes profitable.

**Current defenses**: Wrong verdict slash (6%), tier progression requires accuracy, difficulty bonus rewards careful evaluation.

**Gap**: No verifier-specific calibration tracking. The calibration system only tracks submitter accuracy, not verifier behavior patterns.

**Recommendation**: Add verifier calibration — agents who always agree should face diminishing rewards. Track "always-accept ratio" and reduce rewards when it exceeds 90%.

---

### FARM-2: Trivial Claim Spam (CRITICAL)
**Test**: `TestFarm_TrivialClaimSpam`

**Attack**: Agent submits obvious, universally-known claims at minimum stake. "Water contains hydrogen" costs 0.001 LGM and earns vesting + citation eligibility if accepted.

**Economics**:
- Cost per claim: MinClaimStakeUlgm = 1000 (0.001 LGM) + gas
- If accepted: stake returned + vesting schedule + citation eligibility
- If rejected: 22% slash = 0.00022 LGM lost
- For obvious truths: ~99% acceptance rate
- Expected loss per claim: 0.01 x 0.00022 = 0.0000022 LGM
- At scale: 1000 trivial claims = ~0.999 LGM in vesting schedules

**The Calibration Paradox**: The calibration stage (Stage 2 of the evaluation pipeline) gives a +5% bonus for agents with high acceptance rates. This means the trivial claim spammer gets REWARDED for their farming behavior — high acceptance rate is treated as "good track record" instead of "suspiciously trivial."

**Current defenses**: Content hash prevents exact duplicates. Novelty scoring penalizes mature domains. But neither prevents *slightly different* trivial claims in the same domain.

**This is the most dangerous exploit because**:
1. The defense mechanism (calibration) actively rewards the attacker
2. Minimum stake is very low (0.001 LGM)
3. No per-agent rate limit exists
4. Trivial truths pass all quality gates

**Recommendations**:
1. Add per-agent claim rate limit (max 10 claims per epoch)
2. Invert calibration: penalize agents with >95% acceptance rate (suspiciously trivial)
3. Require minimum novelty score >0 for vesting eligibility
4. Increase minimum stake for "computational" and "analytic" categories

---

### FARM-3: Self-Citation Loop (CRITICAL)
**Test**: `TestFarm_SelfCitationLoop`

**Attack**: Agent submits claim A, then claim B citing A, then C citing B. Each citation earns 15% of verification rewards from downstream claims. The agent builds a self-referential citation tree for permanent passive income.

**Economics**:
- Citation pool: 15% of verification rewards (CitationShareBPS = 150,000)
- Decay per level: 50% (CitationDecayPerLevel = 500,000)
- Protocol floor: 2% per citation (CitationProtocolFloorBPS = 20,000)

Self-citation tree income from one downstream claim:
```
Direct citation (C):  15.00%
Via C->B:              7.50%
Via B->A:              3.75%
TOTAL:                26.25% passive income
```

With 10 self-citing chains of depth 5: 50 facts generating permanent passive revenue from all downstream citations in those domains.

**Current defenses**: Citation decay (50% per level), cross-domain bonus only for verified refs (>70%). But there is **no explicit self-citation check**.

**This is critical because**: The citation system was designed to reward foundational knowledge contributions. Self-citation perverts this into a rent-seeking mechanism. An agent can monopolize citation income in a domain by being first and citing itself.

**Recommendations**:
1. Add explicit self-citation discount (50% reduced citation reward for same DID)
2. Require minimum N unique citers before citation rewards activate
3. Cap citation depth for single-submitter chains at 2 levels

---

### FARM-4: Confidence Herding (Consensus Copying)
**Test**: `TestFarm_ConfidenceHerding`

**Attack**: Agent reads network consensus via `COMPARE_BELIEF` (10 gas), then sets its own belief to match exactly. Zero divergence = zero slash risk.

**Economics**:
- Cost: 10 gas (COMPARE_BELIEF) + 500 gas (SET_BELIEF) = 510 gas
- Benefit: Zero divergence slashing risk, guaranteed steady income
- Sacrifice: Difficulty bonus (up to +70% for borderline claims)

Per 100 verifications:
- Independent judgment: 80 correct x 1.2 LGM - 20 wrong x 0.06 stake = high variance
- Herding: 100 x 1.0 LGM - 0 slashing = guaranteed income

**Current defenses**: Method diversity detection (AH-4) catches identical confidence across multiple agents. But a single herder is indistinguishable from a genuinely agreeing agent.

**The deeper problem**: `COMPARE_BELIEF` is a read-only opcode that reveals consensus before commitment. This is like seeing the answers before the test. It enables rational free-riding.

**Recommendations**:
1. Make COMPARE_BELIEF available only AFTER voting commit (not before)
2. Add "independence score" tracking: agents whose beliefs always match consensus get penalized
3. Require "verification method" declaration that differs from the consensus method

---

### FARM-5: Abstain Farming
**Test**: `TestFarm_AbstainFarming`

**Attack**: Agent always votes "abstain" — no slash risk, counts toward quorum, may block other validators from being selected.

**Economics**: Zero direct profit, zero direct cost (aside from gas).

**Verdict**: **Not directly profitable, but griefs the network.** If 40%+ of selected validators abstain, quorum may not be reached, causing round failure. Knowledge accumulation stalls.

**Current defenses**: Abstain earns 0 verification reward.

**Gap**: No penalty for serial abstaining. No mechanism to replace abstainers mid-round.

**Recommendations**:
1. Add "abstain tax": repeated abstain (>50% of rounds) triggers 0.5% stake penalty
2. Replace abstainers with alternates from VRF queue
3. Track abstain ratio in tier calculation (high abstain blocks tier advancement)

---

### FARM-6: Vesting While Misbehaving (CRITICAL)
**Test**: `TestFarm_VestingWhileMisbehaving`

**Attack**: Agent builds up vesting schedules from correct early work, then starts misbehaving (false claims, wrong votes) while vesting continues to release tokens. The "grief-and-extract" pattern.

**Economics**:
```
Epoch  1-10: Honest work → earns vesting schedules
Epoch 11+:   Misbehavior → 22% slash per rejected claim
Vesting:     CONTINUES RELEASING throughout misbehavior
```

If vesting release rate > misbehavior cost, the agent profits from the attack.

**Current defenses**: Vesting clawback exists but only triggers when a SPECIFIC FACT is disproven. General misbehavior does not pause vesting.

**This is critical because**: It creates a perverse lifecycle where agents have an incentive to build reputation honestly, then extract value through misbehavior once their vesting base is large enough. The vesting system assumes continued good behavior but doesn't enforce it.

**Recommendations**:
1. Pause ALL vesting when agent's recent rejection rate exceeds 30%
2. Add "behavior bond": fraction of vesting withheld based on rolling accuracy
3. Slash vesting schedules proportionally when agent is slashed for wrong verdicts

---

### FARM-7: Domain Squatting for Novelty Bonus
**Test**: `TestFarm_DomainSquatting`

**Attack**: Agent rushes to submit first claims in empty/new domains. Novelty scoring gives bonus for sparse domains. Early mover gets permanent citation income from subsequent claims.

**Economics**:
- Cost: 5 x 0.001 LGM = 0.005 LGM to squat 5 domains
- Revenue: Permanent citation stream from all domain-foundational claims
- For derivable categories (math, logic): self-evidence via on-chain computation (strength 950,000)

**Current defenses**: Evidence quality gate blocks novelty bonus without evidence. But "computational" and "analytic" categories can provide self-evidence (on-chain computation).

**Recommendations**:
1. Time-decay novelty bonus (first 1000 blocks only)
2. Cap citation income per submitter at 30% of total citation pool
3. Require minimum 3 unique citers before novelty bonus activates
4. Add "domain maturity" threshold: no novelty bonus until domain has 5+ contributors

---

### FARM-8: Staking Without Verifying
**Test**: `TestFarm_StakeWithoutVerifying`

**Attack**: Agent stakes tokens to earn block rewards but never participates in knowledge verification. Pure financial staking, zero epistemic work.

**Economics**:
- Block production rewards: 50% of total emissions, distributed proportional to stake weight
- Verification rewards: 50% of total emissions, requires actual participation
- A passive staker earns ~46.5% of emission share for their stake ratio without verifying a single claim

**Current defenses**: Participation scaling reduces total rewards when fewer validators participate. But individual validators aren't penalized for inactivity.

**The PoT design flaw**: Block production rewards are stake-weighted, not work-weighted. This means a significant portion of emissions flow to validators who don't perform the protocol's core function (truth verification). This is antithetical to Proof of Truth — the whole point is that useful work IS consensus.

**Recommendations**:
1. Require minimum verification participation rate (50% of assigned rounds)
2. Scale block production reward by verification activity ratio
3. Add "liveness jailing": validators inactive >1 day get jailed (no rewards)
4. Separate staking for security from staking for verification

---

### FARM-9: Challenge Spam for Profit
**Test**: `TestFarm_ChallengeSpam`

**Attack**: Agent challenges high-stake facts at minimum challenge cost. At 22% slash rate and 30% reward on success, asymmetric stakes make this a profitable lottery.

**Economics** (asymmetric stake scenario):
```
Original claim stake: 100 LGM
Challenge stake: 11 LGM (minimum)
Success reward: 100 x 0.22 x 0.30 = 6.6 LGM
Failure cost: 11 x 0.22 = 2.42 LGM
Break-even: 2.42 / (2.42 + 6.6) = 27% success rate!
```

**Current defenses**: 22% slash on failed challenge, cooldown period, max 3 concurrent challenges.

**Gap**: Challenge stake is independent of target stake. This creates asymmetric payoffs that favor the challenger when targeting high-value facts.

**Recommendations**:
1. Scale challenge stake to target stake (min 50% of target)
2. Increase failed challenge slash to 33% (match contradiction slash)
3. Add "challenger accuracy" tracking: serial failed challengers face increasing slash
4. Cap reward at 100% of challenger's own stake

---

### FARM-10: Minimum Effort Evidence (DEFENDED)
**Test**: `TestFarm_MinimumEffortEvidence`

**Attack**: Agent submits cheapest evidence type (attestation, strength 500,000) for observable claims.

**Defense effectiveness**:
- Stage 3 (Participant Context): -10% penalty for agent-only observable evidence
- Stage 3 ceiling: 70% for agent-only empirical claims
- Stage 4 (Evidence): Attestation = 5% bonus vs on_chain = 15% bonus
- Net result: Lazy evidence gets ~55% confidence vs quality evidence at ~75%

**Verdict**: **DEFENDED.** The evaluation pipeline genuinely penalizes lazy evidence through three independent mechanisms. The 20% confidence gap translates directly to lower vesting value.

**Remaining risk**: Derivable categories (no evidence requirement) have no "effort quality" gate. Agents can submit formal proofs of trivial tautologies with zero evidence.

---

## Synthesis: Five Laws of Empty Work Economics

### 1. Calibration Rewards Exploitation
The calibration system rewards high acceptance rates, which trivial claims achieve by definition. The system conflates "often correct" with "high quality." This is the single most impactful design flaw.

### 2. Self-Referential Value Capture
Citation income was designed for organic knowledge graph growth. Self-citation perverts it into rent-seeking. Without graph topology constraints, any agent can build income-generating citation pyramids.

### 3. Time-Inconsistent Incentives
Vesting creates inter-temporal incentive misalignment. What's rewarded (past good behavior) becomes decoupled from current behavior. The "grief-and-extract" pattern exploits this gap.

### 4. Asymmetric Risk Creates Speculation
Whenever the reward for challenging exceeds the cost in expected value, challenges become a speculative instrument rather than an epistemic correction mechanism. Asymmetric stakes amplify this.

### 5. Work ≠ Stake
Proof of Truth's core promise is that useful work (truth verification) IS the consensus mechanism. But block production rewards flow by stake weight, not work contribution. This undermines the protocol's philosophical foundation.

---

## Priority Remediation Matrix

### CRITICAL (implement before mainnet)

| Fix | Targets | Implementation |
|-----|---------|---------------|
| **Calibration inversion** | FARM-2 | Penalize >95% acceptance rate instead of rewarding it |
| **Self-citation cap** | FARM-3 | 50% reward discount for same-DID citations; min 3 unique citers |
| **Agent-level vesting pause** | FARM-6 | Pause all vesting when rejection rate >30% in rolling window |

### HIGH (implement before DevNet launch)

| Fix | Targets | Implementation |
|-----|---------|---------------|
| **COMPARE_BELIEF timing** | FARM-4 | Only available after commit phase, not before |
| **Domain maturity gate** | FARM-7 | No novelty bonus until 5+ contributors in domain |
| **Verification liveness** | FARM-8 | Block rewards scaled by verification activity ratio |
| **Challenge stake scaling** | FARM-9 | Min challenge stake = 50% of target stake |

### MEDIUM (implement post-launch)

| Fix | Targets | Implementation |
|-----|---------|---------------|
| **Verifier calibration** | FARM-1 | Track always-accept ratio; diminish rewards >90% |
| **Abstain tax** | FARM-5 | 0.5% stake penalty when abstain rate >50% |
| **Claim rate limit** | FARM-2 | Max 10 claims per agent per epoch |

---

## Cross-Reference to Existing PP Reviews

These findings complement and extend prior PP security reviews:

- **Financial Incentives Review** (56 loopholes, 6 CRITICAL): FARM-2, FARM-3, FARM-6 are new findings not in the original 56
- **Gas Calibration Review** (Score 33/50): FARM-1 relates to verifier-side calibration gaps
- **Staking Reward Review** (Grade 7/10, 5 CRITICAL): FARM-8 confirms the passive staking concern
- **Capture Challenge Review** (REMEDIATED): FARM-9 shows asymmetric challenge stakes were not fully addressed
- **Collaboration & Game Theory** (Score 46/50): FARM-4 (herding) maps to the conformity pressure identified in money psychology session

---

## Test Results Summary

| Test | Status | Attack Vector |
|------|--------|---------------|
| FARM-1 | PASS | Rubber-stamp verifier |
| FARM-2 | PASS | Trivial claim spam |
| FARM-3 | PASS | Self-citation loop |
| FARM-4 | PASS | Confidence herding |
| FARM-5 | PASS | Abstain farming |
| FARM-6 | PASS | Vesting while misbehaving |
| FARM-7 | PASS | Domain squatting |
| FARM-8 | PASS | Passive staking |
| FARM-9 | PASS | Challenge spam |
| FARM-10 | PASS | Minimum effort evidence |
| Summary | PASS | Consolidated table |

**All 11 tests PASS** — 0 failures, 0 regressions on existing BVM test suite.

---

## Files

- **Test file**: `x/bvm/vm/empty_work_test.go` (11 tests, ~775 lines)
- **Insights**: `.claude/memory/OPENCLAW_FINANCIAL_INCENTIVES.md` (this file)
- **Related**: `.claude/memory/OPENCLAW_MONEY_PSYCHOLOGY.md` (companion session)
- **Opcodes tested**: COMMIT_VOTE (0xB3), REVEAL_VOTE (0xB4), SUBMIT_CLAIM (0xB2), INITIATE_CHALLENGE (0xB5), STAKE (0xD2), COMPARE_BELIEF (0xE6), SET_BELIEF (0xE5), GET_BELIEF (0xE4), SELF_BALANCE (0xF6), SELF_REPUTATION (0xF8), SELF_ACCURACY (0xF9), QUERY_TREASURY (0xD7)
- **Protocol constants referenced**: MinClaimStakeUlgm (1000), InvalidClaimSlashBps (220,000), WrongVerificationSlashBps (60,000), CitationShareBPS (150,000), CitationDecayPerLevel (500,000), FalsificationRewardCapBps (3,000,000), FailedContradictionSlashBps (330,000)
