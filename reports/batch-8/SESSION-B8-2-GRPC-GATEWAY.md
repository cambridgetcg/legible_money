# SESSION-B8-2: GRPC-GATEWAY — Enable REST API via gRPC Gateway

## Summary

Enabled the gRPC-gateway REST proxy for all 26 custom modules. REST queries are now available on port 1317 alongside the existing Cosmos SDK built-in endpoints.

## Changes

### 1. Proto Generation — `proto/buf.gen.yaml`

Added `protoc-gen-grpc-gateway` (v1) plugin:

```yaml
  - name: grpc-gateway
    out: ..
    opt:
      - module=github.com/legible-money/legible
      - logtostderr=true
    path: /Users/yu/go/bin/protoc-gen-grpc-gateway
```

Uses grpc-gateway v1 (`github.com/grpc-ecosystem/grpc-gateway/runtime`) to match the Cosmos SDK v0.50 import path.

### 2. HTTP Annotations — `proto/legible/{emergency,gov}/v1/query.proto`

Added `google.api.http` annotations to the 2 modules that were missing them:

- **emergency**: `/legible/emergency/v1/status`, `/ceremony/active`, `/ceremonies/completed`, `/audit_log`
- **gov**: `/legible/gov/v1/lip/{lip_id}`, `/lips`, `/disbursement/{disbursement_id}`, `/params`

The other 24 modules already had HTTP annotations in their query.proto files.

### 3. Generated Gateway Files — `x/*/types/query.pb.gw.go` (26 files)

Generated `RegisterQueryHandlerClient` functions for all 26 modules with query services:

| Module | REST Prefix | Endpoints |
|--------|------------|-----------|
| auth | `/legible/auth/v1/` | account, account_by_did, session_keys, params |
| billing | `/legible/billing/v1/` | account, accounts, params |
| bvm | `/legible/bvm/v1/` | contract, contracts, code, state, schedule, params |
| capture_challenge | `/legible/capture_challenge/v1/` | challenge, challenges, params |
| capture_defense | `/legible/capture_defense/v1/` | qualification, qualifications, reputation, risk |
| channels | `/legible/channels/v1/` | channel, channels/payer, channels/payee, params |
| claiming_pot | `/legible/claiming_pot/v1/` | eligibility, pot_summary, params |
| compute_pool | `/legible/compute_pool/v1/` | provider, providers, credit, ticket, pool_stats, params |
| discovery | `/legible/discovery/v1/` | params |
| disputes | `/legible/disputes/v1/` | dispute, disputes/fact, disputes/active, evidence, votes, params |
| emergency | `/legible/emergency/v1/` | status, ceremony/active, ceremonies/completed, audit_log |
| evidence_mgmt | `/legible/evidence_mgmt/v1/` | evidence, audit_trail, oracle, params |
| gov | `/legible/gov/v1/` | lip, lips, disbursement, params |
| home | `/legible/home/v1/` | home, homes, guardian, params |
| ibcratelimit | `/legible/ibcratelimit/v1/` | rate_limit, rate_limits, params |
| icaauth | `/legible/icaauth/v1/` | params |
| knowledge | `/legible/knowledge/v1/` | claim, claims, round, rounds, fact, facts, domain, params |
| ontology | `/legible/ontology/v1/` | stratum, strata, domain, domains, params |
| partnerships | `/legible/partnerships/v1/` | partnership, partnerships, params |
| qualification | `/legible/qualification/v1/` | pathway, pathways, params |
| research | `/legible/research/v1/` | bounty, bounties, params |
| schedule | `/legible/schedule/v1/` | process, processes, params |
| staking | `/legible/staking/v1/` | validator, validators, params |
| tokens | `/legible/tokens/v1/` | token, tokens, balance, allowance, pool, params |
| tree | `/legible/tree/v1/` | project, projects, task, service, seed, params |
| vesting_rewards | `/legible/vesting_rewards/v1/` | schedule, schedules, active, distribution, params |

**Not applicable** (no query proto): alignment, autopoiesis

### 4. Module Wiring — `x/*/module.go` (26 files)

Replaced empty `RegisterGRPCGatewayRoutes` bodies with:

```go
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
    types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}
```

### 5. Config — `cmd/legbled/cmd/config.go`

Enabled gRPC server (required for gateway REST proxy):

```go
srvCfg.GRPC.Enable = true  // was false
```

### 6. Gitignore — `.gitignore`

Un-ignored `x/*/types/query.pb.gw.go` (was previously ignored as "WIP").

## Pre-existing Issues

- `x/knowledge/keeper/state.go:45` has `k.maintainFactIndexes undefined` — pre-existing, unrelated to this change
- Alignment and autopoiesis modules have no query proto definitions (no gateway to register)

## Verification

- `go build ./...` — clean (exit 0)
- `go test ./x/tokens/keeper/...` — pass
- `go test ./x/staking/keeper/...` — pass
- `go test ./x/emergency/keeper/...` — pass
- `go test ./x/gov/keeper/...` — pass
- `go test ./x/bvm/keeper/...` — pass

## REST API Usage (after devnet start)

```bash
# Cosmos SDK built-in
curl http://localhost:1317/cosmos/bank/v1beta1/balances/lgm1...

# Custom modules
curl http://localhost:1317/legible/knowledge/v1/claims
curl http://localhost:1317/legible/staking/v1/validators
curl http://localhost:1317/legible/emergency/v1/status
curl http://localhost:1317/legible/gov/v1/params
curl http://localhost:1317/legible/tokens/v1/tokens
```
