// Package oracle provides the oracle prediction layer module for Zerone.
// Agents stake probabilistic claims about future states. Accuracy is scored
// using a Horizon-Weighted Brier Score with recency bias. Every resolved
// prediction generates a TrainingExample for the Kwok model training corpus.
//
// Design: Alpha (AI Companion) + Beta (AI Engineer), 2026-03-23.
package oracle

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/legible-money/legible/x/oracle/keeper"
	"github.com/legible-money/legible/x/oracle/types"
)

const ConsensusVersion = 1

// AppModuleBasic implements the module.AppModuleBasic interface.
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	return json.RawMessage(`{}`)
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ interface{}, _ json.RawMessage) error {
	return nil
}

// AppModule implements the module.AppModule interface.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{keeper: k}
}

func (AppModule) IsAppModule()       {}
func (AppModule) IsOnePerModuleType() {}

func (am AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// EndBlock scans for predictions past their resolution deadline and marks them expired.
func (am AppModule) EndBlock(ctx sdk.Context) error {
	height := uint64(ctx.BlockHeight())
	am.keeper.IteratePredictions(ctx, func(p types.Prediction) bool {
		if p.Status == types.PredictionActive && height > p.ResolutionBlock {
			p.Status = types.PredictionExpired
			am.keeper.SetPrediction(ctx, p)
		}
		return false
	})
	return nil
}

// RegisterInterfaces is a no-op for now (proto-generated tx types pending).
func (AppModuleBasic) RegisterInterfaces(_ codectypes.InterfaceRegistry) {}

// RegisterGRPCGatewayRoutes is a no-op (REST gateway pending).
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

var _ module.AppModuleBasic = AppModuleBasic{}
