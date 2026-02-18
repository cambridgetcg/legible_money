# GOV-CLI: Custom Governance Module CLI

**Date**: 2026-02-18
**Module**: `x/gov` (ModuleName = `lgm_gov`)
**CLI namespace**: `legbled tx lgm_gov`, `legbled query lgm_gov`

---

## Handler Audit

### Proto-defined RPCs (tx.proto: 25 RPCs)

| # | RPC | Group | Implemented? | CLI created? |
|---|-----|-------|-------------|-------------|
| 1 | `SubmitLIP` | LIP lifecycle | YES | YES |
| 2 | `StakeLIP` | LIP lifecycle | YES | YES |
| 3 | `AmendLIP` | LIP lifecycle | NO (Unimplemented) | NO |
| 4 | `AdvanceLIPStage` | LIP lifecycle | YES | YES |
| 5 | `WithdrawLIP` | LIP lifecycle | YES | YES |
| 6 | `CastVote` | Voting | YES | YES |
| 7 | `LockVote` | Voting | NO (Unimplemented) | NO |
| 8 | `UnlockVote` | Voting | NO (Unimplemented) | NO |
| 9 | `SwitchVote` | Voting | NO (Unimplemented) | NO |
| 10 | `CommitReview` | Review | NO (Unimplemented) | NO |
| 11 | `RevealReview` | Review | NO (Unimplemented) | NO |
| 12 | `FinalizeReview` | Review | NO (Unimplemented) | NO |
| 13 | `SubmitDisbursement` | Disbursement | YES | YES |
| 14 | `StakeDisbursement` | Disbursement | NO (Unimplemented) | NO |
| 15 | `AdvanceDisbursement` | Disbursement | NO (Unimplemented) | NO |
| 16 | `ExecuteDisbursement` | Disbursement | YES | YES |
| 17 | `CancelDisbursement` | Disbursement | YES | YES |
| 18 | `RegisterOperator` | Operator | NO (Unimplemented) | NO |
| 19 | `AddAgent` | Operator | NO (Unimplemented) | NO |
| 20 | `RemoveAgent` | Operator | NO (Unimplemented) | NO |
| 21 | `SlashOperator` | Operator | NO (Unimplemented) | NO |
| 22 | `CreateDeployment` | Deployment | NO (Unimplemented) | NO |
| 23 | `AdvanceDeployment` | Deployment | NO (Unimplemented) | NO |
| 24 | `ApproveDeployment` | Deployment | NO (Unimplemented) | NO |
| 25 | `RollbackDeployment` | Deployment | NO (Unimplemented) | NO |

### Hand-written Msg types (not in proto)

| # | Msg | Implemented? | CLI created? |
|---|-----|-------------|-------------|
| 26 | `MsgUpdateParams` | YES | NO (authority-only) |
| 27 | `MsgAttachUpgradePlan` | YES | YES |

### Query RPCs (hand-written, no query.proto)

| # | Query | Implemented? | CLI created? |
|---|-------|-------------|-------------|
| 1 | `LIP` | YES | YES |
| 2 | `LIPs` | YES | YES |
| 3 | `Disbursement` | YES | YES |
| 4 | `Params` | YES | YES |

### Summary

- **10 implemented Msg handlers** (8 proto + 2 hand-written)
- **15 unimplemented proto RPCs** (fall through to `UnimplementedMsgServer`)
- **4 implemented Query handlers** (all hand-written, no query.proto)
- **9 TX CLI commands created** (all implemented handlers except UpdateParams which is authority-only)
- **4 Query CLI commands created** (all 4 query handlers)

---

## Commands Created

### TX Commands (9)

| Command | Msg type | Args | Flags |
|---------|----------|------|-------|
| `submit-lip` | `MsgSubmitLIP` | title, description, initial-stake | `--lip-type`, `--specification` |
| `stake-lip` | `MsgStakeLIP` | lip-id, amount | — |
| `advance-lip-stage` | `MsgAdvanceLIPStage` | lip-id | — |
| `withdraw-lip` | `MsgWithdrawLIP` | lip-id | — |
| `attach-upgrade-plan` | `MsgAttachUpgradePlan` | lip-id, upgrade-name, height | `--info` |
| `cast-vote` | `MsgCastVote` | proposal-id, option, stake | — |
| `submit-disbursement` | `MsgSubmitDisbursement` | title, description, amount, recipient | `--justification` |
| `execute-disbursement` | `MsgExecuteDisbursement` | disbursement-id | — |
| `cancel-disbursement` | `MsgCancelDisbursement` | disbursement-id | `--reason` |

### Query Commands (4)

| Command | Query RPC | Args | Flags |
|---------|-----------|------|-------|
| `params` | `Params` | — | — |
| `lip` | `LIP` | lip-id | — |
| `lips` | `LIPs` | — | `--stage`, `--limit`, `--offset` |
| `disbursement` | `Disbursement` | disbursement-id | — |

---

## Commands NOT Created (and why)

### Unimplemented proto RPCs (15)

These RPCs are defined in `proto/legible/gov/v1/tx.proto` but their handlers in `x/gov/keeper/msg_server.go` fall through to `types.UnimplementedMsgServer`, returning gRPC `Unimplemented` errors. Creating CLI commands for them would give users a non-functional interface.

**Voting group (3)**: `LockVote`, `UnlockVote`, `SwitchVote` — conviction voting features not yet implemented.

