# CLI-V-02: Deep CLI Wiring Verification (Modules A-L)

**Date**: 2026-02-18
**Binary**: `/tmp/legbled` (built from `go build -o /tmp/legbled ./cmd/legbled`)
**Scope**: 19 modules alphabetically A through L (alignment through knowledge)

---

## Executive Summary

All 19 modules were audited by comparing:
1. CLI subcommands (from `x/{module}/client/cli/tx.go` and `query.go`)
2. Proto service definitions (from `proto/legible/{module}/v1/tx.proto` and `query.proto`)
3. Binary `--help` output for each subcommand

**Key finding**: CLI commands are registered directly in `cmd/legbled/cmd/root.go` (lines 130-214), NOT through module.go's `GetTxCmd()/GetQueryCmd()`. Most modules return nil from these methods but the CLI still works because root.go imports cli packages directly.

### Totals

| Metric | Count |
|--------|-------|
| TX CLI commands | 87 |
| Query CLI commands | 82 |
| Proto Msg RPCs | 81 |
| Proto Query RPCs | 89 |
| **Total CLI surface** | **169 commands** |
| **Total Proto surface** | **170 RPCs** |

### Alignment Summary

| Status | Modules |
|--------|---------|
| Proto-CLI aligned | disputes, claiming_pot, ibcratelimit |
| CLI exceeds proto (hand-written types) | auth, billing, autopoiesis, bvm, capture_challenge, capture_defense, channels, compute_pool, discovery, emergency, icaauth |
| Proto exceeds CLI (missing CLI cmds) | knowledge, home, evidence_mgmt, gov |
| No CLI or proto | alignment |

---

## Architecture Note: CLI Registration Pattern

```
cmd/legbled/cmd/root.go
  |
  +-- queryCmd.AddCommand(
  |     legibleauthcli.NewQueryCmd(),    // line 138
  |     autopoiesiscli.NewQueryCmd(),    // line 139
  |     ...24 more modules...           // lines 140-163
  |   )
  |
  +-- txCmd.AddCommand(
        legibleauthcli.NewTxCmd(),       // line 188
        ...25 more modules...            // lines 189-213
      )
```

Module.go `GetTxCmd()`/`GetQueryCmd()` return nil for most modules (used for autocli, which is not enabled). This is NOT a bug -- root.go handles all CLI registration.

---

## Per-Module Results

### 1. alignment

| | CLI | Proto |
|---|---|---|
| TX | 0 | 0 |
| Query | 0 | 0 |

**Status**: STUB. No `client/cli/` directory. No `proto/legible/alignment/v1/` files. Module has keeper and types but no external interface. Not registered in root.go's command lists.

---

### 2. auth (legible_auth)

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 11 | 4 | +7 CLI |
| Query | 4 | 4 | aligned |

**TX CLI commands**: register-account, rotate-key, create-session, revoke-session, freeze-account, unfreeze-account, set-recovery-config, initiate-recovery, submit-recovery-shard, challenge-recovery, execute-recovery

**Proto Msg RPCs**: RotateKey, CreateSession, RevokeSession, UpdateParams

**CLI-only TX (hand-written types)**: register-account, freeze-account, unfreeze-account, set-recovery-config, initiate-recovery, submit-recovery-shard, challenge-recovery, execute-recovery

**Proto-only Msg (no CLI)**: UpdateParams (governance-only, intentional)

**Query alignment**: Perfect. CLI `account`, `account-by-did`, `session-keys`, `params` match proto `Account`, `AccountByDID`, `SessionKeys`, `Params`.

**Flag/type check**: All commands use standard `flags.AddTxFlagsToCmd`. Account addresses use bech32 encoding. Session key commands accept hex-encoded public keys.

---

### 3. autopoiesis

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 3 | 0 | +3 CLI |
| Query | 0 | 0 | aligned |

**TX CLI commands**: activate, override-multiplier, freeze-multiplier

**No proto files exist**. Module uses hand-written Msg types in `types/`. MsgServer registered in module.go but without proto service definition.

**Missing**: No query CLI and no query proto. Module has no read interface.

---

### 4. billing

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 4 | 0 | +4 CLI |
| Query | 3 | 3 | aligned |

