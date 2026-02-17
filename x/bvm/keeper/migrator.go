package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/bvm/types"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to version 2.
// Sets BvmVersion=1 on all existing contracts that have BvmVersion==0
// (deployed before the versioning upgrade).
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	m.keeper.IterateContracts(ctx, func(contract types.DeployedContract) bool {
		if contract.BvmVersion == 0 {
			contract.BvmVersion = 1
			m.keeper.SetContract(ctx, contract)
		}
		return false
	})
	return nil
}
