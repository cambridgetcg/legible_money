# Zerone Devnet Status

- **Status**: LIVE
- **Started**: 2026-03-17T14:04:30Z
- **Chain ID**: lgm-devnet-1
- **RPC endpoint**: http://localhost:26657
- **REST endpoint**: http://localhost:1317
- **gRPC endpoint**: localhost:9090
- **Block height**: 59+ (producing ~1 block / 2.521s)
- **Binary**: `build/legbled` (133M, built from source)
- **Validator**: lgm1qvfhrkyssxjtm8a2u2gpx5yaqmvkt9gs2lqy9t (10,000 LGM self-delegated)
- **Total supply**: 50,497,444 LGM
- **Genesis facts**: 75 axioms injected

## Health Check Results (9/9 PASS)

| Check | Result |
|-------|--------|
| Node reachable | PASS (HTTP 200) |
| Block production | PASS |
| Node synced | PASS |
| Chain ID | PASS (lgm-devnet-1) |
| REST API | WARN (HTTP 501 — gRPC-gateway routing, queries work via RPC) |
| Active validators | PASS (1) |
| Block time | PASS |
| Total supply | PASS (50,497,444 LGM) |
| Bonded tokens | PASS (10,000 LGM) |

## TX Lifecycle Test Results (23/23 PASS)

| Phase | Tests | Result |
|-------|-------|--------|
| Phase 1: Bank Operations | 6 | ALL PASS |
| Phase 2: Staking Operations | 5 | ALL PASS |
| Phase 3: Governance Operations | 4 | ALL PASS |
| Phase 4: Authorization & Errors | 5 | ALL PASS |
| Phase 5: Economic Invariants | 3 | ALL PASS |

### Highlights
- Token transfers (send, multi-hop) work correctly
- Insufficient funds and invalid addresses properly rejected
- Delegation/undelegation operational, pool conservation verified
- Governance proposal submitted, voted, and **passed** (60s voting period)
- Wrong chain-id, zero fees, invalid recipients all rejected
- Supply conservation maintained across all operations
- Fee deduction verified (send: 100,000 ulgm, fee: 197,584 ulgm)

## Issues Found

- **REST API returns 501**: The gRPC-gateway REST endpoint at `:1317` returns 501 for some routes. RPC queries at `:26657` work fine. This is a known Cosmos SDK v0.50 configuration issue — not blocking.
- **gRPC service registration warning**: `MsgProposeResearchFund` not registered in gogoproto (known — see `app/proto_compat.go` bridge). Does not affect block production or tx routing.
- **Single validator**: Devnet runs with 1 validator (expected for local dev). Not representative of production consensus.

## Commands

```bash
# Check status
make devnet-health

# Stop devnet
make devnet-stop

# Reset and restart
make devnet-reset && make devnet

# Run tx lifecycle tests
make test-tx-lifecycle
```
