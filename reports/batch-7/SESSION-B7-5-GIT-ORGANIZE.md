# Session B7-5: GIT-COMMIT-ORGANIZE

**Goal**: Organize all uncommitted batch 1-7 changes into a clean, reviewable commit history.

## Final Commit List

| # | Hash | Message | Files |
|---|------|---------|-------|
| 1 | `d6f20d4` | `fix(app): add missing module account permissions` | `app/app.go` |
| 2 | `73dda01` | `fix(staking,gov): wire CLI commands and fix gRPC query paths` | `x/staking/module.go`, `x/gov/module.go`, `x/gov/client/cli/query.go` |
| 3 | `72d8cce` | `feat(tokens): add --mintable/--burnable/--pausable/--wrappable CLI flags` | `x/tokens/client/cli/tx.go` |
| 4 | `1383b12` | `feat(knowledge): proto-migrate ChallengeProvisionalFact + UpdateExtendedParams` | 7 files (proto, pb.go, types.go, msg_server.go) |
| 5 | `7b56665` | `fix(e2e): fix event type filters in test-lgm-modules.sh` | `devnet/scripts/test-lgm-modules.sh` |
| 6 | `d569f5f` | `feat(genesis): add production testnet genesis ceremony` | `scripts/testnet-genesis.sh`, `scripts/testnet-genesis-config.json`, `docs/TESTNET-LAUNCH.md` |
| 7 | `4668649` | `chore(reports): batch-6 session reports + ping-pub explorer config` | 8 files (7 reports + 1 config) |
| 8 | `ce76b2a` | `chore(build): set binary version via ldflags` | `Makefile`, `deployment/docker/Dockerfile.cosmos` |
| 9 | `c2835a1` | `chore: add session prompts, batch-7 report, gitignore devnet-data` | 16 files (14 prompts + 1 report + .gitignore) |
| 10 | `628ae39` | `fix(app): add remaining module accounts from B7-1 maccPerms audit` | `app/app.go`, `app/gas.go` |

## Files Per Commit (Detail)

### Commit 1: Module account permissions
- `app/app.go` — knowledge gets `Burner`, lgm_gov gets module account with `Burner`

### Commit 2: CLI wiring + gRPC path fixes
- `x/staking/module.go` — GetTxCmd/GetQueryCmd return actual commands, RegisterCodec wired
- `x/gov/module.go` — GetTxCmd/GetQueryCmd return actual commands
- `x/gov/client/cli/query.go` — fix 4 gRPC paths: `lgm_gov` → `gov`

### Commit 3: Token CLI feature flags
- `x/tokens/client/cli/tx.go` — add `--mintable`/`--burnable`/`--pausable`/`--wrappable` + populate `Features`

### Commit 4: Knowledge proto migration + events
- `proto/legible/knowledge/v1/claim.proto` — add MsgChallengeProvisionalFact + MsgUpdateExtendedParams
- `proto/legible/knowledge/v1/tx.proto` — add 2 RPCs to Msg service
- `x/knowledge/types/claim.pb.go` — regenerated
- `x/knowledge/types/tx.pb.go` — regenerated
- `x/knowledge/types/tx_grpc.pb.go` — regenerated
- `x/knowledge/types/types.go` — remove hand-written structs replaced by proto
- `x/knowledge/keeper/msg_server.go` — add 2 missing events + UpdateExtendedParams JSON decode

### Commit 5: E2E event extraction fix
- `devnet/scripts/test-lgm-modules.sh` — fix 4 event type filters (A3, C2, D2, E2)

### Commit 6: Testnet genesis ceremony
- `scripts/testnet-genesis.sh` (850 lines) — full ceremony script
- `scripts/testnet-genesis-config.json` (744 lines) — 410+ params across 28 modules
- `docs/TESTNET-LAUNCH.md` (222 lines) — launch guide

### Commit 7: Reports + explorer config
- `reports/batch-6/SESSION-B6-{1-7}*.md` — 7 session reports
- `deployment/docker/ping-pub-config/lgm-devnet.json` — Ping.pub explorer config

### Commit 8: Binary version ldflags
- `Makefile` — add VERSION/COMMIT/BUILD_TAGS ldflags to build target
- `deployment/docker/Dockerfile.cosmos` — add ldflags to Docker build

### Commit 9: Prompts + housekeeping
- `prompts/batch-6/*.md` — 7 session prompt files
- `prompts/batch-7/*.md` — 7 session prompt files
- `reports/batch-7/SESSION-B7-2-BINARY-VERSION.md` — binary version report
- `.gitignore` — exclude `deployment/docker/.devnet-data/`

### Commit 10: Remaining maccPerms (B7-1 audit)
- `app/app.go` — 11 module accounts: bvm, discovery, ontology, research, schedule + 6 treasury
- `app/gas.go` — treasury module account name constants

## Build Verification

```
go build ./...     ✅ clean
go vet ./...       ✅ clean
go test ./x/...    ✅ 35 packages, all OK, 0 failures
```

## Intentionally Left Uncommitted

| Path | Reason |
|------|--------|
| `deployment/docker/.devnet-data/` | Ephemeral devnet validator data — gitignored |

## Working Tree

```
nothing to commit, working tree clean
```
