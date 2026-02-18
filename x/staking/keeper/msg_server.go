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
	senderAddr, err := sdk.AccAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	// Check if validator already exists
	if _, found := k.GetValidator(ctx, msg.ValidatorAddress); found {
		return nil, types.ErrValidatorAlreadyExists
	}

	// Extract DID from description identity field
	did := ""
	if msg.Description != nil {
		did = msg.Description.GetIdentity()
	}

	// Check if DID already registered (only if DID is provided)
	if did != "" {
		if _, found := k.GetValidatorByDID(ctx, did); found {
			return nil, types.ErrDuplicateDID
		}
	}

	// Parse initial stake
	initialStake := new(big.Int)
	if msg.SelfDelegation != "" && msg.SelfDelegation != "0" {
		if _, ok := initialStake.SetString(msg.SelfDelegation, 10); !ok {
			return nil, fmt.Errorf("invalid initial stake: %s", msg.SelfDelegation)
		}
	} else {
		initialStake.SetInt64(0)
	}

	// Enforce minimum registration deposit (Sybil defense — uses MinSelfDelegation)
	params := k.GetParams(ctx)
	minDeposit := new(big.Int)
	minDepositStr := params.GetMinSelfDelegation()
	minDeposit.SetString(minDepositStr, 10)
	if initialStake.Cmp(minDeposit) < 0 {
		return nil, types.ErrInsufficientRegistrationDeposit
	}

	// Determine initial tier based on stake
	initialTier := k.ComputeValidatorTierSimple(ctx, initialStake.String(), 0, 0, 0)

	// Check block producer cap: MaxValidators applies only to tier 2+ (bonded/guardian)
	if initialTier >= types.TierBonded {
		blockProducerCount := k.CountBlockProducers(ctx)
		if blockProducerCount >= params.MaxValidators {
			return nil, types.ErrMaxBlockProducersReached
		}
	}

	// Create validator
	validator := &types.LegibleValidator{
		OperatorAddress:        msg.ValidatorAddress,
		Did:                    did,
		ConsensusPubkey:        string(msg.PubKey),
		Stake:                  initialStake.String(),
		DelegatedStake:         "0",
		TotalStake:             initialStake.String(),
		Commission:             uint64(msg.CommissionRate),
		Tier:                   types.ValidatorTierString(initialTier),
		IsActive:               true,
		ActiveSinceHeight:      uint64(ctx.BlockHeight()),
		LastProposedHeight:     0,
		BlocksProposed:         0,
		VerificationsCompleted: 0,
		VerificationsCorrect:   0,
		SlashCount:             0,
		Reputation:             500000, // 0.5 default
	}

	// Transfer initial stake if non-zero
	if initialStake.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(initialStake)))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("failed to transfer initial stake: %w", err)
		}

		// Create self-delegation
		delegation := &types.Delegation{
			DelegatorAddress:  msg.ValidatorAddress,
			ValidatorAddress:  msg.ValidatorAddress,
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
			sdk.NewAttribute("validator", msg.ValidatorAddress),
			sdk.NewAttribute("did", did),
			sdk.NewAttribute("tier", types.ValidatorTierString(initialTier)),
			sdk.NewAttribute("stake", initialStake.String()),
		),
	)

	return &types.MsgRegisterValidatorResponse{
		InitialTier: initialTier,
	}, nil
}

// Delegate handles external delegation to a validator.
func (k Keeper) Delegate(ctx sdk.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	delegatorAddr, err := sdk.AccAddressFromBech32(msg.Delegator)
	if err != nil {
		return nil, err
	}

	// Get validator
	validator, found := k.GetValidator(ctx, msg.Validator)
	if !found {
		return nil, types.ErrValidatorNotFound
	}
	if !validator.IsActive {
		return nil, types.ErrValidatorInactive
	}

	// Parse amount
	amountInt := new(big.Int)
	if _, ok := amountInt.SetString(msg.Amount, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	// Transfer tokens from delegator to staking module
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amountInt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, delegatorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to transfer delegation: %w", err)
	}

	// Update or create delegation
	delegation, found := k.GetDelegation(ctx, msg.Delegator, msg.Validator)
	if !found {
		delegation = &types.Delegation{
			DelegatorAddress:  msg.Delegator,
			ValidatorAddress:  msg.Validator,
			Amount:            amountInt.String(),
			DelegatedAtHeight: uint64(ctx.BlockHeight()),
		}
	} else {
		delInt := new(big.Int)
		delInt.SetString(delegation.Amount, 10)
		delInt.Add(delInt, amountInt)
		delegation.Amount = delInt.String()
	}
	if err := k.SetDelegation(ctx, delegation); err != nil {
		return nil, err
	}

	// Update validator DelegatedStake and TotalStake
	delegatedInt := new(big.Int)
	delegatedInt.SetString(validator.DelegatedStake, 10)
	delegatedInt.Add(delegatedInt, amountInt)
	validator.DelegatedStake = delegatedInt.String()

	stakeInt := new(big.Int)
	stakeInt.SetString(validator.Stake, 10)
	totalStake := new(big.Int).Add(stakeInt, delegatedInt)
	validator.TotalStake = totalStake.String()

	// Check for tier transition
	if newTier, changed := k.CheckTierTransition(ctx, validator); changed {
		oldTier := validator.Tier
		validator.Tier = types.ValidatorTierString(newTier)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("validator_tier_changed",
				sdk.NewAttribute("validator", msg.Validator),
				sdk.NewAttribute("old_tier", oldTier),
				sdk.NewAttribute("new_tier", types.ValidatorTierString(newTier)),
			),
		)
	}

	if err := k.SetValidator(ctx, validator); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("delegation_created",
			sdk.NewAttribute("delegator", msg.Delegator),
			sdk.NewAttribute("validator", msg.Validator),
			sdk.NewAttribute("amount", msg.Amount),
		),
	)

	return &types.MsgDelegateResponse{NewDelegation: delegation.Amount}, nil
}

