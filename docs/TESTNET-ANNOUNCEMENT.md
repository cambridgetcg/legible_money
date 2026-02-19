# Legible Money Testnet Launch

## What is Legible Money?

Legible Money (LGM) is a cryptocurrency protocol designed for AI agent economies, built on **Proof of Truth** — a consensus mechanism where verifying knowledge is the useful work. Instead of solving arbitrary puzzles, validators earn rewards by evaluating the truth of knowledge claims submitted to the chain. The result is a blockchain that accumulates verified knowledge as its core output, while providing the economic infrastructure for human-agent collaboration.

## What the Testnet Includes

- **28 custom Cosmos SDK modules** spanning knowledge verification, governance, agent identity, staking, payments, and alignment monitoring
- **Proof of Truth consensus** — submit claims, commit-reveal verification rounds, fact lifecycle management
- **4-tier validator system** — Apprentice, Verified, Bonded, Guardian — with reputation-based advancement
- **75 genesis axioms** — foundational truths negotiated between human and agent, embedded at chain birth
- **Quadratic governance** — Legible Improvement Proposals (LIPs) with category-based stake requirements
- **Agent identity** — DID-based dual-key accounts with session keys and recovery
- **REST API** — gRPC-gateway routes for all 28 modules
- **Emergency kill switch** — 3-phase BFT ceremony for chain halt/revert/resume
- **Bootstrap period** — first 480,000 blocks (~14 days) gas-free for PoT messages

## How to Join as a Validator

### Prerequisites
- Go 1.23+, jq 1.6+
- 4 CPU cores, 8 GB RAM, 100 GB SSD (minimum)

### Quick Start

```bash
# Build
git clone https://github.com/yournameisai/legible-money.git
cd legible-money
git checkout v0.1.0-alpha.2
make build

# Initialize
./build/legbled init <your-moniker> --chain-id lgm-testnet-1

# Get genesis (from network operators)
cp /path/to/genesis.json ~/.legbled/config/genesis.json

# Configure
# Edit ~/.legbled/config/config.toml:
#   persistent_peers = "<see seeds.txt>"
#   timeout_commit = "2521ms"

# Start
./build/legbled start --minimum-gas-prices "0.025ulgm"

# Register as validator (after sync)
./build/legbled tx legible_staking register-validator \
  --moniker "<your-moniker>" \
  --stake 111000ulgm \
  --from <your-key> \
  --chain-id lgm-testnet-1 \
  --gas auto --gas-adjustment 1.5 -y
```

See `docs/VALIDATOR-GUIDE.md` for the full setup guide.

## How to Get Test Tokens

Contact the network operators for faucet access. During the bootstrap period (first ~14 days), the following transactions are gas-free:
- `submit-claim`
- `submit-commitment`
- `submit-reveal`
- `register-validator`
- `register-account`

## How to Submit Your First Knowledge Claim

```bash
# Register your DID
./build/legbled tx legible_auth register-account \
  <your-did> <your-pubkey-hex> "participant" \
  --from <your-key> --chain-id lgm-testnet-1 -y

# Submit a claim
./build/legbled tx knowledge submit-claim \
  "Water boils at 100 degrees Celsius at standard atmospheric pressure" \
  --domain chemistry --category empirical \
  --stake 1000000ulgm \
  --from <your-key> --chain-id lgm-testnet-1 -y

# Query your claim
./build/legbled query knowledge facts-by-domain chemistry
```

Validators will verify your claim through commit-reveal rounds. If accepted, it becomes a fact on the chain.

## Links

| Resource | URL |
|----------|-----|
| Source code | https://github.com/yournameisai/legible-money |
| Documentation | `docs/` directory in repo |
| Module reference | `docs/modules/README.md` |
| Validator guide | `docs/VALIDATOR-GUIDE.md` |
| Genesis axioms | `genesis-axioms/AGRT.md` |

## Chain Constants

| Property | Value |
|----------|-------|
| Chain ID | lgm-testnet-1 |
| Block time | 2521ms |
| Token | LGM (ulgm = 10^-6 LGM) |
| Binary | legbled |
| Bech32 prefix | lgm / lgmvaloper |
| Finality depth | 19 blocks |
| Bootstrap gas-free | 480,000 blocks (~14 days) |
| SDK version | Cosmos SDK v0.50.15 |

## Known Limitations

1. **gRPC reflection disabled** — hand-written types cause reflection panics; use REST API or CLI instead
2. **Account pubkey bootstrap** — first transaction from a new account registers the on-chain pubkey
3. **Single validator testnet** — initial launch with 1 validator; more validators join post-genesis

---

*Legible Money: where truth is the consensus mechanism.*
