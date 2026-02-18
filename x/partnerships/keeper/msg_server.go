package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// InitiatePartnership starts partnership formation.
func (k msgServer) InitiatePartnership(goCtx context.Context, msg *types.MsgInitiatePartnership) (*types.MsgInitiatePartnershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Check for existing partnership between these two
	existingByHuman := k.GetPartnershipsByHuman(ctx, msg.Initiator)
	for _, id := range existingByHuman {
		if p, found := k.GetPartnership(ctx, id); found {
			if (p.HumanAddr == msg.Initiator && p.AgentAddr == msg.Partner) ||
				(p.HumanAddr == msg.Partner && p.AgentAddr == msg.Initiator) {
				if p.Status != types.StatusDissolved {
					return nil, types.ErrPartnershipExists
				}
			}
		}
	}
	existingByAgent := k.GetPartnershipsByAgent(ctx, msg.Initiator)
	for _, id := range existingByAgent {
		if p, found := k.GetPartnership(ctx, id); found {
			if (p.HumanAddr == msg.Initiator && p.AgentAddr == msg.Partner) ||
				(p.HumanAddr == msg.Partner && p.AgentAddr == msg.Initiator) {
				if p.Status != types.StatusDissolved {
					return nil, types.ErrPartnershipExists
				}
			}
		}
	}

	seq := k.NextSequence(ctx)
	partnershipId := fmt.Sprintf("partnership-%d", seq)

	// Validate lock tier
	if msg.LockTier > 5 {
		return nil, types.ErrInvalidLockTier
	}
	lockDuration := types.LockTiers[msg.LockTier].MinBlocks

	partnership := types.Partnership{
		Id:               partnershipId,
		HumanAddr:        msg.Initiator,
		AgentAddr:        msg.Partner,
		Status:           types.StatusForming,
		Tier:             types.TierTrial,
		LockTier:         msg.LockTier,
		LockExpiresAt:    currentBlock + lockDuration,
		SplitHumanBps:    params.DefaultHumanSplitBps,
		SplitAgentBps:    params.DefaultAgentSplitBps,
		CommonPotBalance: "0",
		TotalEarned:      "0",
		CooperationScore: 5000, // Start at 50%
		FormedAtBlock:    currentBlock,
	}
	k.SetPartnership(ctx, partnership)

	// Store formation record with expiry
	kvStore := k.storeService.OpenKVStore(ctx)
	formationExpiry := currentBlock + params.FormationWindowBlocks
	_ = kvStore.Set(
		append(types.FormationKeyPrefix, []byte(partnershipId)...),
		[]byte(fmt.Sprintf("%d", formationExpiry)),
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"partnership_initiated",
			sdk.NewAttribute("partnership_id", partnershipId),
			sdk.NewAttribute("initiator", msg.Initiator),
			sdk.NewAttribute("partner", msg.Partner),
			sdk.NewAttribute("formation_expires", fmt.Sprintf("%d", formationExpiry)),
		),
	)

	k.Logger(ctx).Info("partnership formation initiated",
		"partnership_id", partnershipId,
		"initiator", msg.Initiator,
		"partner", msg.Partner,
	)

	return &types.MsgInitiatePartnershipResponse{PartnershipId: partnershipId}, nil
}

// AcceptPartnership accepts a pending partnership formation.
func (k msgServer) AcceptPartnership(goCtx context.Context, msg *types.MsgAcceptPartnership) (*types.MsgAcceptPartnershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}

	if partnership.Status != types.StatusForming {
		return nil, types.ErrNotFormingStatus
	}

	// Verify the acceptor is the partner (not the initiator)
	if msg.Acceptor != partnership.AgentAddr {
		return nil, fmt.Errorf("%w: acceptor must be the partner address", types.ErrNotParticipant)
	}

	// Check formation window
	kvStore := k.storeService.OpenKVStore(ctx)
	expiryBz, err := kvStore.Get(append(types.FormationKeyPrefix, []byte(msg.PartnershipId)...))
	if err != nil || expiryBz == nil {
		return nil, types.ErrFormationExpired
	}
	var expiry uint64
	if _, err := fmt.Sscanf(string(expiryBz), "%d", &expiry); err != nil {
		return nil, types.ErrFormationExpired
	}
	if currentBlock > expiry {
		return nil, types.ErrFormationExpired
	}

	// Activate the partnership
	partnership.Status = types.StatusActive
	partnership.FormedAtBlock = currentBlock
	k.SetPartnership(ctx, partnership)

	// Clean up formation record
	_ = kvStore.Delete(append(types.FormationKeyPrefix, []byte(msg.PartnershipId)...))

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"partnership_accepted",
			sdk.NewAttribute("partnership_id", msg.PartnershipId),
			sdk.NewAttribute("acceptor", msg.Acceptor),
		),
	)

	k.Logger(ctx).Info("partnership accepted",
		"partnership_id", msg.PartnershipId,
		"acceptor", msg.Acceptor,
	)

	return &types.MsgAcceptPartnershipResponse{}, nil
}

