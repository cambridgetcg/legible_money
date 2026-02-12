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

	balance := parseBigInt(rai.Balance, "RAI.Balance", k.Logger(ctx))
	balance.Add(balance, amtInt)
	rai.Balance = balance.String()

	totalReceived := parseBigInt(rai.TotalReceived, "RAI.TotalReceived", k.Logger(ctx))
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
//
// Only the amount actually credited to recipients is debited from the RAI pool.
// If a credit fails (e.g., below dust threshold), that recipient's share stays
// in the pool for the next distribution cycle.
func (k Keeper) DistributeRAI(ctx sdk.Context, recipients []string) (types.RAIDistribution, error) {
	if len(recipients) == 0 {
		return types.RAIDistribution{}, fmt.Errorf("no recipients for RAI distribution")
	}

	rai := k.GetRAIAccount(ctx)
	balance := parseBigInt(rai.Balance, "RAI.Balance", k.Logger(ctx))

	if balance.Sign() <= 0 {
		return types.RAIDistribution{}, fmt.Errorf("RAI pool is empty")
	}

	recipientCount := big.NewInt(int64(len(recipients)))
	perRecipient := new(big.Int).Div(balance, recipientCount)

	if perRecipient.Sign() <= 0 {
		return types.RAIDistribution{}, fmt.Errorf("RAI amount per recipient is zero")
	}

	height := uint64(ctx.BlockHeight())

	// Credit each recipient, tracking what was actually distributed.
	// Failed credits are NOT counted — their share stays in the RAI pool.
	actualDistributed := new(big.Int)
	var actualRecipientCount uint32

	for _, recipient := range recipients {
		if err := k.CreditBalance(ctx, recipient, perRecipient.String(), height); err != nil {
			k.Logger(ctx).Error("failed to distribute RAI to recipient",
				"recipient", recipient, "error", err)
			continue
		}
		actualDistributed.Add(actualDistributed, perRecipient)
		actualRecipientCount++
	}

	if actualDistributed.Sign() <= 0 {
		return types.RAIDistribution{}, fmt.Errorf("all RAI credit attempts failed")
	}

	// Only debit from RAI pool what was actually credited
	balance.Sub(balance, actualDistributed)
	rai.Balance = balance.String()

	totalDist := parseBigInt(rai.TotalDistributed, "RAI.TotalDistributed", k.Logger(ctx))
	totalDist.Add(totalDist, actualDistributed)
	rai.TotalDistributed = totalDist.String()
	rai.LastDistBlock = height

	k.SetRAIAccount(ctx, rai)

	distID := generateDistributionID(height)
	dist := types.RAIDistribution{
		Id:             distID,
		Block:          height,
		TotalAmount:    actualDistributed.String(),
		RecipientCount: actualRecipientCount,
		PerRecipient:   perRecipient.String(),
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"rai_distributed",
			sdk.NewAttribute("distribution_id", distID),
			sdk.NewAttribute("total_amount", actualDistributed.String()),
			sdk.NewAttribute("recipient_count", fmt.Sprintf("%d", actualRecipientCount)),
			sdk.NewAttribute("per_recipient", perRecipient.String()),
		),
	)

	k.Logger(ctx).Info("RAI distributed",
		"distribution_id", distID,
		"actual_total", actualDistributed.String(),
		"actual_recipients", actualRecipientCount,
		"attempted_recipients", len(recipients),
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

// GetEligibleRAIRecipients returns all addresses eligible for RAI distribution.
func (k Keeper) GetEligibleRAIRecipients(ctx sdk.Context) []string {
	var recipients []string
	k.IterateExpirableAccounts(ctx, func(acct types.ExpirableAccount) bool {
		if k.IsEligibleForRAI(ctx, acct.Address) {
			recipients = append(recipients, acct.Address)
		}
		return false
	})
	return recipients
}

// ShouldDistributeRAI checks whether enough blocks have passed since the last
// RAI distribution and the pool has a non-zero balance.
func (k Keeper) ShouldDistributeRAI(ctx sdk.Context, distributionRate uint64) bool {
	if distributionRate == 0 {
		return false
	}
	rai := k.GetRAIAccount(ctx)
	balance := new(big.Int)
	balance.SetString(rai.Balance, 10)
	if balance.Sign() <= 0 {
		return false
	}
	height := uint64(ctx.BlockHeight())
	return height-rai.LastDistBlock >= distributionRate
}

// generateDistributionID creates a deterministic distribution ID from the block height.
func generateDistributionID(height uint64) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("LGM.rai.dist.v1:%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))
}
