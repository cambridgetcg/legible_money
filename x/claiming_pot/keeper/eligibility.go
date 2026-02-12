package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// EvaluateEligibility checks if a claimer meets pot eligibility rules.
// Returns (eligible, reason).
func (k Keeper) EvaluateEligibility(ctx sdk.Context, pot types.ClaimingPot, claimer string) (bool, string) {
	elig := pot.Config.Eligibility

	// No eligibility restrictions configured — everyone is eligible
	if elig == nil {
		return true, ""
	}

	// 1. Allowlist check (fast path) — if non-empty, claimer must be in it
	if len(elig.Allowlist) > 0 {
		found := false
		for _, addr := range elig.Allowlist {
			if addr == claimer {
				found = true
				break
			}
		}
		if !found {
			return false, "address not in allowlist"
		}
	}

	// 2. Max balance check — if set, claimer must have balance <= max
	if elig.MaxBalance != "" && k.bankKeeper != nil {
		maxBal, ok := new(big.Int).SetString(elig.MaxBalance, 10)
		if ok && maxBal.Sign() > 0 {
			claimerAddr, err := sdk.AccAddressFromBech32(claimer)
			if err == nil {
				balance := k.bankKeeper.GetBalance(ctx, claimerAddr, "ulgm")
				currentBal := balance.Amount.BigInt()
				if currentBal.Cmp(maxBal) > 0 {
					return false, "balance exceeds maximum"
				}
			}
		}
	}

	// 3. Account age — deferred to future integration (always passes)
	// 4. Activity score — deferred to future integration (always passes)
	// 5. Excluded types — deferred to future integration (always passes)
	// 6. Verification level — deferred to future integration (always passes)

	return true, ""
}
