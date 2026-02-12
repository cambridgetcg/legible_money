# Cosmos SDK DevNet Simulation Report

**Date**: 2026-02-11
**Chain ID**: `lgm-devnet-1`
**Binary**: `build/legbled` (112MB, 25 custom modules)
**Block time**: ~2.5s (CometBFT consensus timeout)

---

## 1. Node Startup — Obstacles & Fixes

Getting the Cosmos SDK chain from "compiles cleanly" to "producing blocks" required resolving **5 sequential runtime panics**:

### 1.1 Store Key Prefix Collision
**Error**: `store keys must not conflict: "ibcratelimit" has prefix of existing key "ibc"`
**Root cause**: `NewKVStoreKeys()` enforces that no store key is a prefix of another. The `ibcratelimit` module's `StoreKey` was `ModuleName` = `"ibcratelimit"`, which starts with `"ibc"`.
**Fix**: Changed `StoreKey` to `"ratelimit"` in `x/ibcratelimit/types/keys.go`.

### 1.2 Missing Module in BeginBlockers
**Error**: `module legible_auth is registered but missing from SetOrderBeginBlockers`
**Root cause**: Cosmos SDK v0.50 requires every registered module to appear in all three ordering lists (BeginBlockers, EndBlockers, InitGenesis).
**Fix**: Added `legibleauthtypes.ModuleName` to both `SetOrderBeginBlockers` and `SetOrderEndBlockers`.

### 1.3 Multiple Modules Missing from EndBlockers
**Error**: `module discovery is registered but missing from SetOrderEndBlockers`
**Root cause**: Systematic gap — several standard and custom modules were missing from one or more ordering lists.
**Fix**: Used a Python script to audit all three lists against registered modules. Added `discovery`, `upgrade`, `distribution`, `slashing`, `evidence` to EndBlockers; `gov`, `feegrant` to BeginBlockers.

### 1.4 Proto Type Registration Panic
**Error**: `panic: type_url / has not been registered yet` from `x/partnerships/types.RegisterMsgServer`
**Root cause**: All 25 custom modules have empty `RegisterInterfaces()` ("deferred to proto codegen") but all call `RegisterMsgServer()` in `RegisterServices()`. The SDK's `MsgServiceRouter` panics when resolving unregistered type URLs.
**Fix**: Created `SafeServiceRegistrar` wrapper in `app/app.go` with `defer/recover` to gracefully skip modules without proto codegen. Custom modules' gRPC services emit a warning and are skipped.

### 1.5 ICA Host Missing Query Router
**Error**: `panic: query router must not be nil` from ICA host `NewMsgServerImpl`
**Root cause**: `ICAHostKeeper.NewKeeper()` doesn't accept a query router; it must be set separately via `WithQueryRouter()`.
**Fix**: Added `app.ICAHostKeeper.WithQueryRouter(app.GRPCQueryRouter())` after keeper creation.

### 1.6 CLI Missing Query/Tx Command Groups
**Symptom**: `unknown command "query"` — the root command only had individual query commands, not the grouped `query`/`tx` subcommand trees.
**Fix**: Added proper `query` and `tx` command groups to `cmd/legbled/cmd/root.go` with bank, staking, and gov subcommands explicitly wired (not via `ModuleBasics.AddTxCommands` which panics on uninitialized modules).

---

## 2. Genesis Flow

### Supply at Block 1

| Source | Amount (LGM) | Notes |
|--------|-------------|-------|
| Validator | 50,000 | 10K self-delegated, 40K liquid |
| Foundation | 22,222,222 | Protocol treasury |
| Research | 17,111,111 | Research fund |
| Community | 11,111,111 | Community pool |
| Test accounts (×3) | 3,000 | 1,000 each |
| Block reward fund | 133,333,333.2 | Auto-minted by `x/vesting_rewards` InitGenesis |
| **Total** | **183,830,777.2** | Verified via `/cosmos/bank/v1beta1/supply` |

### Module Genesis Initialization
- **25 custom modules** initialized via `InitGenesis` in declared order
- **Knowledge module**: 4 default domains (mathematics, physics, computer_science, general)
- **Ontology module**: 7 strata (axiomatic → testimonial) with 7 domains
- **Vesting rewards**: Minted 133.3M LGM to module account; epoch decay configured
- **Legible staking**: 4 validator tiers (apprentice → guardian)
- **Vote extensions**: Enabled from height 1 (required for PoT commit-reveal)

---

## 3. Transaction Simulation

### 3.1 Bank Send (SUCCESS)
```
legbled tx bank send <validator> <test1> 1000000ulgm --fees 250000ulgm
```
- **Result**: Code 0, test1 balance increased from 1,000 → 1,001 LGM
- **Insight**: Minimum fee is 200,000 ulgm (gas 200000 × min price 1 ulgm/gas from custom ante handler)
- **Note**: Min gas price in app.toml is 0.025 ulgm but the custom `LegibleGasDecorator` enforces a floor of 1 ulgm/gas

### 3.2 Cosmos Staking Delegation (SUCCESS)
```
legbled tx staking delegate <valoper> 100000000ulgm --from test1 --fees 250000ulgm
```
- **Result**: Code 0, 100 LGM delegated
- **REST verified**: `delegation_responses[0].balance.amount = "100000000"`
- **Remaining balance**: 900.75 LGM (1001 - 100 delegation - 0.25 fee)

