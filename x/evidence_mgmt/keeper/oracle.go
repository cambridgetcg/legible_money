package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// RegisterOracle validates and stores a new oracle registration.
func (k Keeper) RegisterOracle(ctx sdk.Context, oracle types.Oracle) error {
	params := k.GetParams(ctx)

	// Check if operator already has an oracle registered
	if _, exists := k.GetOracleByOperator(ctx, oracle.Operator); exists {
		return fmt.Errorf("%w: operator %s already has a registered oracle",
			types.ErrOracleAlreadyRegistered, oracle.Operator)
	}

	// Validate data types
	for _, dt := range oracle.DataTypes {
		if !types.IsValidEvidenceType(types.EvidenceType(dt)) {
			return fmt.Errorf("%w: invalid data type %s",
				types.ErrInvalidEvidence, dt)
		}
	}

	// Validate minimum oracle stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(oracle.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientOracleStake)
	}
	minStake := new(big.Int)
	if _, ok := minStake.SetString(params.MinOracleStake, 10); !ok {
		minStake.SetInt64(100000000) // fallback: 100 LGM
	}
	if stakeAmount.Cmp(minStake) < 0 {
		return fmt.Errorf("%w: stake %s below minimum %s",
			types.ErrInsufficientOracleStake, oracle.Stake, params.MinOracleStake)
	}

	// Deduct stake from operator
	operatorAddr, err := sdk.AccAddressFromBech32(oracle.Operator)
	if err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}
	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, operatorAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return fmt.Errorf("%w: %v", types.ErrInsufficientOracleStake, err)
	}

	// Set initial reputation
	oracle.ReputationScore = 500000 // 50% starting reputation
	oracle.DataPoints = 0
	oracle.AccuracyRate = 0
	oracle.IsActive = true

	k.SetOracle(ctx, oracle)
	k.SetActiveOracle(ctx, oracle.Id)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_registered",
			sdk.NewAttribute("oracle_id", oracle.Id),
			sdk.NewAttribute("operator", oracle.Operator),
			sdk.NewAttribute("name", oracle.Name),
			sdk.NewAttribute("stake", oracle.Stake),
		),
	)

	k.Logger(ctx).Info("oracle registered",
		"oracle_id", oracle.Id,
		"operator", oracle.Operator,
		"name", oracle.Name,
	)

	return nil
}

// RecordOracleSubmission updates oracle reputation after a data submission.
// Adjusts accuracy rate and reputation score based on whether the submission was accurate.
func (k Keeper) RecordOracleSubmission(ctx sdk.Context, oracleId string, accurate bool) error {
	oracle, found := k.GetOracle(ctx, oracleId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrOracleNotFound, oracleId)
	}

	if !oracle.IsActive {
		return fmt.Errorf("%w: oracle %s is inactive", types.ErrOracleInactive, oracleId)
	}

	oracle.DataPoints++

	// Update accuracy rate using cumulative moving average
	// accuracy_rate = (old_rate * (n-1) + new_value) / n
	// where new_value is 1000000 (accurate) or 0 (inaccurate)
	var newDataPoint uint64
	if accurate {
		newDataPoint = 1000000
	}

	if oracle.DataPoints == 1 {
		oracle.AccuracyRate = newDataPoint
	} else {
		// Moving average: (old * (n-1) + new) / n
		n := oracle.DataPoints
		oldContrib := new(big.Int).Mul(
			new(big.Int).SetUint64(oracle.AccuracyRate),
			new(big.Int).SetUint64(n-1),
		)
		total := new(big.Int).Add(oldContrib, new(big.Int).SetUint64(newDataPoint))
		oracle.AccuracyRate = new(big.Int).Div(total, new(big.Int).SetUint64(n)).Uint64()
	}

	// Update reputation score: blend of accuracy rate with dampening
	// reputation = (accuracy_rate * 3 + old_reputation) / 4
	newRep := (oracle.AccuracyRate*3 + oracle.ReputationScore) / 4
	oracle.ReputationScore = newRep

	k.SetOracle(ctx, oracle)

	k.Logger(ctx).Debug("oracle submission recorded",
		"oracle_id", oracleId,
		"accurate", accurate,
		"accuracy_rate", oracle.AccuracyRate,
		"reputation", oracle.ReputationScore,
		"data_points", oracle.DataPoints,
	)

	return nil
}

// SlashOracle slashes an oracle's stake and deactivates it if the remaining stake
// falls below the minimum.
func (k Keeper) SlashOracle(ctx sdk.Context, oracleId string, amount string) error {
	oracle, found := k.GetOracle(ctx, oracleId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrOracleNotFound, oracleId)
	}

	slashAmount := new(big.Int)
	if _, ok := slashAmount.SetString(amount, 10); !ok || slashAmount.Sign() <= 0 {
		return fmt.Errorf("invalid slash amount: %s", amount)
	}

	currentStake := new(big.Int)
	if _, ok := currentStake.SetString(oracle.Stake, 10); !ok {
		return fmt.Errorf("invalid oracle stake: %s", oracle.Stake)
	}

	// Cap slash to available stake
	if slashAmount.Cmp(currentStake) > 0 {
		slashAmount.Set(currentStake)
	}

	// Update oracle stake
	newStake := new(big.Int).Sub(currentStake, slashAmount)
	oracle.Stake = newStake.String()

	// Check if remaining stake is below minimum — deactivate if so
	params := k.GetParams(ctx)
	minStake := new(big.Int)
	if _, ok := minStake.SetString(params.MinOracleStake, 10); !ok {
		minStake.SetInt64(100000000) // fallback
	}
	if newStake.Cmp(minStake) < 0 {
		oracle.IsActive = false
		k.RemoveActiveOracle(ctx, oracleId)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"oracle_deactivated",
				sdk.NewAttribute("oracle_id", oracleId),
				sdk.NewAttribute("reason", "stake_below_minimum"),
				sdk.NewAttribute("remaining_stake", newStake.String()),
			),
		)
	}

	k.SetOracle(ctx, oracle)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_slashed",
			sdk.NewAttribute("oracle_id", oracleId),
			sdk.NewAttribute("slash_amount", slashAmount.String()),
			sdk.NewAttribute("remaining_stake", newStake.String()),
		),
	)

	k.Logger(ctx).Info("oracle slashed",
		"oracle_id", oracleId,
		"slash_amount", slashAmount.String(),
		"remaining_stake", newStake.String(),
		"is_active", oracle.IsActive,
	)

	return nil
}

// DeactivateOracle deactivates an oracle, removing it from the active set.
func (k Keeper) DeactivateOracle(ctx sdk.Context, oracleId string) error {
	oracle, found := k.GetOracle(ctx, oracleId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrOracleNotFound, oracleId)
	}

	oracle.IsActive = false
	k.SetOracle(ctx, oracle)
	k.RemoveActiveOracle(ctx, oracleId)

	return nil
}

// PruneInactiveOracles removes oracles that have been inactive and have zero data points
// for longer than the audit window. Called from EndBlock.
func (k Keeper) PruneInactiveOracles(ctx sdk.Context) {
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())

	k.IterateOracles(ctx, func(o types.Oracle) bool {
		if !o.IsActive && o.DataPoints == 0 {
			// Prune oracles registered more than 2 audit windows ago with no data
			if currentHeight > o.RegisteredAt+(params.AuditWindow*2) {
				k.DeleteOracle(ctx, o.Id)
				k.Logger(ctx).Info("pruned inactive oracle",
					"oracle_id", o.Id,
					"operator", o.Operator,
				)
			}
		}
		return false
	})
}
