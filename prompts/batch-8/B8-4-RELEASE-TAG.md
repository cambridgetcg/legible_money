# B8-4: Release Tagging + CI Verification

## Context
We're preparing to tag v0.1.0-alpha for the deterministic testnet launch. All P0 blockers are resolved. Need clean tag, verified build, and release artifacts.

## Task

### 1. Pre-tag verification
Run the full verification suite:
```bash
make build          # clean build with ldflags
go vet ./...        # zero warnings
go test ./... -count=1 -timeout 600s  # all tests pass
./build/legbled version --long        # version metadata correct
```

### 2. Verify git is clean
```bash
git status          # nothing uncommitted
git log --oneline -15  # review recent history
```

If there are uncommitted changes from other batch-8 sessions, commit them with appropriate messages first.

### 3. Tag
```bash
git tag -a v0.1.0-alpha -m "Legible Money v0.1.0-alpha: Deterministic Testnet

Features:
- 28 custom modules (knowledge, staking, gov, tokens, home, bvm, etc.)
- Proof of Truth consensus (deterministic path: protocol, computational, formal, analytic)
- Full CLI: 216 tx + 163 query commands
- 2800+ tests, all passing
- Production genesis ceremony
- Docker multi-validator support
- Binary version metadata via ldflags

Known limitations:
- Arbiter path (non-deterministic claims) not yet wired
- gRPC gateway disabled (CLI and gRPC direct work)
- E2E test race conditions in rapid-fire sequences"
```

### 4. Docker build verification
```bash
docker build -f deployment/docker/Dockerfile.cosmos \
  --build-arg VERSION=v0.1.0-alpha \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  -t legible/legbled:v0.1.0-alpha .
```

Verify the container:
```bash
docker run --rm legible/legbled:v0.1.0-alpha version --long
```

### 5. Create release notes
Generate `RELEASE-v0.1.0-alpha.md` summarizing:
- What's included
- What's not (arbiter path, gRPC gateway)
- How to run (single validator, Docker)
- Known issues

## File Ownership
- Git tags
- `RELEASE-v0.1.0-alpha.md` (new)
- Docker image build verification

## Report
Write `reports/batch-8/SESSION-B8-4-RELEASE-TAG.md`.
