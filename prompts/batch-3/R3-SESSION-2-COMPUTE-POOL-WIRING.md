# Session 2: Wire compute_pool Module into app.go

## Objective
`x/compute_pool` is **completely unwired** — it has a full implementation (keeper, CLI, gas entries, 43 passing tests) but is not registered in `app.go`. Wire it fully.

## Context
- The module has: store key, keeper code, CLI commands (8 tx, 7 query), gas entries (8), proto files, codec registration
- What's missing: app.go registration (store key, keeper field, module manager, blocker ordering, init genesis)
- 8 gas entries in `app/gas.go` are currently orphaned — they'll become live once wired

## Tasks

### 1. Study the Module
- Read `x/compute_pool/module.go` to understand its AppModule interface
- Read `x/compute_pool/keeper/keeper.go` for constructor signature and dependencies
- Read `x/compute_pool/types/expected_keepers.go` for required keeper interfaces
- Check if it needs BeginBlocker/EndBlocker

### 2. Wire into app.go
Follow the pattern of other wired modules in `app/app.go`:

1. **Store key**: Add `compute_pool` store key to `KVStoreKeys`
2. **Keeper field**: Add `ComputePoolKeeper` field to the App struct
3. **Keeper initialization**: Create the keeper with correct dependencies (bank, account, staking, etc.)
4. **Module manager**: Register `compute_pool.NewAppModule(...)` in the module manager
5. **Blocker ordering**: Add to BeginBlocker/EndBlocker order if needed
6. **InitGenesis ordering**: Add to InitGenesis/ExportGenesis order
7. **gRPC registration**: Ensure query/tx servers are registered

### 3. Adapter (if needed)
- If compute_pool needs cross-module interfaces, create `app/compute_pool_adapter.go` following the pattern of other adapter files in `app/`
- Check `app/billing_adapters.go`, `app/staking_adapter.go` for reference

### 4. Verify Gas Entries
- Confirm the 8 gas entries in `app/gas.go` for compute_pool are correct and match the handler operations
- No orphaned keys should remain

## Verification
```bash
go build ./...
go test ./x/compute_pool/...
go test ./app/...
go vet ./...
```

## Report
Write completion report to `/reports/SESSION-R3-2-COMPUTE-POOL-WIRING.md` with:
- Full list of changes made to app.go
- Any adapter files created
- Dependencies resolved
- Test results
- Any issues or design decisions encountered
