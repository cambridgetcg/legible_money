# CLI-V-01: Build & Root Structure

**Date**: 2026-02-18
**Binary**: `legbled`
**Source**: `cmd/legbled/cmd/root.go`

## Build Result

**PASS** — clean build, zero warnings.

```
$ go build -o /tmp/legbled ./cmd/legbled
(exit 0, no output)
```

- Go version: 1.26.0 darwin/arm64
- Binary size: ~133 MB

## Registered Subcommands

### Root Commands

| Command | Source | Description |
|---------|--------|-------------|
| `init` | `genutilcli.InitCmd` | Initialize node config/genesis |
| `start` | `server.AddCommands` | Run full node |
| `debug` | `debug.Cmd` | Debugging tools |
| `keys` | `keys.Commands` | Key management |
| `query` (alias: `q`) | Custom `&cobra.Command` | Query router |
| `tx` | Custom `&cobra.Command` | Transaction router |
| `comet` | `server.AddCommands` | CometBFT subcommands |
| `export` | `server.AddCommands` | Export state to JSON |
| `rollback` | `server.AddCommands` | Rollback 1 height |
| `version` | `server.AddCommands` | Print version |
| `collect-gentxs` | `genutilcli.CollectGenTxsCmd` | Collect genesis txs |
| `gentx` | `genutilcli.GenTxCmd` | Generate genesis self-delegation tx |
| `validate` | `genutilcli.ValidateGenesisCmd` | Validate genesis.json |
| `add-genesis-account` | `AddGenesisAccountCmd` (custom) | Add account to genesis |
| `migrate-genesis` | `MigrateGenesisCmd` (custom) | Migrate TS chain state to Cosmos genesis |
| `module-hash-by-height` | `server.AddCommands` | Module hashes at height |
| `completion` | cobra built-in | Shell autocomplete |
| `help` | cobra built-in | Help |

### Query Subcommands (`legbled query`)

**SDK built-in:**
- `block`, `block-results`, `blocks` — block queries
- `tx`, `txs`, `wait-tx` — transaction queries

**Legible custom modules (25):**

| CLI Name | Import Alias | Source Package |
|----------|--------------|----------------|
| `autopoiesis` | `autopoiesiscli` | `x/autopoiesis/client/cli` |
| `billing` | `billingcli` | `x/billing/client/cli` |
| `bvm` | `bvmcli` | `x/bvm/client/cli` |
| `capture_challenge` | `capturechallengecli` | `x/capture_challenge/client/cli` |
| `capture_defense` | `capturedefensecli` | `x/capture_defense/client/cli` |
| `channels` | `channelscli` | `x/channels/client/cli` |
| `claiming_pot` | `claimingpotcli` | `x/claiming_pot/client/cli` |
| `compute_pool` | `computepoolcli` | `x/compute_pool/client/cli` |
| `discovery` | `discoverycli` | `x/discovery/client/cli` |
| `disputes` | `disputescli` | `x/disputes/client/cli` |
| `emergency` | `emergencycli` | `x/emergency/client/cli` |
| `evidence_mgmt` | `evidencemgmtcli` | `x/evidence_mgmt/client/cli` |
| `ibcratelimit` | `ibcratelimitcli` | `x/ibcratelimit/client/cli` |
| `icaauth` | `icaauthcli` | `x/icaauth/client/cli` |
| `knowledge` | `knowledgecli` | `x/knowledge/client/cli` |
| `legible_auth` | `legibleauthcli` | `x/auth/client/cli` |
| `legible_staking` | `legiblestakingcli` | `x/staking/client/cli` |
| `ontology` | `ontologycli` | `x/ontology/client/cli` |
| `partnerships` | `partnershipscli` | `x/partnerships/client/cli` |
| `qualification` | `qualificationcli` | `x/qualification/client/cli` |
| `research` | `researchcli` | `x/research/client/cli` |
| `schedule` | `schedulecli` | `x/schedule/client/cli` |
| `tokens` | `tokenscli` | `x/tokens/client/cli` |
| `tree` | `treecli` | `x/tree/client/cli` |
| `vesting_rewards` | `vestingrewardscli` | `x/vesting_rewards/client/cli` |

### Transaction Subcommands (`legbled tx`)

**SDK built-in:**
- `bank` — `bankcli.NewTxCmd` (SDK bank module)
- `staking` — `stakingcli.NewTxCmd` (SDK staking module)
- `gov` — `govcli.NewTxCmd` (SDK governance module)
- `sign`, `sign-batch`, `multi-sign`, `multisign-batch` — signing
- `broadcast`, `encode`, `decode`, `simulate` — tx utilities
- `validate-signatures` — signature validation

**Legible custom modules (26):**

Same 25 as query, plus:
- `home` — `homecli.NewTxCmd` (10 commands: create-home, update-home, update-memory, configure-guardian, register-key, revoke-key, start-session, end-session, set-spending-limit, ack-alert)

## Modules in x/