// Undelegate handles undelegation from a validator (initiates unbonding).
func (k Keeper) Undelegate(ctx sdk.Context, msg *types.MsgUndelegate) (*types.MsgUndelegateResponse, error) {
	// Get delegation
	delegation, found := k.GetDelegation(ctx, msg.Delegator, msg.Validator)
	if !found {
		return nil, types.ErrDelegationNotFound
	}

	// Parse amounts
	amountInt := new(big.Int)
	if _, ok := amountInt.SetString(msg.Amount, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	delInt := new(big.Int)
	delInt.SetString(delegation.Amount, 10)
	if delInt.Cmp(amountInt) < 0 {
		return nil, types.ErrInsufficientStake
	}

	// Get validator
	validator, found := k.GetValidator(ctx, msg.Validator)
	if !found {
		return nil, types.ErrValidatorNotFound
	}

	// Reduce delegation
	newDelAmount := new(big.Int).Sub(delInt, amountInt)
	if newDelAmount.Sign() == 0 {
		// Delete delegation entirely
		if err := k.DeleteDelegation(ctx, msg.Delegator, msg.Validator); err != nil {
			return nil, err
		}
	} else {
		delegation.Amount = newDelAmount.String()
		if err := k.SetDelegation(ctx, delegation); err != nil {
			return nil, err
		}
	}

	// Create unbonding entry
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	seq := k.NextUnbondingSeq(ctx)
	unbondingId := fmt.Sprintf("%s_%s_%d_%d", msg.Delegator, msg.Validator, currentHeight, seq)

	unbonding := &types.UnbondingEntry{
		Id:                unbondingId,
		DelegatorAddress:  msg.Delegator,
		ValidatorAddress:  msg.Validator,
		Amount:            msg.Amount,
		InitiatedAtHeight: currentHeight,
		CompletesAtHeight: currentHeight + params.UnbondingPeriod,
		Status:            "pending",
	}
	if err := k.SetUnbonding(ctx, unbonding); err != nil {
		return nil, err
	}

	// Update validator DelegatedStake and TotalStake
	delegatedInt := new(big.Int)
	delegatedInt.SetString(validator.DelegatedStake, 10)
	delegatedInt.Sub(delegatedInt, amountInt)
	if delegatedInt.Sign() < 0 {
		delegatedInt.SetInt64(0)
	}
	validator.DelegatedStake = delegatedInt.String()

	stakeInt := new(big.Int)
	stakeInt.SetString(validator.Stake, 10)
	totalStake := new(big.Int).Add(stakeInt, delegatedInt)
	validator.TotalStake = totalStake.String()

	// Check for tier transition
	if newTier, changed := k.CheckTierTransition(ctx, validator); changed {
		oldTier := validator.Tier
		validator.Tier = types.ValidatorTierString(newTier)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("validator_tier_changed",
				sdk.NewAttribute("validator", msg.Validator),
				sdk.NewAttribute("old_tier", oldTier),
				sdk.NewAttribute("new_tier", types.ValidatorTierString(newTier)),
			),
		)
	}

	if err := k.SetValidator(ctx, validator); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("delegation_unbonding",
			sdk.NewAttribute("delegator", msg.Delegator),
			sdk.NewAttribute("validator", msg.Validator),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("completes_at", fmt.Sprintf("%d", unbonding.CompletesAtHeight)),
		),
	)

	return &types.MsgUndelegateResponse{
		UnbondingId:       unbondingId,
		CompletesAtHeight: unbonding.CompletesAtHeight,
	}, nil
}

