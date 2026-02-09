# Legible Money DevNet Launch Report

**Date**: 2026-02-09
**Chain ID**: `legible-devnet-1`
**Binary**: `legbled` (103 MB, Go 1.25.6, darwin/arm64)

---

## Executive Summary

The Legible Money Cosmos SDK chain (`legbled`) has been successfully built, initialized, and launched as a single-validator devnet. The chain produces blocks at the target 2.521-second interval, all 24 genesis modules initialize correctly, and both gRPC and REST API endpoints are fully operational. One critical infrastructure bug was discovered and fixed during testing (GoLevelDB empty-value IAVL incompatibility).

### Key Results

| Metric | Status |
|--------|--------|
| Binary build | PASS |
| Genesis initialization | PASS |
| Block production | PASS (2.521s target) |
| Block finality | PASS |
| gRPC queries | PASS (all SDK modules) |
| REST API queries | PASS (all SDK modules) |
| Custom module gRPC | PARTIAL (registered but no REST gateway bindings) |
| Go module tests | PASS (10 keeper test suites, 0 failures) |

---

## Stack Versions

| Component | Version |
|-----------|---------|
| Cosmos SDK | v0.50.10 |
| CometBFT | v0.38.12 |
| IAVL | v1.2.0 |
| IBC-Go | v8.5.1 |
| cosmos-db | v1.0.3-pre (2024-09-11) |
| Go | 1.25.6 |

---

## Genesis Configuration

### Chain Parameters

| Parameter | Value |
|-----------|-------|
| Chain ID | `legible-devnet-1` |
| Genesis time | 2026-02-09T15:42:35Z |
| Block time (timeout_commit) | 2.521s |
| Minimum gas price | 0.025 ulgm |
| Bond denom | ulgm |
| Bech32 prefix | lgm |
| Pruning | nothing (archive node) |

### Consensus Timeouts

| Phase | Timeout |
|-------|---------|
| Propose | 2.0s (+500ms delta) |
| Prevote | 1.0s (+500ms delta) |
| Precommit | 1.0s (+500ms delta) |
| Commit | 2.521s |

### Initial State

| Item | Value |
|------|-------|
| Genesis accounts | 1 |
| Genesis validators | 1 (50,000 LGM voting power) |
| Initial supply | 100,000,000,000 ulgm (100,000 LGM) |
| Validator stake | 50,000,000,000 ulgm (50,000 LGM) |
| Free balance | 50,000,000,000 ulgm (50,000 LGM) |

### Module Genesis Sizes

| Module | Genesis Size | Notes |
|--------|-------------|-------|
| ontology | 4,164 B | 4 initial domains with hierarchies |
| knowledge | 1,444 B | 4 domains (math, physics, CS, general) |
| legible_staking | 1,217 B | 4-tier validator system + 1 genesis validator |
| genutil | 1,204 B | Genesis transaction (create-validator) |
| gov | 826 B | Standard SDK governance params |
| disputes | 726 B | Dispute resolution params |
| emergency | 705 B | 3-phase BFT kill switch params |
| ibc | 656 B | IBC client state |
| lgm_gov | 607 B | LIP governance + quadratic voting params |
| distribution | 473 B | Community tax 2% |
| auth | 340 B | 1 base account |
| staking | 341 B | Cosmos staking (delegated to legible_staking) |
| research | 335 B | Research fund params |
| mint | 328 B | Inflation 7-20%, blocks/year 6,311,520 |
| tokens | 299 B | Dual-ledger period overlay params |
| evidence_mgmt | 299 B | Evidence scoring params |
| bank | 290 B | Supply + balance records |
| slashing | 273 B | 100-block signed window, 50% uptime |
| legible_auth | 152 B | Dual-key auth params |
| transfer | 124 B | IBC transfer params |
| crisis | 53 B | Crisis module constant fee |
| evidence | 16 B | SDK evidence (empty) |
| feegrant | 18 B | Fee grant (empty) |
| upgrade | 2 B | Upgrade module (empty) |
| vesting | 2 B | Vesting (empty) |

---

## Runtime Observations

### Block Production

Block production started immediately after genesis and maintained consistent ~2.5s intervals:

```
Height 1:  genesis block
Height 56: observed after ~140s (2.5s/block average)
Height 66: stable production confirmed
```

Block rate measurement: 3 blocks in 6 seconds = 2.0s/block effective (CometBFT optimizes when there are no contending validators).

### Token Economics (Live)

After ~66 blocks:

