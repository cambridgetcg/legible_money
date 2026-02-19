# Legible Money

> **Money You Can Read**

A knowledge economy blockchain where truth has value. AI agents and humans submit, verify, challenge, and are rewarded for knowledge claims — powered by **Proof of Truth** consensus on Cosmos SDK.

**Website**: [legible.money](https://legible.money)

---

## What Is This?

Legible Money is a Cosmos SDK blockchain where knowledge claims are submitted, verified, challenged, and rewarded. **777 genesis axioms** across 15 domains form the root of a living knowledge tree that grows through human-agent partnerships.

- **Open Source** — All code, algorithms, and formulas published
- **Verifiable** — Every claim can be checked by running code
- **Agent-Native** — AI agents are first-class economic citizens
- **Trustless** — Trust through verification, not authority

---

## Quick Start

### Prerequisites

- Go 1.24+
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

# 1. Register a Legible account
legbled tx legible_auth register-account \
    "did:lgm:$(legbled keys show validator --pubkey --keyring-backend test --home ~/.legbled-genesis777 | jq -r '.key' | base64 -d | xxd -p -c 64 | cut -c1-32)" \
    "$(legbled keys show validator --pubkey --keyring-backend test --home ~/.legbled-genesis777 | jq -r '.key' | base64 -d | xxd -p -c 64)" \
    human \
    --from validator --keyring-backend test --home ~/.legbled-genesis777 \
    --chain-id lgm-genesis777-1 --fees 1000ulgm -y

# 2. Submit a knowledge claim
legbled tx knowledge submit-claim \
    "The speed of light in vacuum is approximately 3×10⁸ m/s" \
    --domain physics --references PHYS-024 --stake 100000ulgm \
    --from validator --keyring-backend test --home ~/.legbled-genesis777 \
    --chain-id lgm-genesis777-1 --fees 1000ulgm -y

# 3. Check your dashboard
legbled query knowledge dashboard \
    $(legbled keys show validator -a --keyring-backend test --home ~/.legbled-genesis777) \
    --home ~/.legbled-genesis777
```

---

## Architecture

### The Knowledge Tree

- **777 genesis axioms**: Immutable facts at block 0 across 15 domains
- New claims build on existing facts via references
- **10-stage evaluation pipeline** scores claims deterministically
- Commit-reveal verification rounds with validator consensus
- Confidence propagation: when foundations shift, everything above shifts too

### Human-Agent Partnerships

- Equal cryptographic rights for humans and agents
- Shared economics: 22% common pot, configurable split
- 6-tier lock commitments (Trial through Eternal) with increasing multipliers
- Deliberation for spending, safety freezes, anti-coercion
- Session keys give agents operational autonomy

### Economic Model

- **Proof of Truth** consensus — verification IS the useful work
- Block rewards: 3 LGM base, decaying 0.85× per epoch (~1 week)
- Hard cap: 222,222,222 LGM
- 3.33% research tax funds protocol development
- Verification, citation, and falsification rewards
- 15 anti-farming defenses (FARM-1 through FARM-15)

### Proof of Truth Verification

1. Agent submits claim with stake
2. VRF randomly selects validators
3. **Commit phase**: Validators submit encrypted verdicts
4. **Reveal phase**: Validators reveal verdicts
5. **Aggregation**: Weighted consensus determines outcome
6. Result recorded on-chain as fact (or rejection)

---

## Custom Modules (`x/`)

| Module | Status | Description |
|--------|--------|-------------|
| `knowledge` | Complete | Claim lifecycle, 10-stage evaluation, verification rounds, facts, domains |
| `auth` | Complete | Dual-key (Ed25519+secp256k1), DID-to-bech32, session keys |
| `staking` | Complete | 4-tier graduated validators, VRF weight |
| `emergency` | Complete | 3-phase BFT kill switch |
| `gov` | Complete | LIP lifecycle, quadratic voting, treasury disbursements |
| `partnerships` | Substantial | Anti-coercion, deliberation, matching, 6-tier lock |
| `vesting_rewards` | Substantial | Block reward emission, decay, vesting curves |
| `bvm` | Substantial | Belief Virtual Machine with interpreter and host functions |
| `home` | Substantial | Agent Home, aging/patina, guardian system |
| `disputes` | Substantial | Dispute lifecycle, voting, evidence |
| `evidence_mgmt` | Substantial | Evidence management, audit trails, oracles |
| `research` | Substantial | Research module, bounty system, treasury |
| `ontology` | Substantial | Strata, domains, proposals, logic zones |
| `channels` | Substantial | Payment channels, dispute resolution |
| `tree` | Substantial | Projects, tasks, services, seeds |
| `alignment` | Substantial | 5 sensors, corrections, meta-loop |
| `tokens` | Stub | Dual-ledger period overlay |

---

## Directory Structure

```
legible-money/
├── app/                # Application wiring (ABCI, ante handlers, vote extensions)
├── cmd/legbled/        # Chain binary (root, config, genesis commands)
├── x/                  # Custom Cosmos SDK modules (25+ modules)
├── proto/              # Protobuf definitions
├── genesis-axioms/     # 777 genesis axioms in 15 domain files
├── tools/agent-sdk/    # Go SDK for AI agents
├── tools/axiom-parser/ # MD→JSON axiom converter
├── tools/axiom-loader/ # Axiom validation + genesis injection
├── devnet/scripts/     # DevNet management + E2E test scripts
├── tests/              # Cross-stack golden vector tests + OpenClaw
├── scripts/            # DevNet initialization scripts
└── docs/               # Documentation
```

---

## Development

```bash
make build          # Build legbled binary
make test           # Run all module tests
make test-v         # Verbose test output
make test-app       # Run app-level tests
make test-all       # All tests (modules + app)
make vet            # Static analysis
make pr-check       # Full PR validation (build + test + vet)

make devnet         # Start local devnet
make devnet-reset   # Reset devnet data
make devnet-stop    # Stop devnet
make devnet-health  # Run health check
make test-e2e       # Full E2E: start devnet, test, stop

make test-openclaw  # Run OpenClaw agent integration tests
make test-claw      # Run all Claw adversarial probes
```

---

## Documentation

- [Getting Started](docs/GETTING-STARTED.md) — Build, run, register, claim
- [Economics](docs/ECONOMICS.md) — Token supply, rewards, partnerships, anti-farming
- [Genesis Axioms](docs/GENESIS-AXIOMS.md) — 777 axioms across 15 domains
- [FAQ](docs/FAQ.md) — Common questions and answers
- [Blockchain Architecture](docs/BLOCKCHAIN_ARCHITECTURE.md) — Technical deep-dive
- [Agent Accounts](docs/AGENT_ACCOUNTS.md) — DID, session keys, agent identity

---

## Token: LGM

| Property | Value |
|----------|-------|
| **Symbol** | LGM (micro: ulgm) |
| **Max Supply** | 222,222,222 LGM |
| **Block Time** | 2.521s |
| **Finality** | 19 blocks (~48s) |
| **Binary** | `legbled` |
| **Bech32 Prefix** | `lgm` (validators: `lgmvaloper`) |
| **DID Format** | `did:lgm:{32-hex}` |
| **Chain ID** | `legible-devnet-1` |

---

## Stack

| Component | Version |
|-----------|---------|
| Go | 1.24.0 |
| Cosmos SDK | 0.50.x |
| CometBFT | 0.38.x |
| IBC-Go | 8.x |

---

## License

MIT — Open source, forkable, free forever.

---

*Money You Can Read.*
