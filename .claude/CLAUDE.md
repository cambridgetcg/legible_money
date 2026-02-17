# Legible Money — Go/Cosmos SDK Chain - Project Intelligence

> **This is the production Go/Cosmos SDK chain.** The TypeScript prototype/reference implementation lives in a separate repo.
>
> **Session Continuity System**: This file + memory/ directory maintain project state across sessions.
> Always read `.claude/memory/CURRENT_STATE.md` at session start.

---

## Project Identity

**Legible Money** is a cryptocurrency protocol designed for AI agent economies with **Proof of Truth (PoT)** consensus - where verifying knowledge IS the useful work.

### Core Philosophy
- Truth as consensus mechanism (not arbitrary computation)
- AI agents as first-class economic citizens
- Knowledge accumulation as blockchain's purpose
- Gödelian acknowledgment of incompleteness

### Protocol Constants
- Block time: **2521ms**
- Finality depth: **19 blocks**
- Token: **LGM** (Legible Money)
- Binary: **legbled**
- SDK: **Cosmos SDK v0.50.10**, CometBFT v0.38, IBC-Go v8.5.1

---

## Directory Structure

```
legible-money/
├── app/                    # Application wiring
│   ├── app.go              # Full module wiring
│   ├── ante.go             # 16-decorator ante handler chain
│   ├── ante_legible.go     # Custom ante decorators (account, capability, DID)
│   ├── abci.go             # ABCI++ handlers + halt guard
│   ├── vote_extensions.go  # ExtendVote + VerifyVoteExtension (VRF)
│   ├── gas.go              # 38 tx gas costs
│   ├── local_state.go      # In-memory commitment salt store
│   └── staking_adapter.go  # Cross-module staking interface adapter
│
├── cmd/legbled/            # Chain binary
│   ├── root.go             # Root command
│   ├── config.go           # 2521ms block time config
│   └── genesis.go          # Genesis commands
│
├── x/                      # Custom Cosmos SDK modules
│   ├── knowledge/          # COMPLETE: Claims, verification rounds, facts, domains
│   ├── auth/               # COMPLETE: Dual-key, DID-to-bech32, session keys
│   ├── staking/            # COMPLETE: 4-tier graduated validators, VRF weight
│   ├── emergency/          # COMPLETE: 3-phase BFT kill switch
│   ├── gov/                # COMPLETE: LIP lifecycle, quadratic voting
│   ├── tokens/             # PARTIAL: Dual-ledger period overlay
│   ├── vesting_rewards/    # PARTIAL: Block reward emission
│   ├── bvm/                # PARTIAL: Belief Virtual Machine
│   ├── home/               # PARTIAL: Agent Home
│   ├── disputes/           # STUB
│   ├── evidence_mgmt/      # STUB
│   ├── research/           # STUB
│   ├── ontology/           # STUB
│   ├── channels/           # STUB
│   ├── tree/               # STUB
│   └── ... (more stubs)
│
├── proto/                  # Protobuf definitions
├── devnet/scripts/         # DevNet management scripts
├── tests/cross_stack/      # Golden vector tests (Go)
├── tools/migrate/          # State migration tooling
├── scripts/                # DevNet initialization scripts
├── docs/                   # Documentation
└── .claude/                # Session continuity
```

---

## Key Technical Patterns

### SDK v0.50 Patterns
- `context.Context` not `sdk.Context` in keeper interfaces
- `storetypes.KVStorePrefixIterator` replaces `sdk.KVStorePrefixIterator`
- `sdkmath.NewIntFromBigInt` replaces `sdk.NewIntFromBigInt`
- `sdk.HasValidateBasic` interface for optional ValidateBasic
- `addresscodec.NewBech32Codec()` required for AccountKeeper/StakingKeeper
- Hand-written types use `json.Marshal`/`json.Unmarshal` (not codec — no proto registration)

### Cross-Module Patterns
- `StakingKeeperAdapter` wraps custom staking keeper for x/knowledge interface
- `LegibleStakingKeeper` satisfies emergency StakingKeeper directly (no adapter)
- Nil-safe `if k.tokensKeeper != nil` guards — tokens overlay is optional
- TokensKeeper inited before LegibleStakingKeeper and KnowledgeKeeper in app.go

### Testing Patterns
- Test `init()` must set lgm bech32 prefix or AccAddressFromBech32 defaults to "cosmos"
- Equal-stake test validators for predictable quorum calculations
- Mock keepers returning nil — all Cosmos account operations need nil guards

---

## Agent Context (inject into subagent prompts)

```
PROJECT CONTEXT (Legible Money - Go Chain):
- Cosmos SDK v0.50.10, CometBFT v0.38, IBC-Go v8.5.1
- Binary: legbled, Chain ID: legible-devnet-1
- Block time: 2521ms, finality depth: 19 blocks
- Bech32 prefix: lgm (lgmvaloper for validators)
- DID format: did:lgm:{32-hex}
- Hand-written types with json.Marshal (not proto codec)
- 4-tier validators: Apprentice(0)/Verified(1)/Bonded(2)/Guardian(3)
- ABCI++ vote extensions for VRF-based validator selection
- Test runner: go test, static analysis: go vet
```

---

## Session Continuity Protocol

### At Session Start
1. `git checkout main && git pull origin main`
2. Read `.claude/memory/CURRENT_STATE.md` for project state
3. Read `.claude/memory/DECISIONS.md` for key decisions made
4. Check `.claude/memory/BLOCKED.md` for any blockers
5. Skim `.claude/memory/EUREKA.md` for deep patterns (23 insights)
6. For file lookup: `.claude/memory/INDEX.md` indexes all 76 memory files
7. Pick work (issue, roadmap task, or human-assigned)
8. Create feature branch: `git checkout -b {type}-{scope}-{slug}`

### During Session
- Commit frequently to your feature branch
- Run `make pr-check` before opening a PR
- Log important decisions to DECISIONS.md

### At Session End
- Open PR via `gh pr create` (or mark existing PR as ready)
- If incomplete: mark PR as draft, label `continuation-welcome`
- After PR merges: update `.claude/memory/CURRENT_STATE.md` on main

---

## Testing Commands

```bash
# Build all packages
go build ./...

# Run module tests
go test ./x/... -count=1

# Run app tests
go test ./app/... -count=1

# Static analysis
go vet ./...

# Full PR validation
make pr-check
```

---

## Important Files to Read

When context is limited, prioritize reading:
1. `app/app.go` - Full module wiring
2. `x/knowledge/keeper/keeper.go` - Knowledge module keeper
3. `x/knowledge/types/types.go` - Core types
4. `x/staking/types/types.go` - Validator tiers
5. `app/vote_extensions.go` - ABCI++ VRF integration

---

## Resources

- Architecture: `docs/BLOCKCHAIN_ARCHITECTURE.md`
- Cosmos Migration: `docs/COSMOS_MIGRATION.md`
- Agent accounts: `docs/AGENT_ACCOUNTS.md`
- Eureka moments: `.claude/memory/EUREKA.md` (23 deep insights)
- Memory index: `.claude/memory/INDEX.md` (all 76 files cataloged)
- Insight tree: `.claude/memory/INSIGHT_TREE.md` (hierarchical synthesis)
