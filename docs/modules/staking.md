# x/staking — Validator Staking & Reputation

## Purpose

Custom staking module for Proof of Truth consensus. Implements a 4-tier validator system with reputation scoring, VRF-based selection via ABCI++ vote extensions, delegation, and anti-grief slashing controls.

## Key Concepts

- **4 Tiers**: Apprentice (0, limited) → Verified (1) → Bonded (2) → Guardian (3)
- **Reputation**: Per-validator correctness tracking; accumulates on correct verifications, decays on incorrect
- **VRF Selection**: Vote extensions carry ECVRF proofs for validator selection in verification rounds
- **Delegation**: Delegators stake to validators, share rewards proportionally
- **Anti-Grief**: Max slashes per epoch, escalation on repeat offenses, deactivation threshold

## CLI Commands

### Transactions
```bash
legbled tx legible_staking register-validator --moniker "my-node" --stake 111000ulgm --from alice
legbled tx legible_staking update-validator-stake --amount 500000ulgm --from alice
legbled tx legible_staking delegate [validator-addr] 1000000ulgm --from bob
legbled tx legible_staking undelegate [validator-addr] 500000ulgm --from bob
legbled tx legible_staking redelegate [src-validator] [dst-validator] 500000ulgm --from bob
```

### Queries
```bash
legbled query legible_staking params
legbled query legible_staking validator [address]
legbled query legible_staking validators
legbled query legible_staking delegation [delegator] [validator]
legbled query legible_staking delegations-for-validator [validator]
legbled query legible_staking unbondings [delegator]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| max_validators | 100 | Max total validators |
| unbonding_period | 268,560 blocks | ~7 days unbonding |
| min_self_delegation | "111000" ulgm | Min self-delegation |
| max_slashes_per_epoch | 2 | Anti-grief limit |
| reputation_correct_delta | 100 BPS | +0.01% per correct |
| reputation_incorrect_delta | 200 BPS | -0.02% per incorrect |
| redelegation_cooldown_blocks | 1,111 | ~46 min between redelegations |

15 total governance-adjustable parameters.

## Events

`validator_registered`, `validator_tier_changed`, `delegation_created`, `delegation_unbonding`, `delegation_redelegated`, `validator_slashed`, `reputation_updated`

## Dependencies

AccountKeeper, BankKeeper, LegibleAuthKeeper, AutopoiesisKeeper (optional)
