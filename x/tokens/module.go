package tokens

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

	"github.com/legible-money/legible/x/tokens/keeper"
	"github.com/legible-money/legible/x/tokens/types"
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
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
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
// Sweeps expired tokens and moves them to the RAI holding pool.
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := am.keeper.GetParams(sdkCtx)

	totalSwept, accountsProcessed := am.keeper.SweepExpiredTokens(sdkCtx, params.SweepBatchSize)
	if accountsProcessed > 0 {
		am.keeper.Logger(sdkCtx).Debug("BeginBlock sweep completed",
			"swept", totalSwept,
			"accounts", accountsProcessed,
		)
	}

	// Distribute RAI if enough blocks have passed and pool has balance
	if am.keeper.ShouldDistributeRAI(sdkCtx, params.RaiDistributionRate) {
		recipients := am.keeper.GetEligibleRAIRecipients(sdkCtx)
		if len(recipients) > 0 {
			dist, err := am.keeper.DistributeRAI(sdkCtx, recipients)
			if err != nil {
				am.keeper.Logger(sdkCtx).Error("RAI distribution failed", "error", err)
			} else {
				am.keeper.Logger(sdkCtx).Info("BeginBlock RAI distribution",
					"distribution_id", dist.Id,
					"recipients", dist.RecipientCount,
					"total", dist.TotalAmount,
				)
			}
		}
	}

	return nil
}

// EndBlock executes end-block logic.
func (am AppModule) EndBlock(ctx context.Context) error {
	return nil
}