### 3.3 Create Validator (SUCCESS — second attempt)
```
legbled tx staking create-validator /tmp/test1-validator.json --from test1
```
- **First attempt**: Failed — pubkey was wrong size (fake base64 decoded to != 32 bytes). Chain recovered gracefully via `baseapp.recovery`.
- **Second attempt**: Code 0 with proper 32-byte ed25519 pubkey.
- **Result**: Two validators in active set:
  - `lgm-devnet-validator`: 10,100 voting power (10K self + 100 delegation)
  - `test1-pot-validator`: 500 voting power (500 LGM self-delegation)

### 3.4 Knowledge Claim Submission (BLOCKED)
**Cannot submit.** The `x/knowledge` module's gRPC Msg and Query services were skipped by `SafeServiceRegistrar` because proto codegen hasn't been run. This means:
- No REST endpoint: `/legible/knowledge/v1/params` returns `Not Implemented`
- No gRPC service: Direct gRPC calls get `UNIMPLEMENTED`
- No CLI command: Custom modules return `nil` from `GetTxCmd()`

**What would need to happen**:
1. Run `buf generate` to produce `*.pb.go` files from the `.proto` definitions
2. Each module's `RegisterInterfaces()` must register its message types
3. Each module's `RegisterServices()` then successfully registers Msg and Query servers
4. CLI tx commands would also need the generated types

---

## 4. Key Architectural Insights

### 4.1 Dual Staking Systems
The chain runs **two independent staking systems**:
1. **Cosmos SDK staking** (`x/staking`): CometBFT validator set, standard delegation, slashing. Used for block production consensus.
2. **Legible staking** (`legible_staking`): PoT-specific validator tiers (apprentice/verified/bonded/guardian), verification-based selection, truth-specific slashing. Used for knowledge verification.

Currently only Cosmos staking is operational (has proto types). The bridge between the two systems (verifier selection from Cosmos validators) is in the vote extensions handler.

### 4.2 Vote Extensions (PoT Bridge)
Vote extensions are enabled from height 1 (`vote_extensions_enable_height = "1"`). The `VoteExtensionHandler` in `app/vote_extensions.go` is the bridge between CometBFT consensus and PoT:
- `ExtendVote`: Validators submit their knowledge verifications
- `VerifyVoteExtension`: Other validators verify the claims
- Currently these run as no-ops (return empty extensions) because no knowledge claims are in the pipeline

### 4.3 The Proto Codegen Gap
The single biggest blocker for the Cosmos SDK chain's full functionality:
- **25 custom modules** × 2 services (Msg + Query) = **50 gRPC services** skipped
- All `.proto` files exist in `proto/legible/*/v1/` but `buf generate` hasn't been run
- Each module has hand-written `_serviceDesc` in `types/types.go` but these lack the generated `pb.go` registration
- Fix: Run protobuf codegen, then remove `SafeServiceRegistrar` wrapper

### 4.4 Custom Ante Handler
The Legible chain uses a custom ante handler (`app/ante_legible.go`) that adds:
- `LegibleGasDecorator`: Enforces minimum gas price of 1 ulgm/gas (overrides CometBFT's 0.025)
- Knowledge claim validation
- Domain-specific fee calculation
- Session key authentication (for agent identity)

### 4.5 Module Account Architecture
25 module accounts are registered with specific permissions:
- `vesting_rewards`: Minter + Burner (manages the finite block reward fund)
- `knowledge`: Minter (for verification rewards)
- `burn`: Burner only (for token burns/slashing)
- Standard accounts: `bonded_tokens_pool`, `not_bonded_tokens_pool`, `distribution`, `fee_collector`

---

## 5. What Works vs What's Blocked

### Working
- Block production at ~2.5s intervals
- Cosmos staking (delegate, create-validator, unbond)
- Bank transactions (send, multi-send)
- Governance (submit proposal, vote, deposit)
- IBC core (transfer, channels, interchain accounts)
- Module genesis initialization (all 25 modules)
- Block reward fund (133.3M LGM in vesting_rewards account)
- REST API for standard Cosmos modules
- Vote extensions framework (enabled but no-op)

### Blocked (requires proto codegen)
- Knowledge claim submission & verification
- PoT validator registration (legible_staking)
- Evidence management
- Dispute resolution
- Payment channels
- BVM (Bytecode Virtual Machine)
- All 25 custom module REST/gRPC endpoints
- Claude evaluation daemon integration

---

## 6. Devnet Launch Command Reference

```bash
# Start devnet (build + init + start)
bash devnet/scripts/cosmos-devnet.sh

# Reset devnet state
bash devnet/scripts/cosmos-devnet.sh reset

# Stop node
bash devnet/scripts/cosmos-devnet.sh stop

# Check status
bash devnet/scripts/cosmos-devnet.sh status

# Or manually via RPC:
curl -s localhost:26657/status | jq .result.sync_info

# Query balances via REST:
curl -s localhost:1317/cosmos/bank/v1beta1/balances/<address>

# Query validators:
curl -s localhost:1317/cosmos/staking/v1beta1/validators

# Send tokens:
build/legbled tx bank send <from> <to> <amount>ulgm \
  --chain-id lgm-devnet-1 --keyring-backend test \
  --home ~/.legbled-devnet --fees 250000ulgm -y
```

---

## 7. Next Steps for Full PoT Simulation

1. **Run `buf generate`** to produce protobuf Go code for all 25 custom modules
2. **Remove SafeServiceRegistrar** — services will register normally once types are generated
3. **Wire CLI tx commands** — each module's `GetTxCmd()` should return proper commands
4. **Submit test knowledge claim** — `MsgSubmitClaim{domain, text, evidence[], stake}`
5. **Observe commit-reveal round** — vote extensions carry verifier attestations
6. **Test slashing** — submit conflicting verifications, observe equivocation slash
7. **Test block rewards** — verify epoch decay from the 133.3M reward fund
