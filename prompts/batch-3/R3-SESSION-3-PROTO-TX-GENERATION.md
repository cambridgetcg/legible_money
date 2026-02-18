# Session 3: Add tx.proto for Modules Missing gRPC TX Surface

## Objective
13 modules have working CLI tx handlers but no `tx.proto`, meaning they lack gRPC/REST transaction endpoints. Add `tx.proto` definitions and generate the Go code so agents and external clients can submit transactions via gRPC.

## Modules Needing tx.proto

| Module | TX Handlers | Priority |
|--------|:-----------:|:--------:|
| autopoiesis | 4 | HIGH — core hormone system |
| billing | 5 | HIGH — distribution/pricing |
| compute_pool | 9 | HIGH — being wired in Session 2 |
| discovery | 7 | MEDIUM |
| ontology | 6 | MEDIUM |
| partnerships | 11 | MEDIUM |
| qualification | 5 | MEDIUM |
| schedule | 6 | MEDIUM |
| capture_challenge | 4 | MEDIUM |
| capture_defense | 3 | MEDIUM |
| icaauth | 3 | LOW — IBC utility |
| alignment | 0 | SKIP — no user txs by design |
| staking | 6 | LOW — uses keeper directly |

## Tasks

### 1. For Each Module (skip alignment)
- Read the existing CLI tx commands in `x/<module>/client/cli/tx.go`
- Read the Msg types in `x/<module>/types/msgs.go` or `types/tx.pb.go` (if partial)
- Create `proto/legible/<module>/v1/tx.proto` with:
  - `service Msg { ... }` matching each handler
  - Request/Response message types matching existing Go types
  - Proper cosmos.msg.v1 annotations
- Reference existing tx.proto files (e.g., `proto/legible/knowledge/v1/tx.proto`) for the pattern

### 2. Generate Go Code
```bash
# Use buf or protoc — check Makefile for the project's proto generation command
make proto-gen  # or equivalent
```

### 3. Wire msg_server.go (if missing)
- Some modules route handlers through keeper methods directly instead of a `msg_server.go`
- If a module lacks `msg_server.go`, create one that delegates to the keeper methods
- Register the `MsgServer` in the module's `RegisterServices`

### 4. Priority Order
Start with HIGH priority modules (autopoiesis, billing, compute_pool), then MEDIUM. Skip alignment entirely. Do staking/icaauth only if time permits.

## Verification
```bash
go build ./...
go test ./...
```

## Report
Write completion report to `/reports/SESSION-R3-3-PROTO-TX-GENERATION.md` with:
- Table of modules processed: proto created, msg_server wired, RPCs added
- Any modules skipped and why
- Total new gRPC endpoints added
- Test results
