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
// P0-2 fix: passes contested flag so ContestedVerificationsCorrect is tracked.
func (h StakingVerificationHooks) AfterVerificationCompleted(ctx sdk.Context, validatorAddr, _ string, correct, contested bool) error {
	return h.k.RecordVerification(ctx, validatorAddr, correct, contested)
}
