# R5 Session 5: Docker 3-Validator Testnet Verification

**Date**: 2026-02-18
**Verdict**: PASS (with fixes applied)

---

## Summary

Successfully built, initialized, and ran a 3-validator Docker devnet. All validators reached consensus, produced blocks in sync, and a cross-validator bank transfer was confirmed on all nodes.

---

## 1. Docker Image Build

**Result**: PASS (after fixes)

### Fixes Applied to `Dockerfile.cosmos`
1. **Go version**: `golang:1.22-alpine` → `golang:1.24-alpine` (matches `go.mod` requirement `go 1.24.0`)
2. **Local replace module**: Added `COPY tools/agent-sdk/go.mod ./tools/agent-sdk/` before `go mod download` (required for `replace` directive in `go.mod`)
3. **CGO_ENABLED**: Already set correctly to `1` for Alpine musl builds

### Build Prerequisite: Pointer/Value Type Mismatches

The build uncovered **14 failing packages** with pointer/value type mismatches caused by the proto code generation (`buf generate`) changing types from value to pointer semantics. All were fixed:

| Category | Packages | Fix Pattern |
|----------|----------|-------------|
| Type redeclarations | auth, channels, emergency, gov (`types/`) | Removed hand-written Msg/Query types duplicated by `.pb.go` files |
| `&var` → `**Type` | bvm, staking, compute_pool, partnerships, qualification, knowledge, research, claiming_pot (`keeper/`) | Removed `&` where getter already returns `*Type` |
| Value → pointer | capture_defense, ibcratelimit, ontology (`keeper/`) | Added `&` where setter expects `*Type`, updated iterator callbacks |
| Module genesis | compute_pool, qualification, evidence_mgmt (`module.go`) | `InitGenesis(ctx, genState)` → `InitGenesis(ctx, &genState)` |

**Total files modified**: ~40+ (keepers, tests, types, modules)

### Build Output
```
docker build -f deployment/docker/Dockerfile.cosmos -t legbled:latest .
# Build time: ~57s
# Image size: ~30MB (alpine runtime)
```

---

## 2. DevNet Initialization

**Result**: PASS (after fixes)

### Fixes Applied to `init-cosmos-docker.sh`
1. **Entrypoint bypass**: `docker run` commands now use `--entrypoint legbled` to bypass the shell entrypoint for init operations
2. **Node ID capture**: `tendermint show-node-id` outputs to stderr; changed `2>/dev/null` to `2>&1 | tail -1`
3. **Self-copy guard**: Skip `cp genesis.json` when source == destination (validator-1 → validator-1)
4. **macOS sed syntax**: Removed `{...}` grouping in address ranges (not supported by BSD sed)
5. **Volume naming**: Set `COMPOSE_PROJECT="lgm-devnet"` and `name: lgm-devnet` in compose file for predictable volume names

### Genesis Ceremony Output
```
Validators:
  validator-1: lgm1au8txkapvfjc8j52fpmjnmg6kmz47llvmws08e
  validator-2: lgm1lqtfrt8xm85sef0yhlueetdtsxg0zn2u2hnx62
  validator-3: lgm1v8zxwm7e4ezmqvsrtnnnjazvu993x0prwvz62l
Faucet:       lgm14rs5llqnmet9twk8ur3rk897rrkqkwesmnksjc

Genesis: valid ✓
Gentxs: collected (3/3) ✓
Peers: configured ✓
Volumes: populated (3/3) ✓
```

---

## 3. Network Startup

**Result**: PASS

All 3 containers started and reported `healthy` within 30 seconds.

```
NAME              STATUS                    PORTS
lgm-validator-1   Up 36 seconds (healthy)   26656-26657, 1317, 9090
lgm-validator-2   Up 36 seconds (healthy)   26666-26667, 1318, 9091
lgm-validator-3   Up 36 seconds (healthy)   26676-26677, 1319, 9092
```

---

## 4. Consensus Verification

**Result**: PASS

All 3 validators producing blocks in sync, none catching up:

| Validator | Port  | Block Height | Catching Up |
|-----------|-------|-------------|-------------|
| validator-1 | 26657 | 41 | false |
| validator-2 | 26667 | 41 | false |
| validator-3 | 26677 | 41 | false |