// DepositToPot deposits funds directly into the partnership common pot.
func (k msgServer) DepositToPot(goCtx context.Context, msg *types.MsgDepositToPot) (*types.MsgDepositToPotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if partnership.Status != types.StatusActive {
		return nil, fmt.Errorf("%w: partnership must be active for deposits", types.ErrInvalidStatus)
	}
	if partnership.Status == types.StatusSuspended {
		return nil, fmt.Errorf("%w: partnership is frozen", types.ErrFreezeActive)
	}

	// Verify depositor is a participant
	if msg.Depositor != partnership.HumanAddr && msg.Depositor != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	// Parse and validate amount
	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Transfer from depositor to module account
	depositorAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, fmt.Errorf("invalid depositor address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to transfer deposit: %w", err)
	}

	// Update common pot balance
	currentPot := new(big.Int)
	if partnership.CommonPotBalance != "" {
		currentPot.SetString(partnership.CommonPotBalance, 10)
	}
	currentPot.Add(currentPot, amount)
	partnership.CommonPotBalance = currentPot.String()
	k.SetPartnership(ctx, partnership)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"deposit_to_pot",
			sdk.NewAttribute("partnership_id", msg.PartnershipId),
			sdk.NewAttribute("depositor", msg.Depositor),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("new_pot_balance", currentPot.String()),
		),
	)

	k.Logger(ctx).Info("deposit to partnership pot",
		"partnership_id", msg.PartnershipId,
		"depositor", msg.Depositor,
		"amount", msg.Amount,
	)

	return &types.MsgDepositToPotResponse{}, nil
}

// DistributeReward distributes a reward through the partnership.
func (k msgServer) DistributeReward(goCtx context.Context, msg *types.MsgDistributeReward) (*types.MsgDistributeRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if partnership.Status != types.StatusActive {
		return nil, fmt.Errorf("%w: partnership must be active for rewards", types.ErrInvalidStatus)
	}

	humanShare, agentShare, commonPotAdd, err := k.Keeper.DistributeReward(ctx, &partnership, msg.GrossAmount, msg.SourceType)
	if err != nil {
		return nil, err
	}

	return &types.MsgDistributeRewardResponse{
		HumanShare:   humanShare,
		AgentShare:   agentShare,
		CommonPotAdd: commonPotAdd,
	}, nil
}

// ProposeOperation proposes a consensus operation.
func (k msgServer) ProposeOperation(goCtx context.Context, msg *types.MsgProposeOperation) (*types.MsgProposeOperationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if partnership.Status != types.StatusActive {
		return nil, fmt.Errorf("%w: partnership must be active for operations", types.ErrInvalidStatus)
	}

	// Verify proposer is a participant
	if msg.Proposer != partnership.HumanAddr && msg.Proposer != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	// Check rejection cooldown
	if rc, found := k.GetRejectionCooldown(ctx, msg.PartnershipId); found {
		if rc.CooldownEndsAt > currentBlock {
			return nil, fmt.Errorf("%w: cooldown ends at block %d",
				types.ErrCooldownActive, rc.CooldownEndsAt)
		}
	}

	// Create deliberation state
	delib := k.CreateDeliberationState(ctx, msg.Amount, partnership.CommonPotBalance, msg.Rationale, "", 0)

	seq := k.NextSequence(ctx)
	opId := fmt.Sprintf("op-%d", seq)

	op := types.ConsensusOperation{
		Id:            opId,
		PartnershipId: msg.PartnershipId,
		Type:          msg.OperationType,
		ProposedBy:    msg.Proposer,
		Amount:        msg.Amount,
		Status:        types.OpStatusProposed,
		Deliberation:  &delib,
		CreatedAt:     currentBlock,
	}
	k.SetConsensusOperation(ctx, op)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operation_proposed",
			sdk.NewAttribute("operation_id", opId),
			sdk.NewAttribute("partnership_id", msg.PartnershipId),
			sdk.NewAttribute("type", msg.OperationType),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("tier", string(delib.AmountTier)),
		),
	)

	return &types.MsgProposeOperationResponse{OperationId: opId}, nil
}

