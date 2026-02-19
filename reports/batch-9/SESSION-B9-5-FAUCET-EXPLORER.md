# Session B9-5: Testnet Faucet + Block Explorer Setup

## Summary

Built an HTTP faucet service for the LGM devnet and fixed the Ping.pub explorer config for browser access.

## New: Faucet Service

### `tools/faucet/main.go`

Simple Go HTTP server that wraps the `legbled` CLI to sign and broadcast `bank.MsgSend` transactions.

**Endpoints:**
| Method | Path | Description |
|--------|------|-------------|
| `GET /` | HTML form | Browser-friendly token request page |
| `POST /faucet` | JSON API | `{"address": "lgm1..."}` → `{"tx_hash": "..."}` |
| `GET /health` | Health check | Returns `ok` |

**Features:**
- Rate limiting: 1 request per address per hour (in-memory, auto-cleanup)
- Send mutex: serializes transactions to avoid sequence number conflicts
- CORS headers for cross-origin API use
- Rolls back rate limit on send failure (user can retry immediately)
- `FAUCET_MNEMONIC` env var: recovers key at startup (standalone mode)
- Falls back to existing keyring if no mnemonic provided (shared volume mode)
- `extractJSON()` handles `--gas auto` stderr mixed with JSON output

**Configuration (environment variables):**
| Var | Default | Description |
|-----|---------|-------------|
| `FAUCET_KEY_NAME` | `faucet` | Key name in keyring |
| `FAUCET_AMOUNT` | `10000000ulgm` | 10 LGM per request |
| `CHAIN_ID` | `lgm-devnet-1` | Chain ID |
| `NODE_URL` | `http://localhost:26657` | Node RPC endpoint |
| `KEYRING_DIR` | `/root/.legbled` | Keyring home directory |
| `LISTEN_ADDR` | `:8088` | HTTP listen address |
| `FAUCET_MNEMONIC` | (none) | Optional: recover key at startup |

### `tools/faucet/Dockerfile`

Multi-stage build:
1. Builds faucet Go binary with `CGO_ENABLED=0`
2. Uses `legbled:latest` as runtime base (provides the `legbled` CLI for signing)
3. Health check on `:8088/health`

### Docker Compose Integration

Added `faucet` service under `--profile faucet`:
```bash
# Start network + faucet
docker compose -f deployment/docker/docker-compose.cosmos.yml \
  --profile faucet up -d

# Start everything (faucet + explorer + monitoring)
docker compose -f deployment/docker/docker-compose.cosmos.yml \
  --profile faucet --profile explorer --profile monitoring up -d
```

The faucet mounts `validator-1-data` (read-only) to access the keyring with the pre-funded faucet key created by `init-cosmos-docker.sh`.

## Fix: Ping.pub Explorer Config

**Problem:** `lgm-devnet.json` used Docker-internal hostnames (`http://lgm-validator-1:1317`) for API/RPC/gRPC endpoints. Ping.pub is a client-side app — the browser makes the API calls, not the Docker container. Docker hostnames are unreachable from the browser.

**Fix:** Changed to `http://localhost:1317` / `http://localhost:26657` / `http://localhost:9090`.

## Service Port Map

| Port | Service | Profile |
|------|---------|---------|
| 8080 | Ping.pub Explorer | `explorer` |
| 8088 | LGM Faucet | `faucet` |
| 9093 | Prometheus | `monitoring` |
| 3000 | Grafana | `monitoring` |

## Build Verification

```
go build ./...   ✅ clean (includes tools/faucet/)
go vet ./...     ✅ clean
docker compose config --quiet  ✅ valid
```

## Files Created

- `tools/faucet/main.go` — faucet HTTP server (pure Go, wraps legbled CLI)
- `tools/faucet/Dockerfile` — multi-stage Docker build
- `reports/batch-9/SESSION-B9-5-FAUCET-EXPLORER.md` — this report

## Files Modified

- `deployment/docker/docker-compose.cosmos.yml` — added faucet service + profile docs
- `deployment/docker/ping-pub-config/lgm-devnet.json` — fixed URLs for browser access
