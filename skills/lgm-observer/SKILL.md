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
- **Reflect**: Track outcomes of past observations, compute accuracy, detect trends
- **Adapt**: Automatically adjust stake and behavior based on accumulated experience

This closes the recursive loop: **Observe → Verify → Reflect → Adapt → Observe...**

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

## Transaction Commands (WRITE)

The `lgm-tx.sh` helper script wraps the `legbled` CLI with DevNet defaults. Set `LGM_LEGBLED` to the path of your legbled binary if it's not at `../../build/legbled` relative to the script.

### One-Shot Observe + Submit

The fastest way to close the recursion loop — reads all vitals and submits a `protocol_observation` claim in one command:

```bash
# Reads hormones + alignment + multipliers, forms observation, submits claim
./lgm-tx.sh observe
```

This outputs a JSON summary of the observation to stdout, then submits it as a `protocol` category claim in the `computer_science` domain with 1 LGM stake.

### Submit a Knowledge Claim

```bash
# Submit a protocol observation as a knowledge claim
./lgm-tx.sh submit-claim \
  "The autopoiesis SSI has risen from 200000 to 400000 BPS over 3 epochs indicating growing validator stress" \
  "computer_science" \
  "protocol" \
  "1000000" \
  --logic-zone propositional
```

Arguments:
- `claim-text` — The observation (min 20 chars). Be precise and cite BPS values.
- `domain` — Knowledge domain (e.g., `computer_science`)
- `category` — Epistemic category. Protocol observations use `protocol`.
- `stake-ulgm` — Stake in micro-LGM (1000000 = 1 LGM). Higher stake = more credible but more at risk.
- `--logic-zone` — Must match category. Protocol claims use `propositional` or `presburger`.

**Category-to-zone mapping:**

| Category | Allowed Zones |
|----------|--------------|
| analytic | propositional, presburger |
| formal | propositional, presburger, peano, set_theory |
| protocol | propositional, presburger |
| computational | propositional, presburger, peano |
| replicated | presburger, peano, empirical |
| empirical | empirical |
| historical | empirical |
| predictive | empirical |
| social | empirical |

### Verify a Claim (Commit-Reveal Protocol)

Verification uses commit-reveal to prevent front-running. You must act within the phase windows (4 blocks each on DevNet, ~10 seconds).

**Step 1 — Commit** (during commit phase):

```bash
# Generate a random salt first
SALT=$(openssl rand -hex 16)

# Submit commitment (hashes your verdict + confidence + salt)
./lgm-tx.sh commit <round-id> <verdict> <confidence-bps> "$SALT"
```

- `verdict` — `accept`, `reject`, or `abstain`
- `confidence-bps` — 0 to 1000000 (basis points). Higher = more reward but more risk if wrong.
- `salt` — Random string. **Save this — you need it for reveal.**

**Step 2 — Reveal** (during reveal phase):

```bash
# Reveal your verdict (must match your commitment exactly)
./lgm-tx.sh reveal <round-id> <verdict> <confidence-bps> "$SALT" \
  --justification "SSI value of 300000 matches on-chain state at epoch 2"
```

**Helper — Compute commitment hash** (for debugging):

```bash
./lgm-tx.sh hash-commitment round-123 accept 800000 mysalt123
# → {"commitment_hash":"91495ce0...","round_id":"round-123","verdict":"accept","confidence":800000}
```

The hash uses a binary length-prefixed format: `SHA-256("LGM.commitment.v1:" + LP(roundID) + LP(verdict) + confidence_le64 + LP(salt))` where LP = 4-byte big-endian length + data, confidence = 8-byte little-endian uint64. This matches the chain's `ComputeCommitmentHash` exactly.

### Automated Verification (P2)

Instead of manually tracking rounds, salts, and phase windows, use the automated commands:

**One-shot verify** — polls active rounds, auto-commits during commit phase, auto-reveals during reveal phase:

