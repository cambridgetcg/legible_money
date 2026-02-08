package keeper

import (
	"fmt"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

// CreditBalance credits tokens to the current period for an address.
// This is used for system rewards, minting, etc. where tokens receive fresh expiration.
func (k Keeper) CreditBalance(ctx sdk.Context, address string, amount string, blockHeight uint64) error {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return fmt.Errorf("%w: %s", types.ErrInvalidAmount, amount)
	}

	params := k.GetParams(ctx)
	if params.PeriodBlocks == 0 {
		return fmt.Errorf("%w: period_blocks is zero", types.ErrInvalidPeriod)
	}

	period := blockHeight / params.PeriodBlocks
	return k.CreditBalanceToSpecificPeriod(ctx, address, amount, period)
}

// CreditBalanceToSpecificPeriod credits tokens to a specific period for an address.
// Used for period-preserving transfers (sender's periods are maintained on the receiver).
func (k Keeper) CreditBalanceToSpecificPeriod(ctx sdk.Context, address string, amount string, period uint64) error {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return fmt.Errorf("%w: %s", types.ErrInvalidAmount, amount)
	}

	params := k.GetParams(ctx)

	// Check dust threshold
	dust := new(big.Int)
	dust.SetString(params.DustThreshold, 10)
	if amtInt.Cmp(dust) < 0 {
		return fmt.Errorf("%w: amount %s below threshold %s", types.ErrDustAmount, amount, params.DustThreshold)
	}

	acct, found := k.GetExpirableAccount(ctx, address)
	if !found {
		acct = types.ExpirableAccount{
			Address:        address,
			TotalBalance:   "0",
			PeriodBalances: []types.PeriodBalance{},
			EarliestPeriod: period,
			LatestPeriod:   period,
			TotalExpired:   "0",
			AccountType:    "standard",
		}
	}

	// Find or create the period balance entry
	expiresAt := k.PeriodToExpirationBlock(period, params)
	periodFound := false
	for i := range acct.PeriodBalances {
		if acct.PeriodBalances[i].Period == period {
			bal := new(big.Int)
			bal.SetString(acct.PeriodBalances[i].Balance, 10)
			bal.Add(bal, amtInt)
			acct.PeriodBalances[i].Balance = bal.String()
			periodFound = true
			break
		}
	}

	if !periodFound {
		acct.PeriodBalances = append(acct.PeriodBalances, types.PeriodBalance{
			Period:    period,
			Balance:   amount,
			ExpiresAt: expiresAt,
			Swept:     false,
		})
		// Keep sorted by period
		sort.Slice(acct.PeriodBalances, func(i, j int) bool {
			return acct.PeriodBalances[i].Period < acct.PeriodBalances[j].Period
		})
	}

	// Update aggregate fields
	totalBal := new(big.Int)
	totalBal.SetString(acct.TotalBalance, 10)
	totalBal.Add(totalBal, amtInt)
	acct.TotalBalance = totalBal.String()

	// Update earliest/latest
	if period < acct.EarliestPeriod || acct.EarliestPeriod == 0 {
		acct.EarliestPeriod = period
	}
	if period > acct.LatestPeriod {
		acct.LatestPeriod = period
	}

	// Register in sweep index
	k.AddToSweepIndex(ctx, period, address)

	k.SetExpirableAccount(ctx, acct)
	return nil
}

// DebitBalance debits tokens from an account using FIFO (oldest non-expired period first).
// Returns an error if the account has insufficient active balance.
func (k Keeper) DebitBalance(ctx sdk.Context, address string, amount string) error {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return fmt.Errorf("%w: %s", types.ErrInvalidAmount, amount)
	}

	acct, found := k.GetExpirableAccount(ctx, address)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrAccountNotFound, address)
	}

	params := k.GetParams(ctx)
	currentPeriod := k.GetCurrentPeriod(ctx)

	// First, compute total active balance (excluding expired)
	activeBalance := new(big.Int)
	for _, pb := range acct.PeriodBalances {
		if pb.Swept || k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			continue
		}
		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)
		activeBalance.Add(activeBalance, bal)
	}

	if activeBalance.Cmp(amtInt) < 0 {
		return fmt.Errorf("%w: active balance %s < requested %s",
			types.ErrInsufficientBalance, activeBalance.String(), amount)
	}

	// FIFO debit from oldest non-expired period
	remaining := new(big.Int).Set(amtInt)
	var updatedPeriods []types.PeriodBalance

	for _, pb := range acct.PeriodBalances {
		if pb.Swept || k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			updatedPeriods = append(updatedPeriods, pb)
			continue
		}

		if remaining.Sign() <= 0 {
			updatedPeriods = append(updatedPeriods, pb)
			continue
		}

		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)

		if bal.Cmp(remaining) <= 0 {
			// Consume entire period balance
			remaining.Sub(remaining, bal)
			pb.Balance = "0"
		} else {
			// Partial debit
			bal.Sub(bal, remaining)
			pb.Balance = bal.String()
			remaining.SetInt64(0)
		}

		// Only keep non-zero balances
		pbBal := new(big.Int)
		pbBal.SetString(pb.Balance, 10)
		if pbBal.Sign() > 0 {
			updatedPeriods = append(updatedPeriods, pb)
		}
	}

	acct.PeriodBalances = updatedPeriods

	// Recalculate total balance
	totalBal := new(big.Int)
	totalBal.SetString(acct.TotalBalance, 10)
	totalBal.Sub(totalBal, amtInt)
	acct.TotalBalance = totalBal.String()

	// Update earliest/latest period
	k.recalculatePeriodBounds(&acct)

	k.SetExpirableAccount(ctx, acct)
	return nil
}

