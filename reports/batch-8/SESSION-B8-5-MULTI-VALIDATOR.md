# Session B8-5: Multi-Validator Docker Hardening

## Summary

Hardened the 3-validator Docker devnet for testnet use: Prometheus metrics exposure, monitoring stack, graceful shutdown, and improved health checks.

## Changes

### 1. Dockerfile.cosmos

| Change | Before | After |
|--------|--------|-------|
| EXPOSE ports | 26656 26657 9090 1317 | + 26660 (Prometheus) |
| HEALTHCHECK | `curl -sf .../status` | + `grep -q '"catching_up"'` (verifies response contains sync info) |
| STOPSIGNAL | (default SIGTERM) | Explicit `STOPSIGNAL SIGTERM` (documents intent) |

### 2. docker-compose.cosmos.yml

| Change | Detail |
|--------|--------|
| Prometheus ports | val1→26660, val2→26661, val3→26662 |
| `stop_grace_period` | 30s on all 3 validators |
| Prometheus service | `prom/prometheus:v2.51.0` on `:9093` (monitoring profile) |
| Grafana service | `grafana/grafana:10.4.0` on `:3000` (monitoring profile) |
| Named volumes | + `prometheus-data`, `grafana-data` |
| Usage docs | Updated header with profile docs + `down` vs `down -v` |

### 3. prometheus.yml (NEW)

Prometheus scrape config targeting all 3 validators on port 26660, with `cometbft` and `cosmos-sdk` job names.

### 4. entrypoint-cosmos.sh

| Change | Detail |
|--------|--------|
| `prometheus_listen_addr` | Explicitly set to `:26660` (ensures 0.0.0.0 binding) |
| Start log line | Now includes `metrics=:26660` |

### 5. init-cosmos-docker.sh

| Change | Detail |
|--------|--------|
| `prometheus_listen_addr` | Added to per-node config patching |

## Persistent Volumes (Already Working)

Named volumes were already in place from B6-4:
- `validator-{1,2,3}-data` → `/root/.legbled` (entire home dir)
- Data survives `docker compose down` (without `-v`)
- Config lives inside volume, patched by entrypoint on each start

## Peer Discovery (Already Working)

- `persistent_peers` configured per-node by init script (excludes self)
- Entrypoint supports `SEEDS` and `PERSISTENT_PEERS` env vars for runtime override
- Docker container names (`lgm-validator-{1,2,3}`) resolve via `lgm-net` bridge network

## Graceful Shutdown

- `STOPSIGNAL SIGTERM` explicit in Dockerfile
- `stop_grace_period: 30s` in compose gives CometBFT time to flush WAL and close LevelDB
- `exec` in entrypoint ensures legbled is PID 1 (receives SIGTERM directly)

## Monitoring Stack

Start with monitoring:
```bash
docker compose -f deployment/docker/docker-compose.cosmos.yml --profile monitoring up -d
```

| Service | URL | Credentials |
|---------|-----|-------------|
| Prometheus | http://localhost:9093 | — |
| Grafana | http://localhost:3000 | admin / lgmdevnet |
| Validator 1 metrics | http://localhost:26660/metrics | — |
| Validator 2 metrics | http://localhost:26661/metrics | — |
| Validator 3 metrics | http://localhost:26662/metrics | — |

Key CometBFT metrics:
- `cometbft_consensus_height` — current block height
- `cometbft_consensus_rounds` — rounds per height (should be 0-1)
- `cometbft_consensus_validators` — active validator count
- `cometbft_p2p_peers` — connected peers (should be 2 for each)
- `cometbft_mempool_size` — pending txs

## Port Map (Complete)

| Port | Service | Validator |
|------|---------|-----------|
| 26656 / 26666 / 26676 | P2P | 1 / 2 / 3 |
| 26657 / 26667 / 26677 | RPC | 1 / 2 / 3 |
| 1317 / 1318 / 1319 | REST API | 1 / 2 / 3 |
| 9090 / 9091 / 9092 | gRPC | 1 / 2 / 3 |
| 26660 / 26661 / 26662 | Prometheus | 1 / 2 / 3 |
| 9093 | Prometheus UI | monitoring profile |
| 3000 | Grafana | monitoring profile |
| 8080 | Ping.pub Explorer | explorer profile |

## Build Verification

```
go build ./...   ✅ clean
go vet ./...     ✅ clean
docker compose config --quiet  ✅ valid
docker build --check           ✅ no warnings
```

## Files Modified

- `deployment/docker/Dockerfile.cosmos` — EXPOSE 26660, HEALTHCHECK upgrade, STOPSIGNAL
- `deployment/docker/docker-compose.cosmos.yml` — Prometheus ports, stop_grace_period, monitoring stack
- `deployment/docker/entrypoint-cosmos.sh` — prometheus_listen_addr, log line
- `deployment/docker/init-cosmos-docker.sh` — prometheus_listen_addr in init

## Files Created

- `deployment/docker/prometheus.yml` — Prometheus scrape config
- `reports/batch-8/SESSION-B8-5-MULTI-VALIDATOR.md` — this report
