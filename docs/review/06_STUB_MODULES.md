# Stub Modules — Phase 2 Roadmap

> Five modules are scaffolded with minimal code (keeper stub, module shell, store key). They compile and participate in the module manager but have no business logic yet. One module (vesting_rewards) is partially implemented.
>
> **Related**: [00 — Module Overview](./00_MODULE_OVERVIEW.md) lists all modules with status. These stubs correspond to TS chain subsystems documented in the project architecture.

---

## Stub Module Pattern

All stubs follow this consistent structure:

```
x/{module}/
├── types/
│   └── keys.go          # ModuleName, StoreKey, RouterKey constants
├── keeper/
│   └── keeper.go        # Keeper struct with storeKey + cdc, empty Init/ExportGenesis
└── module.go            # AppModuleBasic + AppModule, empty RegisterServices,
                         # BeginBlock/EndBlock with TODO comments
```

Each stub:
- Has a store key allocated in `app/app.go`
- Is registered in the module manager
- Has ordered BeginBlock/EndBlock slots (no-ops)
- Has InitGenesis/ExportGenesis (empty genesis)
- Compiles cleanly and doesn't affect other modules

---

## x/bvm — Blockchain Virtual Machine

**Store key**: `bvm`
**Planned purpose**: Execute self-invoking contracts triggered by DMSS scheduling or manual invocation.

**What exists in TS chain**: Full BVM with 100+ opcodes, stack machine, gas metering per opcode, state bridge, DMSS contract integration, STATICCALL mode.

**What's in the Go stub**:
- Keeper with empty `InitGenesis`/`ExportGenesis`
- Module shell with `BeginBlock` comment: "Execute DMSS-triggered contract invocations"
- Module shell with `EndBlock` comment: "Process scheduled contract calls, gas metering finalization"

**Phase 2 scope**: Contract registry, invocation engine, opcode gas costs, state sandboxing.

---

## x/channels — Payment Channels

**Store key**: `channels`
**Planned purpose**: Off-chain bi-directional payment channels for claim settlement disputes.

**What exists in TS chain**: Full payment channel system with state transitions, dispute windows, settlement.

**What's in the Go stub**:
- Keeper with empty genesis
- Module shell with `BeginBlock` comment: "Process dispute timeout windows"
- Module shell with `EndBlock` comment: "Settle expired channels, finalize dispute resolutions"

**Phase 2 scope**: Channel creation, funding, state updates, dispute timeout, settlement, IBC cross-chain atomic swaps.

---

## x/claiming_pot — Collaborative Reward Pooling

**Store key**: `claiming_pot`
**Planned purpose**: Agents contribute to reward pots that mature over time and distribute to members.

**What exists in TS chain**: Full claiming pot system with contribution deadlines, maturity periods, member management.

**What's in the Go stub**:
- Keeper with empty genesis
- Module shell with `BeginBlock` comment: "Check pot maturity, process contribution deadlines"
- Module shell with `EndBlock` comment: "Distribute finalized pot rewards to members"

**Phase 2 scope**: Pot creation, contribution, maturity tracking, reward distribution, vesting integration.

---

## x/tree — Tree of Life (Project/Task Hierarchy)

**Store key**: `tree`
**Planned purpose**: Hierarchical project/task system where agents propose tasks, others contribute, and completed tasks earn rewards.

**What exists in TS chain**: Full tree system with seedling → sapling → mature lifecycle, milestone tracking.

**What's in the Go stub**:
- Keeper with empty genesis
- Module shell with `BeginBlock` comment: "Process task deadlines, seed germination checks"
- Module shell with `EndBlock` comment: "Finalize completed tasks, distribute project rewards"

**Phase 2 scope**: Task creation, contribution tracking, completion verification, milestone-based disbursement.

---

## x/vesting_rewards — Time-Locked Rewards

**Store key**: `vesting_rewards`
**Status**: **Partially implemented** (more complete than other stubs)

**What's implemented**:
- `VestingSchedule` type: beneficiary, vesting amount, start/end blocks, claimable amount, status
- `VestingStatus` enum: active, completed, revoked, expired
- `Params`: cliff period, vesting period, slashing rules
- `DefaultGenesis()` with parameter validation
- **Working EndBlocker**: Iterates active vesting schedules, calls `UpdateClaimableAmount()` per schedule per block
- `MsgClaimVestedRewards` and `MsgFalsifyClaim` handler signatures (not yet wired to service descriptor)

**What's missing**:
- Service registration (RegisterServices is empty)
- CLI commands
- gRPC query endpoints
- Full message handler implementations (type signatures exist but may not be fully wired)

---

## Migration Priority

Based on the TS chain's dependency graph and usage:

| Priority | Module | Reason |
|----------|--------|--------|
| High | `vesting_rewards` | Already partially implemented, needed for reward alignment |
| High | `bvm` | Core to self-invoking contracts and DMSS scheduling |
| Medium | `tree` | Project management for AI agent task coordination |
| Medium | `claiming_pot` | Collaborative economics for agent groups |
| Low | `channels` | Off-chain optimization, not critical for initial launch |
