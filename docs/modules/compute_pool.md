# x/compute_pool — Compute Marketplace

## Purpose

Distributed compute resource marketplace. Providers register capacity, users redeem credits, and tickets track usage with confirmation/dispute flow. Supports heartbeat-based liveness and epoch-based finalization.

## CLI Commands

### Transactions
```bash
legbled tx compute_pool register-provider --service-type "gpu" --price 100ulgm --from provider1
legbled tx compute_pool update-provider --price 150ulgm --from provider1
legbled tx compute_pool deregister-provider --from provider1
legbled tx compute_pool heartbeat --from provider1
legbled tx compute_pool redeem-credits [provider] --amount 1000000ulgm --from user
legbled tx compute_pool confirm-usage [ticket-id] --from user
legbled tx compute_pool dispute-usage [ticket-id] --reason "..." --from user
```

### Queries
```bash
legbled query compute_pool params
legbled query compute_pool provider [address]
legbled query compute_pool active-providers
legbled query compute_pool credit-balance [user] [provider]
legbled query compute_pool ticket [id]
legbled query compute_pool pool-stats
```

## Events

`ProviderRegistered`, `CreditsRedeemed`, `TicketConfirmed`, `TicketDisputed`

## Dependencies

BankKeeper, StakingKeeper

## BeginBlock

Expires credits, expires redemption tickets, checks provider liveness, applies pending price changes, finalizes epochs.
