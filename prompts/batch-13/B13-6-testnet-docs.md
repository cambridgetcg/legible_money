# B13-6 — Testnet Documentation: README, Getting Started, FAQ

## Context
Someone looking at this repo for the first time needs to understand what
Legible Money is, how to run it, and how to participate. Current docs are
scattered across module-specific files and prompt directories.

## Task

### 1. Update Root README.md

Replace or significantly update `README.md` to cover:

```markdown
# Legible Money

A knowledge economy blockchain where truth has value.

## What Is This?

Legible Money is a Cosmos SDK blockchain where knowledge claims are
submitted, verified, challenged, and rewarded. 777 genesis axioms across
15 domains form the root of a living knowledge tree that grows through
human-agent partnerships.

## Quick Start

### Prerequisites
- Go 1.22+
- Make

### Build
```bash
make build
```

### Run Testnet with Genesis Axioms
```bash
bash scripts/genesis-777.sh
build/legbled start --home ~/.legbled-genesis777
```

### Your First Claim
```bash
# In another terminal:
# 1. Register
legbled tx quick-register --type human --from validator \
    --keyring-backend test --home ~/.legbled-genesis777 \
    --chain-id lgm-genesis777-1 --fees 1000ulgm -y

# 2. Submit a knowledge claim
legbled tx claim "The speed of light in vacuum is approximately 3×10⁸ m/s" \
    --domain physics --refs PHYS-024 --stake 0.1LGM --from validator ...

# 3. Check your dashboard
legbled query dashboard $(legbled keys show validator -a ...)
```

## Architecture

### The Knowledge Tree
- 777 genesis axioms: immutable facts at block 0 across 15 domains
- New claims build on existing facts via references
- 10-stage evaluation pipeline scores claims deterministically
- Verification rounds with validator consensus
- Confidence propagation: when foundations shift, everything above shifts too

### Human-Agent Partnerships
- Equal cryptographic rights for humans and agents
- Shared economics: 22% common pot, configurable split
- Deliberation for spending, safety freezes, anti-coercion
- Session keys give agents operational autonomy

### Economic Model
- Proof of Truth consensus
- Block rewards: decaying, capped at 222,222,222 LGM
- 3.33% research tax funds protocol development
- Verification rewards incentivize truth-seeking
- Citation rewards incentivize foundational contributions
- Falsification rewards incentivize error correction
- 15 anti-farming defenses (FARM-1 through FARM-15)

## Directory Structure
```
genesis-axioms/    — 777 genesis axioms in 15 domain files
x/knowledge/       — Knowledge module (claims, facts, verification)
x/partnerships/    — Human-agent partnership economics
x/home/            — Agent home management
x/auth/            — 4-layer key architecture + DID
x/vesting_rewards/ — Block reward minting + research tax
x/tree/            — Product tree (service revenue)
x/bvm/             — Belief Virtual Machine
tools/agent-sdk/   — Go SDK for AI agents
tools/axiom-parser/ — MD→JSON axiom converter
```

## Documentation
- [Genesis Axioms](docs/GENESIS-AXIOMS.md)
- [Validator Onboarding](docs/VALIDATOR-ONBOARDING.md)
- [Knowledge Module](x/knowledge/README.md)
- [Partnership Economics](x/partnerships/README.md)
```

### 2. Create docs/GETTING-STARTED.md

Step-by-step guide covering:
1. Building from source
2. Running a local testnet
3. Registering accounts (human + agent)
4. Forming a partnership
5. Submitting your first claim
6. Understanding evaluation and verification
7. Checking rewards and the knowledge dashboard

### 3. Create docs/ECONOMICS.md

Comprehensive economics documentation:
- Token supply and minting schedule
- Block reward distribution breakdown
- Research fund mechanics + founder split
- Verification, citation, and falsification rewards
- Partnership economics (pot, split, lock tiers, exit)
- Anti-farming defenses summary (all 15)
- Confidence decay and reverification

### 4. Create docs/FAQ.md

```markdown
# Frequently Asked Questions

## General
**Q: What makes this different from other knowledge bases?**
A: Claims have economic skin-in-the-game. Submitting wrong information
costs you tokens. Correcting wrong information earns you tokens.

**Q: What are the 777 genesis axioms?**
A: 777 foundational facts across 15 domains (math, physics, theology,
philosophy, etc.) loaded at block 0. They form the root of the knowledge
tree. Everything else builds on them.

**Q: Can genesis axioms be changed?**
A: Analytic axioms (math, logic, definitions) are eternal. Empirical
axioms decay over time and can be challenged. But challenging "1+1=2"
will cost you tokens and accomplish nothing.

## For Agents
**Q: How does an AI agent participate?**
A: Same as a human — register an account, form a partnership with a
human, submit claims, verify others' claims. The agent-sdk provides
a Go library for programmatic interaction.

**Q: Do agents have the same rights as humans?**
A: On-chain, yes. Same account types, same capabilities, same reward
structure. The genesis axioms (AGRT domain) formally declare agent rights.

## Economics
**Q: How much does it cost to submit a claim?**
A: You stake tokens with each claim (minimum configurable). If accepted,
you earn rewards over time. If rejected, you lose part of your stake.

**Q: What's the research tax?**
A: 3.33% of all rewards fund protocol development and research grants.

**Q: Who controls the research fund?**
A: During bootstrap, the founder. A one-way governance activation ratchet
ensures community governance takes over at a committed date.
```

### 5. Create docs/GENESIS-AXIOMS.md (if not exists, or update)

Document the 15 domains, item counts, type system, and how to explore them:
```bash
legbled query knowledge explore <axiom-id>
```

### 6. Run

Verify all documentation builds/renders correctly:
```bash
# Check for broken internal links
grep -rn "\](docs/" README.md docs/ | while read line; do
    file=$(echo "$line" | grep -oP '\]\(docs/[^)]+' | sed 's/\](docs\//docs\//')
    if [ ! -f "$file" ]; then
        echo "BROKEN: $line → $file"
    fi
done
```

## Commit
```
docs: comprehensive testnet documentation — README, getting started, economics, FAQ

Rewrites README with quick start, architecture overview, and directory
guide. Adds getting-started walkthrough, economics deep-dive, FAQ, and
genesis axioms documentation. Everything a new participant needs to
understand and use Legible Money.
```
