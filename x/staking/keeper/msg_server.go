package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/types"
)

// RegisterValidator handles the registration of a new validator
func (k Keeper) RegisterValidator(ctx sdk.Context, msg *types.MsgRegisterValidator) (*types.MsgRegisterValidatorResponse, error) {
	// Validate sender
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check if validator already exists
	if _, found := k.GetValidator(ctx, msg.Sender); found {
		return nil, types.ErrValidatorAlreadyExists
	}

	// Check if DID already registered
	if _, found := k.GetValidatorByDID(ctx, msg.Did); found {
		return nil, types.ErrDuplicateDID
	}

	// Parse initial stake
	initialStake := new(big.Int)
	if msg.InitialStake != "" && msg.InitialStake != "0" {
		if _, ok := initialStake.SetString(msg.InitialStake, 10); !ok {
			return nil, fmt.Errorf("invalid initial stake: %s", msg.InitialStake)
		}
	} else {
		initialStake.SetInt64(0)
	}

	// Determine initial tier based on stake
	initialTier := k.ComputeValidatorTier(ctx, initialStake.String(), 0, 0, 0)

	// Check apprentice cap if tier 0
	if initialTier == types.TierApprentice {
		params := k.GetParams(ctx)
		apprenticeCount := k.CountApprenticeValidators(ctx)
		if apprenticeCount >= params.MaxApprenticeValidators {
			return nil, types.ErrMaxApprenticesReached
		}
	}

	// Check max validators
	params := k.GetParams(ctx)
	totalValidators := uint32(0)
	k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
		totalValidators++
		return false
	})
	if totalValidators >= params.MaxValidators {
		return nil, types.ErrMaxValidatorsReached
	}

	// Create validator
	validator := types.LegibleValidator{
		OperatorAddress:       msg.Sender,
		Did:                   msg.Did,
		ConsensusPubkey:       msg.ConsensusPubkey,
		Stake:                 initialStake.String(),
		DelegatedStake:        "0",
		TotalStake:            initialStake.String(),
		Commission:            msg.Commission,
		Tier:                  initialTier,
		IsActive:              true,
		ActiveSinceHeight:     uint64(ctx.BlockHeight()),
		LastProposedHeight:    0,
		BlocksProposed:        0,
		VerificationsCompleted: 0,
		VerificationsCorrect:  0,
		SlashCount:            0,
		Reputation:            500000, // 0.5 default
	}

	// Transfer initial stake if non-zero
	if initialStake.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(initialStake)))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("failed to transfer initial stake: %w", err)
		}

		// Create self-delegation
		delegation := types.Delegation{
			DelegatorAddress:  msg.Sender,
			ValidatorAddress:  msg.Sender,
			Amount:            initialStake.String(),
			DelegatedAtHeight: uint64(ctx.BlockHeight()),
		}
		if err := k.SetDelegation(ctx, delegation); err != nil {
			return nil, err
		}
	}

	// Store validator
	if err := k.SetValidator(ctx, validator); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_registered",
			sdk.NewAttribute("validator", msg.Sender),
			sdk.NewAttribute("did", msg.Did),
			sdk.NewAttribute("tier", initialTier.String()),
			sdk.NewAttribute("stake", initialStake.String()),
		),
	)

	return &types.MsgRegisterValidatorResponse{
		ValidatorAddress: msg.Sender,
		InitialTier:      initialTier,
	}, nil
}

