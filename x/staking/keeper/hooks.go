package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingVerificationHooks implements knowledge.VerificationHooks to update
// validator reputation and verification stats after each verification round.
type StakingVerificationHooks struct {
	k Keeper
}

// NewStakingVerificationHooks creates a new StakingVerificationHooks.
func NewStakingVerificationHooks(k Keeper) StakingVerificationHooks {
	return StakingVerificationHooks{k: k}
}

// AfterVerificationCompleted delegates to the staking keeper's RecordVerification
// to update validator stats (verification count, reputation, tier transitions).
func (h StakingVerificationHooks) AfterVerificationCompleted(ctx sdk.Context, validatorAddr, _ string, correct bool) error {
	return h.k.RecordVerification(ctx, validatorAddr, correct)
}
