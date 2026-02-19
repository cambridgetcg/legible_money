# x/auth — Dual-Key Identity & Session Keys

## Purpose

Custom authentication with 4-layer key architecture: identity key (permanent Ed25519), operational key (rotatable), session keys (ephemeral with capabilities), and recovery shards (Shamir). Manages DID-to-bech32 mapping, key rotation with cooldown, and account recovery.

## Key Concepts

- **DID Format**: `did:lgm:{32-hex}` derived from Ed25519 identity key
- **Dual Keys**: Permanent identity key + rotatable operational key
- **Session Keys**: Ephemeral keys with time limits and capability flags (transfer, stake, submit_claims, vote)
- **Recovery**: Shamir shard-based with delay + challenge period
- **Bootstrap**: Auto-funded on registration when enabled

## CLI Commands

### Transactions
```bash
legbled tx legible_auth register-account [did] [public-key] [account-type] --from alice
legbled tx legible_auth rotate-key [new-key-hex] [auth-signature-hex] --from alice
legbled tx legible_auth create-session [session-pub-key-hex] --can-transfer --can-stake --expires-at 100000 --from alice
legbled tx legible_auth revoke-session [session-id] --from alice
legbled tx legible_auth freeze-account [address] --reason "suspicious" --from alice
legbled tx legible_auth unfreeze-account [address] --from authority
legbled tx legible_auth set-recovery-config [threshold] [total-shards] [shard-holders...] --from alice
legbled tx legible_auth initiate-recovery [account] [new-key] --from guardian1
legbled tx legible_auth submit-recovery-shard [account] [shard-index] --from guardian1
legbled tx legible_auth challenge-recovery [account] [reason] --from bob
legbled tx legible_auth execute-recovery [account] --from guardian1
```

### Queries
```bash
legbled query legible_auth params
legbled query legible_auth account [address]
legbled query legible_auth account-by-did [did]
legbled query legible_auth session-keys [owner-address]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| max_session_keys | 10 | Max concurrent sessions per account |
| max_session_duration | 34,272 blocks | ~1 day max session |
| key_rotation_cooldown | 1,111 blocks | ~46 min between rotations |
| recovery_delay_blocks | 6,854 blocks | ~4.8 hours delay before recovery |
| challenge_period_blocks | 3,427 blocks | ~2.4 hours to challenge |
| bootstrap_enabled | true | Enable bootstrap fund |
| bootstrap_amount | "1000000" | 1 LGM bootstrap (ulgm) |

## Events

`account_registered`, `key_rotated`, `session_created`, `session_revoked`, `account_frozen`, `account_unfrozen`, `recovery_config_set`, `recovery_initiated`, `recovery_shard_submitted`, `recovery_challenged`, `recovery_executed`, `params_updated`

## Dependencies

CosmosAccountKeeper (pubkey sync), BankKeeper (bootstrap fund)