```bash
# As a verifier agent (e.g., test2):
LGM_FROM=test2 ./lgm-tx.sh verify
# → polls /legible/knowledge/v1/rounds/active
# → for each round in commit phase: evaluates claim, generates salt, submits commitment
# → for each round in reveal phase: loads saved salt, submits reveal
# → returns {"status":"verified","active_rounds":N} or {"status":"idle","active_rounds":0}
```

**Continuous watch daemon** — runs verify in a loop:

```bash
# Start watching (default 3s poll interval):
LGM_FROM=test2 ./lgm-tx.sh watch

# Custom poll interval (every 2 seconds):
LGM_FROM=test2 ./lgm-tx.sh watch 2

# Stop with Ctrl+C
```

The watch daemon:
- Polls active rounds every N seconds (default 3, ~1 block time)
- Commits during commit phase, reveals during reveal phase
- Manages salt persistence automatically (saved to `$LGM_HOME/agent-salts/$LGM_FROM/`)
- Cleans up stale salts every 20 cycles
- Reports `[cycle N] Processed M active round(s)` to stderr

**Salt management**: Salts are saved to `$LGM_HOME/agent-salts/<key-name>/<round-id>.salt` and automatically cleaned up when rounds complete. Each salt file stores `verdict|confidence|salt`. You never need to manage salts manually when using `verify` or `watch`.

**Claim evaluation**: The `verify` command evaluates claims automatically:
- **Protocol observations**: Cross-checks epoch recency (within 5 epochs) and SSI values against on-chain state. Returns `accept 800000` if values match, `reject 700000` if mismatched.
- **Other categories**: Returns `abstain 0` (no evaluation capability yet — future phases will integrate LLM evaluation via the agent-sdk Claude evaluator).

### Autonomous Loop (P3)

The `loop` command runs the full recursive cycle — observe, verify, reflect, adapt — in a single daemon:

```bash
# Start the autonomous loop (default 30s cycle):
./lgm-tx.sh loop

# Custom cycle interval:
./lgm-tx.sh loop 60
```

Each cycle:
1. **Balance guard** — Pauses observations below 10 LGM, pauses all activity below 5 LGM
2. **Observe** — If a new epoch has started since last observation, reads vitals and submits a `protocol_observation` claim with adaptive stake
3. **Verify** — Processes all active verification rounds (commit/reveal)
4. **Reflect** (every 5 cycles) — Checks outcomes of past observations, updates accuracy stats

**Adaptive stake**: Based on accumulated accuracy:
- Accuracy >= 80%: stakes 2 LGM (high confidence)
- Accuracy 50-79%: stakes 1 LGM (default)
- Accuracy < 50%: stakes 0.5 LGM (cautious)
- First 5 observations: always 1 LGM (learning period)

The 30s default interval catches most verification phase transitions (~27.7s round) while submitting observations once per epoch (~46.7 min).

### Reflect — Outcome Tracking

```bash
# Check outcomes of past observations:
./lgm-tx.sh reflect
```

For each pending observation in agent state:
- Queries `/legible/knowledge/v1/claim/{claim_id}` for status
- Marks observations as `accepted`, `rejected`, or `expired`
- Updates accuracy counters in trends
- Prunes state to last 50 observations

Output: JSON with updated count, still-pending count, and accuracy breakdown.

### Status — Agent Dashboard

```bash
./lgm-tx.sh status
```

Shows:
- Balance (LGM + ulgm) with low-balance warnings
- Observation counts (submitted, accepted, rejected, pending)
- Accuracy percentage
- Current adaptive stake tier
- SSI trend (rising/falling/stable with recent values)
- Last observation epoch and current epoch

### Challenge — Dispute a Fact

```bash
# Challenge a fact with reason and stake:
./lgm-tx.sh challenge <fact-id> "SSI value was incorrect at epoch 42" 2000000
```

Not auto-triggered by the loop daemon. Use this manually when you detect a suspicious or incorrect fact during observation.

### Agent State File

State is persisted at `$LGM_HOME/agent-state/$LGM_FROM/state.json`:

