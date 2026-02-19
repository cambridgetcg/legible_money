# B8-2: Enable gRPC Gateway (REST API)

## Context
The gRPC gateway is currently disabled. REST queries are unavailable. This is a P2 but important for testnet usability — block explorers, wallets, and external tools typically use REST.

## Task

### 1. Understand current state
Read `app/app.go` — find where gRPC gateway routes are (or should be) registered. Check if there's a `RegisterGRPCGatewayRoutes` method or similar.

Read the Cosmos SDK v0.50 docs/examples for how gRPC gateway is typically wired:
- `app.go` RegisterGRPCGatewayRoutes
- API server config in `app.toml`

### 2. Enable the gateway
Wire up the gRPC-gateway mux in the app. This typically involves:
- Registering all module gateway routes
- Ensuring the API server is enabled in config
- Setting up CORS if needed

### 3. Verify all 28 custom modules register their gateway routes
For each module in `x/`, check if the module implements `RegisterGRPCGatewayRoutes`. If any are missing, add them.

### 4. Test
- Start local devnet
- Verify REST endpoints respond: `curl http://localhost:1317/cosmos/bank/v1beta1/balances/<addr>`
- Verify at least one custom module endpoint: `curl http://localhost:1317/legible/knowledge/v1/claims`
- Verify Swagger/OpenAPI if available

### 5. Config
Ensure `app.toml` has API server enabled by default for testnet genesis config.

## File Ownership
- `app/app.go` (gateway registration section only)
- Any module `module.go` files that need `RegisterGRPCGatewayRoutes`
- Config templates if modified

## Report
Write `reports/batch-8/SESSION-B8-2-GRPC-GATEWAY.md`.
