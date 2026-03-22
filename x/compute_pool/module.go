package compute_pool

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/legible-money/legible/x/compute_pool/keeper"
	"github.com/legible-money/legible/x/compute_pool/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

// ---------- AppModuleBasic ----------

// AppModuleBasic implements the AppModuleBasic interface for the compute_pool module.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's types on the legacy amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)) //nolint:errcheck
}

// DefaultGenesis returns the module's default genesis state as JSON.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("failed to marshal default genesis: " + err.Error())
	}
	return bz
}

// ValidateGenesis validates the module's genesis state.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// ---------- AppModule ----------

// AppModule implements the AppModule interface for the compute_pool module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object.
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

// RegisterServices registers the module's gRPC services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
}

// InitGenesis initializes the module's genesis state.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genState types.GenesisState
	if err := json.Unmarshal(data, &genState); err != nil {
		panic("failed to unmarshal genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, &genState)
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

// BeginBlock executes start-of-block logic for the compute_pool module.
//
// Operations:
//  1. Expire stale compute credits
//  2. Expire unconfirmed redemption tickets
//  3. Expire unassigned/overdue compute tasks
//  4. Check provider liveness (suspend inactive)
//  5. Apply pending price changes
//  6. At epoch boundary: finalize epoch statistics
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Expire credits past their ExpiresAt block
	am.keeper.ExpireCredits(sdkCtx)

	// 2. Expire unconfirmed redemption tickets
	am.keeper.ExpireRedemptionTickets(sdkCtx)

	// 3. Expire unassigned/overdue compute tasks
	am.keeper.ExpireUnassignedTasks(sdkCtx)

	// 4. Check provider heartbeats, complete unbondings
	am.keeper.CheckProviderLiveness(sdkCtx)

	// 5. Apply pending price changes
	am.keeper.ApplyPendingPriceChanges(sdkCtx)

	// 6. Finalize epoch stats at epoch boundary
	am.keeper.FinalizeEpoch(sdkCtx)

	return nil
}

// EndBlock executes end-of-block logic (no-op for compute_pool).
func (am AppModule) EndBlock(ctx context.Context) error {
	return nil
}