```
alignment          autopoiesis        billing
bvm                capture_challenge   capture_defense
channels           claiming_pot        compute_pool
discovery          disputes            emergency
evidence_mgmt      gov                 home
ibcratelimit       icaauth             knowledge
ontology           partnerships        qualification
research           schedule            staking
testutil           tokens              tree
vesting_rewards
```

**Total: 28 directories** (27 modules + 1 test utility)

## Coverage Matrix

| Module | Query CLI | TX CLI | Notes |
|--------|-----------|--------|-------|
| `alignment` | NO | NO | **No `client/cli/` directory at all** |
| `auth` | YES (`legible_auth`) | YES (`legible_auth`) | |
| `autopoiesis` | YES | YES | |
| `billing` | YES | YES | |
| `bvm` | YES | YES | |
| `capture_challenge` | YES | YES | |
| `capture_defense` | YES | YES | |
| `channels` | YES | YES | |
| `claiming_pot` | YES | YES | |
| `compute_pool` | YES | YES | |
| `discovery` | YES | YES | |
| `disputes` | YES | YES | |
| `emergency` | YES | YES | |
| `evidence_mgmt` | YES | YES | |
| `gov` | NO | NO (SDK `govcli` provides `tx gov`) | **No custom `client/cli/` directory; SDK gov CLI covers tx only** |
| `home` | NO | YES (10 commands) | **Missing query CLI — has `cli/tx.go` only** |
| `ibcratelimit` | YES | YES | |
| `icaauth` | YES | YES | |
| `knowledge` | YES | YES | |
| `ontology` | YES | YES | |
| `partnerships` | YES | YES | |
| `qualification` | YES | YES | |
| `research` | YES | YES | |
| `schedule` | YES | YES | |
| `staking` | YES (`legible_staking`) | YES (`legible_staking`) | SDK staking CLI also present as `tx staking` |
| `testutil` | N/A | N/A | Test utility, not a runtime module |
| `tokens` | YES | YES | |
| `tree` | YES | YES | |
| `vesting_rewards` | YES | YES | |

**Summary**: 25/27 modules have both query + tx CLI. Two gaps below.

## How Subcommands Are Registered

Registration is centralized in `cmd/legbled/cmd/root.go:105-225` via `initRootCmd()`:

1. **Query commands** (root.go:130-163): `queryCmd.AddCommand(...)` explicitly lists each module's `NewQueryCmd()` plus SDK block/tx query helpers. Uses `addresscodec.NewBech32Codec("lgm")` and `"lgmvaloper"` for address codecs.

2. **TX commands** (root.go:174-214): `txCmd.AddCommand(...)` explicitly lists each module's `NewTxCmd()` plus SDK auth signing, bank, staking, and gov CLIs. Uses same address codecs.

3. **Genesis commands** (root.go:218-224): `collect-gentxs`, `gentx`, `validate`, `add-genesis-account`, `migrate-genesis`.

4. **Server commands** (root.go:111): `server.AddCommands(rootCmd, ...)` adds `start`, `export`, `rollback`, `comet`, `version`, `module-hash-by-height`.

Pattern: every module with CLI needs an explicit import + `AddCommand` call — no automatic discovery.

## Issues Found

### Issue 1: `alignment` module has NO CLI at all

- **Severity**: Medium
- **Details**: `x/alignment/` has no `client/cli/` directory. Not imported in `root.go`. Users cannot query alignment sensors, corrections, or meta-loop state, nor submit alignment-related transactions via CLI.
- **Impact**: Alignment module functionality is only accessible programmatically or via REST API (if gRPC/REST wiring exists).

### Issue 2: `home` module has TX CLI but NO query CLI

- **Severity**: Medium
- **Details**: `x/home/client/cli/tx.go` exists with 10 tx commands (create-home, update-home, configure-guardian, register-key, etc.), but there is no `query.go` file and `homecli.NewQueryCmd()` is not registered in `root.go`'s query command list.
- **Impact**: Users can create/modify homes via CLI but cannot query home state, guardian config, sessions, alerts, or keys via CLI. Must use REST API.

### Issue 3: `gov` module has NO custom CLI

- **Severity**: Low
- **Details**: `x/gov/` has no `client/cli/` directory. The SDK's `govcli.NewTxCmd(nil)` is registered for generic governance transactions, but there are no custom LIP-specific query or tx commands.
- **Impact**: LIP lifecycle (submit-lip, vote-lip, etc.) and quadratic voting are not exposed via dedicated CLI subcommands. SDK gov provides generic proposal/vote/deposit commands only.

### Observation: Naming inconsistency

- `x/auth/` registers as `legible_auth` in CLI
- `x/staking/` registers as `legible_staking` in CLI
- SDK's `x/staking` also present as `tx staking` (separate from `tx legible_staking`)
- SDK's `x/gov` present as `tx gov` (no custom `lgm_gov` equivalent)

This dual registration (SDK staking + custom legible_staking) could confuse users. No functional conflict since they wire to different keepers, but the naming overlap deserves documentation.
