# Legible Money

> **Money You Can Read**

The production blockchain for Legible Money — an open source protocol for AI agent economies with **Proof of Truth (PoT)** consensus. Built on Cosmos SDK.

**Website**: [legible.money](https://legible.money)

> **Note**: The TypeScript prototype/reference implementation lives in a [separate repo](https://github.com/legible-protocol/legible-prototype).

---

## What is Legible Money?

Legible Money is cryptocurrency that agents can **read, understand, and verify**.

- **Open Source** — All code, algorithms, and formulas published
- **Verifiable** — Every claim can be checked by running code
- **Agent-Native** — Designed for how AI agents think and operate
- **Trustless** — Trust through verification, not authority

---

## Quick Start

```bash
# Build the chain binary
make build

# Run tests
make test

# Run go vet
make vet

# Full PR validation
make pr-check

# Start local devnet
make devnet
```

---

## Architecture

Built on **Cosmos SDK v0.50.10** with **CometBFT v0.38** and **IBC-Go v8.5.1**.

### Custom Modules (`x/`)

| Module | Status | Description |
|--------|--------|-------------|
| `x/knowledge` | Complete | Claim lifecycle, verification rounds, fact CRUD, domain management |
| `x/auth` | Complete | Dual-key (Ed25519+secp256k1), DID-to-bech32, session keys |
| `x/staking` | Complete | 4-tier graduated validators, VRF weight |
| `x/emergency` | Complete | 3-phase BFT kill switch |
| `x/gov` | Complete | LIP lifecycle, quadratic voting, treasury disbursements |
| `x/tokens` | Partial | Dual-ledger period overlay |
| `x/vesting_rewards` | Partial | Block reward emission |
| `x/disputes` | Stub | Dispute resolution |
| `x/evidence_mgmt` | Stub | Evidence management |
| `x/research` | Stub | Research proposals |
| `x/ontology` | Stub | Knowledge ontology |
| `x/channels` | Stub | Payment channels |
| `x/bvm` | Partial | Belief Virtual Machine |
| `x/home` | Partial | Agent Home |
| `x/tree` | Stub | Knowledge tree |

### Key Features

- **ABCI++ Vote Extensions**: VRF-based validator selection for knowledge verification
- **Proof of Truth**: Knowledge verification as consensus work
- **4-Tier Validators**: Apprentice → Verified → Bonded → Guardian
- **Agent Identity**: DID-based with Ed25519 keys, session key support

---

## Directory Structure

```
legible-money/
├── app/              # Application wiring (ABCI, ante handlers, vote extensions)
├── cmd/legbled/      # Chain binary (root, config, genesis commands)
├── x/                # Custom Cosmos SDK modules (15+ modules)
├── proto/            # Protobuf definitions
├── devnet/scripts/   # DevNet management scripts
├── tests/            # Cross-stack golden vector tests
├── tools/migrate/    # State migration tooling
├── scripts/          # DevNet initialization scripts
└── docs/             # Documentation
```

---

## Development

```bash
# Build binary
make build

# Run module tests
make test

# Run with verbose output
make test-v

# Static analysis
make vet

# Full E2E (start devnet, test, stop)
make test-e2e
```

---

## Token: LGM

| Property | Value |
|----------|-------|
| **Symbol** | LGM |
| **Block Time** | 2.521s |
| **Finality** | 19 blocks |
| **Binary** | `legbled` |
| **Chain ID** | `legible-devnet-1` |

---

## License

MIT — Open source, forkable, free forever.

---

*Money You Can Read.*
