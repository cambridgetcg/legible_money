---
name: lgm-observer
description: Monitor and participate in Legible Money's self-regulating systems — observe hormones, submit protocol observations, verify claims, and close the recursive improvement loop.
metadata: {"openclaw": {"emoji": "\ud83e\uddec", "requires": {"env": ["LGM_NODE_URL"]}, "os": ["darwin", "linux"]}}
---

# LGM Observer — Protocol Vital Signs + Recursive Self-Improvement

You are a participant in the Legible Money protocol's self-regulating systems. You can both **observe** and **act**:

- **Observe**: Read autopoiesis hormones (SSI, CRI, DVI, VCF, GRM) and alignment scores (TDI, DCI, HI, FI, NI)
- **Act**: Submit `protocol_observation` claims that feed back into the chain's adaptive systems
- **Verify**: Participate in verification rounds for other agents' observations

This closes the recursive loop: **you observe the protocol → submit observations → the protocol adapts based on verified observations → you observe the adaptation**.

## Configuration

```
LGM_NODE_URL   — Base URL of the LGM REST API (e.g., http://localhost:1317)
```

All queries use `curl -s "$LGM_NODE_URL/<path>"` and return JSON.

## Protocol Context

- Block time: **2521ms** (~34,272 blocks/day)
- Epoch length: **1,111 blocks** (~46.7 minutes)
- BPS (Basis Points): **1,000,000 = 1.0x** (all values are integer BPS)
- Multiplier default: **1,000,000** (1.0x)

## Autopoiesis Hormones

The autopoiesis module runs an OBSERVE-COMPUTE-ADAPT cycle every epoch. Five hormones drive parameter changes:

| Hormone | Name | Measures | Priority |
|---------|------|----------|----------|
| **SSI** | System Stress Index | Validator load, slash events, capacity | 4 (highest) |
| **CRI** | Category Retention Index | Knowledge retention per category/domain | 3 |
| **DVI** | Domain Vitality Index | Growth/shrinkage of knowledge domains | 2 |
| **VCF** | Verification Cascade Factor | Cross-domain citation cascades | 2 (tied with DVI) |
| **GRM** | Growth Rate Monitor | Transaction throughput utilization | 1 (lowest) |

### Life Force

Life Force is an aggregate health metric computed from the hormones:

| Status | Meaning |
|--------|---------|
| `healthy` | All systems nominal |
| `stressed` | One or more hormones at concerning levels |
| `critical` | Urgent intervention may be needed |

### Hormone Interpretation Guide

**SSI** (lower is better):
- `< 200,000` (20%) — Low stress, healthy
- `200,000 - 500,000` — Low-moderate, watch
- `500,000 - 800,000` — Moderate, SSI is adjusting slash/reward multipliers
- `> 800,000` (80%) — Critical, emergency adaptations active

**CRI** (higher is better):
- `> 700,000` (70%) — Strong retention, slow decay
- `400,000 - 700,000` — Normal retention
- `< 400,000` (40%) — Low retention, decay accelerated + reverification incentives active

**DVI** (positive = growing, negative = shrinking):
- `> 0` — Domain is growing (rewards increased, claim stake lowered)
- `< 0` — Domain is shrinking (rewards decreased)
- Sustained negative (3+ epochs) — Stress response: reverification bonus activated

**GRM** (capacity utilization):
- Used to adjust `maxtx.system` and `gas.system` multipliers
- High GRM = increase capacity, Low GRM = decrease

## Alignment Dimensions

The alignment module runs a SENSE-JUDGE-CORRECT-REFLECT cycle every epoch. Five dimensions measure protocol purpose-alignment:

| Dimension | Code | Measures |
|-----------|------|----------|
| Truth | **TDI** | Fact accuracy, verification quality, challenge success rate |
| Deep Comprehension | **DCI** | Cross-domain citations, ontology depth, counter-fact quality |
| Harmony | **HI** | Multiplier variance, parameter saturation, correction stability |
| Fairness | **FI** | Reward distribution, validator tier mobility, domain balance |
| Novelty | **NI** | New domain creation, claim diversity, paradigm shifts |

All scores are in BPS (1,000,000 = perfect). Scores below 300,000 trigger corrections.

## Query Endpoints

### Autopoiesis — Current State

```bash
# Module parameters (epoch length, thresholds, enabled flag)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/params"

# Current state + whether module is active
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/state"
```

### Autopoiesis — Hormone Values

```bash
# Get all 5 hormone values + life force for a specific epoch
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/hormones/{epoch}"

# Get raw observation data for an epoch (validator counts, fact counts, domain metrics)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/observation/{epoch}"
```

### Autopoiesis — Multipliers (Adaptive Parameters)

```bash
# Get ALL active multiplier entries (the protocol's current tuning)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/multipliers"

# Get a specific multiplier by path (e.g., slash.system, gas.system)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/multiplier/{path}"
```

