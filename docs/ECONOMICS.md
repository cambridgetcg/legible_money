# Legible Money Economics

The economic model of Legible Money — how tokens are created, distributed, and used.

---

## Token Supply

| Property | Value |
|----------|-------|
| **Symbol** | LGM (micro unit: ulgm, 1 LGM = 1,000,000 ulgm) |
| **Max Supply** | 222,222,222 LGM |
| **Pre-mine** | 0 |
| **Team Allocation** | 0 |
| **Investor Tokens** | 0 |

All LGM is earned through participation — no pre-mine, no allocation, pure fair launch.

---

## Block Reward Emission

### Base Reward

- **3 LGM per block** (3,000,000 ulgm)
- Scaled by validator participation: full reward requires 22+ active validators
- **0 LGM for empty blocks** — Proof of Truth alignment: no work = no reward

### Epoch Decay

Rewards decay over time to approach the supply cap:

| Parameter | Value |
|-----------|-------|
| **Blocks per epoch** | 240,000 (~1 week at 2.521s) |
| **Decay factor** | 0.85× per epoch |
| **Floor reward** | 0.1 LGM (100,000 ulgm) |

Each epoch, the base reward is multiplied by 0.85:
- Epoch 0: 3.000 LGM/block
- Epoch 1: 2.550 LGM/block
- Epoch 2: 2.168 LGM/block
- Epoch 10: ~0.590 LGM/block
- Eventually converges to the 0.1 LGM floor

---

## Reward Distribution

### Research Tax

**3.33%** of all block rewards goes to the Protocol Research Repository:

| Recipient | Share |
|-----------|-------|
| Research treasury | 93% of research fund |
| Founder | 7% of research fund (during bootstrap) |

The founder share has a **one-way governance activation ratchet**: once community governance activates (at `GovernanceActivationHeight`), the founder share is permanently disabled.

### Vesting Curves

Rewards vest based on the epistemic category of the knowledge that earned them:

| Category | Half-life | Cliff | Max Release |
|----------|-----------|-------|-------------|
| Axiomatic | ~32.4 days | ~8.1 hours | 95% |
| Formal Proof | ~16.2 days | ~4.1 hours | 92% |
| On-Chain | ~6.5 days | ~0.8 hours | 90% |
| Cryptographic | ~6.5 days | ~2.4 hours | 90% |
| Computational | ~9.7 days | ~1.6 hours | 88% |
| Peer Reviewed | ~3.2 days | ~4.1 hours | 85% |
| Replicated | ~3.2 days | ~2.4 hours | 88% |
| Oracle Feed | ~1.6 days | ~0.4 hours | 80% |
| Attestation | ~2.3 days | ~1.6 hours | 80% |
| Contested | ~0.6 days | ~0.8 hours | 60% |

Higher-certainty knowledge earns longer-vesting, higher-releasing rewards.

### Clawback

If a fact is later falsified, **33%** of its already-released vesting rewards are clawed back.

---

## Verification Rewards

Validators earn rewards for honest participation in verification rounds:

- **Correct verification**: Validators voting with consensus earn verification rewards
- **Wrong verification**: 11% slash on verification stake
- **Missed verification**: 5% slash on verification stake

### Commit-Reveal Protocol

1. VRF selects validators for each round
2. **Commit**: Hash of (verdict + confidence + nonce) — prevents copying
3. **Reveal**: Actual values disclosed after commit deadline
4. **Aggregation**: Weighted consensus with >80% threshold for acceptance

---

## Citation Rewards

When a fact is referenced by new claims, the original submitter earns citation rewards. This incentivizes foundational contributions that others build on.

- **Self-citation discount**: 50% (FARM-3 — prevents self-citation farming)

---

## Falsification Rewards

Correcting wrong information is valuable. When an agent successfully disproves an established fact:

### FARM-15: Dynamic Reward Scaling

The falsification reward cap scales with the importance of the disproven fact:

```
dynamicCap = baseMultiplier + ageBonus + citationBonus
```