**TX CLI commands**: register-provider, deregister-provider, request-quote, execute-payment

**No Msg service in proto**. TX commands use hand-written types.

**Query alignment**: Perfect. CLI `params`, `provider`, `quote` match proto `Params`, `Provider`, `Quote`.

---

### 5. bvm

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 4 | 3 | +1 CLI |
| Query | 6 | 6 | aligned |

**TX CLI commands**: deploy-contract, call-contract, schedule-contract, cancel-schedule

**Proto Msg RPCs**: DeployContract, CallContract, ScheduleExecution

**CLI-only TX**: `cancel-schedule` (hand-written type, no proto Msg)

**Query alignment**: Perfect. CLI `params`, `contract`, `contracts-by-creator`, `code`, `contract-state`, `schedule` match proto exactly.

---

### 6. capture_challenge

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 3 | 0 | +3 CLI |
| Query | 3 | 3 | aligned |

**TX CLI commands**: submit-challenge, add-evidence, resolve-challenge

**No Msg service in proto**. TX commands use hand-written types.

**Query alignment**: Perfect. CLI `challenge`, `active-challenges`, `bounty-pool` match proto `Challenge`, `ActiveChallenges`, `BountyPool`.

---

### 7. capture_defense

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 2 | 0 | +2 CLI |
| Query | 4 | 4 | aligned |

**TX CLI commands**: request-qualification, endorse-qualification

**No Msg service in proto**. TX commands use hand-written types.

**Query alignment**: Perfect. CLI `qualification`, `validator-qualifications`, `domain-reputation`, `capture-risk` match proto exactly.

---

### 8. channels

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 8 | 5 | +3 CLI |
| Query | 5 | 5 | aligned |

**TX CLI commands**: open-channel, update-channel, close-channel, settle-channel, dispute-channel, deposit-channel, update-state, claim-expired

**Proto Msg RPCs**: OpenChannel, UpdateChannel, CloseChannel, SettleChannel, DisputeChannel

**CLI-only TX (hand-written)**: deposit-channel, update-state, claim-expired

**Query alignment**: Perfect. CLI `channel`, `channels-by-payer`, `channels-by-provider`, `active-dispute`, `params` match proto exactly.

**Flag check**: Pagination flags (--limit, --offset) present on channels-by-payer and channels-by-provider.

---

### 9. claiming_pot

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 4 | 4 | aligned |
| Query | 6 | 6 | aligned |

**Full alignment**. CLI commands map 1:1 to proto RPCs.

- TX: create-pot/CreatePot, fund-pot/FundPot, claim-from-pot/ClaimFromPot, close-pot/ClosePot
- Query: pot/Pot, pots/Pots, pots-by-creator/PotsByCreator, eligibility/Eligibility, claim-records/ClaimRecords, params/Params

**Flag check**: Pots query has --status, --limit, --offset filters matching proto fields.

---

### 10. compute_pool

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 7 | 0 | +7 CLI |
| Query | 6 | 6 | aligned |

**TX CLI commands**: register-provider, update-provider, deregister-provider, heartbeat, redeem-credits, confirm-usage, dispute-usage

**No Msg service in proto**. All 7 TX commands use hand-written types.

**Query alignment**: Perfect. CLI `params`, `provider`, `active-providers`, `credit-balance`, `ticket`, `pool-stats` match proto exactly.

**Flag check**: active-providers has --service-type filter matching proto's service_type field.

---

### 11. discovery

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 6 | 0 | +6 CLI |
| Query | 4 | 4 | aligned |

**TX CLI commands**: register-agent, update-profile, add-capability, remove-capability, deregister-agent, heartbeat

**No Msg service in proto** (no tx.proto). All TX commands use hand-written types.

**Query alignment**: Perfect. CLI `params`, `agent`, `agents-by-domain`, `agents-by-capability` match proto exactly.

---

### 12. disputes

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 7 | 7 | aligned |
| Query | 6 | 6 | aligned |

**Full alignment**. CLI commands map 1:1 to proto RPCs.