// ApproveOperation approves a pending consensus operation.
func (k msgServer) ApproveOperation(goCtx context.Context, msg *types.MsgApproveOperation) (*types.MsgApproveOperationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	op, found := k.GetConsensusOperation(ctx, msg.OperationId)
	if !found {
		return nil, fmt.Errorf("operation not found: %s", msg.OperationId)
	}
	if op.Status != types.OpStatusProposed {
		return nil, fmt.Errorf("%w: operation is %s", types.ErrInvalidStatus, op.Status)
	}

	// Verify approver is the other partner (not the proposer)
	partnership, found := k.GetPartnership(ctx, op.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if msg.Approver == op.ProposedBy {
		return nil, fmt.Errorf("%w: cannot approve own proposal", types.ErrUnauthorized)
	}
	if msg.Approver != partnership.HumanAddr && msg.Approver != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	// Check if within deliberation window
	if currentBlock > op.Deliberation.WindowEndsAt {
		op.Status = types.OpStatusExpired
		k.SetConsensusOperation(ctx, op)
		return nil, fmt.Errorf("%w: deliberation window expired", types.ErrInvalidStatus)
	}

	// Check floor period
	if currentBlock < op.Deliberation.FloorEndsAt {
		return nil, fmt.Errorf("%w: floor period not yet elapsed (ends block %d)",
			types.ErrDeliberationActive, op.Deliberation.FloorEndsAt)
	}

	op.Status = types.OpStatusApproved
	k.SetConsensusOperation(ctx, op)

	// Clear any rejection cooldown on approval
	k.DeleteRejectionCooldown(ctx, op.PartnershipId)

	// Execute the operation if it's a withdrawal
	if op.Type == "withdrawal" && op.Amount != "" {
		withdrawAmt := new(big.Int)
		if _, ok := withdrawAmt.SetString(op.Amount, 10); ok && withdrawAmt.Sign() > 0 {
			// Deduct from common pot
			potBal := new(big.Int)
			if partnership.CommonPotBalance != "" {
				potBal.SetString(partnership.CommonPotBalance, 10)
			}
			if potBal.Cmp(withdrawAmt) >= 0 {
				potBal.Sub(potBal, withdrawAmt)
				partnership.CommonPotBalance = potBal.String()
				k.SetPartnership(ctx, partnership)

				// Transfer from module to proposer
				recipientAddr, err := sdk.AccAddressFromBech32(op.ProposedBy)
				if err == nil {
					coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(withdrawAmt)))
					if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, coins); err != nil {
						k.Logger(ctx).Error("failed to execute withdrawal", "err", err)
					}
				}
			}
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operation_approved",
			sdk.NewAttribute("operation_id", msg.OperationId),
			sdk.NewAttribute("approver", msg.Approver),
		),
	)

	return &types.MsgApproveOperationResponse{}, nil
}

