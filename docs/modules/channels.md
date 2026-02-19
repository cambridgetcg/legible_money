# x/channels — Payment Channels

## Purpose

Off-chain payment channels with on-chain settlement. Supports opening channels with deposits, off-chain balance updates with Ed25519 signatures, cooperative close, dispute resolution with challenge windows, and auto-settlement of expired channels.

## Key Concepts

- **Channel Lifecycle**: open → active → closing → settled (or disputed)
- **Ed25519 Signatures**: All state mutations require cryptographic proof
- **Dispute Window**: Counterparty can challenge close with newer signed state
- **Auto-Settlement**: Expired dispute windows settle automatically

## CLI Commands

### Transactions
```bash
legbled tx channels open-channel [provider] --deposit 1000000ulgm --from payer
legbled tx channels update-channel [channel-id] [amount] [signature] --from payer
legbled tx channels close-channel [channel-id] [final-state] [signature] --from payer
legbled tx channels settle-channel [channel-id] --from anyone
legbled tx channels dispute-channel [channel-id] [newer-state] [proof-signature] --from provider
legbled tx channels deposit-channel [channel-id] [amount] --from payer
legbled tx channels update-state [channel-id] [state-json] [payer-sig] [provider-sig] --from either
legbled tx channels claim-expired [channel-id] --from provider
```

### Queries
```bash
legbled query channels channel [id]
legbled query channels channels-by-payer [address]
legbled query channels channels-by-provider [address]
legbled query channels active-dispute [channel-id]
legbled query channels params
```

## Parameters

Dispute window duration, settlement periods, fee structures.

## Events

`ChannelOpened`, `ChannelUpdated`, `ChannelDisputed`, `ChannelSettled`

## Dependencies

BankKeeper, AccountKeeper

## BeginBlock

Auto-settles channels past their dispute window.