// UpdateValidatorStake handles stake increase/decrease
func (k Keeper) UpdateValidatorStake(ctx sdk.Context, msg *types.MsgUpdateValidatorStake) (*types.MsgUpdateValidatorStakeResponse, error) {
	// Validate validator address
	validatorAddr, err := sdk.AccAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, err
	}

	// Get validator
	validator, found := k.GetValidator(ctx, msg.Validator)
	if !found {
		return nil, types.ErrValidatorNotFound
	}

	// Parse amount
	amountInt := new(big.Int)
	if _, ok := amountInt.SetString(msg.Amount, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	// Parse current stake
	stakeInt := new(big.Int)
	stakeInt.SetString(validator.Stake, 10)

	var newStake *big.Int
	if msg.IsIncrease {
		// Increase stake - transfer tokens to module
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amountInt)))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, validatorAddr, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("failed to transfer stake: %w", err)
		}

		newStake = new(big.Int).Add(stakeInt, amountInt)

		// Update self-delegation
		delegation, found := k.GetDelegation(ctx, msg.Validator, msg.Validator)
		if !found {
			// Create new self-delegation
			delegation = types.Delegation{
				DelegatorAddress:  msg.Validator,
				ValidatorAddress:  msg.Validator,
				Amount:            amountInt.String(),
				DelegatedAtHeight: uint64(ctx.BlockHeight()),
			}
		} else {
			// Increase existing delegation
			delInt := new(big.Int)
			delInt.SetString(delegation.Amount, 10)
			delInt.Add(delInt, amountInt)
			delegation.Amount = delInt.String()
		}
		if err := k.SetDelegation(ctx, delegation); err != nil {
			return nil, err
		}

	} else {
		// Decrease stake - initiate unbonding
		if stakeInt.Cmp(amountInt) < 0 {
			return nil, types.ErrInsufficientStake
		}

		newStake = new(big.Int).Sub(stakeInt, amountInt)

		// Create unbonding entry
		params := k.GetParams(ctx)
		unbondingId := fmt.Sprintf("%s_%d", msg.Validator, ctx.BlockHeight())
		unbonding := types.UnbondingEntry{
			Id:                unbondingId,
			DelegatorAddress:  msg.Validator,
			ValidatorAddress:  msg.Validator,
			Amount:            msg.Amount,
			InitiatedAtHeight: uint64(ctx.BlockHeight()),
			CompletesAtHeight: uint64(ctx.BlockHeight()) + params.UnbondingPeriod,
			Status:            "pending",
		}
		if err := k.SetUnbonding(ctx, unbonding); err != nil {
			return nil, err
		}

		// Update self-delegation
		delegation, found := k.GetDelegation(ctx, msg.Validator, msg.Validator)
		if found {
			delInt := new(big.Int)
			delInt.SetString(delegation.Amount, 10)
			delInt.Sub(delInt, amountInt)
			delegation.Amount = delInt.String()
			if err := k.SetDelegation(ctx, delegation); err != nil {
				return nil, err
			}
		}
	}

	// Update validator stake
	validator.Stake = newStake.String()

	// Recalculate total stake (stake + delegated)
	delegatedInt := new(big.Int)
	delegatedInt.SetString(validator.DelegatedStake, 10)
	totalStake := new(big.Int).Add(newStake, delegatedInt)
	validator.TotalStake = totalStake.String()

	// Check for tier transition
	newTier, changed := k.CheckTierTransition(ctx, validator)
	if changed {
		oldTier := validator.Tier
		validator.Tier = newTier

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"validator_tier_changed",
				sdk.NewAttribute("validator", msg.Validator),
				sdk.NewAttribute("old_tier", oldTier.String()),
				sdk.NewAttribute("new_tier", newTier.String()),
			),
		)
	}

	// Check if validator becomes inactive due to insufficient stake
	if newTier == types.TierApprentice && newStake.Sign() == 0 && validator.VerificationsCompleted < 22 {
		validator.IsActive = false
	}

	// Store updated validator
	if err := k.SetValidator(ctx, validator); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_stake_updated",
			sdk.NewAttribute("validator", msg.Validator),
			sdk.NewAttribute("new_stake", newStake.String()),
			sdk.NewAttribute("is_increase", fmt.Sprintf("%t", msg.IsIncrease)),
		),
	)

	return &types.MsgUpdateValidatorStakeResponse{
		NewStake: newStake.String(),
		NewTier:  validator.Tier,
	}, nil
}