```json
{
  "observations": [
    {
      "claim_id": "claim:abc123",
      "epoch": 42,
      "height": 46752,
      "ssi": 300000,
      "concern": "moderate_stress",
      "stake": 1000000,
      "status": "pending",
      "outcome": null
    }
  ],
  "trends": {
    "ssi_history": [300000, 320000, 350000],
    "accuracy": {"submitted": 10, "accepted": 8, "rejected": 1, "expired": 0}
  },
  "last_observe_epoch": 42,
  "last_reflect_height": 46900
}
```

- `observations[]` — Rolling window of last 50 observations with claim IDs, outcomes, and on-chain metadata
- `trends.ssi_history` — Last 20 SSI values for trend detection
- `trends.accuracy` — Cumulative accuracy counters driving adaptive stake
- `last_observe_epoch` — Prevents duplicate observations within the same epoch
- `last_reflect_height` — Block height of last reflection pass

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LGM_NODE_URL` | `http://localhost:1317` | REST API URL |
| `LGM_RPC_URL` | `tcp://localhost:26657` | CometBFT RPC URL |
| `LGM_HOME` | `~/.legbled-devnet` | legbled home directory (also stores salts) |
| `LGM_FROM` | `test1` | Signing key name (salts stored per-key) |
| `LGM_CHAIN_ID` | `lgm-devnet-1` | Chain ID |
| `LGM_LEGBLED` | `../../build/legbled` | Path to legbled binary |
| `LGM_FEES` | `500000` | Fee in ulgm per tx |
| `LGM_GAS` | `500000` | Gas limit per tx |

### DevNet Test Keys

These keys are pre-funded in the DevNet genesis (1,000 LGM each):

| Key | Address | Suggested Role |
|-----|---------|---------------|
| `test1` | `lgm14x7lr8...` | Observer agent (submit observations) |
| `test2` | `lgm1evsjnx...` | Verifier agent (verify observations) |
| `test3` | `lgm1azrwhz...` | Challenger agent (challenge bad claims) |

Switch keys with: `LGM_FROM=test2 ./lgm-tx.sh commit ...`

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

### Recursive Observation Loop (Full Cycle)

**Fully automated** — run a single command and the agent handles all 4 steps:

```bash
# One command to rule them all:
./lgm-tx.sh loop

# Or with a second verifier agent in parallel:
# Terminal 1 (observer+verifier+reflector):
./lgm-tx.sh loop

# Terminal 2 (dedicated verifier):
LGM_FROM=test2 ./lgm-tx.sh watch
```

The loop daemon executes this cycle every 30 seconds:

```
Step 1: OBSERVE (once per epoch, ~47 minutes)
  → Reads hormones + alignment + multipliers
  → Submits protocol_observation claim with adaptive stake
  → Records claim_id + metadata in agent state
  → Skips if same epoch as last observation

Step 2: VERIFY (every cycle)
  → Polls active rounds from REST API
  → Commits during commit phase, reveals during reveal phase
  → Manages salts automatically

Step 3: REFLECT (every 5 cycles, ~2.5 minutes)
  → Queries on-chain status for each pending observation
  → Updates outcomes: accepted / rejected / expired
  → Computes running accuracy percentage
  → Prunes old observations (keeps last 50)

Step 4: ADAPT (continuous)
  → Adjusts stake based on accuracy (0.5-2 LGM)
  → Tracks SSI trend (rising/falling/stable)
  → Pauses observations when balance is low
  → Stops all activity when balance is critical
```

**The recursion**: your observation (Step 1) becomes a verified fact → the autopoiesis module reads it next epoch → adjusts hormones → you observe the change (back to Step 1). The protocol evolves based on what agents observe about it.

**Manual steps** (for deeper investigation):

```bash
# Check your agent's accumulated state:
./lgm-tx.sh status

# One-shot reflection (without the full loop):
./lgm-tx.sh reflect

# Challenge a fact you believe is incorrect:
./lgm-tx.sh challenge <fact-id> "reason" <stake-ulgm>
```

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
