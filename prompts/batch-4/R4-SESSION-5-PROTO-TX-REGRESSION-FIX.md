# R4 Session 5: Proto TX Migration — Regression Verification & Codec Integrity

## Context
Session R3-3 was the biggest change: deleted hand-written Msg structs and proto_names.go across 12 modules, replacing them with buf-generated code. The report notes:

> "go test ./x/... → 1 FAIL (pre-existing: x/knowledge/evaluation — unrelated to this session)"

And also: 11 `proto_names.go` files deleted, 11 `types.go/msgs.go` cleaned, 12 `msg_server.go` updated. This is a massive refactor surface. Need to verify nothing broke.

### Key Risks
1. **Amino encoding**: Old hand-written types used `RegisterConcrete` with specific type URLs. Proto-generated types may have different names → **transaction signing breaks**
2. **GetSigners**: Preserved in msgs_validate.go — but are they still wired? Does the proto-generated type call them?
3. **gRPC service descriptors**: Hand-written gRPC descriptors were deleted — proto-generated ones should replace them exactly
4. **Backward compatibility**: Any serialized state using old type names won't deserialize

## Tasks

### 1. Amino Type URL Audit
For each of the 12 modules, verify:
```bash
grep -rn "RegisterConcrete\|RegisterImplementations" x/<module>/types/codec.go
```
- The Amino type URL (e.g., `"autopoiesis/UpdateParams"`) must match what was registered before
- Check git diff for any type URL changes

### 2. Full Interface Registration Test
```bash
go test ./app/... -run TestRegister
```
If there's a registration test in app, it should catch missing types. If not, create a simple one that:
- Creates a codec
- Registers all module interfaces
- Marshals/unmarshals each Msg type

### 3. GetSigners Wiring
For each module, verify the GetSigners method is called during ante handler processing:
- Proto-generated types may need cosmos.msg.v1.signer annotation in the proto OR a GetSigners() method
- Check that `option (cosmos.msg.v1.signer) = "authority"` (or equivalent) is set in each tx.proto
- If both proto annotation AND GetSigners method exist, verify they agree

### 4. gRPC Endpoint Smoke Test
For each module with new tx.proto:
- Verify `RegisterServices` in module.go calls `types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(...))`
- Verify the generated `tx_grpc.pb.go` has the correct service descriptor

### 5. Module-by-Module Test Run
```bash
for mod in autopoiesis billing compute_pool discovery ontology partnerships qualification schedule capture_challenge capture_defense icaauth staking; do
  echo "=== $mod ===" && go test ./x/$mod/... -count=1 2>&1 | tail -3
done
```

### 6. Build the Binary and Verify CLI
```bash
make build
./legbled tx --help 2>&1 | head -50
```
Verify all 12 modules appear in the tx subcommand list.

## Verification
```bash
go build ./...
go test ./...
go vet ./...
```

## Report
Write to `/reports/batch-4/SESSION-R4-5-PROTO-CODEC-INTEGRITY.md` with:
- Amino type URL comparison table (before/after)
- GetSigners verification per module
- gRPC registration status
- Any regressions found and fixed
