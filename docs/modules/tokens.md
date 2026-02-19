# x/tokens — LGM-20 Token Standard

## Purpose

Custom token management for LGM-20 compatible tokens. Supports token creation with feature flags (mintable, burnable, pausable, wrappable), ERC-20 style approvals, and liquidity pool operations (swap, add/remove liquidity).

## Key Concepts

- **Feature Flags**: `--mintable`, `--burnable`, `--pausable`, `--wrappable` at creation
- **Approvals**: ERC-20 style allowance and transfer-from
- **Liquidity Pools**: Token pair pools for AMM swaps
- **Pause/Unpause**: Creator can pause token transfers

## CLI Commands

### Transactions
```bash
legbled tx tokens create-token [name] [symbol] [supply] --mintable --burnable --from alice
legbled tx tokens mint-token [token-id] [amount] --from creator
legbled tx tokens burn-token [token-id] [amount] --from holder
legbled tx tokens transfer-token [token-id] [to] [amount] --from alice
legbled tx tokens approve-token [token-id] [spender] [amount] --from alice
legbled tx tokens transfer-from [token-id] [from] [to] [amount] --from spender
legbled tx tokens pause-token [token-id] --from creator
legbled tx tokens unpause-token [token-id] --from creator
legbled tx tokens add-liquidity [pool-id] [amount-a] [amount-b] --from alice
legbled tx tokens remove-liquidity [pool-id] [shares] --from alice
legbled tx tokens swap [pool-id] [input-token] [amount] --min-output [amount] --from alice
```

### Queries
```bash
legbled query tokens params
```

## Parameters

Module params configurable via governance.

## Events

Token creation, transfer, mint, burn, approval, pool operations.

## Dependencies

BankKeeper, AccountKeeper

## Note

Token message handlers are proto-skeleton stubs (14 handlers). Core token operations work through the Cosmos SDK bank module for ulgm.