Block advancement: Height 16 at T+30s, Height 41 at T+60s (~25 blocks/30s ≈ 1.2s/block).

---

## 5. gRPC Registration

**Result**: PASS — Zero failures

```bash
docker logs lgm-validator-1 | grep "gRPC service registration FAILED"
# (no output)
```

Only benign errors: P2P address book warnings about non-routable Docker internal IPs (expected in container networking).

---

## 6. Cross-Validator TX

**Result**: PASS

```bash
# Send 100000 ulgm from validator-1 to validator-2
legbled tx bank send <val1> <val2> 100000ulgm --fees 250000ulgm
# code: 0, txhash: 7F55BF38...

# Verify on validator-2 REST API
curl localhost:1318/cosmos/bank/v1beta1/balances/<val2>
# amount: "90000100000" ulgm ✓
```

Initial attempt with 5000ulgm fee failed (`insufficient fee` — minimum is 200000ulgm for gas 200000 × min price 1ulgm). Adjusted to 250000ulgm and succeeded.

---

## 7. API Endpoints

| Endpoint | Port | Result |
|----------|------|--------|
| REST `/cosmos/staking/v1beta1/validators` | 1317 | PASS (3 validators) |
| REST `/cosmos/bank/v1beta1/supply` | 1317 | PASS (1,000,330,000,000,000 ulgm) |
| REST `/cosmos/bank/v1beta1/balances/{addr}` | 1317/1318/1319 | PASS |
| RPC `/status` | 26657/26667/26677 | PASS |

Note: `/cosmos/base/tendermint/v1beta1/node_info` returns null for some fields — this is a known SDK v0.50 issue where the legacy Tendermint REST path requires CometBFT-specific registration.

---

## 8. Cleanup

**Result**: PASS

```bash
docker compose -f deployment/docker/docker-compose.cosmos.yml down -v
# All containers, volumes, and networks removed cleanly
```

---

## Files Modified

### Docker Infrastructure
- `deployment/docker/Dockerfile.cosmos` — Go 1.22→1.24, agent-sdk go.mod copy
- `deployment/docker/docker-compose.cosmos.yml` — Added `name: lgm-devnet`
- `deployment/docker/init-cosmos-docker.sh` — 5 fixes (entrypoint, node ID, cp, sed, volumes)

### Build Fixes (pointer/value mismatches from proto migration)
- `x/auth/types/{codec.go,types.go}` — Removed 16 redeclared Msg types + 8 handlers
- `x/auth/keeper/msg_server.go` — ExtendedMsgServer → MsgServer
- `x/channels/types/{codec.go,types.go}` — Removed 7 redeclared Msg types + 3 handlers
- `x/channels/keeper/msg_server.go` — FullMsgServer → MsgServer
- `x/emergency/types/types.go` — Removed Msg/Query redeclarations
- `x/gov/types/{query.go,codec.go}` — Removed query types + handlers
- `x/bvm/keeper/{grpc_query.go,migrator.go,msg_server.go}` — Pointer fixes
- `x/staking/keeper/{grpc_query.go,keeper.go,msg_server.go,tiers.go}` — Pointer fixes
- `x/compute_pool/{module.go,keeper/*.go}` — Pointer fixes
- `x/partnerships/keeper/{msg_server.go,grpc_query.go,*_test.go}` — Pointer fixes
- `x/qualification/{module.go,keeper/*.go}` — Pointer fixes
- `x/knowledge/keeper/{bvm_bridge.go,challenge.go,decay.go,expiration.go,grpc_query.go,msg_server.go,*_test.go}` — Pointer fixes
- `x/capture_defense/keeper/{msg_server.go,reputation.go}` — Pointer fixes
- `x/claiming_pot/keeper/{grpc_query.go,msg_server.go}` — Pointer fixes
- `x/ontology/keeper/query_support.go` — Pointer fixes
- `x/research/keeper/grpc_query.go` — Pointer fixes
- `x/ibcratelimit/keeper/msg_server.go` — UnimplementedMsgServer embedding
- `x/evidence_mgmt/{module.go,keeper/keeper_test.go}` — Pointer fixes
- `x/billing/keeper/keeper_test.go` — Pointer fix
