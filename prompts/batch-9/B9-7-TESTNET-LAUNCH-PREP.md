# B9-7: Testnet Launch Preparation — Final Checklist

## Context
This is the final session before testnet goes live. Everything from batches 1-9 should be integrated. This session does the final verification and prepares launch artifacts.

## Task

### 1. Full build + test
```bash
make build
go vet ./...
go test ./... -count=1 -timeout 600s
```
Record: exact test count, any warnings, build time.

### 2. Git status
```bash
git status  # must be clean
git log --oneline -20  # review recent history
```

### 3. Version tag
If B9-1 resulted in changes after v0.1.0-alpha:
```bash
git tag -a v0.1.0-alpha.2 -m "Pre-testnet launch"
```

### 4. Genesis generation
Run the full ceremony for the testnet genesis:
```bash
bash scripts/testnet-genesis.sh init
bash scripts/testnet-genesis.sh add-validator val1
# Inject genesis axioms (once we have them in final form)
bash scripts/testnet-genesis.sh finalize
```
Validate the genesis file.

### 5. Docker image
```bash
docker build -f deployment/docker/Dockerfile.cosmos \
  --build-arg VERSION=$(git describe --tags) \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  -t legible/legbled:testnet .
```

### 6. Seed node configuration
- Document the seed node's address and peer ID
- Create a `seeds.txt` file for distribution
- Verify persistent peer configuration

### 7. Create testnet launch announcement
`docs/TESTNET-ANNOUNCEMENT.md`:
- What is Legible Money (1 paragraph)
- What the testnet includes
- How to join as a validator
- How to get test tokens (faucet)
- How to submit your first knowledge claim
- Links: explorer, faucet, docs, Discord/community

### 8. Launch readiness matrix

| Requirement | Status | Notes |
|-------------|--------|-------|
| Binary builds clean | | |
| All tests pass | | |
| Genesis validates | | |
| Docker image works | | |
| REST API responds | | |
| E2E tests pass | | |
| Validator can join | | |
| Faucet works | | |
| Explorer works | | |
| Docs complete | | |
| Security audit done | | |
| Monitoring works | | |

Fill in each row with PASS/FAIL/PARTIAL.

## File Ownership
- `docs/TESTNET-ANNOUNCEMENT.md` (new)
- Git tags
- Docker images

## Report
Write `reports/batch-9/SESSION-B9-7-LAUNCH-PREP.md`.
