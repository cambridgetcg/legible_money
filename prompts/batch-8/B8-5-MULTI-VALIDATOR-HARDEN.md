# B8-5: Multi-Validator Docker Hardening

## Context
B6-4 verified Docker 3-validator devnet works. Need to harden for testnet use: persistent volumes, proper peer discovery, monitoring endpoints, graceful shutdown.

## Task

### 1. Read current Docker setup
- `deployment/docker/Dockerfile.cosmos`
- `deployment/docker/docker-compose*.yml` (all variants)
- `deployment/docker/scripts/` (any init scripts)

### 2. Add persistent volumes
Ensure validator data survives container restarts:
- `/root/.legbled/data` → named volume per validator
- `/root/.legbled/config` → bind mount for config overrides

### 3. Peer discovery
- Configure persistent peers in `config.toml` template
- Add seed node support
- Ensure validators find each other on container restart

### 4. Health checks
Add Docker HEALTHCHECK to Dockerfile:
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD legbled status 2>&1 | grep -q '"catching_up":false' || exit 1
```

### 5. Monitoring
- Expose Prometheus metrics (port 26660)
- Add Prometheus + Grafana to docker-compose (optional service)
- Verify `legbled` serves `/metrics` endpoint

### 6. Graceful shutdown
- Ensure SIGTERM is properly handled
- Add `stop_grace_period: 30s` to compose

### 7. Test
- `docker compose up -d` — all 3 validators start
- `docker compose restart val1` — val1 recovers and catches up
- `docker compose down && docker compose up -d` — full restart, state persists
- Submit tx, verify consensus across all 3

## File Ownership
- `deployment/docker/Dockerfile.cosmos`
- `deployment/docker/docker-compose*.yml`
- `deployment/docker/config/` (templates)

## Report
Write `reports/batch-8/SESSION-B8-5-MULTI-VALIDATOR.md`.
