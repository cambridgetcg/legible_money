package knowledge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

// AppModuleBasic implements the AppModuleBasic interface.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("failed to marshal default genesis: " + err.Error())
	}
	return bz
}

// ValidateGenesis performs genesis state validation.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// Phase 2: Register query service gateway routes
}

// GetTxCmd returns the module's tx CLI command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil // Phase 2: CLI commands
}

// GetQueryCmd returns the module's query CLI command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil // Phase 2: CLI commands
}

// AppModule implements the AppModule interface.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule.
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterExtendedMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	migrator := keeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to register %s migration: %v", types.ModuleName, err))
	}
}

// InitGenesis initializes the module from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genState types.GenesisState
	if err := json.Unmarshal(data, &genState); err != nil {
		panic("failed to unmarshal genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, genState)
}

// ExportGenesis exports the module's genesis state.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock executes begin-block logic.
// In adversarial mode: structural eval → challenge resolution → window expiry → growth.
// In legacy mode: advances verification round phases.
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := am.keeper.BeginBlockerAdversarial(sdkCtx); err != nil {
		am.keeper.Logger(sdkCtx).Error("failed to run BeginBlocker",
			"height", sdkCtx.BlockHeight(), "error", err)
	}

	return nil
}

// EndBlock executes end-block logic.
// 1. Cleans up completed rounds older than 100 blocks.
// 2. Expires stale facts (max 50 per block, amortized).
// 3. Scans for facts needing reverification (max 10 per block).
// 4. Applies cycle penalties every 1000 blocks.
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.CleanupExpiredRounds(sdkCtx, 100)

	// Expire facts that have decayed below threshold or exceeded staleness
	expired := am.keeper.ExpireStaleFactsInEndBlocker(sdkCtx, 50)
	if expired > 0 {
		am.keeper.Logger(sdkCtx).Info("expired stale facts",
			"count", expired,
			"height", sdkCtx.BlockHeight())
	}

	// Scan for facts needing reverification (emit events for external tooling)
	triggered := am.keeper.TriggerReverificationScan(sdkCtx, 10)
	if triggered > 0 {
		am.keeper.Logger(sdkCtx).Info("reverification needed",
			"count", triggered,
			"height", sdkCtx.BlockHeight())
	}

	// Apply cycle penalties every 1000 blocks
	if sdkCtx.BlockHeight()%1000 == 0 {
		penalized := am.keeper.ApplyCyclePenalties(sdkCtx)
		if penalized > 0 {
			am.keeper.Logger(sdkCtx).Info("cycle penalties applied",
				"count", penalized,
				"height", sdkCtx.BlockHeight())
		}
	}

	return nil
}
