package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QualificationVerificationHooks implements knowledge.VerificationHooks to update
// domain qualification metrics after each verification round. This fixes the bug
// where RecordVerificationOutcome was never called — qualification metrics were frozen.
type QualificationVerificationHooks struct {
	k Keeper
}

// NewQualificationVerificationHooks creates a new QualificationVerificationHooks.
func NewQualificationVerificationHooks(k Keeper) QualificationVerificationHooks {
	return QualificationVerificationHooks{k: k}
}

// AfterVerificationCompleted delegates to the qualification keeper's
// RecordVerificationOutcome to update domain-specific accuracy metrics,
// consecutive counts, and suspension triggers.
func (h QualificationVerificationHooks) AfterVerificationCompleted(ctx sdk.Context, validatorAddr, domain string, correct, _ bool) error {
	return h.k.RecordVerificationOutcome(ctx, validatorAddr, domain, correct)
}