// Redelegate moves delegation from one validator to another.
func (k Keeper) Redelegate(ctx sdk.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	// Enforce redelegation cooldown to prevent rapid stake-shifting attacks (P1-A: parameterized)
	params := k.GetParams(ctx)
	cooldownBlocks := params.RedelegationCooldownBlocks
	currentHeight := uint64(ctx.BlockHeight())
	lastRedelHeight := k.GetLastRedelegationHeight(ctx, msg.Delegator)
	if lastRedelHeight > 0 && currentHeight < lastRedelHeight+cooldownBlocks {
		return nil, types.ErrRedelegationCooldown
	}

	// Check source delegation exists
	srcDelegation, found := k.GetDelegation(ctx, msg.Delegator, msg.SrcValidator)
	if !found {
		return nil, types.ErrDelegationNotFound
	}

	// Parse amounts
	amountInt := new(big.Int)
	if _, ok := amountInt.SetString(msg.Amount, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	srcDelInt := new(big.Int)
	srcDelInt.SetString(srcDelegation.Amount, 10)
	if srcDelInt.Cmp(amountInt) < 0 {
		return nil, types.ErrInsufficientStake
	}

	// Check destination validator exists and is active
	dstValidator, found := k.GetValidator(ctx, msg.DstValidator)
	if !found {
		return nil, types.ErrValidatorNotFound
	}
	if !dstValidator.IsActive {
		return nil, types.ErrValidatorInactive
	}

	// Get source validator
	srcValidator, found := k.GetValidator(ctx, msg.SrcValidator)
	if !found {
		return nil, types.ErrValidatorNotFound
	}

	// Reduce source delegation
	newSrcAmount := new(big.Int).Sub(srcDelInt, amountInt)
	if newSrcAmount.Sign() == 0 {
		if err := k.DeleteDelegation(ctx, msg.Delegator, msg.SrcValidator); err != nil {
			return nil, err
		}
	} else {
		srcDelegation.Amount = newSrcAmount.String()
		if err := k.SetDelegation(ctx, srcDelegation); err != nil {
			return nil, err
		}
	}

	// Increase destination delegation
	dstDelegation, found := k.GetDelegation(ctx, msg.Delegator, msg.DstValidator)
	if !found {
		dstDelegation = &types.Delegation{
			DelegatorAddress:  msg.Delegator,
			ValidatorAddress:  msg.DstValidator,
			Amount:            amountInt.String(),
			DelegatedAtHeight: uint64(ctx.BlockHeight()),
		}
	} else {
		dstDelInt := new(big.Int)
		dstDelInt.SetString(dstDelegation.Amount, 10)
		dstDelInt.Add(dstDelInt, amountInt)
		dstDelegation.Amount = dstDelInt.String()
	}
	if err := k.SetDelegation(ctx, dstDelegation); err != nil {
		return nil, err
	}

	// Update source validator DelegatedStake
	srcDelegatedInt := new(big.Int)
	srcDelegatedInt.SetString(srcValidator.DelegatedStake, 10)
	srcDelegatedInt.Sub(srcDelegatedInt, amountInt)
	if srcDelegatedInt.Sign() < 0 {
		srcDelegatedInt.SetInt64(0)
	}
	srcValidator.DelegatedStake = srcDelegatedInt.String()
	srcStakeInt := new(big.Int)
	srcStakeInt.SetString(srcValidator.Stake, 10)
	srcValidator.TotalStake = new(big.Int).Add(srcStakeInt, srcDelegatedInt).String()

	if newTier, changed := k.CheckTierTransition(ctx, srcValidator); changed {
		srcValidator.Tier = types.ValidatorTierString(newTier)
	}
	if err := k.SetValidator(ctx, srcValidator); err != nil {
		return nil, err
	}

	// Update destination validator DelegatedStake
	dstDelegatedInt := new(big.Int)
	dstDelegatedInt.SetString(dstValidator.DelegatedStake, 10)
	dstDelegatedInt.Add(dstDelegatedInt, amountInt)
	dstValidator.DelegatedStake = dstDelegatedInt.String()
	dstStakeInt := new(big.Int)
	dstStakeInt.SetString(dstValidator.Stake, 10)
	dstValidator.TotalStake = new(big.Int).Add(dstStakeInt, dstDelegatedInt).String()

	if newTier, changed := k.CheckTierTransition(ctx, dstValidator); changed {
		dstValidator.Tier = types.ValidatorTierString(newTier)
	}
	if err := k.SetValidator(ctx, dstValidator); err != nil {
		return nil, err
	}

	// Record redelegation height for cooldown enforcement
	if err := k.SetLastRedelegationHeight(ctx, msg.Delegator, currentHeight); err != nil {
		return nil, fmt.Errorf("failed to record redelegation cooldown: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("delegation_redelegated",
			sdk.NewAttribute("delegator", msg.Delegator),
			sdk.NewAttribute("src_validator", msg.SrcValidator),
			sdk.NewAttribute("dst_validator", msg.DstValidator),
			sdk.NewAttribute("amount", msg.Amount),
		),
	)

	return &types.MsgRedelegateResponse{}, nil
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
			delegation = &types.Delegation{
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
		seq := k.NextUnbondingSeq(ctx)
		unbondingId := fmt.Sprintf("%s_%d_%d", msg.Validator, ctx.BlockHeight(), seq)

		unbonding := &types.UnbondingEntry{
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
		validator.Tier = types.ValidatorTierString(newTier)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"validator_tier_changed",
				sdk.NewAttribute("validator", msg.Validator),
				sdk.NewAttribute("old_tier", oldTier),
				sdk.NewAttribute("new_tier", types.ValidatorTierString(newTier)),
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
		NewTier:  types.ValidatorTierFromString(validator.Tier),
	}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter update.
func (k Keeper) UpdateParams(ctx sdk.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, fmt.Errorf("failed to set params: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"staking_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