- TX: initiate/InitiateDispute, commit-evidence/CommitEvidence, reveal-evidence/RevealEvidence, vote/ArbiterVote, escalate/EscalateDispute, settle/SettleDispute, timeout/TimeoutDispute
- Query: params/Params, dispute/Dispute, disputes-by-fact/DisputesByFact, active-disputes/ActiveDisputes, evidence/Evidence, votes/Votes

---

### 13. emergency

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 6 | 6 | aligned |
| Query | 4 | 0 | +4 CLI |

**TX alignment**: Perfect. CLI matches proto exactly: propose-halt/ProposeHalt, vote-halt/VoteHalt, propose-revert/ProposeRevert, vote-revert/VoteRevert, propose-resume/ProposeResume, vote-resume/VoteResume.

**Query CLI commands**: status, active-ceremony, completed-ceremonies, audit-log

**No Query service in proto**. Query commands use hand-written types. Query proto file exists but has no Query service definition.

**Flag check**: completed-ceremonies and audit-log have --limit, --offset pagination flags.

---

### 14. evidence_mgmt

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 4 | 5 | -1 CLI |
| Query | 8 | 8 | aligned |

**TX CLI commands**: submit-evidence, audit-evidence, register-oracle, submit-oracle-data

**Proto Msg RPCs**: SubmitEvidence, AuditEvidence, RegisterOracle, SubmitOracleData, UpdateParams

**Missing TX CLI**: `update-params` (governance-only, intentional omission)

**Query alignment**: Perfect. All 8 query commands match proto RPCs exactly.

---

### 15. gov

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 0 | 25 | -25 CLI |
| Query | 0 | 0 | aligned |

**No client/cli/ directory exists**.

**Proto Msg RPCs (25)**: SubmitLIP, StakeLIP, AmendLIP, AdvanceLIPStage, WithdrawLIP, CastVote, LockVote, UnlockVote, SwitchVote, CommitReview, RevealReview, FinalizeReview, SubmitDisbursement, StakeDisbursement, AdvanceDisbursement, ExecuteDisbursement, CancelDisbursement, RegisterOperator, AddAgent, RemoveAgent, SlashOperator, CreateDeployment, AdvanceDeployment, ApproveDeployment, RollbackDeployment

**Design note**: Custom governance is accessed via standard Cosmos `legbled tx gov` commands (submit-proposal, vote, etc.). Direct CLI for the 25 custom Msg types is not yet implemented. This is the largest CLI gap across all modules.

**No Query proto**: Query capabilities not yet defined.

---

### 16. home

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 10 | 10 | aligned |
| Query | 0 | 5 | -5 CLI |

**TX alignment**: Perfect. All 10 CLI commands match proto Msg RPCs: create-home/CreateHome, update-home/UpdateHome, update-memory/UpdateMemoryCID, configure-guardian/ConfigureGuardian, register-key/RegisterKey, revoke-key/RevokeKey, start-session/StartSession, end-session/EndSession, set-spending-limit/SetSpendingLimit, ack-alert/AcknowledgeAlert.

**Missing Query CLI**: No `query.go` file exists. Proto defines 5 Query RPCs: Home, Homes, Sessions, Alerts, Params. Not registered in root.go's queryCmd (line 163 — home is NOT in the query list).

**Action needed**: Create `x/home/client/cli/query.go` with 5 query commands and add `homecli.NewQueryCmd()` to root.go line ~163.

---

### 17. ibcratelimit

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 0 | 0 | aligned |
| Query | 2 | 2 | aligned |

**Full alignment**. Governance-only module (MsgUpdateParams via gov proposals). Query CLI `params`, `channel-flow` match proto `Params`, `ChannelFlow`.

---

### 18. icaauth

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 2 | 0 | +2 CLI |
| Query | 2 | 2 | aligned |

**TX CLI commands**: register (register-interchain-account), send-tx (send-ica-transaction)

**No Msg service in proto** (tx.proto exists but Msg service not defined). TX commands use hand-written types with `tx.GenerateOrBroadcastTxCLI`.

**Query alignment**: Perfect. CLI `accounts`, `params` match proto `InterchainAccounts`→`Accounts`, `Params`.

**Flag check**: register has --version and --allowed-msgs flags. send-tx has --timeout flag (uint64 nanoseconds, default 10 minutes).

---

