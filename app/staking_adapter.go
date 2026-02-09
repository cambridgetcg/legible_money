package app

import (
	"context"
	"encoding/hex"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	evidencemgmtkeeper "github.com/legible-money/legible/x/evidence_mgmt/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	vestingrewardstypes "github.com/legible-money/legible/x/vesting_rewards/types"
)

// StakingKeeperAdapter wraps the Legible staking keeper to satisfy
// the knowledge module's StakingKeeper interface.
type StakingKeeperAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

// NewStakingKeeperAdapter creates a new adapter.
func NewStakingKeeperAdapter(keeper legiblestakingkeeper.Keeper) *StakingKeeperAdapter {
	return &StakingKeeperAdapter{keeper: keeper}
}

var _ knowledgetypes.StakingKeeper = (*StakingKeeperAdapter)(nil)

// GetActiveValidatorInfos returns active validators as knowledge module ValidatorInfo.
func (a *StakingKeeperAdapter) GetActiveValidatorInfos(ctx context.Context) []knowledgetypes.ValidatorInfo {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validators := a.keeper.GetActiveValidatorSet(sdkCtx)

	infos := make([]knowledgetypes.ValidatorInfo, 0, len(validators))
	for _, v := range validators {
		pubkey, _ := hex.DecodeString(v.ConsensusPubkey)
		totalStake := bigIntStringToUint64(v.TotalStake)

		infos = append(infos, knowledgetypes.ValidatorInfo{
			Address:    v.OperatorAddress,
			Stake:      totalStake,
			Tier:       uint32(v.Tier),
			PublicKey:  pubkey,
			Reputation: v.Reputation,
		})
	}
	return infos
}

// GetValidatorInfo returns info for a single validator.
func (a *StakingKeeperAdapter) GetValidatorInfo(ctx context.Context, operatorAddr string) (knowledgetypes.ValidatorInfo, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operatorAddr)
	if !found {
		return knowledgetypes.ValidatorInfo{}, false
	}

	pubkey, _ := hex.DecodeString(v.ConsensusPubkey)
	totalStake := bigIntStringToUint64(v.TotalStake)

	return knowledgetypes.ValidatorInfo{
		Address:    v.OperatorAddress,
		Stake:      totalStake,
		Tier:       uint32(v.Tier),
		PublicKey:  pubkey,
		Reputation: v.Reputation,
	}, true
}

// GetEffectiveStake returns the effective selection stake for VRF.
func (a *StakingKeeperAdapter) GetEffectiveStake(ctx context.Context, operatorAddr string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operatorAddr)
	if !found {
		return 0
	}
	stakeStr := a.keeper.GetEffectiveSelectionStake(sdkCtx, v)
	return bigIntStringToUint64(stakeStr)
}

// GetTotalStake returns total active stake.
func (a *StakingKeeperAdapter) GetTotalStake(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validators := a.keeper.GetActiveValidatorSet(sdkCtx)

	var total uint64
	for _, v := range validators {
		total += bigIntStringToUint64(v.TotalStake)
	}
	return total
}

// RecordVerification delegates to the staking keeper.
func (a *StakingKeeperAdapter) RecordVerification(ctx context.Context, operatorAddr string, correct bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.RecordVerification(sdkCtx, operatorAddr, correct)
}

// SlashValidator delegates to the staking keeper.
func (a *StakingKeeperAdapter) SlashValidator(ctx context.Context, operatorAddr string, amount string, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.SlashValidator(sdkCtx, operatorAddr, amount, reason)
}

// VestingStakingAdapter wraps the Legible staking keeper to satisfy
// the vesting_rewards module's StakingKeeper interface.
type VestingStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewVestingStakingAdapter(keeper legiblestakingkeeper.Keeper) *VestingStakingAdapter {
	return &VestingStakingAdapter{keeper: keeper}
}

var _ vestingrewardstypes.StakingKeeper = (*VestingStakingAdapter)(nil)

// GetActiveValidatorCount returns the number of active validators.
func (a *VestingStakingAdapter) GetActiveValidatorCount(ctx sdk.Context) uint32 {
	validators := a.keeper.GetActiveValidatorSet(ctx)
	return uint32(len(validators))
}

// ---------- Evidence Management Keeper Adapter ----------

// EvidenceMgmtKeeperAdapter wraps the evidence_mgmt keeper to satisfy
// the knowledge module's EvidenceMgmtKeeper interface.
// It bridges x/evidence_mgmt types to x/knowledge/types.EvidenceRecord.
type EvidenceMgmtKeeperAdapter struct {
	keeper evidencemgmtkeeper.Keeper
}

// NewEvidenceMgmtKeeperAdapter creates a new adapter.
func NewEvidenceMgmtKeeperAdapter(keeper evidencemgmtkeeper.Keeper) *EvidenceMgmtKeeperAdapter {
	return &EvidenceMgmtKeeperAdapter{keeper: keeper}
}

var _ knowledgetypes.EvidenceMgmtKeeper = (*EvidenceMgmtKeeperAdapter)(nil)

// GetEvidenceByClaimId returns all evidence for a claim, mapped to EvidenceRecord.
func (a *EvidenceMgmtKeeperAdapter) GetEvidenceByClaimId(ctx sdk.Context, claimId string) []knowledgetypes.EvidenceRecord {
	evidences := a.keeper.GetEvidenceByClaimId(ctx, claimId)
	records := make([]knowledgetypes.EvidenceRecord, len(evidences))
	for i, e := range evidences {
		records[i] = knowledgetypes.EvidenceRecord{
			Id:           e.Id,
			ClaimId:      e.ClaimId,
			EvidenceType: string(e.EvidenceType),
			Status:       string(e.Status),
			Reliability:  e.Reliability,
		}
	}
	return records
}

// ComputeReliability delegates to the evidence_mgmt keeper.
func (a *EvidenceMgmtKeeperAdapter) ComputeReliability(ctx sdk.Context, evidenceId string) (uint64, error) {
	return a.keeper.ComputeReliability(ctx, evidenceId)
}

// bigIntStringToUint64 converts a bigint string to uint64, capping at MaxUint64.
func bigIntStringToUint64(s string) uint64 {
	bi := new(big.Int)
	if _, ok := bi.SetString(s, 10); !ok {
		return 0
	}
	if !bi.IsUint64() {
		return ^uint64(0) // MaxUint64
	}
	return bi.Uint64()
}
