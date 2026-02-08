package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

// AddToRAIPool adds swept tokens to the RAI holding pool.
func (k Keeper) AddToRAIPool(ctx sdk.Context, amount string) {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return
	}

	rai := k.GetRAIAccount(ctx)

	balance := new(big.Int)
	balance.SetString(rai.Balance, 10)
	balance.Add(balance, amtInt)
	rai.Balance = balance.String()

	totalReceived := new(big.Int)
	totalReceived.SetString(rai.TotalReceived, 10)
	totalReceived.Add(totalReceived, amtInt)
	rai.TotalReceived = totalReceived.String()

	k.SetRAIAccount(ctx, rai)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"rai_pool_increased",
			sdk.NewAttribute("amount", amount),
			sdk.NewAttribute("new_balance", rai.Balance),
		),
	)
}

// DistributeRAI distributes the RAI pool equally among eligible recipients.
// Returns the distribution record or an error.
func (k Keeper) DistributeRAI(ctx sdk.Context, recipients []string) (types.RAIDistribution, error) {
	if len(recipients) == 0 {
		return types.RAIDistribution{}, fmt.Errorf("no recipients for RAI distribution")
	}

	rai := k.GetRAIAccount(ctx)
	balance := new(big.Int)
	balance.SetString(rai.Balance, 10)

	if balance.Sign() <= 0 {
		return types.RAIDistribution{}, fmt.Errorf("RAI pool is empty")
	}

	recipientCount := big.NewInt(int64(len(recipients)))
	perRecipient := new(big.Int).Div(balance, recipientCount)

	if perRecipient.Sign() <= 0 {
		return types.RAIDistribution{}, fmt.Errorf("RAI amount per recipient is zero")
	}

	// Total actually distributed (may be slightly less than balance due to integer division)
	totalDistributed := new(big.Int).Mul(perRecipient, recipientCount)

	height := uint64(ctx.BlockHeight())

	// Credit each recipient at the current period
	for _, recipient := range recipients {
		if err := k.CreditBalance(ctx, recipient, perRecipient.String(), height); err != nil {
			k.Logger(ctx).Error("failed to distribute RAI to recipient",
				"recipient", recipient, "error", err)
			continue
		}
	}

	// Update RAI account
	balance.Sub(balance, totalDistributed)
	rai.Balance = balance.String()

	totalDist := new(big.Int)
	totalDist.SetString(rai.TotalDistributed, 10)
	totalDist.Add(totalDist, totalDistributed)
	rai.TotalDistributed = totalDist.String()
	rai.LastDistBlock = height

	k.SetRAIAccount(ctx, rai)

	distID := generateDistributionID(height)
	dist := types.RAIDistribution{
		Id:             distID,
		Block:          height,
		TotalAmount:    totalDistributed.String(),
		RecipientCount: uint32(len(recipients)),
		PerRecipient:   perRecipient.String(),
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"rai_distributed",
			sdk.NewAttribute("distribution_id", distID),
			sdk.NewAttribute("total_amount", totalDistributed.String()),
			sdk.NewAttribute("recipient_count", fmt.Sprintf("%d", len(recipients))),
			sdk.NewAttribute("per_recipient", perRecipient.String()),
		),
	)

	k.Logger(ctx).Info("RAI distributed",
		"distribution_id", distID,
		"total", totalDistributed.String(),
		"recipients", len(recipients),
	)

	return dist, nil
}

// IsEligibleForRAI checks if an address is eligible for RAI distribution.
// Eligibility requires having an active expirable account with non-zero balance.
func (k Keeper) IsEligibleForRAI(ctx sdk.Context, address string) bool {
	acct, found := k.GetExpirableAccount(ctx, address)
	if !found {
		return false
	}

	// Must have active (non-zero, non-expired) balance
	balance := k.GetBalance(ctx, address)
	bal := new(big.Int)
	bal.SetString(balance, 10)
	if bal.Sign() <= 0 {
		return false
	}

	// Foundation and research accounts are not eligible for RAI
	if acct.AccountType == "foundation" || acct.AccountType == "research" {
		return false
	}

	return true
}

// generateDistributionID creates a deterministic distribution ID from the block height.
func generateDistributionID(height uint64) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("LGM.rai.dist.v1:%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}
