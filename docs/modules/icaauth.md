# x/icaauth — Interchain Accounts Auth

## Purpose

Controls Legible agent accounts' ability to execute transactions on remote chains via IBC Interchain Accounts. Enforces per-account message type whitelists, registration limits, and DID-based ownership.

## Key Concepts

- **Per-Account Whitelist**: `allowed_msg_types` restricts which proto URLs can be sent
- **Global Whitelist**: `allowed_host_msg_types` restricts all ICA messages
- **Registration Limit**: Max ICAs per DID with cooldown
- **Operational Key Only**: Session keys cannot register or send ICA transactions

## CLI Commands

### Transactions
```bash
legbled tx icaauth register [connection-id] --allowed-msgs "/cosmos.bank.v1beta1.MsgSend" --from alice
legbled tx icaauth send-tx [connection-id] [msg-bytes-hex] --timeout 600000000000 --from alice
```

### Queries
```bash
legbled query icaauth accounts [did]
legbled query icaauth params
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| max_remote_accounts_per_did | varies | Max ICAs per DID |
| allowed_host_msg_types | varies | Global proto URL whitelist |
| registration_cooldown | varies | Blocks between registrations |

## Events

`register_interchain_account`, `send_ica_transaction`, `icaauth_params_updated`

## Dependencies

LegibleAuthKeeper (DID verification), ICAControllerKeeper (IBC ICA operations)