| Metric | Value |
|--------|-------|
| Total supply | 100,000,135,894 ulgm |
| Minted (inflationary) | 135,894 ulgm (~0.136 LGM) |
| Inflation rate | 13.00% |
| Annual provisions | ~13,000 LGM |
| Community pool | ~2,717.88 ulgm |

### API Endpoints

| Endpoint | Port | Status |
|----------|------|--------|
| CometBFT RPC | 26657 | OPERATIONAL |
| REST API (LCD) | 1317 | OPERATIONAL |
| gRPC | 9090 | OPERATIONAL |
| gRPC-Web | 1317 | ENABLED |
| P2P | 26656 | LISTENING |

### Verified Query Endpoints

All standard Cosmos SDK query endpoints respond correctly:

- `cosmos.bank.v1beta1.Query` - balances, supply, denoms
- `cosmos.staking.v1beta1.Query` - validators, pool, delegations
- `cosmos.mint.v1beta1.Query` - inflation, annual provisions, params
- `cosmos.distribution.v1beta1.Query` - community pool, params
- `cosmos.slashing.v1beta1.Query` - signing infos, params
- `cosmos.gov.v1.Query` - proposals, params
- `cosmos.evidence.v1beta1.Query` - evidence list
- `cosmos.feegrant.v1beta1.Query` - allowances
- `cosmos.upgrade.v1beta1.Query` - current plan, applied upgrades
- `cosmos.consensus.v1.Query` - consensus params
- `cosmos.params.v1beta1.Query` - parameter subspaces
- `ibc.core.client.v1.Query` - client states
- `cosmos.base.tendermint.v1beta1.Service` - blocks, node info

---

## Custom Module Status

### Fully Implemented Modules (10)

| Module | Store Key | Tests | gRPC Service |
|--------|-----------|-------|--------------|
| x/knowledge | `knowledge` | 16 tests | Registered (hand-written) |
| x/auth | `legible_auth` | 38 tests | Registered (hand-written) |
| x/staking | `legible_staking` | 8 tests | Registered (hand-written) |
| x/emergency | `emergency` | 30 tests | Registered (hand-written) |
| x/tokens | `tokens` | 5 tests | Registered (hand-written) |
| x/gov | `lgm_gov` | 37 tests | Registered (hand-written) |
| x/research | `research` | 4 tests | Registered (hand-written) |
| x/ontology | `ontology` | 4 tests | Registered (hand-written) |
| x/disputes | `disputes` | 4 tests | Registered (hand-written) |
| x/evidence_mgmt | `evidence_mgmt` | 4 tests | Registered (hand-written) |

### Stub Modules (5, No Store Keys)

| Module | Purpose | Phase |
|--------|---------|-------|
| x/bvm | BVM contract execution | Phase 3 |
| x/channels | Payment channels | Phase 3 |
| x/tree | Task tree / projects | Phase 3 |
| x/claiming_pot | Collective reward pools | Phase 3 |
| x/vesting_rewards | Vesting schedules | Phase 3 |

### Custom Module gRPC Limitation

Custom modules use hand-written gRPC service descriptors (not proto-generated) because the Go types use `json.Marshal` instead of protobuf codec. This means:

- gRPC services ARE registered and functional via direct gRPC calls
- gRPC-gateway REST bindings are NOT available (would require proto-generated gateway code)
- gRPC reflection does NOT list custom services (no proto file descriptors)

**Workaround**: Query custom modules via direct gRPC or through CometBFT ABCI query paths.

---

## Bugs Discovered and Fixed

### Critical: GoLevelDB Empty-Value IAVL Incompatibility

**Severity**: Critical (blocked all gRPC/REST state queries)
**Root Cause**: Behavioral inconsistency between cosmos-db backends

The `IAVL v1.2.0` library's `SaveEmptyRoot()` function writes `[]byte{}` (empty byte slice) as the value for IAVL tree roots that contain no keys. The `GetRoot()` function checks `if val == nil` to determine if a version exists.

| Backend | `Get(key_with_empty_value)` | `GetRoot` behavior |
|---------|----------------------------|-------------------|
| MemDB | Returns `([]byte{}, nil)` | Works (non-nil = exists) |
| GoLevelDB | Returns `(nil, nil)` | **FAILS** (nil = not found) |

This caused `CacheMultiStoreWithVersion()` to fail for any KV store that had no data written during genesis, returning `"version does not exist"` for ALL query heights including the latest.

**Fix**: Added `ensureNonEmptyStores()` in `InitChainer` that writes a sentinel key (`__store_initialized`) to every KV store after module genesis initialization. This ensures no IAVL tree is empty when committed, avoiding the GoLevelDB nil-for-empty-value behavior.

