package keeper

// Migrator handles module version migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator creates a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}
