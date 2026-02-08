package app

import (
	"context"
	"encoding/hex"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
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