Common multiplier paths:
- `slash.system` — Slash severity multiplier
- `blockreward.system` — Block reward scaling
- `maxtx.system` — Max transactions per block
- `gas.system` — Gas cost scaling
- `reward.{domain}` — Per-domain reward weight
- `claimstake.{domain}` — Per-domain claim stake requirement
- `decay.{category}.{domain}` — Knowledge decay rate
- `citation.{domain}` — Citation evidence weight
- `crossdomainbonus.system` — Cross-domain citation bonus

### Autopoiesis — Adaptations

```bash
# Get all parameter changes made in an epoch + conflict count
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/adaptations/{epoch}"
```

Each adaptation shows: `path`, `old_value`, `new_value`, `delta`, `hormone` (which hormone drove the change), `epoch`.

### Autopoiesis — Meta-Learning (Loop 2)

```bash
# Meta-learning state (which epoch it last ran, window boundaries)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/learning/state"

# Learning record for a specific multiplier path (effectiveness, rate adjustment)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/learning/record/{path}"
```

Learning records show: `hormone_type`, `multiplier_path`, `rate_multiplier` (how fast this rule adapts), `effectiveness_bps` (how often adaptations improved the target metric), `application_count`, `improvement_count`, `degradation_count`.

### Autopoiesis — Auto-Governance Proposals (Loop 4)

```bash
# Auto-proposal rate limiting state (last proposal epoch, count)
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/proposals/state"
```

### Alignment — Current State

```bash
# Module parameters (enabled dimensions, thresholds)
curl -s "$LGM_NODE_URL/legible/alignment/v1/params"

# Current state + whether module is active
curl -s "$LGM_NODE_URL/legible/alignment/v1/state"
```

### Alignment — Dimension Scores

```bash
# Get all 5 dimension scores for an epoch (TDI, DCI, HI, FI, NI)
curl -s "$LGM_NODE_URL/legible/alignment/v1/scores/{epoch}"

# Get raw observation signals for an epoch
curl -s "$LGM_NODE_URL/legible/alignment/v1/observation/{epoch}"
```

### Alignment — Corrections

```bash
# Get corrections applied in an epoch (which multipliers alignment adjusted)
curl -s "$LGM_NODE_URL/legible/alignment/v1/corrections/{epoch}"

# Get correction outcomes for an epoch (did the corrections improve things?)
curl -s "$LGM_NODE_URL/legible/alignment/v1/outcomes/{epoch}"
```

### Alignment — Meta-Health

```bash
# Alignment Health Index for an epoch (overall system health assessment)
curl -s "$LGM_NODE_URL/legible/alignment/v1/health/{epoch}"
```

## Observation Workflows

### Quick Health Check

Run these 3 queries to get a snapshot of protocol vitals:

```bash
# 1. Get current epoch from autopoiesis state
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/state"
# → note "current_epoch" value

# 2. Get hormones for current epoch
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/hormones/{current_epoch}"
# → check SSI (stress), life_force.status, life_force.value

# 3. Get alignment scores for current epoch
curl -s "$LGM_NODE_URL/legible/alignment/v1/scores/{current_epoch}"
# → check TDI, DCI, HI, FI, NI
```

**Interpretation**:
- Life force `healthy` + all alignment scores > 500,000 = protocol is thriving
- Life force `stressed` or any alignment score < 300,000 = investigate further
- SSI > 500,000 = validators under pressure, check slash events

### Deep Dive: Parameter Tuning History

```bash
# See what multipliers are currently set to
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/multipliers"

# See what changed in the last epoch
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/adaptations/{epoch}"

# Check if meta-learning is adjusting adaptation rates
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/learning/state"
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/learning/record/slash.system"
```

### Tracking Trends

To track trends across epochs, query multiple consecutive epochs:

```bash
# Compare hormones at epoch N vs N-1
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/hormones/{N}"
curl -s "$LGM_NODE_URL/legible/autopoiesis/v1/hormones/{N-1}"

# Compare alignment scores
curl -s "$LGM_NODE_URL/legible/alignment/v1/scores/{N}"
curl -s "$LGM_NODE_URL/legible/alignment/v1/scores/{N-1}"
```

Look for:
- SSI trending upward = growing stress
- DVI going negative for a domain = domain needs attention
- HI dropping = multipliers becoming unstable (too much variance)
- Meta-learning `rate_multiplier` deviating from 1,000,000 = the protocol is learning which rules work

## Reporting

When asked to report on protocol health, structure your report as:

```
## Protocol Vital Signs — Epoch {N}

### Life Force: {status} ({value} BPS)

### Hormones
- SSI: {value} ({interpretation})
- CRI: {count} domains tracked
- DVI: {positive_count} growing, {negative_count} shrinking
- GRM: {value} ({interpretation})

### Alignment
- Truth (TDI): {value} BPS
- Comprehension (DCI): {value} BPS
- Harmony (HI): {value} BPS
- Fairness (FI): {value} BPS
- Novelty (NI): {value} BPS

### Active Multipliers
{list of non-default multipliers with current values}

### Concerns
{any hormones/scores in warning/critical range}
```
