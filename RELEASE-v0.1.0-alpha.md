# Legible Money v0.1.0-alpha — Deterministic Testnet

**Release Date**: 2026-02-19
**Tag**: `v0.1.0-alpha`
**Commit**: `c868cbe`
**Go**: 1.26.0
**Cosmos SDK**: v0.50.15 / CometBFT v0.38 / IBC-Go v8.5.1

---

## What's Included

### Core Chain
- **29 custom modules** built on Cosmos SDK v0.50
- **Proof of Truth (PoT) consensus** — deterministic verification path for protocol, computational, formal, and analytic claims
- **ABCI++ vote extensions** with VRF-based validator selection (scaffolded for arbiter path)
- **4-tier validator system**: Apprentice → Verified → Bonded → Guardian
- **Knowledge lifecycle**: claim submission → verification round → commit-reveal → fact creation
- **Dual-key authentication** with DID-based identity (`did:lgm:{hex}`)

### CLI
- **47 transaction module groups** and **42 query module groups**
- Full module coverage: knowledge, staking, governance (LIPs), tokens, home, BVM, emergency, channels, ontology, disputes, evidence, research, tree, and more
- Binary version metadata via ldflags

### Testing
- **2583 passing tests** across 29 modules + app-level tests
- Zero `go vet` warnings
- OpenClaw adversarial probes (macro game theory, precision attacks, zero-cost attacks)

### Infrastructure
- **Docker multi-validator support**: 3-node devnet via docker compose
- **Prometheus + Grafana** monitoring (optional `--profile monitoring`)
- Production genesis ceremony with chain ID `legible-devnet-1`
- Entrypoint with graceful SIGTERM shutdown
- Genesis axioms scaffolding (777 foundational knowledge claims)

### Documentation
- 29 module reference docs (`docs/modules/`)
- Validator operator guide (`docs/VALIDATOR-GUIDE.md`)
- Blockchain architecture docs
- CHANGELOG

---

## What's Not Included

| Feature | Status | Notes |
|---------|--------|-------|
| Arbiter path (non-deterministic claims) | Designed, not wired | Vote extensions filter on `RoundTypeChallenge` but no round creation sets this type — see `reports/batch-7/SESSION-B7-4-ARBITER-POT.md` |
| gRPC gateway (REST API) | Disabled | `tendermint/v1beta1` returns 501 (SDK v0.50 deprecation); custom module REST stubs not yet generated |
| `MsgRegisterValidator` | Blocked | `ValidatorDescription` proto codec registration issue — nested proto v2 type not in registry |
| IBC transfers | Untested | IBC modules wired but no relayer testing done |
| Tokens module handlers | Stub | 14 msg handlers defined but unimplemented |

---

## How to Run

### Single Validator (Local)

```bash
# Build
make build

# Initialize
./build/legbled init my-node --chain-id legible-devnet-1
./build/legbled keys add validator

# Configure genesis (see scripts/genesis-ceremony.sh for full ceremony)
./build/legbled genesis add-genesis-account $(./build/legbled keys show validator -a) 100000000000ulgm

# Start
./build/legbled start --minimum-gas-prices 0.001ulgm
```

### Docker Multi-Validator (3-Node Devnet)

```bash
# Build image
docker build -f deployment/docker/Dockerfile.cosmos \
  --build-arg VERSION=v0.1.0-alpha \
  -t legible/legbled:v0.1.0-alpha .

# Initialize 3 validators
bash deployment/docker/init-cosmos-docker.sh

# Start devnet
docker compose -f deployment/docker/docker-compose.cosmos.yml up -d

# Optional: with monitoring
docker compose -f deployment/docker/docker-compose.cosmos.yml --profile monitoring up -d

# Check status
curl -s localhost:26657/status | jq .result.sync_info.latest_block_height

# Tear down
docker compose -f deployment/docker/docker-compose.cosmos.yml down -v
```

### Submit a Claim

```bash
legbled tx knowledge submit-claim \
  "Water boils at 100 degrees Celsius at standard atmospheric pressure" \
  --domain science --category empirical \
  --from validator --chain-id legible-devnet-1 \
  --gas auto --gas-adjustment 1.3 -y
```

---

## Known Issues

1. **E2E race conditions**: Rapid-fire transaction sequences can fail due to nonce/sequence race conditions in test scripts
2. **`legible_staking register-validator`**: TX parsing fails on nested `ValidatorDescription` proto type (P1)
3. **SDK v0.50 REST deprecation**: `/cosmos/base/tendermint/v1beta1/*` endpoints return 501 — use CometBFT RPC directly at `:26657`
4. **Secondary index WIP**: Fact/claim secondary indexes (B8-6) are in progress — query performance for large datasets will improve in v0.1.1

---

## Build Verification

| Check | Result |
|-------|--------|
| `make build` | PASS — clean build, 0 errors |
| `go vet ./...` | PASS — 0 warnings |
| `go test ./... -count=1` | PASS — 2583 tests, 0 failures |
| `legbled version` | `v0.1.0-alpha` |
| Docker image | Built successfully |
| 3-validator consensus | Verified (B6-4) |
| Cross-validator TX | Verified (B6-4) |

---

## Stats

| Metric | Value |
|--------|-------|
| Go source files | 770 |
| Test files | 130 |
| Total Go lines | ~318K |
| Commits | 383 |
| Custom modules | 29 |
| Tests passing | 2583 |