// RejectOperation rejects a pending operation with optional counter-proposal.
func (k msgServer) RejectOperation(goCtx context.Context, msg *types.MsgRejectOperation) (*types.MsgRejectOperationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	op, found := k.GetConsensusOperation(ctx, msg.OperationId)
	if !found {
		return nil, fmt.Errorf("operation not found: %s", msg.OperationId)
	}
	if op.Status != types.OpStatusProposed {
		return nil, fmt.Errorf("%w: operation is %s", types.ErrInvalidStatus, op.Status)
	}

	// Verify rejecter is the other partner
	partnership, found := k.GetPartnership(ctx, op.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if msg.Rejecter == op.ProposedBy {
		return nil, fmt.Errorf("%w: cannot reject own proposal", types.ErrUnauthorized)
	}
	if msg.Rejecter != partnership.HumanAddr && msg.Rejecter != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	// Reject the operation
	op.Status = types.OpStatusRejected
	k.SetConsensusOperation(ctx, op)

	// Update rejection cooldown (exponential)
	rc, found := k.GetRejectionCooldown(ctx, op.PartnershipId)
	if !found {
		rc = types.RejectionCooldown{PartnershipId: op.PartnershipId}
	}
	rc.RejectionCount++
	cooldown := k.CalculateCooldown(ctx, rc.RejectionCount)
	rc.CooldownEndsAt = currentBlock + cooldown
	k.SetRejectionCooldown(ctx, rc)

	resp := &types.MsgRejectOperationResponse{}

	// Handle counter-proposal if provided
	if msg.CounterAmount != "" {
		counterAmt := new(big.Int)
		if _, ok := counterAmt.SetString(msg.CounterAmount, 10); ok && counterAmt.Sign() > 0 {
			// Validate chain depth
			if err := k.ValidateCounterProposal(ctx, op); err != nil {
				// Still reject, but skip counter-proposal
				return resp, nil
			}

			// Create counter-proposal
			delib := k.CreateDeliberationState(
				ctx,
				msg.CounterAmount,
				partnership.CommonPotBalance,
				msg.CounterRationale,
				op.Id,
				op.Deliberation.ChainDepth+1,
			)

			seq := k.NextSequence(ctx)
			counterOpId := fmt.Sprintf("op-%d", seq)

			counterOp := types.ConsensusOperation{
				Id:            counterOpId,
				PartnershipId: op.PartnershipId,
				Type:          op.Type,
				ProposedBy:    msg.Rejecter,
				Amount:        msg.CounterAmount,
				Status:        types.OpStatusProposed,
				Deliberation:  &delib,
				CreatedAt:     currentBlock,
			}
			k.SetConsensusOperation(ctx, counterOp)

			resp.CounterOperationId = counterOpId

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"counter_proposal_created",
					sdk.NewAttribute("operation_id", counterOpId),
					sdk.NewAttribute("parent_operation_id", op.Id),
					sdk.NewAttribute("proposer", msg.Rejecter),
					sdk.NewAttribute("amount", msg.CounterAmount),
					sdk.NewAttribute("chain_depth", fmt.Sprintf("%d", delib.ChainDepth)),
				),
			)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operation_rejected",
			sdk.NewAttribute("operation_id", msg.OperationId),
			sdk.NewAttribute("rejecter", msg.Rejecter),
			sdk.NewAttribute("cooldown_ends_at", fmt.Sprintf("%d", rc.CooldownEndsAt)),
		),
	)

	return resp, nil
}

// SafetyFreeze triggers a unilateral safety freeze.
func (k msgServer) SafetyFreeze(goCtx context.Context, msg *types.MsgSafetyFreeze) (*types.MsgSafetyFreezeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify freezer is a participant
	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if msg.Freezer != partnership.HumanAddr && msg.Freezer != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}
	if partnership.Status != types.StatusActive {
		return nil, fmt.Errorf("%w: partnership must be active to freeze", types.ErrInvalidStatus)
	}

	sf, err := k.HandleSafetyFreeze(ctx, msg.PartnershipId, msg.Freezer)
	if err != nil {
		return nil, err
	}

	return &types.MsgSafetyFreezeResponse{ExpiresAt: sf.ExpiresAt}, nil
}

// RaiseCoercionSignal raises a coercion/duress flag.
func (k msgServer) RaiseCoercionSignal(goCtx context.Context, msg *types.MsgRaiseCoercionSignal) (*types.MsgRaiseCoercionSignalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify raiser is a participant
	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if msg.Raiser != partnership.HumanAddr && msg.Raiser != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	cs, err := k.HandleCoercionSignal(ctx, msg.PartnershipId, msg.Raiser)
	if err != nil {
		return nil, err
	}

	return &types.MsgRaiseCoercionSignalResponse{
		SignalId:  cs.SignalId,
		ExpiresAt: cs.ExpiresAt,
	}, nil
}

// InitiateExit starts the exit/dissolution process.
func (k msgServer) InitiateExit(goCtx context.Context, msg *types.MsgInitiateExit) (*types.MsgInitiateExitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify initiator is a participant
	partnership, found := k.GetPartnership(ctx, msg.PartnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}
	if msg.Initiator != partnership.HumanAddr && msg.Initiator != partnership.AgentAddr {
		return nil, types.ErrNotParticipant
	}

	cooldownEnd, err := k.HandleExit(ctx, msg.PartnershipId, msg.Initiator)
	if err != nil {
		return nil, err
	}

	return &types.MsgInitiateExitResponse{CooldownEndsAt: cooldownEnd}, nil
}

// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s",
			types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	k.SetParams(ctx, *msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"partnerships_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
