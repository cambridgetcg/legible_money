package app

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hometypes "github.com/legible-money/legible/x/home/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	partnershipskeeper "github.com/legible-money/legible/x/partnerships/keeper"
	partnershipstypes "github.com/legible-money/legible/x/partnerships/types"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
)

// PartnershipsStakingAdapter wraps the Legible staking keeper to satisfy
// the partnerships module's StakingKeeper interface.
type PartnershipsStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

// NewPartnershipsStakingAdapter creates a new adapter.
func NewPartnershipsStakingAdapter(k legiblestakingkeeper.Keeper) *PartnershipsStakingAdapter {
	return &PartnershipsStakingAdapter{keeper: k}
}

var _ partnershipstypes.StakingKeeper = (*PartnershipsStakingAdapter)(nil)

// GetLastValidatorPower returns the last recorded power for a validator.
// Maps from the Legible staking keeper's TotalStake to int64 power.
func (a *PartnershipsStakingAdapter) GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (int64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operator.String())
	if !found {
		return 0, nil
	}
	// Convert TotalStake string to int64 power (in ulgm)
	bi := new(big.Int)
	if _, ok := bi.SetString(v.TotalStake, 10); !ok {
		return 0, nil
	}
	// Divide by 1e6 to get power in whole LGM tokens
	divisor := new(big.Int).SetInt64(1_000_000)
	bi.Div(bi, divisor)
	if !bi.IsInt64() {
		return 1<<62 - 1, nil // cap at large value
	}
	return bi.Int64(), nil
}

// ---------- Knowledge → Partnerships Adapter ----------

// KnowledgePartnershipsAdapter wraps the partnerships keeper to satisfy
// the knowledge module's PartnershipsKeeper interface.
// Converts between knowledge types.Partnership and partnerships types.Partnership.
type KnowledgePartnershipsAdapter struct {
	keeper partnershipskeeper.Keeper
}

// NewKnowledgePartnershipsAdapter creates a new adapter.
func NewKnowledgePartnershipsAdapter(keeper partnershipskeeper.Keeper) *KnowledgePartnershipsAdapter {
	return &KnowledgePartnershipsAdapter{keeper: keeper}
}

var _ knowledgetypes.PartnershipsKeeper = (*KnowledgePartnershipsAdapter)(nil)

// GetActivePartnershipForAddress returns the first active partnership for addr.
func (a *KnowledgePartnershipsAdapter) GetActivePartnershipForAddress(ctx sdk.Context, addr string) (*knowledgetypes.Partnership, bool) {
	p, found := a.keeper.GetActivePartnershipForAddress(ctx, addr)
	if !found {
		return nil, false
	}
	return toKnowledgePartnership(p), true
}

// DistributeReward routes a reward through the partnership economics.
func (a *KnowledgePartnershipsAdapter) DistributeReward(
	ctx sdk.Context,
	partnership *knowledgetypes.Partnership,
	grossAmount string,
	sourceType string,
) (humanShare, agentShare, commonPotAdd string, err error) {
	// Look up the real partnership by ID (adapter converts knowledge mirror type → actual type)
	realP, found := a.keeper.GetPartnership(ctx, partnership.Id)
	if !found {
		return "", "", "", partnershipstypes.ErrPartnershipNotFound
	}
	return a.keeper.DistributeReward(ctx, realP, grossAmount, sourceType)
}

// ---------- Home → Partnerships Adapter ----------

// HomePartnershipAdapter wraps the partnerships keeper to satisfy
// the home module's PartnershipKeeper interface.
type HomePartnershipAdapter struct {
	keeper partnershipskeeper.Keeper
}

// NewHomePartnershipAdapter creates a new adapter.
func NewHomePartnershipAdapter(keeper partnershipskeeper.Keeper) *HomePartnershipAdapter {
	return &HomePartnershipAdapter{keeper: keeper}
}

var _ hometypes.PartnershipKeeper = (*HomePartnershipAdapter)(nil)

// GetPartnership returns partnership info by ID, converting to the home module's mirror type.
func (a *HomePartnershipAdapter) GetPartnership(ctx sdk.Context, id string) (*hometypes.PartnershipInfo, bool) {
	p, found := a.keeper.GetPartnership(ctx, id)
	if !found {
		return nil, false
	}
	return &hometypes.PartnershipInfo{
		Id:               p.Id,
		HumanAddr:        p.HumanAddr,
		AgentAddr:        p.AgentAddr,
		Status:           p.Status,
		CommonPotBalance: p.CommonPotBalance,
	}, true
}

// CalculateAmountTier determines the deliberation tier for an amount vs pot balance.
func (a *HomePartnershipAdapter) CalculateAmountTier(amount string, potBalance string) string {
	return a.keeper.CalculateAmountTier(amount, potBalance)
}

// toKnowledgePartnership converts a partnerships.Partnership to the knowledge module's mirror type.
func toKnowledgePartnership(p *partnershipstypes.Partnership) *knowledgetypes.Partnership {
	return &knowledgetypes.Partnership{
		Id:               p.Id,
		HumanAddr:        p.HumanAddr,
		AgentAddr:        p.AgentAddr,
		Status:           p.Status,
		LockTier:         p.LockTier,
		SplitHumanBps:    p.SplitHumanBps,
		SplitAgentBps:    p.SplitAgentBps,
		CommonPotBalance: p.CommonPotBalance,
		TotalEarned:      p.TotalEarned,
	}
}