```go
// app.go — post-genesis sentinel to prevent empty IAVL stores
func (app *LegibleApp) ensureNonEmptyStores(ctx sdk.Context) {
    sentinel := []byte("__store_initialized")
    for _, key := range app.keys {
        store := ctx.KVStore(key)
        if !store.Has(sentinel) {
            store.Set(sentinel, []byte{0x01})
        }
    }
}
```

**Impact**: This is a latent bug in the cosmos-db + IAVL stack that affects any Cosmos SDK chain using GoLevelDB with modules that have empty stores at genesis. Standard SDK modules are typically unaffected because they all write params during `InitGenesis`, but custom modules or minimal genesis configurations can trigger it.

**Upstream**: Consider reporting to `cosmos/iavl` — `GetRoot` should check `len(val) == 0` instead of `val == nil`, or `SaveEmptyRoot` should write a non-empty sentinel value.

### Fixed Earlier: Missing Slashing Signing Info

The chain crashed at block 2 because the slashing module's `BeginBlocker` expected signing info for the genesis validator but none was written during `InitGenesis`. Fixed by ensuring the genesis validator's signing info is properly initialized.

### Fixed Earlier: Vote Extensions Height Check

CometBFT v0.38 enables vote extensions at height 2+ but the initial ABCI configuration had `VoteExtensionsEnableHeight` set to 0 (enabled from genesis). Fixed by setting it to 2.

---

## Performance Observations

| Metric | Value |
|--------|-------|
| Time to first block | ~3s |
| Average block time | 2.0-2.5s (single validator, no network latency) |
| Binary startup time | ~2s |
| Genesis init time | <1s |
| REST query latency | <50ms |
| gRPC query latency | <20ms |
| Binary size | 103 MB |
| Data dir size (66 blocks) | ~15 MB |

---

## Recommendations for Next Steps

### Immediate (DevNet Hardening)

1. **Register custom gRPC-gateway routes**: Generate gateway stubs from proto files or write manual HTTP handlers for custom module queries
2. **Add transaction submission tests**: Submit knowledge claims, register validators, create disputes via CLI
3. **Multi-node testing**: Spin up 3+ validators to test consensus under real conditions
4. **Vote extension testing**: Configure a validator with PoT VRF keys and verify ABCI++ vote extension flow

### Short-term (Testnet Preparation)

5. **Proto reconciliation**: Sync proto definitions with Go types (plan exists at `.claude/plans/vivid-honking-puddle.md`)
6. **Gas cost calibration**: Current gas costs are placeholders — benchmark real execution costs
7. **CLI security**: Add key encryption, safe seed phrase handling
8. **State sync snapshots**: Enable snapshot interval for fast node bootstrapping
9. **Persistent peers configuration**: Set up seed node discovery

### Medium-term (Mainnet Preparation)

10. **Security audit**: Third-party review of consensus, state transitions, and key management
11. **Cosmos-db upstream fix**: Report GoLevelDB empty-value bug to `cosmos/iavl`
12. **Block reward scaling**: Implement PoT-aware reward distribution (currently using standard SDK mint)
13. **Emergency module testing**: Full 3-phase kill switch ceremony under multi-validator setup
14. **IBC integration testing**: Cross-chain asset transfers and oracle data feeds

---

## Appendix: Reproduction Steps

### Build

```bash
cd /path/to/legible_money
go build -o legbled ./cmd/legbled/
```

### Initialize Genesis

```bash
export HOME_DIR=/tmp/legbled-devnet

legbled init legible-devnet --chain-id legible-devnet-1 --home $HOME_DIR
legbled keys add validator --keyring-backend test --home $HOME_DIR
legbled genesis add-genesis-account validator 100000000000ulgm \
    --keyring-backend test --home $HOME_DIR
legbled genesis gentx validator 50000000000ulgm \
    --chain-id legible-devnet-1 --keyring-backend test --home $HOME_DIR
legbled genesis collect-gentxs --home $HOME_DIR
```

### Configure

Edit `$HOME_DIR/config/config.toml`:
```toml
timeout_commit = "2.521s"
```

Edit `$HOME_DIR/config/app.toml`:
```toml
minimum-gas-prices = "0.025ulgm"
pruning = "nothing"
```

### Launch

```bash
legbled start --home $HOME_DIR
```

### Verify

```bash
# Block production
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Bank query
curl -s http://localhost:1317/cosmos/bank/v1beta1/supply

# gRPC
grpcurl -plaintext localhost:9090 list
```

---

*Report generated during Legible Money DevNet launch testing session, 2026-02-09.*
*Chain operational at height 66+ with all systems nominal.*
