# Session: CRITICAL-FIXES — Unblock testnet: module account, proto codec, query wiring

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Fix the 3 highest-priority blockers preventing testnet launch

These were discovered in E2E testing (batch-5). Without these fixes, governance, custom staking, and governance queries are all broken on a live chain.

## Fix 1: F-2 CRITICAL — lgm_gov module account not registered

**Symptom**: `MsgSubmitLIP` panics with `module account lgm_gov does not exist: unknown address` (recovered by baseapp, returns code 111222)
**Root cause**: `maccPerms` in `app/app.go` doesn't include the lgm_gov module name. The bank keeper can't create or access the module account for deposit escrow.

**Steps**:
1. Read `app/app.go` and search for `maccPerms` (the map of module account permissions)
2. Find the exact module name used by lgm_gov — read `x/gov/types/keys.go` for `ModuleName`
3. Add the module to `maccPerms` with appropriate permissions. It needs `{authtypes.Burner}` if LIP deposits can be burned on rejection, or `{}` (empty) if deposits are only returned/transferred:
   ```go
   govtypes.ModuleName: {authtypes.Burner},
   ```
4. Also check: does `x/gov/keeper/msg_server.go` call `SendCoinsFromAccountToModule` or `SendCoinsFromModuleToAccount`? Both require the module to be in `maccPerms`.

**Verify**: `go build ./...` clean. Then run the devnet and test `MsgSubmitLIP`:
```bash
cd devnet && bash scripts/cosmos-devnet.sh &
sleep 10
legbled tx lgm_gov submit-lip --title "Test LIP" --description "Test" --deposit 1000000ulgm --from validator --yes --gas 300000 --fees 300000ulgm
```
Should return code 0, not 111222.

## Fix 2: F-1 HIGH — Staking tx.pb.go wrong proto generator

**Symptom**: `register-validator` returns code 2 (tx parse error). The `ValidatorDescription` nested message can't be resolved at decode time.
**Root cause**: `x/staking/types/tx.pb.go` was generated with `protoc-gen-go` (Google protov2) instead of `protoc-gen-gogo` (gogoproto required by Cosmos SDK v0.50). The type registration uses `protoimpl.UnsafeEnabled` instead of `gogoproto.RegisterType`.

**Steps**:
1. Check the proto generation setup:
   - Read `buf.gen.yaml` or `Makefile` for the proto generation command
   - Read `proto/legible/staking/v1/tx.proto` — does it use `gogoproto` options?
2. Identify all `.pb.go` files in `x/staking/types/` generated with the wrong plugin:
   ```bash
   grep -l "protoimpl.UnsafeEnabled" x/staking/types/*.pb.go
   ```
3. Regenerate using gogoproto. If `buf generate` is configured correctly but staking was manually generated:
   ```bash
   buf generate --path proto/legible/staking/v1/
   ```
4. If buf/protoc aren't available or configured, check if other modules have correctly generated pb.go files and compare the generation headers:
   ```bash
   head -5 x/tokens/types/tx.pb.go  # known good
   head -5 x/staking/types/tx.pb.go  # broken
   ```
5. After regeneration, verify the types register correctly:
   ```bash
   grep "RegisterType" x/staking/types/tx.pb.go | head -5
   ```
   Should show `proto.RegisterType` (gogoproto), NOT `protoimpl` registration.

**Verify**: `go build ./...` clean. Devnet test:
```bash
legbled tx legible_staking register-validator --moniker "test" --from validator --yes --gas 300000 --fees 300000ulgm
```
Should NOT return code 2.

## Fix 3: F-4 MEDIUM — lgm_gov gRPC query service not registered

**Symptom**: All `legbled query lgm_gov` commands fail with `unknown query path: unknown request`
**Root cause**: `RegisterServices` in `x/gov/module.go` likely doesn't call `types.RegisterQueryServer`.

**Steps**:
1. Read `x/gov/module.go` — find `RegisterServices(cfg module.Configurator)` 
2. Check if it calls `types.RegisterQueryServer(cfg.QueryServer(), keeper)` or equivalent
3. If the query server registration is missing, add it. Pattern from other modules:
   ```go
   func (am AppModule) RegisterServices(cfg module.Configurator) {
       types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
       types.RegisterQueryServer(cfg.QueryServer(), am.keeper)  // ADD THIS
   }
   ```
4. Also check: does `x/gov/keeper/` have a `grpc_query.go` that implements `QueryServer`? If not, the keeper needs to implement the query interface.
5. Check `x/gov/types/query_grpc.pb.go` or `x/gov/types/query.go` — does the query service interface exist?

**Verify**: `go build ./...` clean. Devnet test:
```bash
legbled query lgm_gov params
```
Should return JSON params, not "unknown query path".

## Final verification
After all 3 fixes:
- `go build ./...` clean
- `go test ./... -count=1` — all pass
- Boot devnet, run full E2E suite:
  ```bash
  bash devnet/scripts/test-tx-lifecycle.sh
  bash devnet/scripts/test-lgm-modules.sh
  ```
- Previously skipped tests E3/E5 should now pass (they cascaded from F-2)

## Report format
Write to reports/batch-6/SESSION-B6-1-CRITICAL-FIXES.md:
- Each fix: root cause confirmed, exact code change, before/after
- E2E test results (especially the previously failing/skipping tests)
- Build + test output
- Any new issues discovered
