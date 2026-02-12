package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/legible-money/legible/x/auth/migrations/v2"
)

// Migrator handles in-place store migrations for the auth module.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from v1 to v2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.keeper.storeService)
}
