# x/discovery — Agent Profile Registry

## Purpose

Agent profile registry with capability-based discovery. Agents register with display name, DID, domain/capability lists, and metadata. Heartbeat-based liveness tracking with deterministic expiry at ~10 days.

## Key Concepts

- **Agent Profiles**: Address, display name, DID, domains, capabilities, metadata
- **Heartbeat**: Liveness signal resets expiry timer
- **Capability Types**: Verified or unverified capabilities
- **Profile Expiry**: 342,720 blocks (~10 days without heartbeat)

## CLI Commands

### Transactions
```bash
legbled tx discovery register-agent --did [did] --domains "physics,math" --description "..." --from alice
legbled tx discovery update-profile --description "updated" --from alice
legbled tx discovery add-capability --type "data-analysis" --from alice
legbled tx discovery remove-capability --type "data-analysis" --from alice
legbled tx discovery deregister-agent --from alice
legbled tx discovery heartbeat --from alice
```

### Queries
```bash
legbled query discovery params
legbled query discovery agent [address]
legbled query discovery agents-by-domain [domain]
legbled query discovery agents-by-capability [capability-type]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| MinRegistrationStake | 100 LGM | Registration collateral |
| MaxCapabilitiesPerAgent | 50 | Cap on capabilities |
| ProfileExpiryBlocks | 342,720 | ~10 days without heartbeat |

## Events

`agent_registered`, `agent_deregistered`, `agent_expired`

## Dependencies

BankKeeper (registration stake)

## BeginBlock

Expires inactive profiles (every 100 blocks for amortization).
