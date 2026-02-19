# B9-4: Validator Onboarding — Session Report

## Summary

Mapped the full validator journey, tested the complete validator lifecycle on a fresh devnet, created comprehensive onboarding documentation, and built a join-testnet script.

## Step 1: Validator Journey Mapping

Explored the full validator lifecycle across:
- `x/staking/` — 4-tier graduated system (Apprentice → Verified → Bonded → Guardian)
- `x/staking/keeper/msg_server.go` — Registration, delegation, undelegation, redelegate handlers
- `x/staking/types/types.go` — Tier configs, validator struct, delegation struct
- `x/vesting_rewards/` — Block reward emission and vesting
- `docs/TESTNET-LAUNCH.md` — Existing launch docs

### Key Findings

**4-Tier System:**

| Tier | Min Stake | Min Verifications | Min Accuracy | Role |
|------|-----------|-------------------|--------------|------|
| Apprentice | 111,000 ulgm | 0 | 0% | Verifier only |
| Verified | 1,110,000 ulgm | 22 | 77% | Verifier only |
| Bonded | 1,111,000,000 ulgm | 11 | 50% | Block producer |
| Guardian | 11,111,000,000 ulgm | 333 | 77% | Block producer + contested |

**Registration Flow:**
1. `MsgRegisterValidator` creates `LegibleValidator` with initial Reputation=500000
2. Self-delegation parsed as plain integer (no denom suffix)
3. Minimum deposit: 111,000 ulgm (0.111 LGM)
4. Commission in basis points (1000 = 10%)
5. Tier computed automatically from stake/performance

**SDK vs Custom Staking:**
- Cosmos SDK staking (gentx) creates standard validators for consensus
- `legible_staking` module is a parallel system for PoT validator management
- Both coexist — SDK staking handles CometBFT consensus, legible_staking handles PoT tiers

## Step 2: DevNet Lifecycle Test

### Test Results (all on fresh devnet, chain ID `lgm-devnet-1`)

| Operation | TX Hash (prefix) | Code | Height | Result |
|-----------|-------------------|------|--------|--------|
| Fund val2 (500 LGM from test1) | `18737F95...` | 0 | ~10 | Balance: 500,000,000 ulgm |
| Register val2 | `870B4A26...` | 0 | 14 | Validator created, Apprentice tier |
| Delegate 100 LGM (test2→val2) | `E5E6D461...` | 0 | 20 | delegated_stake: 100,000,000 |
| Undelegate 50 LGM (test2←val2) | `C5DF6A4C...` | 0 | 25 | delegated_stake: 50,000,000 |
| Update stake +50 LGM (val2) | `2E78CE4C...` | 0 | 31 | stake: 250,000,000 |

### Final Validator State

```
operator_address: lgm1je536yah25zh9sfd9pptp6zgxv0mkpstw4phry
stake: 250,000,000 (250 LGM)
delegated_stake: 50,000,000 (50 LGM)
total_stake: 300,000,000 (300 LGM)
tier: apprentice
commission: 1000 (10%)
reputation: 500,000
is_active: true
```

### Issues Discovered

1. **Self-delegation denom suffix**: `register-validator` takes a plain number, not a coin string. Passing `200000000ulgm` fails with "invalid initial stake" — must pass `200000000`.

2. **Bank queries not in CLI**: `legbled query bank` returns "unknown command". SDK bank module queries not registered as custom CLI commands. Workaround: REST API at `localhost:1317/cosmos/bank/v1beta1/balances/<addr>`.

3. **Chain can stall on restart with stale state**: After a crash, restarting against the old data directory sometimes produced a stuck chain (blocks stopped advancing). Fresh init resolves it.

4. **SDK + custom staking coexistence**: The genesis gentx validator appears only in SDK staking, not in legible_staking. Legible validators must be registered via `register-validator` TX.

## Step 3: Consensus Participation

On the single-node devnet, only the SDK genesis validator produces blocks. The registered legible_staking validator (val2) is at the Apprentice tier and is a **verifier only** — it does not produce blocks. Consensus participation for PoT verification rounds would require active knowledge claims to verify.

Reputation starts at 500,000 and adjusts based on verification performance:
- Correct verification: +100 reputation
- Incorrect verification: -200 reputation
- Slash event: -10,000 reputation

## Step 4: Validator Operations

All tested successfully:
- **Register**: `tx legible_staking register-validator`
- **Increase stake**: `tx legible_staking update-stake <amount> --increase`
- **Delegate**: `tx legible_staking delegate <validator> <amount>`
- **Undelegate**: `tx legible_staking undelegate <validator> <amount>`
- **Query validators**: `query legible_staking validators`
- **Query delegations**: `query legible_staking delegations-for-validator <addr>`
- **Query unbondings**: `query legible_staking unbondings <delegator>`
- **Query params**: `query legible_staking params`

Not tested (not implemented or would require multi-node setup):
- Unjailing (requires slashing, which requires sustained downtime)
- Redelegate (requires two validators both registered)
- Commission update (no dedicated TX in current CLI)

## Step 5: Documentation Created

### `docs/VALIDATOR-ONBOARDING.md`
- Prerequisites (hardware, software)
- Step-by-step join instructions (8 steps)
- Tier advancement table
- Common operations (stake, delegate, undelegate, redelegate)
- Monitoring setup (Prometheus endpoints, useful queries)
- Slashing details
- Troubleshooting (common errors with solutions)

### `scripts/join-testnet.sh`
- Argument parsing (moniker, chain-id, peers, genesis-url, etc.)
- Builds binary with version ldflags
- Initializes node
- Downloads and validates genesis
- Configures CometBFT (block time, peers, prometheus)
- Configures app (gas prices)
- Prints next-steps summary
- Optional `--start` to begin syncing

## File Ownership

| File | Status |
|------|--------|
| `docs/VALIDATOR-ONBOARDING.md` | **NEW** — Comprehensive onboarding guide |
| `scripts/join-testnet.sh` | **NEW** — One-liner join script |
| `reports/batch-9/SESSION-B9-4-VALIDATOR-ONBOARDING.md` | **NEW** — This report |

## Deferred / Future Work

1. **Multi-node devnet testing**: Full consensus participation with 3+ validators, tier advancement, VRF selection
2. **Commission update TX**: No CLI command exists yet for updating commission after registration
3. **Unjail flow**: Need to test slashing → jail → unjail lifecycle
4. **State sync**: `join-testnet.sh` could be enhanced with state sync configuration for faster sync
5. **Docker-based join**: Provide a Docker image + compose file for validators who don't want to build from source