### 19. knowledge

| | CLI | Proto | Delta |
|---|---|---|---|
| TX | 6 | 12 | -6 CLI |
| Query | 9 | 20 | -11 CLI |

**TX CLI commands**: submit-claim, submit-commitment, submit-reveal, challenge-fact, submit-contradiction, patronize-fact

**Missing TX CLI (6)**:
- `add-fact` (MsgAddFact) — used by validators to add verified facts
- `propose-domain` (MsgProposeDomain) — domain governance
- `endorse-domain-proposal` (MsgEndorseDomainProposal)
- `challenge-domain-proposal` (MsgChallengeDomainProposal)
- `register-stratum` (MsgRegisterStratum)
- `update-params` (MsgUpdateParams) — governance-only

**Query CLI commands**: params, fact, facts, facts-by-domain, claim, round, active-rounds, domain, counter-fact

**Missing Query CLI (11)**:
- `stratum` / `strata` / `stratum-registry` — ontology stratum queries
- `domains` / `domains-by-stratum` / `domain-children` / `domain-path` — domain hierarchy navigation
- `domain-proposal` / `domain-proposals` — domain governance queries
- `validate-dependency` — dependency validation
- `calculate-confidence` — confidence calculation

**Priority**: The core verification flow (submit-claim, submit-commitment, submit-reveal) works. Missing commands are primarily for domain/stratum governance and advanced queries.

---

## Gap Analysis

### High Priority Gaps (Functional Impact)

| Module | Gap | Impact |
|--------|-----|--------|
| **home** | 0/5 query commands | Cannot query agent homes, sessions, alerts via CLI |
| **knowledge** | 6/12 tx, 11/20 query missing | Domain governance and ontology navigation unavailable |
| **gov** | 0/25 tx commands | Custom governance only via generic gov proposals |

### Medium Priority (Hand-Written Types Need Proto)

These modules have working CLI but use hand-written Go types instead of proto-generated types. This means they lack gRPC-gateway REST endpoints and proto codec registration.

| Module | CLI TX (hand-written) | CLI Query (hand-written) |
|--------|----------------------|-------------------------|
| auth | 7 | 0 |
| compute_pool | 7 | 0 |
| discovery | 6 | 0 |
| channels | 3 | 0 |
| billing | 4 | 0 |
| autopoiesis | 3 | 0 |
| capture_challenge | 3 | 0 |
| capture_defense | 2 | 0 |
| emergency | 0 | 4 |
| icaauth | 2 | 0 |
| bvm | 1 | 0 |

**Total hand-written TX**: 38 commands without proto Msg service definitions.

### Low Priority (Intentional Omissions)

- `evidence_mgmt` missing `update-params` TX CLI — governance-only, expected
- `ibcratelimit` has no TX CLI — governance-only module, expected

---

## Module.go GetTxCmd/GetQueryCmd Status

All modules except **knowledge** return nil from `GetTxCmd()` and `GetQueryCmd()`. This is NOT a bug — root.go handles CLI registration directly. However, it's a deviation from standard Cosmos SDK module patterns and means autocli cannot be enabled.

| Returns CLI | Returns nil |
|-------------|-------------|
| knowledge | alignment, auth, autopoiesis, billing, bvm, capture_challenge, capture_defense, channels, claiming_pot, compute_pool, discovery, disputes, emergency, evidence_mgmt, gov, home, ibcratelimit, icaauth |

---

## Recommendations

1. **Create `x/home/client/cli/query.go`** — 5 query commands for Home, Homes, Sessions, Alerts, Params. Add `homecli.NewQueryCmd()` to root.go.

2. **Add missing knowledge CLI commands** — At minimum: propose-domain, register-stratum, stratum, strata, domains, domain-proposals.

3. **Add gov CLI commands** — At minimum the most common operations: submit-lip, cast-vote, create-deployment.

4. **Migrate hand-written Msg types to proto** — 38 TX commands use hand-written types. Priority modules: auth (7 cmds), compute_pool (7 cmds), discovery (6 cmds).

5. **Standardize module.go** — Wire `GetTxCmd()/GetQueryCmd()` to return actual commands (matching knowledge's pattern) for future autocli support.
