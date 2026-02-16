package vesting_rewards

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

	"github.com/legible-money/legible/x/vesting_rewards/keeper"
	"github.com/legible-money/legible/x/vesting_rewards/types"
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
	cdc.RegisterConcrete(&types.MsgClaimVesting{}, "vesting_rewards/ClaimVesting", nil)
	cdc.RegisterConcrete(&types.MsgFalsifyVesting{}, "vesting_rewards/FalsifyVesting", nil)
}

// RegisterInterfaces registers the module's interface types.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesis()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default genesis: %v", err))
	}
	return bz
}

// ValidateGenesis performs genesis state validation.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}
	return (&gs).Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd returns the module's tx CLI command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd returns the module's query CLI command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
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
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	migrator := keeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to register %s migration: %v", types.ModuleName, err))
	}
}

// InitGenesis initializes the module from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		// Fall back to defaults if genesis data is empty/invalid
		defGs := types.DefaultGenesis()
		am.keeper.InitGenesis(ctx, defGs)
		return
	}
	am.keeper.InitGenesis(ctx, &gs)
}

// ExportGenesis exports the module's genesis state.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal genesis state: %v", err))
	}
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock executes begin-block logic.
// 1. Routes 22% of accumulated transaction fees to research fund.
// 2. Distributes block rewards to the block producer with 22% research tax.
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := sdkCtx.BlockHeight()

	// Skip genesis block
	if height <= 1 {
		return nil
	}

	// Route accumulated transaction fees: 22% to research fund, 78% stays in fee_collector.
	// Must run BEFORE x/distribution's BeginBlocker sweeps fee_collector to validators.
	if err := am.keeper.RouteFees(sdkCtx); err != nil {
		am.keeper.Logger(sdkCtx).Error("failed to route fees to research fund",
			"block", height, "error", err)
	}

	// Get block proposer from header
	proposerAddr := sdkCtx.BlockHeader().ProposerAddress
	if len(proposerAddr) == 0 {
		return nil
	}
	producer := sdk.AccAddress(proposerAddr).String()

	// Get active validator count for reward scaling
	var activeValidatorCount uint32
	if sk := am.keeper.GetStakingKeeper(); sk != nil {
		activeValidatorCount = sk.GetActiveValidatorCount(sdkCtx)
	}

	// Check if block has transactions (PoT: 0% for empty blocks)
	hasTransactions := len(sdkCtx.BlockHeader().DataHash) > 0 && activeValidatorCount > 0

	_, err := am.keeper.DistributeBlockReward(sdkCtx, producer, activeValidatorCount, hasTransactions)
	if err != nil {
		am.keeper.Logger(sdkCtx).Error("failed to distribute block reward",
			"block", height, "error", err)
	}

	return nil
}

// EndBlock executes end-block logic.
// Processes vesting releases for active schedules.
//
// L3 DESIGN NOTE: This EndBlocker proactively updates claimable amounts each block.
// This keeps query results accurate but costs gas proportional to active schedule count.
// An alternative "lazy evaluation" approach (only compute on ClaimVestedRewards) would
// be more gas-efficient at scale but less query-accurate. The proactive approach is chosen
// because: (1) accuracy matters for governance decisions referencing vesting state,
// (2) the exponential release curve means most updates are small/zero and cheap,
// (3) if schedule count grows large, this can be rate-limited to every N blocks.
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Update claimable amounts for active vesting schedules
	schedules := am.keeper.GetAllActiveVestingSchedules(sdkCtx)
	for _, schedule := range schedules {
		if schedule.Status == string(types.VestingStatusActive) {
			if err := am.keeper.UpdateClaimableAmount(sdkCtx, schedule.Id); err != nil {
				am.keeper.Logger(sdkCtx).Error("failed to update claimable amount",
					"vesting_id", schedule.Id, "error", err)
			}
		}
	}

	return nil
}
