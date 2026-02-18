# R5 Session 1: Fix gRPC Service Registration Failures (Testnet Blocker)

## Context — CRITICAL LAUNCH BLOCKER
When the node starts, 6 gRPC services fail to register:

```
ERR gRPC service registration FAILED
  1. legible.ibcratelimit.v1.Msg — "type_url /legible.ibcratelimit.v1.MsgUpdateParams has not been registered yet"
  2. legible.auth.v1.ExtendedMsg — "cannot find method descriptor legible.auth.v1.ExtendedMsg.RegisterAccount"
  3. legible.emergency.v1.Msg — "cannot find method descriptor legible.emergency.v1.Msg.UpdateParams"
  4. legible.emergency.v1.Query — "cannot find method descriptor legible.emergency.v1.Query.Status"
  5. legible.lgm_gov.v1.Query — "cannot find method descriptor legible.lgm_gov.v1.Query.LIP"
  6. legible.channels.v1.SupplementalMsg — "cannot find method descriptor legible.channels.v1.SupplementalMsg.DepositChannel"
```

The node still starts and produces blocks (the errors are non-fatal), but these modules have no gRPC/REST endpoints. For testnet, ALL modules must be queryable via gRPC.

## Root Cause Analysis

These fall into two patterns:

### Pattern A: Missing RegisterInterfaces (ibcratelimit)
The error says `MsgUpdateParams has not been registered yet`. This means `ibcratelimit/types/codec.go` doesn't call `RegisterImplementations` for its Msg types, OR the types aren't being generated from proto.

### Pattern B: Missing Method Descriptors (auth, emergency, gov, channels)
The error says `cannot find method descriptor`. This means the `.proto` file defines the service, but the generated `_grpc.pb.go` doesn't have matching method descriptors. Likely causes:
- Proto was never generated (`buf generate` was never run for these protos)
- Proto file references methods that don't exist in the `.proto`
- Hand-written gRPC descriptors conflict with proto-generated ones

## Tasks

### For Each Failing Module:

1. **Check if proto files exist**: `ls proto/legible/<module>/v1/{tx,query}.proto`
2. **Check if generated files exist**: `ls x/<module>/types/{tx,query}.pb.go` and `{tx,query}_grpc.pb.go`
3. **Check RegisterServices in module.go**: What service descriptors is it trying to register?
4. **Fix the root cause**:
   - If proto exists but wasn't generated → run `buf generate`
   - If proto doesn't exist → create it matching the hand-written service
   - If RegisterInterfaces is missing types → add them to codec.go
   - If hand-written descriptors conflict → remove them (use proto-generated)

### Module-Specific Guidance:

**ibcratelimit**: Likely needs `RegisterImplementations` in codec.go for `MsgUpdateParams`. Check if `tx.proto` exists.

**auth (ExtendedMsg)**: Has a supplemental `ExtendedMsg` service alongside the main `Msg`. The proto may not define `RegisterAccount` in the `ExtendedMsg` service. Check `proto/legible/auth/v1/tx.proto` for the service definition.

**emergency**: Both Msg and Query fail. Likely no proto was generated. Check if `proto/legible/emergency/v1/` has `tx.proto` and `query.proto`.

**lgm_gov (Query)**: Query service fails. Check `proto/legible/lgm_gov/v1/query.proto`.

**channels (SupplementalMsg)**: Has a supplemental Msg service. Check if proto defines `DepositChannel` in the `SupplementalMsg` service.

## Verification
```bash
# Build and start devnet
make build
build/legbled start --home $HOME/.legbled-devnet 2>&1 | head -20
# Should see ZERO "gRPC service registration FAILED" lines

# Verify gRPC endpoints work
grpcurl -plaintext localhost:9090 list | grep -i "legible"
```

Also run:
```bash
go build ./...
go test ./x/ibcratelimit/... ./x/auth/... ./x/emergency/... ./x/gov/... ./x/channels/...
```

## Report
Write to `/reports/batch-5/SESSION-R5-1-GRPC-FIXES.md` with:
- Each failing service: root cause, fix applied
- Before/after gRPC registration output
- Full list of now-available gRPC services