// GetBalance returns the total active balance (excluding expired) for an address.
func (k Keeper) GetBalance(ctx sdk.Context, address string) string {
	acct, found := k.GetExpirableAccount(ctx, address)
	if !found {
		return "0"
	}

	params := k.GetParams(ctx)
	currentPeriod := k.GetCurrentPeriod(ctx)

	activeBalance := new(big.Int)
	for _, pb := range acct.PeriodBalances {
		if pb.Swept || k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			continue
		}
		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)
		activeBalance.Add(activeBalance, bal)
	}

	return activeBalance.String()
}

// GetPeriodBalances returns all period balances for an address.
func (k Keeper) GetPeriodBalances(ctx sdk.Context, address string) []types.PeriodBalance {
	acct, found := k.GetExpirableAccount(ctx, address)
	if !found {
		return nil
	}
	return acct.PeriodBalances
}

// TransferPreservingPeriods performs a FIFO debit from sender and credits to
// receiver at the sender's original periods (period preservation).
// This prevents the ping-pong exploit where transferring tokens resets expiration.
func (k Keeper) TransferPreservingPeriods(ctx sdk.Context, from, to, amount string) error {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return fmt.Errorf("%w: %s", types.ErrInvalidAmount, amount)
	}

	fromAcct, found := k.GetExpirableAccount(ctx, from)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrAccountNotFound, from)
	}

	params := k.GetParams(ctx)
	currentPeriod := k.GetCurrentPeriod(ctx)

	// Compute active balance
	activeBalance := new(big.Int)
	for _, pb := range fromAcct.PeriodBalances {
		if pb.Swept || k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			continue
		}
		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)
		activeBalance.Add(activeBalance, bal)
	}

	if activeBalance.Cmp(amtInt) < 0 {
		return fmt.Errorf("%w: active balance %s < requested %s",
			types.ErrInsufficientBalance, activeBalance.String(), amount)
	}

	// Collect the period→amount pairs we will debit (FIFO)
	type periodDebit struct {
		period uint64
		amount *big.Int
	}
	var debits []periodDebit

	remaining := new(big.Int).Set(amtInt)
	var updatedPeriods []types.PeriodBalance

	for _, pb := range fromAcct.PeriodBalances {
		if pb.Swept || k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			updatedPeriods = append(updatedPeriods, pb)
			continue
		}

		if remaining.Sign() <= 0 {
			updatedPeriods = append(updatedPeriods, pb)
			continue
		}

		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)

		if bal.Cmp(remaining) <= 0 {
			// Consume entire period balance
			debits = append(debits, periodDebit{period: pb.Period, amount: new(big.Int).Set(bal)})
			remaining.Sub(remaining, bal)
			pb.Balance = "0"
		} else {
			// Partial debit
			debits = append(debits, periodDebit{period: pb.Period, amount: new(big.Int).Set(remaining)})
			bal.Sub(bal, remaining)
			pb.Balance = bal.String()
			remaining.SetInt64(0)
		}

		pbBal := new(big.Int)
		pbBal.SetString(pb.Balance, 10)
		if pbBal.Sign() > 0 {
			updatedPeriods = append(updatedPeriods, pb)
		}
	}

	// Update sender
	fromAcct.PeriodBalances = updatedPeriods
	totalBal := new(big.Int)
	totalBal.SetString(fromAcct.TotalBalance, 10)
	totalBal.Sub(totalBal, amtInt)
	fromAcct.TotalBalance = totalBal.String()
	k.recalculatePeriodBounds(&fromAcct)
	k.SetExpirableAccount(ctx, fromAcct)

	// Credit receiver at sender's original periods
	for _, d := range debits {
		if err := k.CreditBalanceToSpecificPeriod(ctx, to, d.amount.String(), d.period); err != nil {
			return fmt.Errorf("failed to credit receiver for period %d: %w", d.period, err)
		}
	}

	return nil
}

