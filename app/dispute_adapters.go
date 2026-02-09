package app

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	disputestypes "github.com/legible-money/legible/x/disputes/types"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	vestingrewardskeeper "github.com/legible-money/legible/x/vesting_rewards/keeper"
)

// ---------- KnowledgeKeeper Adapter ----------

// DisputeKnowledgeAdapter adapts the knowledge keeper to the disputes module's
// KnowledgeKeeper interface, converting between knowledge types and dispute types.
type DisputeKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

func NewDisputeKnowledgeAdapter(k knowledgekeeper.Keeper) *DisputeKnowledgeAdapter {
	return &DisputeKnowledgeAdapter{keeper: k}
}

func (a *DisputeKnowledgeAdapter) GetFact(ctx sdk.Context, factID string) (disputestypes.Fact, bool) {
	kFact, found := a.keeper.GetFact(ctx, factID)
	if !found {
		return disputestypes.Fact{}, false
	}
	return convertKnowledgeFact(kFact), true
}

func (a *DisputeKnowledgeAdapter) SetFact(ctx sdk.Context, fact disputestypes.Fact) {
	kFact, found := a.keeper.GetFact(ctx, fact.Id)
	if !found {
		return
	}
	// Update mutable fields from the dispute resolution
	kFact.Status = fact.Status
	kFact.Confidence = fact.Confidence
	kFact.ChallengeCount = fact.ChallengeCount
	a.keeper.SetFact(ctx, kFact)
}

func (a *DisputeKnowledgeAdapter) GetReferringFacts(ctx sdk.Context, factID string) []string {
	refs := a.keeper.GetIncomingReferences(ctx, factID)
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.FromFactId)
	}
	return ids
}

func (a *DisputeKnowledgeAdapter) GetClaim(ctx sdk.Context, claimID string) (disputestypes.Claim, bool) {
	kClaim, found := a.keeper.GetClaim(ctx, claimID)
	if !found {
		return disputestypes.Claim{}, false
	}
	return disputestypes.Claim{
		Id:        kClaim.Id,
		Submitter: kClaim.Submitter,
		Status:    kClaim.Status,
	}, true
}

func convertKnowledgeFact(f knowledgetypes.Fact) disputestypes.Fact {
	return disputestypes.Fact{
		Id:                 f.Id,
		ClaimText:          f.ClaimText,
		Domain:             f.Domain,
		EpistemicCategory:  f.EpistemicCategory,
		Confidence:         f.Confidence,
		Status:             f.Status,
		Submitter:          f.Submitter,
		References:         f.References,
		VerifiedAtHeight:   f.VerifiedAtHeight,
		VerifierCount:      f.VerifierCount,
		LogicZone:          f.LogicZone,
		CreatedAtHeight:    f.CreatedAtHeight,
		LastVerifiedHeight: f.LastVerifiedHeight,
		ChallengeCount:     f.ChallengeCount,
	}
}

// ---------- StakingKeeper Adapter ----------

// DisputeStakingAdapter adapts the legible staking keeper to the disputes module's
// StakingKeeper interface.
type DisputeStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewDisputeStakingAdapter(k legiblestakingkeeper.Keeper) *DisputeStakingAdapter {
	return &DisputeStakingAdapter{keeper: k}
}

func (a *DisputeStakingAdapter) GetActiveValidatorInfos(ctx context.Context) []disputestypes.ValidatorInfo {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validators := a.keeper.GetActiveValidatorSet(sdkCtx)

	infos := make([]disputestypes.ValidatorInfo, 0, len(validators))
	for _, v := range validators {
		stake := parseStakeToUint64(v.TotalStake)
		infos = append(infos, disputestypes.ValidatorInfo{
			Address: v.OperatorAddress,
			Stake:   stake,
			Tier:    uint32(v.Tier),
		})
	}
	return infos
}

func (a *DisputeStakingAdapter) GetValidatorInfo(ctx context.Context, operatorAddr string) (disputestypes.ValidatorInfo, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operatorAddr)
	if !found {
		return disputestypes.ValidatorInfo{}, false
	}
	return disputestypes.ValidatorInfo{
		Address: v.OperatorAddress,
		Stake:   parseStakeToUint64(v.TotalStake),
		Tier:    uint32(v.Tier),
	}, true
}

func (a *DisputeStakingAdapter) GetEffectiveStake(ctx context.Context, operatorAddr string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operatorAddr)
	if !found {
		return 0
	}
	return parseStakeToUint64(v.TotalStake)
}

// ---------- StakeQuery Adapter ----------
// Satisfies gov and research StakingKeeper interface:
//   GetValidatorStake(ctx sdk.Context, address string) (string, error)

type StakeQueryAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewStakeQueryAdapter(k legiblestakingkeeper.Keeper) *StakeQueryAdapter {
	return &StakeQueryAdapter{keeper: k}
}

func (a *StakeQueryAdapter) GetValidatorStake(ctx sdk.Context, address string) (string, error) {
	v, found := a.keeper.GetValidator(ctx, address)
	if !found {
		return "0", nil
	}
	if v.TotalStake != "" {
		return v.TotalStake, nil
	}
	return "0", nil
}

// ---------- VestingKeeper Adapter ----------

// DisputeVestingAdapter adapts the vesting_rewards keeper to the disputes module's
// VestingKeeper interface, converting ClawbackRecord types across module boundaries.
type DisputeVestingAdapter struct {
	keeper vestingrewardskeeper.Keeper
}

func NewDisputeVestingAdapter(k vestingrewardskeeper.Keeper) *DisputeVestingAdapter {
	return &DisputeVestingAdapter{keeper: k}
}

var _ disputestypes.VestingKeeper = (*DisputeVestingAdapter)(nil)

func (a *DisputeVestingAdapter) PauseVestingByClaimId(ctx sdk.Context, claimId string) error {
	return a.keeper.PauseVestingByClaimId(ctx, claimId)
}

func (a *DisputeVestingAdapter) ResumeVestingByClaimId(ctx sdk.Context, claimId string) error {
	return a.keeper.ResumeVestingByClaimId(ctx, claimId)
}

func (a *DisputeVestingAdapter) FalsifyClaim(ctx sdk.Context, claimId string, challenger string) (disputestypes.ClawbackRecord, error) {
	record, err := a.keeper.FalsifyClaim(ctx, claimId, challenger)
	if err != nil {
		return disputestypes.ClawbackRecord{}, err
	}
	return disputestypes.ClawbackRecord{
		Id:                record.Id,
		ClaimId:           record.VestingId,
		ReleasedClawback:  record.ReleasedClawback,
		UnvestedForfeited: record.UnvestedForfeited,
		ChallengerReward:  record.ChallengerReward,
		BlockHeight:       record.BlockHeight,
	}, nil
}

// parseStakeToUint64 converts a bigint string to uint64 (capped at MaxUint64).
func parseStakeToUint64(stakeStr string) uint64 {
	if stakeStr == "" {
		return 0
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(stakeStr, 10); !ok {
		return 0
	}
	if !bi.IsUint64() {
		return ^uint64(0) // MaxUint64
	}
	return bi.Uint64()
}
