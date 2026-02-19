# x/ibcratelimit — IBC Rate Limiting

## Purpose

Middleware for IBC token transfers. Tracks inflow/outflow per channel per epoch with configurable limits. Prevents cross-chain liquidity drains by enforcing max transfer volumes.

## Key Concepts

- **Epoch-Based**: Flow tracking resets at epoch boundaries
- **Sliding Window**: Previous epoch flow preserved for smooth transitions
- **Per-Channel**: Independent tracking per IBC channel
- **Asymmetric**: Separate inflow and outflow limits

## CLI Commands

### Queries
```bash
legbled query ibcratelimit params
legbled query ibcratelimit channel-flow [channel-id]
```

No transaction commands (governance-only via MsgUpdateParams).

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| epoch_length_blocks | 34,272 | ~1 day per epoch |
| max_outflow_per_epoch | "10000000000" | 10,000 LGM max outflow |
| max_inflow_per_epoch | "10000000000" | 10,000 LGM max inflow |
| tracked_denom | "ulgm" | Tracked denomination |

## Events

`update_ibcratelimit_params`

## Dependencies

None (standalone middleware).

## BeginBlock

Resets expired flow records at epoch boundaries.