// SweepExpiredTokens batch-sweeps expired token balances and moves them to the RAI pool.
// Returns the total amount swept and number of accounts processed.
func (k Keeper) SweepExpiredTokens(ctx sdk.Context, maxAccounts uint32) (string, uint32) {
	params := k.GetParams(ctx)
	currentPeriod := k.GetCurrentPeriod(ctx)

	totalSwept := new(big.Int)
	var accountsProcessed uint32

	// Iterate expired periods in the sweep index
	type sweepEntry struct {
		period  uint64
		address string
	}
	var entries []sweepEntry

	k.IterateExpiredPeriods(ctx, currentPeriod, params.PeriodsUntilExpiry, func(period uint64, address string) bool {
		entries = append(entries, sweepEntry{period: period, address: address})
		return uint32(len(entries)) >= maxAccounts
	})

	for _, entry := range entries {
		acct, found := k.GetExpirableAccount(ctx, entry.address)
		if !found {
			k.RemoveFromSweepIndex(ctx, entry.period, entry.address)
			continue
		}

		swept := k.sweepAccountPeriod(ctx, &acct, entry.period, currentPeriod, params)
		if swept.Sign() > 0 {
			totalSwept.Add(totalSwept, swept)
			k.SetExpirableAccount(ctx, acct)
		}

		k.RemoveFromSweepIndex(ctx, entry.period, entry.address)
		accountsProcessed++
	}

	// Add swept amount to RAI pool
	if totalSwept.Sign() > 0 {
		k.AddToRAIPool(ctx, totalSwept.String())

		k.Logger(ctx).Info("swept expired tokens",
			"amount", totalSwept.String(),
			"accounts", accountsProcessed,
			"period", currentPeriod,
		)
	}

	return totalSwept.String(), accountsProcessed
}

// sweepAccountPeriod marks expired period balances as swept and returns the total swept amount.
func (k Keeper) sweepAccountPeriod(ctx sdk.Context, acct *types.ExpirableAccount, period, currentPeriod uint64, params types.Params) *big.Int {
	swept := new(big.Int)

	for i := range acct.PeriodBalances {
		pb := &acct.PeriodBalances[i]
		if pb.Period != period {
			continue
		}
		if pb.Swept {
			continue
		}
		if !k.IsExpired(pb.Period, currentPeriod, params.PeriodsUntilExpiry) {
			continue
		}

		bal := new(big.Int)
		bal.SetString(pb.Balance, 10)
		if bal.Sign() <= 0 {
			continue
		}

		swept.Add(swept, bal)
		pb.Swept = true

		// Update account total expired
		totalExpired := new(big.Int)
		totalExpired.SetString(acct.TotalExpired, 10)
		totalExpired.Add(totalExpired, bal)
		acct.TotalExpired = totalExpired.String()

		// Reduce total balance
		totalBal := new(big.Int)
		totalBal.SetString(acct.TotalBalance, 10)
		totalBal.Sub(totalBal, bal)
		acct.TotalBalance = totalBal.String()
	}

	// Clean up zero/swept period balances
	var activePeriods []types.PeriodBalance
	for _, pb := range acct.PeriodBalances {
		pbBal := new(big.Int)
		pbBal.SetString(pb.Balance, 10)
		if !pb.Swept && pbBal.Sign() > 0 {
			activePeriods = append(activePeriods, pb)
		}
	}
	acct.PeriodBalances = activePeriods
	k.recalculatePeriodBounds(acct)

	return swept
}

// recalculatePeriodBounds updates EarliestPeriod and LatestPeriod on an account.
func (k Keeper) recalculatePeriodBounds(acct *types.ExpirableAccount) {
	if len(acct.PeriodBalances) == 0 {
		acct.EarliestPeriod = 0
		acct.LatestPeriod = 0
		return
	}
	acct.EarliestPeriod = acct.PeriodBalances[0].Period
	acct.LatestPeriod = acct.PeriodBalances[len(acct.PeriodBalances)-1].Period
}
