# Legible Money Testnet Launch Guide

## Chain Identity

| Property | Value |
|----------|-------|
| Chain ID | `lgm-testnet-1` |
| Binary | `legbled` |
| Denom | `ulgm` (micro-LGM, 6 decimals) |
| Block time | 2521ms |
| Finality depth | 19 blocks |
| Bech32 prefix | `lgm` / `lgmvaloper` |
| Consensus | Proof of Truth (PoT) + ECVRF |

## Hardware Requirements

### Minimum (testnet validator)
- **CPU**: 4 cores
- **RAM**: 8 GB
- **Disk**: 100 GB SSD
- **Network**: 100 Mbps

### Recommended
- **CPU**: 8 cores
- **RAM**: 16 GB
- **Disk**: 500 GB NVMe SSD
- **Network**: 1 Gbps

### Software
- Go 1.23+
- jq 1.6+
- CometBFT (bundled)

## Genesis Ceremony

### Overview

The genesis ceremony creates the initial `genesis.json` with:
- ALL 28 module parameters explicitly configured
- Bootstrap accounts (foundation, research, faucet, test)
- Validator gentxs with self-delegation
- Denom metadata for LGM token

### Step 1: Build the binary

```bash
make build
```

This injects version metadata (name, version, commit) via ldflags. Do not use bare `go build` — the binary will report empty version fields.

### Step 2: Initialize genesis

```bash
./scripts/testnet-genesis.sh init
```

This creates:
- Chain config at `~/.lgm/testnet/`
- Genesis with all params patched (26 modules, 300+ parameters)
- Bootstrap accounts (foundation, research, faucet, test1-3)
- Validated genesis

### Step 3: Add validators

Each validator operator runs:

```bash
./scripts/testnet-genesis.sh add-validator <validator-name>
```

Repeat for each validator. Each gets:
- 100,000 LGM initial balance
- 10,000 LGM self-delegation
- Unique consensus key (Ed25519)
- Unique node key (P2P identity)

### Step 4: Finalize genesis

```bash
./scripts/testnet-genesis.sh finalize
```

Collects all gentxs and re-validates genesis.

### Step 5: Export and distribute

```bash
./scripts/testnet-genesis.sh export
```

Copies `genesis.json` to the project root. Distribute this file to all validators.

## Joining as a Validator

### If you participated in the genesis ceremony

1. Copy your validator keys from the ceremony:
   ```bash
   cp ~/.lgm/testnet/validators/<name>/config/priv_validator_key.json ~/.legbled/config/
   cp ~/.lgm/testnet/validators/<name>/config/node_key.json ~/.legbled/config/
   ```

2. Copy the finalized genesis:
   ```bash
   cp genesis.json ~/.legbled/config/genesis.json
   ```

3. Configure CometBFT (in `~/.legbled/config/config.toml`):
   ```toml
   # Set persistent peers (comma-separated list of node IDs + addresses)
   # TODO: populate after genesis ceremony — obtain node IDs with:
   #   legbled comet show-node-id --home ~/.legbled/config
   # Planned seed nodes: TBD (the previously planned Hetzner hosts, Sentry and Patch, were retired 2026-10-13)
   # Node IDs blocked on chain restart. See Known Limitations #10.
   persistent_peers = "node_id1@ip1:26656,node_id2@ip2:26656"

   # Block time
   timeout_commit = "2521ms"
   timeout_propose = "2000ms"
   ```

4. Configure app (in `~/.legbled/config/app.toml`):
   ```toml
   minimum-gas-prices = "0.025ulgm"
   ```

5. Start the node:
   ```bash
   legbled start --minimum-gas-prices "0.025ulgm"
   ```

### Joining after genesis (post-launch)

1. Initialize your node:
   ```bash
   legbled init <moniker> --chain-id lgm-testnet-1
   ```

2. Replace genesis.json with the canonical one.

3. Sync from peers (state sync or full sync).

4. Create validator transaction:
   ```bash
   legbled tx staking create-validator \
     --amount=10000000000ulgm \
     --pubkey=$(legbled comet show-validator) \
     --moniker=<your-moniker> \
     --chain-id=lgm-testnet-1 \
     --commission-rate="0.10" \
     --commission-max-rate="0.20" \
     --commission-max-change-rate="0.01" \
     --from=<your-key> \
     --gas auto \
     --gas-adjustment 1.5 \
     -y
   ```

## Verifying Params

After the chain starts, verify params are correct:

```bash
./scripts/testnet-genesis.sh verify
```

Or query individual modules:

```bash
legbled query knowledge params --output json | jq '.params'
legbled query legible_staking params --output json | jq '.params'
legbled query lgm_gov params --output json | jq '.params'
legbled query emergency params --output json | jq '.params'
legbled query vesting_rewards params --output json | jq '.params'
```

## Testnet-Specific Configuration

These values differ from production defaults for faster iteration:

