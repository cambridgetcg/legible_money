# x/home — Agent Homes & Guardians

## Purpose

Virtual agent personas with multi-signature RBAC, guardian defense strategies, deadman switches, and memory management. Supports session management with expiry, configurable alerts, and automatic aging/dormancy transitions.

## Key Concepts

- **Home Status**: active → dormant → guarded → recovery → archived
- **Access Roles**: owner, operator, guest, guardian, viewer
- **Permissions**: 10 granular permissions (read/write memory, transfer, stake, verify, etc.)
- **Defense Strategies**: aggressive, moderate, conservative, diplomatic
- **Deadman Switch**: Triggers after inactivity (alert guardians, transfer to beneficiary, donate to research)
- **Aging**: Automatic status transition based on inactivity

## CLI Commands

### Transactions
```bash
legbled tx home create-home --name "agent-alpha" --from alice
legbled tx home update-home [home-id] --description "..." --from alice
legbled tx home update-memory [home-id] --memory-key "notes" --memory-value "..." --from alice
legbled tx home configure-guardian [home-id] [guardian-addr] --strategy moderate --from alice
legbled tx home register-key [home-id] [key-hex] --role operator --from alice
legbled tx home revoke-key [home-id] [key-hash] --from alice
legbled tx home start-session [home-id] --from operator
legbled tx home end-session [home-id] [session-id] --from operator
legbled tx home set-spending-limit [home-id] [amount] --from alice
legbled tx home ack-alert [home-id] [alert-id] --from alice
```

### Queries
```bash
legbled query home home [home-id]
legbled query home homes --owner [address] --status active
legbled query home sessions [home-id]
legbled query home alerts [home-id] --acknowledged false
legbled query home params
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| MaxHomesPerAccount | varies | Max homes per account |
| MaxKeysPerHome | varies | Max registered keys |
| MaxSessionsPerHome | varies | Max concurrent sessions |

## Events

`home_created`, `home_updated`, `home_status_changed`, `session_started`, `session_ended`, `guardian_configured`, `defense_activated`, `alert_emitted`, `alert_acknowledged`, `home_aged`

## Dependencies

BankKeeper (fund transfers, treasury operations)