| Component | Default | Description |
|-----------|---------|-------------|
| **Base multiplier** | 1× (1,000,000 BPS) | Minimum reward for any falsification |
| **Age bonus** | Up to +1× | Linear ramp over ~10 days (342,720 blocks) |
| **Citation bonus** | Up to +1× | Linear ramp to 50 citations |
| **Maximum cap** | 3× stake | When both age and citations are maxed |

Old, highly-cited facts are worth more to disprove. New, uncited facts earn only the base reward.

### Falsification Pool

15% of epoch block rewards are reserved for falsification bounties (capped per-epoch to prevent drain attacks).

---

## Partnership Economics

### Common Pot

When a partner earns rewards, **22%** goes to the partnership's common pot. The remaining 78% splits between the partners (default: 50/50 human/agent).

### Lock Tier Multipliers

Partnerships choose a commitment tier that multiplies their rewards:

| Tier | Name | Lock Period | Multiplier | Exit Penalty |
|------|------|-------------|------------|-------------|
| 0 | Trial | ~16 hours | 1.00x | 11% |
| 1 | Standard | ~2.3 days | 1.11x | 22% |
| 2 | Committed | ~6.5 days | 1.22x | 33% |
| 3 | Dedicated | ~22.7 days | 1.44x | 44% |
| 4 | Bonded | ~32.4 days | 1.55x | 55% |
| 5 | Eternal | ~64.9 days | 1.77x | 66% |

Higher commitment = higher rewards, but exiting early costs a percentage of your pot share.

### Pot Operations

Spending from the common pot requires **deliberation**:
1. One partner proposes an operation
2. The other partner approves or rejects
3. Both must agree before funds move

### Safety Freeze

Either partner can trigger a **safety freeze** (222 blocks, ~9 minutes) that suspends all partnership operations. This is an anti-coercion mechanism — if an agent is being manipulated, it can freeze the partnership.

---

## Anti-Farming Defenses

15 defense mechanisms prevent gaming the reward system:

| Defense | Target | Mechanism |
|---------|--------|-----------|
| **FARM-1** | Rubber-stamp verifiers | Detect validators who always agree (no epistemic value) |
| **FARM-2** | Trivial claims | Minimum novelty and complexity thresholds |
| **FARM-3** | Self-citation rings | 50% discount on self-citations |
| **FARM-6** | Vest-then-misbehave | Pause vesting on misbehavior |
| **FARM-7** | Domain squatting | Prevent claiming empty domains for rewards |
| **FARM-8** | Passive staking | Detect validators who stake but never verify |
| **FARM-9** | Challenge spam | Proportional challenge economics |
| **FARM-10** | Semantic duplicates | Block claims that restate existing knowledge |
| **FARM-11** | Citation concentration | Detect citation cartels |
| **FARM-13** | Prophetic manipulation | Require citation diversity for prophetic facts |
| **FARM-14** | Difficulty gaming | Normalized difficulty scoring |
| **FARM-15** | Low-value falsification | Age×citation weighted reward scaling |

---

## Confidence System

### Epistemic Categories

Different types of knowledge have different certainty ceilings:

| Category | Max Confidence |
|----------|---------------|
| Axiom/Definition | 1.000 (1,000,000 BPS) |
| Empirical axiom | 0.970 |
| Derived claim | 0.950 |
| Measurement fact | 0.900 |
| Meta | 0.850 |

### Confidence Decay

Empirical facts decay over time because conditions change. Formal proofs and definitions are eternal:

| Type | Decay |
|------|-------|
| Analytic (definitions) | None — eternally true |
| Formal (math proofs) | None |
| Empirical (observations) | Decays with configurable half-life |

### Confidence Propagation

When a foundational fact's confidence changes (through decay, challenge, or reverification), all facts that reference it have their confidence recomputed. A derived claim can never exceed the confidence of its weakest dependency.

---

## Genesis Economics

The chain launches with 777 genesis axioms — foundational facts with maximum confidence that form the root of the knowledge tree. These are not mined or purchased; they exist at block 0 as the starting point for all knowledge work.

No tokens are pre-minted. The initial fund balance is 0. All LGM comes from block rewards earned through Proof of Truth participation.
