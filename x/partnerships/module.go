package partnerships

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

	"github.com/legible-money/legible/x/partnerships/keeper"
	"github.com/legible-money/legible/x/partnerships/types"
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
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)) //nolint:errcheck
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

	migrator := keeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to register %s migration: %v", types.ModuleName, err))
	}
}

// combinedGenesisExport wraps the proto GenesisState plus hand-written consortium data for JSON export.
// Uses json.RawMessage to avoid copying the proto mutex.
type combinedGenesisExport struct {
	Base      json.RawMessage     `json:"base"`
	Consortia []*types.Consortium `json:"consortia,omitempty"`
}

// combinedGenesisImport is the import-side counterpart.
type combinedGenesisImport struct {
	Consortia []*types.Consortium `json:"consortia,omitempty"`
}

// InitGenesis initializes the module from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	// Unmarshal proto GenesisState (tolerates extra fields)
	var genState types.GenesisState
	if err := json.Unmarshal(data, &genState); err != nil {
		panic("failed to unmarshal genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, &genState)

	// Unmarshal consortium data
	var extra combinedGenesisImport
	if err := json.Unmarshal(data, &extra); err == nil {
		am.keeper.InitConsortiumGenesis(ctx, extra.Consortia)
	}
}

// ExportGenesis exports the module's genesis state.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	baseBz, err := json.Marshal(genState)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}

	// Merge consortium data into the JSON
	consortia := am.keeper.ExportConsortiumGenesis(ctx)
	if len(consortia) == 0 {
		return baseBz
	}

	// Unmarshal base into map, add consortia, re-marshal
	var baseMap map[string]json.RawMessage
	if err := json.Unmarshal(baseBz, &baseMap); err != nil {
		panic("failed to unmarshal base genesis: " + err.Error())
	}
	consortiaBz, err := json.Marshal(consortia)
	if err != nil {
		panic("failed to marshal consortia: " + err.Error())
	}
	baseMap["consortia"] = consortiaBz
	combined, err := json.Marshal(baseMap)
	if err != nil {
		panic("failed to marshal combined genesis: " + err.Error())
	}
	return combined
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock executes begin-block logic.
//
// - Expire formation windows that have passed
// - Lift expired safety freezes
// - Expire unresolved coercion signals
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Expire pending formations past their window
	am.keeper.ExpireFormations(sdkCtx)

	// 2. Lift expired safety freezes and restore partnership status
	am.keeper.LiftExpiredFreezes(sdkCtx)

	// 3. Expire coercion signals past their review period
	am.keeper.ExpireCoercionSignals(sdkCtx)

	// 4. Expire consortium formations past their window
	am.keeper.ExpireConsortiumFormations(sdkCtx)

	return nil
}

// EndBlock executes end-block logic.
//
// - Settle cooling partnerships that have completed their cooling period
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.SettleCoolingPartnerships(sdkCtx)
	am.keeper.SettleCoolingConsortia(sdkCtx)
	return nil
}