| Parameter | Testnet | Production |
|-----------|---------|------------|
| Knowledge commit phase | 50 blocks (~2 min) | 100 blocks (~4 min) |
| Knowledge reveal phase | 50 blocks (~2 min) | 100 blocks (~4 min) |
| Staking unbonding | 102,672 blocks (~3 days) | 268,560 blocks (~7 days) |
| Gov voting period | 17,136 blocks (~12 hours) | 100,000 blocks (~2.9 days) |
| Gov review period | 8,568 blocks (~6 hours) | 50,000 blocks (~1.4 days) |
| Min validators for full reward | 3 | 22 |
| Emergency min distinct voters | 3 | 4 |

All other parameters match production defaults. See `scripts/testnet-genesis-config.json` for the complete reference.

## Knowledge ExtendedParams

The knowledge module has 72 additional parameters (ExtendedParams) that are NOT in genesis.json. They are initialized to `DefaultExtendedParams()` during chain startup and can be updated via governance:

```bash
legbled tx knowledge update-extended-params \
  --from authority \
  --extended-params '{"citation_share_bps": 150000, ...}' \
  --chain-id lgm-testnet-1
```

See `scripts/testnet-genesis-config.json` under `modules.knowledge.extended_params` for the complete reference of all 72 fields.

## Key Files

| File | Purpose |
|------|---------|
| `scripts/testnet-genesis.sh` | Genesis ceremony script |
| `scripts/testnet-genesis-config.json` | Complete parameter reference |
| `scripts/genesis-ceremony.sh` | Production genesis ceremony |
| `devnet/scripts/cosmos-devnet.sh` | Single-node devnet launcher |

## Known Limitations

1. **Single-process testnet**: The ceremony script generates all keys locally. For a distributed testnet, each validator should generate their own keys and share only the gentx.

2. **gRPC disabled**: gRPC is disabled due to v2alpha1 reflection panics on hand-written message types. Use REST API or CLI for queries.

3. **ExtendedParams not in genesis**: Knowledge module's 72 ExtendedParams are initialized to defaults and require a governance transaction to change.

4. **No state migration**: `migrate-genesis` command is a Phase 5 stub. Cannot import state from the TypeScript prototype.

5. **Vote extensions required**: `vote_extensions_enable_height = "1"` is mandatory. Without it, VRF-based validator selection for PoT will not function.

6. **Bootstrap period**: First 480,000 blocks (~14 days), essential PoT transactions (claims, commitments, reveals, validator registration, account registration) are gas-free to bootstrap the knowledge verification network.

7. **Account pubkey bootstrap**: The first transaction from a new account must be broadcast to register the on-chain pubkey. Until then, `legbled query auth account <addr>` returns a null pubkey and the account cannot sign multi-message transactions.

8. **Module account permissions**: Before launch, verify all modules that use `SendCoinsFromModuleToAccount` or `SendCoinsFromAccountToModule` are registered in `maccPerms` (`app/app.go`). Missing entries will cause runtime panics.

9. **PoT verification path**: The deterministic commit/reveal path (submit claim → commit → reveal → fact creation) is fully tested end-to-end. The arbiter-based dispute resolution path needs additional testing before production use.

10. **Seed node IDs not finalized**: The `persistent_peers` config in the validator onboarding section uses placeholder node IDs. Real node IDs can only be obtained after chain startup via `legbled comet show-node-id --home ~/.legbled/config`. Planned seed nodes: TBD (the previously planned Hetzner hosts, Sentry and Patch, were retired 2026-10-13). The seed nodes must be running and their IDs extracted before distributing validator onboarding docs. Blocked until devnet/testnet chain restarts under Yu's approval.

11. **Submitter-exclusion gap**: Validators selected as verifiers for a round are not explicitly excluded from submitting facts in the same round. This creates a weak conflict-of-interest surface. Document or implement exclusion logic before testnet launch.

12. **Flag #31 — IncrementBurned dead code (Sprint A blocker)**: `MeterBurnedKey` (0x11) is defined in the token meter but `IncrementBurned` is never called at any of the 14 `BurnCoins` sites in the codebase. `burn_pressure_ratio = TotalBurned / TotalMinted` is therefore uncomputable and the burned counter will read 0 until Sprint A lands (~14 LOC fix). **Pre-testnet required**: Sprint A must be applied before launch, or token-meter dashboards and ECONOMY-DESIGN-003 will report false data.

13. **Flag #43 — Dual minted tracking (divergence risk)**: `TotalMintedKey` (0x0B) is written via `keeper.go` block-reward path; `MeterMintedKey` (0x10) is written via `IncrementMinted` in `rewards.go`. Both count minted LGM but via independent code paths — they can diverge if one path is called without the other. **Pre-testnet required**: audit that every mint site calls both paths, or consolidate to a single authoritative counter.

14. **Flag #45 — FARM-11 citation-cartel detection (CLOSED)**: `UpdateSubmitterCitationGraph` and `GetCitationConcentrationDiscount` are fully implemented in `x/knowledge/keeper/citation_defense.go` and wired into the citation reward path at `citation.go:122`. Tests exist. Not a launch blocker. Verified 2026-05-06 beat 58.

12. **Stake concentration undocumented**: No documentation of submitter concentration risks or maximum single-submitter stake share in TESTNET-LAUNCH.md. Add governance parameter guidance before external validators join.
