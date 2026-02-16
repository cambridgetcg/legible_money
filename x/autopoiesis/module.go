package autopoiesis

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

	"github.com/legible-money/legible/x/autopoiesis/keeper"
	"github.com/legible-money/legible/x/autopoiesis/types"
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

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("failed to marshal default genesis: " + err.Error())
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Params.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

func (a AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// AppModule implements the AppModule interface.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
	}
}

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}

func (am AppModule) RegisterServices(cfg module.Configurator) {}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genState types.GenesisState
	if err := json.Unmarshal(data, &genState); err != nil {
		panic("failed to unmarshal genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, genState)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}
	return bz
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock — observation is deferred to EndBlock (one-block delay from TS spec).
func (am AppModule) BeginBlock(ctx context.Context) error {
	return nil
}

// EndBlock runs the OBSERVE → COMPUTE → ADAPT cycle at epoch boundaries.
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.CollectAndAdapt(sdkCtx)
	return nil
}