**Review group (3)**: `CommitReview`, `RevealReview`, `FinalizeReview` — code/spec review commit-reveal scheme not yet implemented.

**LIP lifecycle (1)**: `AmendLIP` — LIP amendment not yet implemented.

**Disbursement (2)**: `StakeDisbursement`, `AdvanceDisbursement` — disbursement staking and manual advancement not yet implemented. (Disbursements go directly to voting stage on submission.)

**Operator registry (4)**: `RegisterOperator`, `AddAgent`, `RemoveAgent`, `SlashOperator` — entire operator subsystem not yet implemented.

**Deployment (4)**: `CreateDeployment`, `AdvanceDeployment`, `ApproveDeployment`, `RollbackDeployment` — entire deployment subsystem not yet implemented.

### Authority-only handler (1)

**`MsgUpdateParams`**: Not exposed as CLI command. Governance parameter updates are authority-only (module account). This matches the convention in other modules (`x/tokens`, `x/tree`, etc.) where `update-params` is not exposed as a user-facing CLI subcommand.

---

## Module Name Resolution

- `types.ModuleName = "lgm_gov"` (from `x/gov/types/keys.go:4`)
- CLI registers as `tx lgm_gov` and `query lgm_gov`
- SDK governance remains as `tx gov` (via `govcli.NewTxCmd(nil)` in root.go)
- **No collision** — the two governance modules use distinct CLI namespaces
- Import alias in root.go: `lgmgovcli "github.com/legible-money/legible/x/gov/client/cli"`

---

## Files Created/Modified

| File | Action |
|------|--------|
| `x/gov/client/cli/tx.go` | **Created** — 9 tx commands |
| `x/gov/client/cli/query.go` | **Created** — 4 query commands |
| `cmd/legbled/cmd/root.go` | **Modified** — added import + query/tx registration |

---

## Build Output

```
$ go build -o /tmp/legbled ./cmd/legbled
(exit 0, no output)
```

**PASS** — clean build, zero errors.

Note: `go build ./x/gov/...` also passes independently. Full binary build required temporarily stashing unrelated pre-existing knowledge CLI WIP (undefined function references in `x/knowledge/client/cli/query.go` from a prior session).

---

## CLI Help Outputs

### `legbled tx lgm_gov --help`

```
Legible governance module transaction commands

Usage:
  legbled tx lgm_gov [flags]
  legbled tx lgm_gov [command]

Available Commands:
  advance-lip-stage    Advance a LIP to its next stage
  attach-upgrade-plan  Attach a software upgrade plan to a consensus-category LIP
  cancel-disbursement  Cancel a disbursement proposal
  cast-vote            Cast a quadratic vote on a LIP or disbursement
  execute-disbursement Execute an approved disbursement after timelock
  stake-lip            Stake tokens on a LIP to advance it from draft to review
  submit-disbursement  Submit a treasury disbursement proposal
  submit-lip           Submit a new Legible Improvement Proposal
  withdraw-lip         Withdraw a LIP (proposer only)
```

### `legbled query lgm_gov --help`

```
Legible governance module query commands

Usage:
  legbled query lgm_gov [flags]
  legbled query lgm_gov [command]

Available Commands:
  disbursement Query a disbursement by ID (includes votes)
  lip          Query a LIP by ID (includes votes)
  lips         List LIPs (optionally filtered by stage)
  params       Query the governance module parameters
```

---

## Related Issues Discovered (Not Fixed)

### 1. `addBigIntStrings` in module.go uses uint64 (P0, already tracked)

`x/gov/module.go:244-251`: `addBigIntStrings` parses big-int strings to `uint64` using `fmt.Sscanf`. This overflows for vote weights exceeding `2^64`. Already fixed in Session 14 for `msg_server.go`'s `addBigInts` (which uses `math/big`), but the duplicate in `module.go` still uses uint64. Same issue applies to `bpsFraction` at line 253-259 (`n * 10000` can overflow).

### 2. `parseBigIntString` silently returns 0 on parse failure

`x/gov/module.go:262-268`: Uses `fmt.Sscanf` which silently returns 0 on parse failure. Should use `big.Int.SetString` with ok check, consistent with `msg_server.go`.

### 3. `MsgUpdateParams` and `MsgAttachUpgradePlan` missing from `RegisterCodec`

`x/gov/types/codec.go:13-24`: `RegisterCodec` registers 10 types including `MsgUpdateParams` and `MsgAttachUpgradePlan`. However, `RegisterInterfaces` (line 27-55) registers 25 proto types but **omits** `MsgUpdateParams` and `MsgAttachUpgradePlan` (the two hand-written Msg types). These should be added to `RegisterInterfaces` for consistency with the pattern used in `x/staking/types/codec.go`.

### 4. Pre-existing knowledge CLI build failure

`x/knowledge/client/cli/query.go` references 11 undefined functions (`NewQueryDomainsCmd`, `NewQueryDomainsByStratumCmd`, etc.) added in a prior session without their implementations. This prevents `go build ./...` from succeeding.

### 5. No query.proto for governance

The governance module has 4 query RPCs implemented with hand-written gRPC service descriptors (`x/gov/types/codec.go:60-131`) and no `proto/legible/gov/v1/query.proto`. While functional, this means the queries don't have proto-generated client code and the gRPC service name uses `legible.lgm_gov.v1.Query` (matching the hand-written descriptor). Creating a `query.proto` would bring this in line with other modules.
