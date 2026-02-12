package gov

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

	"github.com/legible-money/legible/x/gov/keeper"
	"github.com/legible-money/legible/x/gov/types"
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
	gs := types.DefaultGenesis()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic("failed to marshal default genesis: " + err.Error())
	}
	return bz
}

// ValidateGenesis performs genesis state validation.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis: %w", types.ModuleName, err)
	}
	return gs.Validate()
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
}

// InitGenesis initializes the module from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal %s genesis: %v", types.ModuleName, err))
	}
	am.keeper.InitGenesis(ctx, gs)
}

// ExportGenesis exports the module's genesis state.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return am.keeper.ExportGenesisJSON(ctx)
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock checks voting deadlines and advances expired proposals.
func (am AppModule) BeginBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := uint64(ctx.BlockHeight())
	params := am.keeper.GetParams(ctx)

	// Check LIPs in voting stage for expired voting periods
	votingLIPs := am.keeper.GetLIPsByStage(ctx, types.StageVoting)
	for _, lip := range votingLIPs {
		if lip.VotingEndsAt > 0 && currentHeight >= lip.VotingEndsAt {
			// Voting period expired — tally and resolve
			am.tallyAndResolveLIP(ctx, lip, params)
		}
	}

	// Check disbursements in voting stage for expired voting periods
	am.keeper.IterateDisbursements(ctx, func(disb types.DisbursementProposal) bool {
		if disb.Stage == string(types.DisbStageVoting) && disb.VotingEndsAt > 0 && currentHeight >= disb.VotingEndsAt {
			am.tallyAndResolveDisbursement(ctx, disb, params)
		}
		return false
	})

	return nil
}

// EndBlock executes end-block logic.
func (am AppModule) EndBlock(ctx context.Context) error {
	return nil
}

// tallyAndResolveLIP tallies votes and moves a LIP to accepted/rejected.
// If a consensus-category LIP is accepted and has an attached upgrade plan,
// ScheduleUpgrade is called on the upgrade keeper to halt the chain at the target height.
func (am AppModule) tallyAndResolveLIP(ctx sdk.Context, lip types.LIP, params types.Params) {
	totalWeight := addBigIntStrings(lip.SupportWeight, lip.OpposeWeight, lip.AbstainWeight)
	if totalWeight == "0" {
		lip.Stage = string(types.StageRejected)
	} else {
		supportBps := bpsFraction(lip.SupportWeight, totalWeight)
		if supportBps >= params.SupportThresholdBps {
			lip.Stage = string(types.StageAccepted)
		} else {
			lip.Stage = string(types.StageRejected)
		}
	}
	lip.UpdatedAtBlock = uint64(ctx.BlockHeight())
	am.keeper.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_resolved",
			sdk.NewAttribute("lip_id", lip.Id),
			sdk.NewAttribute("stage", lip.Stage),
		),
	)

	// Bridge to x/upgrade: if an accepted consensus LIP has an upgrade plan, schedule it
	if lip.Stage == string(types.StageAccepted) {
		if plan, found := am.keeper.GetUpgradePlan(ctx, lip.Id); found {
			uk := am.keeper.GetUpgradeKeeper()
			if uk != nil {
				if err := uk.ScheduleUpgrade(ctx, plan); err != nil {
					am.keeper.Logger(ctx).Error("failed to schedule upgrade from LIP",
						"lip_id", lip.Id,
						"upgrade_name", plan.Name,
						"error", err,
					)
				} else {
					ctx.EventManager().EmitEvent(
						sdk.NewEvent(
							"upgrade_scheduled",
							sdk.NewAttribute("lip_id", lip.Id),
							sdk.NewAttribute("upgrade_name", plan.Name),
							sdk.NewAttribute("height", fmt.Sprintf("%d", plan.Height)),
						),
					)
					am.keeper.Logger(ctx).Info("software upgrade scheduled via LIP governance",
						"lip_id", lip.Id,
						"upgrade_name", plan.Name,
						"height", plan.Height,
					)
				}
			}
		}
	}
}

// tallyAndResolveDisbursement tallies votes and moves a disbursement to approved/rejected.
func (am AppModule) tallyAndResolveDisbursement(ctx sdk.Context, disb types.DisbursementProposal, params types.Params) {
	totalWeight := addBigIntStrings(disb.SupportWeight, disb.OpposeWeight, disb.AbstainWeight)
	if totalWeight == "0" {
		disb.Stage = string(types.DisbStageRejected)
	} else {
		supportBps := bpsFraction(disb.SupportWeight, totalWeight)
		if supportBps >= params.DisbursementSupportBps {
			disb.Stage = string(types.DisbStageApproved)
			disb.ApprovedAtBlock = uint64(ctx.BlockHeight())
			disb.TimelockExpiry = disb.ApprovedAtBlock + params.DisbursementTimelockBlocks
		} else {
			disb.Stage = string(types.DisbStageRejected)
		}
	}
	am.keeper.SetDisbursement(ctx, disb)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"disbursement_resolved",
			sdk.NewAttribute("disbursement_id", disb.Id),
			sdk.NewAttribute("stage", string(disb.Stage)),
		),
	)
}

// --- BigInt string helpers ---

func addBigIntStrings(vals ...string) string {
	var total uint64
	for _, v := range vals {
		n := parseBigIntString(v)
		total += n
	}
	return fmt.Sprintf("%d", total)
}

func bpsFraction(numerator, denominator string) uint64 {
	n := parseBigIntString(numerator)
	d := parseBigIntString(denominator)
	if d == 0 {
		return 0
	}
	return (n * 10000) / d
}

func parseBigIntString(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	return n
}

