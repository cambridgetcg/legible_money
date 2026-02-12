# Software Upgrade Guide

How Legible Money handles chain upgrades via LIP governance and the Cosmos SDK x/upgrade module.

## Architecture

```
  LIP Submitted (consensus category)
       │
       ▼
  Upgrade Plan Attached (MsgAttachUpgradePlan)
       │
       ▼
  LIP Voting (quadratic votes)
       │
       ├── Rejected → upgrade plan discarded
       │
       └── Accepted → x/upgrade.ScheduleUpgrade(name, height)
                            │
                            ▼
                  Chain halts at target height
                            │
                            ▼
                  Validators swap binary
                            │
                            ▼
                  SetUpgradeHandler runs migrations
                            │
                            ▼
                  Chain resumes with new binary
```

## Dual Governance Systems

Legible Money has two governance modules:

| Feature | SDK x/gov ("gov") | LGM x/gov ("lgm_gov") |
|---------|-------------------|----------------------|
| Store key | `gov` | `lgm_gov` |
| Voting | 1-token-1-vote | Quadratic (cost = votes^2) |
| Proposals | Text, params, upgrade | LIPs (4 categories) + disbursements |
| Upgrades | Direct `MsgSoftwareUpgrade` | LIP → AttachUpgradePlan → bridge |
| Use case | SDK compatibility, IBC | Protocol governance |

Both are registered in `app/app.go` and run independently.

## How to Perform a Software Upgrade

### 1. Prepare the new binary

Build and test the new version. Ensure `ConsensusVersion()` is incremented in any module whose state schema changed.

### 2. Register the upgrade handler

In `app/upgrades.go`, add the handler:

```go
const UpgradeNameV3 = "v3"

app.UpgradeKeeper.SetUpgradeHandler(
    UpgradeNameV3,
    func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
        // Custom migration logic here
        return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
    },
)
```

### 3. Submit a consensus LIP

```
MsgSubmitLIP {
    proposer: "lgm1...",
    title: "Upgrade to v3",
    abstract: "Adds cross-chain knowledge queries...",
    category: "consensus"
}
```

### 4. Attach the upgrade plan

```
MsgAttachUpgradePlan {
    proposer: "lgm1...",
    lip_id: "LIP-42",
    upgrade_name: "v3",       // Must match UpgradeNameV3
    height: 1000000,          // Target halt height
    info: "https://..."       // Optional binaries URL
}
```

### 5. Advance through LIP lifecycle

The LIP follows the standard lifecycle: draft → review → last_call → voting.

### 6. LIP accepted → upgrade scheduled

When the LIP is accepted, `tallyAndResolveLIP` in `x/gov/module.go` calls `ScheduleUpgrade` on the x/upgrade keeper via the `LGMGovUpgradeAdapter`.

### 7. Chain halts at target height

The x/upgrade module's `PreBlocker` (first in BeginBlocker ordering) halts the chain at the scheduled height.

### 8. Validators swap binaries and restart

Validators install the new binary (which has the `SetUpgradeHandler` for "v3") and restart. `RunMigrations` executes per-module state migrations.

## Adding/Removing Stores

If an upgrade adds or removes KV stores (new modules), configure `UpgradeStoreLoader` in `app/upgrades.go`:

```go
upgradeInfo, _ := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
if upgradeInfo.Name == UpgradeNameV3 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
    storeUpgrades := storetypes.StoreUpgrades{
        Added:   []string{"new_module"},
        Deleted: []string{"old_module"},
    }
    app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
}
```

## Module Migration Pattern

When a module's state schema changes:

1. Increment `ConsensusVersion()` in `x/<module>/module.go` (e.g. 1 → 2)
2. Create `x/<module>/migrations/v2/migrate.go` with the migration function
3. Register in the upgrade handler or via `app.configurator.RegisterMigration()`

## Zero-Height Genesis Export

For chain restarts without an upgrade:

```bash
legible export --for-zero-height > genesis.json
```

This calls `prepForZeroHeightGenesis` which:
- Withdraws all outstanding rewards and commissions
- Clears validator slash events and historical rewards
- Resets delegation and signing info start heights to 0
- Sweeps remaining rewards into the community pool

## Key Files

| File | Purpose |
|------|---------|
| `app/upgrades.go` | Upgrade handlers and store loader config |
| `app/gov_upgrade_adapter.go` | LIP governance → x/upgrade bridge adapter |
| `x/gov/module.go` | `tallyAndResolveLIP` calls ScheduleUpgrade |
| `x/gov/keeper/msg_server.go` | `AttachUpgradePlan` handler |
| `x/gov/keeper/state.go` | Upgrade plan CRUD storage |
| `x/gov/types/expected_keepers.go` | `UpgradeKeeper` interface definition |
