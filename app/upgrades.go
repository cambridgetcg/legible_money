package app

// This file establishes the upgrade handler pattern for Legible Money.
// Each chain upgrade gets a named handler registered here. The x/upgrade
// module's PreBlocker halts the chain when the scheduled height is reached,
// validators swap binaries, and the handler runs on the first block of the
// new binary — executing any state migrations needed.
//
// To add a new upgrade:
//  1. Define a const for the upgrade name (e.g. UpgradeNameV3)
//  2. Register a handler in registerUpgradeHandlers()
//  3. If the upgrade adds/removes stores, add an UpgradeStoreLoader entry
//  4. Submit a consensus-category LIP with the upgrade name + target height
//
// The LIP governance → x/upgrade bridge (in x/gov) automatically calls
// ScheduleUpgrade when an accepted LIP carries an upgrade plan.

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// Upgrade name constants. Each binary release that requires state migration
// should define a unique name here.
const (
	// UpgradeNameV2 is a placeholder for the first post-genesis upgrade.
	// When the v2 binary is ready, implement the migration logic in the
	// handler below and submit a consensus LIP to schedule the upgrade.
	UpgradeNameV2 = "v2"
)

// registerUpgradeHandlers registers all upgrade handlers. Called during app
// construction, after RegisterServices and before LoadLatestVersion.
func (app *LegibleApp) registerUpgradeHandlers() {
	// ---- v2 upgrade handler (skeleton) ----
	//
	// This handler runs exactly once, on the first block after the chain
	// halts at the scheduled upgrade height and validators restart with
	// the v2 binary.
	//
	// To implement:
	//   1. Increment ConsensusVersion() in any module whose state schema changed
	//   2. Register per-module migrations via app.configurator.RegisterMigration()
	//   3. Add migration logic below (or in x/<module>/migrations/v2/)
	//
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameV2,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			// RunMigrations compares each module's current ConsensusVersion
			// with the version stored on-chain. If a module bumped its version,
			// RunMigrations calls the registered migration function for that
			// module's version transition (e.g. v1→v2).
			return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
		},
	)

	// ---- UpgradeStoreLoader ----
	//
	// If an upgrade adds or removes KV stores, configure the store loader
	// so the multistore knows about the change BEFORE the upgrade handler
	// runs. This is required for any upgrade that adds new modules.
	//
	// Example (uncomment when v2 adds/removes stores):
	//
	//   upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	//   if err != nil {
	//       panic("failed to read upgrade info: " + err.Error())
	//   }
	//   if upgradeInfo.Name == UpgradeNameV2 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
	//       storeUpgrades := storetypes.StoreUpgrades{
	//           Added:   []string{"new_module_store_key"},
	//           Deleted: []string{"deprecated_module_store_key"},
	//       }
	//       app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	//   }
	//

	// Suppress unused import warnings for the skeleton
	_ = storetypes.StoreUpgrades{}
	_ = upgradetypes.Plan{}
}
