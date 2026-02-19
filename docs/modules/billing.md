# x/billing — Provider Pricing & Distribution

## Purpose

Knowledge API provider management and quote-based pricing. Providers register with domain specialization, clients request quotes, execute payments with protocol-level fee distribution.

## CLI Commands

### Transactions
```bash
legbled tx billing register-provider --domains "physics,math" --stake 1000000ulgm --from alice
legbled tx billing deregister-provider --from alice
legbled tx billing request-quote [provider] [fact-ids] --from bob
legbled tx billing execute-payment [quote-id] --from bob
```

### Queries
```bash
legbled query billing params
legbled query billing provider [address]
legbled query billing quote [id]
```

## Events

`ProviderRegistered`, `QuoteRequested`, `PaymentExecuted`

## Dependencies

StakingKeeper, BankKeeper, KnowledgeKeeper

## EndBlock

Expires stale quotes.
