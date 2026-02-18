# Session: TESTNET-READINESS — Final checklist and comprehensive verification

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Final testnet readiness assessment — comprehensive checklist

This is a READ-ONLY assessment session. Do NOT modify code. Generate the definitive readiness report.

### Checklist 1: Build & Test
```bash
go build -o build/legbled ./cmd/legbled
go test ./... -count=1 -timeout 600s
go vet ./...
```
All must pass. Document any failures.

### Checklist 2: Binary capabilities
```bash
build/legbled version --long
build/legbled init test-check --chain-id test-check 2>&1 | head -5
build/legbled validate-genesis ~/.legbled/config/genesis.json

# CLI completeness
build/legbled tx --help 2>&1 | grep -c "Available Commands"
build/legbled query --help 2>&1 | grep -c "Available Commands"
```

### Checklist 3: Module account verification
```bash
# Check maccPerms includes ALL modules that need module accounts
grep -A 30 "maccPerms" app/app.go
```
Cross-reference against modules that call `SendCoinsFromAccountToModule` or `SendCoinsFromModuleToAccount`:
```bash
grep -rn "SendCoinsFromAccountToModule\|SendCoinsFromModuleToAccount\|SendCoinsFromModuleToModule" x/ --include="*.go" | grep -v "_test.go"
```
Every module name in those calls MUST be in `maccPerms`. If any are missing → BLOCKER.

### Checklist 4: RegisterServices completeness
For every module in `app/app.go`'s ModuleManager:
```bash
grep -l "RegisterServices" x/*/module.go
```
For each, verify both MsgServer AND QueryServer are registered:
```bash
for mod in x/*/module.go; do
  echo "=== $mod ==="
  grep "RegisterMsgServer\|RegisterQueryServer\|MsgServer\|QueryServer" $mod
done
```
Any module with handlers but no MsgServer registration → BLOCKER.
Any module with query endpoints but no QueryServer registration → WARNING.

### Checklist 5: Genesis round-trip
```bash
# Init, export genesis, validate
build/legbled init test-genesis --chain-id test-genesis
# Patch vote extensions
cat ~/.legbled/config/genesis.json | jq '.consensus.params.abci.vote_extensions_enable_height = "1"' > /tmp/genesis.json
mv /tmp/genesis.json ~/.legbled/config/genesis.json
build/legbled validate-genesis ~/.legbled/config/genesis.json
```

### Checklist 6: Gas coverage
```bash
# Count handlers vs gas entries
grep -c "func (ms msgServer)" x/*/keeper/msg_server.go 2>/dev/null
grep -c "GasKey\|gasKey\|Gas(" app/gas.go 2>/dev/null
```
Any handler without a gas entry that isn't UpdateParams → WARNING.

### Checklist 7: Proto registration
```bash
# Check for any modules still using hand-written types without RegisterInterfaces
for mod in x/*/types/codec.go; do
  echo "=== $mod ==="
  grep -c "RegisterInterfaces" $mod
  grep -c "RegisterCodec" $mod
done
```

### Checklist 8: Known issues from all batches
Read through ALL reports in reports/batch-{1..5}/ and audits/:
- Compile a list of ALL findings marked "not fixed", "deferred", "known issue", "not in scope"
- For each: is it a testnet blocker? Rate: BLOCKER / WARNING / OK-FOR-TESTNET

### Checklist 9: Security P0 status
Verify ALL P0 findings from audits are resolved:
- BVM Engine: 4 P0s (memory truncation, CALLCODE, context prop, call depth) — verify tests exist
- BVM Consensus: 3 P0s (gas bridge, panic recovery, schedule gas) — verify tests exist
- ABCI: 2 P0s (false equivocation, reveal hash) — verify tests exist
- Staking: 3 P0s (reputation underflow, contested tracking, epoch length) — verify tests exist
- IBC: 6 P0s (rate limit bypass, single-denom, epoch burst, allowlist, SendTx, MsgTransfer) — verify tests exist
- Channels: P0-1 (signatures) — verify tests exist
- Ante: P0-1 (SDK v0.50 bypass) — verify tests exist

```bash
# Quick check: all P0 test functions exist
grep -rn "TestP0_\|Test.*P0\|Test.*Critical\|Test.*Security" x/*/keeper/*_test.go x/*/vm/*_test.go app/*_test.go | wc -l
```

### Checklist 10: Operational readiness
- Does `legbled start` boot cleanly in single-validator mode?
- Does the REST API respond? (`curl localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest`)
- Does gRPC respond? (`grpcurl -plaintext localhost:9090 list`)
- Are Prometheus metrics exposed? (`curl localhost:26660/metrics | head -5`)

## Report format
Write to reports/batch-6/SESSION-B6-7-TESTNET-READINESS.md:
- Checklist results: PASS / FAIL / WARNING per item
- BLOCKER list (must fix before testnet)
- WARNING list (should fix but not blocking)
- OK-FOR-TESTNET list (deferred items acceptable for testnet)
- Final verdict: READY / NOT READY (with specific blockers if not ready)
- Estimated effort to fix blockers (hours)
