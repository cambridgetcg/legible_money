package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VerificationHooks defines callbacks triggered after verification outcomes.
// Modules that need to react to verification results (e.g. staking for reputation,
// qualification for domain metrics) implement this interface and register via
// Keeper.SetVerificationHooks.
type VerificationHooks interface {
	AfterVerificationCompleted(ctx sdk.Context, validatorAddr, domain string, correct, contested bool) error
}

// MultiVerificationHooks composes multiple hook implementations into one.
// Standard Cosmos SDK hooks pattern — each hook is called in order.
type MultiVerificationHooks []VerificationHooks

// NewMultiVerificationHooks creates a new MultiVerificationHooks from the given hooks.
func NewMultiVerificationHooks(hooks ...VerificationHooks) MultiVerificationHooks {
	return hooks
}

// AfterVerificationCompleted calls AfterVerificationCompleted on all composed hooks.
// Errors are logged but do not prevent subsequent hooks from being called.
func (h MultiVerificationHooks) AfterVerificationCompleted(ctx sdk.Context, validatorAddr, domain string, correct, contested bool) error {
	for _, hook := range h {
		if err := hook.AfterVerificationCompleted(ctx, validatorAddr, domain, correct, contested); err != nil {
			ctx.Logger().Error("verification hook failed",
				"validator", validatorAddr,
				"domain", domain,
				"error", err,
			)
		}
	}
	return nil
}
