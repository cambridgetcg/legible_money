package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/types"
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
var _ types.MsgServerUpdateParams = msgServer{}

// ProposeHalt handles a Guardian's proposal to halt the chain.
func (k msgServer) ProposeHalt(goCtx context.Context, msg *types.MsgProposeHalt) (*types.MsgProposeHaltResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// 1. Guardian check
	if !k.IsGuardian(ctx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}

	// 2. Status check — must be normal
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusNormal {
		return nil, fmt.Errorf("%w: current status is %s, must be normal", types.ErrStatusConflict, status)
	}

	// 3. No active ceremony
	if _, found := k.GetActiveCeremony(ctx); found {
		return nil, fmt.Errorf("%w", types.ErrCeremonyActive)
	}

	// 4. Guardian stake floor (skip if council active — H-5)
	if !k.isCouncilActive(ctx, params) {
		guardianStake := k.GetGuardianStake(ctx)
		minStake, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if ok && guardianStake.Cmp(minStake) < 0 {
			return nil, fmt.Errorf("%w: total guardian stake %s < minimum %s", types.ErrInsufficientGuardians, guardianStake.String(), params.MinGuardianStake)
		}
	}

	// 5. Per-guardian limit
	if k.GetGuardianProposalCount(ctx, msg.Proposer) >= params.MaxProposalsPerGuardianPerEpoch {
		return nil, fmt.Errorf("%w: guardian %s already proposed this epoch", types.ErrProposalLimitExceeded, msg.Proposer)
	}

	// 6. Global limit
	if k.GetEpochProposalCount(ctx) >= params.MaxProposalsPerEpoch {
		return nil, fmt.Errorf("%w: epoch proposal limit reached", types.ErrProposalLimitExceeded)
	}

	// 7. Cooldown
	currentBlock := uint64(ctx.BlockHeight())
	lastBlock := k.GetLastProposalBlock(ctx)
	if lastBlock > 0 && (currentBlock-lastBlock) < params.CooldownBlocks {
		return nil, fmt.Errorf("%w: %d blocks remaining", types.ErrCooldownActive, params.CooldownBlocks-(currentBlock-lastBlock))
	}

	// 8. Create proposal
	proposalId := fmt.Sprintf("halt-%d-%s", ctx.BlockHeight(), msg.Proposer)
	proposal := types.EmergencyHaltProposal{
		Id:              proposalId,
		Proposer:        msg.Proposer,
		Reason:          msg.Reason,
		ProposedAtBlock: currentBlock,
	}

	// 9. Create ceremony
	ceremony, err := k.CreateHaltCeremony(ctx, proposal)
	if err != nil {
		return nil, err
	}

	// 10. Update status
	k.SetEmergencyStatus(ctx, types.StatusHaltVoting)

	// 11. Update anti-abuse counters
	k.IncrementGuardianProposalCount(ctx, msg.Proposer)
	k.IncrementEpochProposalCount(ctx)
	k.SetLastProposalBlock(ctx, currentBlock)

	// 12. Audit
	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: currentBlock,
		Action:      string(types.AuditHaltProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details:     msg.Reason,
	})

	// 13. Event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("emergency_halt_proposed",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	k.Logger(ctx).Info("halt ceremony proposed",
		"ceremony_id", ceremony.Id,
		"proposer", msg.Proposer,
	)

	return &types.MsgProposeHaltResponse{ProposalId: ceremony.Id}, nil
}

// VoteHalt handles a Guardian's vote on a halt ceremony.
func (k msgServer) VoteHalt(goCtx context.Context, msg *types.MsgVoteHalt) (*types.MsgVoteHaltResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGuardian(ctx, msg.Voter) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Voter)
	}

	ceremony, found := k.GetCeremony(ctx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if ceremony.Type != string(types.CeremonyHalt) {
		return nil, fmt.Errorf("%w: ceremony %s is not a halt ceremony", types.ErrInvalidPhase, msg.ProposalId)
	}

	// Determine whether this is a prevote or precommit based on ceremony phase.
	// In the proto schema there is no IsPrecommit field — the phase determines the action.
	if ceremony.Phase == string(types.PhasePrecommit) {
		err := k.AddPrecommit(ctx, msg.ProposalId, types.EmergencyPrecommit{
			Voter: msg.Voter,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditHaltPrecommit, msg.Voter, msg.ProposalId)
	} else {
		err := k.AddPrevote(ctx, msg.ProposalId, types.EmergencyVote{
			Voter:   msg.Voter,
			Approve: msg.Approve,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditHaltPrevote, msg.Voter, msg.ProposalId)
	}

	// Check progress after vote
	finalized, _ := k.CheckCeremonyProgress(ctx, msg.ProposalId)

	// Get updated ceremony for response
	ceremony, _ = k.GetCeremony(ctx, msg.ProposalId)

	if finalized {
		k.HandleCeremonyFinalization(ctx, msg.ProposalId)
	} else if ceremony.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(ctx, msg.ProposalId)
	}

	return &types.MsgVoteHaltResponse{
		QuorumReached: ceremony.Phase == string(types.PhasePrecommit) || ceremony.Phase == string(types.PhaseFinalized),
		ChainHalted:   finalized,
	}, nil
}

// ProposeRevert handles a Guardian's proposal to revert chain state.
func (k msgServer) ProposeRevert(goCtx context.Context, msg *types.MsgProposeRevert) (*types.MsgProposeRevertResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	if !k.IsGuardian(ctx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		return nil, fmt.Errorf("%w: chain must be halted to propose revert", types.ErrHaltRequired)
	}

	if _, found := k.GetActiveCeremony(ctx); found {
		return nil, fmt.Errorf("%w", types.ErrCeremonyActive)
	}

	// Validate revert depth
	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > msg.RevertToHeight && (currentBlock-msg.RevertToHeight) > params.MaxRevertDepth {
		return nil, fmt.Errorf("%w: depth %d exceeds max %d", types.ErrRevertDepthExceeded, currentBlock-msg.RevertToHeight, params.MaxRevertDepth)
	}

	proposalId := fmt.Sprintf("revert-%d-%s", ctx.BlockHeight(), msg.Proposer)
	proposal := types.EmergencyRevertProposal{
		Id:                proposalId,
		Proposer:          msg.Proposer,
		TargetBlockNumber: msg.RevertToHeight,
		Justification:     msg.Justification,
	}

	ceremony, err := k.CreateRevertCeremony(ctx, proposal)
	if err != nil {
		return nil, err
	}

	k.SetEmergencyStatus(ctx, types.StatusRevertVoting)

	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: currentBlock,
		Action:      string(types.AuditRevertProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details:     fmt.Sprintf("revert to block %d", msg.RevertToHeight),
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("emergency_revert_proposed",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("target_block", fmt.Sprintf("%d", msg.RevertToHeight)),
		),
	)

	k.Logger(ctx).Info("revert ceremony proposed",
		"ceremony_id", ceremony.Id,
		"target_block", msg.RevertToHeight,
	)

	return &types.MsgProposeRevertResponse{ProposalId: ceremony.Id}, nil
}

// VoteRevert handles a Guardian's vote on a revert ceremony.
func (k msgServer) VoteRevert(goCtx context.Context, msg *types.MsgVoteRevert) (*types.MsgVoteRevertResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGuardian(ctx, msg.Voter) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Voter)
	}

	ceremony, found := k.GetCeremony(ctx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if ceremony.Type != string(types.CeremonyRevert) {
		return nil, fmt.Errorf("%w: ceremony %s is not a revert ceremony", types.ErrInvalidPhase, msg.ProposalId)
	}

	// Phase-based prevote/precommit determination
	if ceremony.Phase == string(types.PhasePrecommit) {
		err := k.AddPrecommit(ctx, msg.ProposalId, types.EmergencyPrecommit{
			Voter: msg.Voter,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditRevertPrecommit, msg.Voter, msg.ProposalId)
	} else {
		err := k.AddPrevote(ctx, msg.ProposalId, types.EmergencyVote{
			Voter:   msg.Voter,
			Approve: msg.Approve,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditRevertPrevote, msg.Voter, msg.ProposalId)
	}

	finalized, _ := k.CheckCeremonyProgress(ctx, msg.ProposalId)
	ceremony, _ = k.GetCeremony(ctx, msg.ProposalId)

	if finalized {
		k.HandleCeremonyFinalization(ctx, msg.ProposalId)
	} else if ceremony.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(ctx, msg.ProposalId)
	}

	return &types.MsgVoteRevertResponse{
		QuorumReached:  ceremony.Phase == string(types.PhasePrecommit) || ceremony.Phase == string(types.PhaseFinalized),
		RevertExecuted: finalized,
	}, nil
}

// ProposeResume handles a Guardian's proposal to resume chain operations.
func (k msgServer) ProposeResume(goCtx context.Context, msg *types.MsgProposeResume) (*types.MsgProposeResumeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGuardian(ctx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		return nil, fmt.Errorf("%w: chain must be halted to propose resume", types.ErrHaltRequired)
	}

	if _, found := k.GetActiveCeremony(ctx); found {
		return nil, fmt.Errorf("%w", types.ErrCeremonyActive)
	}

	proposalId := fmt.Sprintf("resume-%d-%s", ctx.BlockHeight(), msg.Proposer)
	proposal := types.EmergencyResumeProposal{
		Id:       proposalId,
		Proposer: msg.Proposer,
	}

	ceremony, err := k.CreateResumeCeremony(ctx, proposal)
	if err != nil {
		return nil, err
	}

	k.SetEmergencyStatus(ctx, types.StatusResumeVoting)

	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: uint64(ctx.BlockHeight()),
		Action:      string(types.AuditResumeProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details:     "resume proposed",
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("emergency_resume_proposed",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute("proposer", msg.Proposer),
		),
	)

	k.Logger(ctx).Info("resume ceremony proposed",
		"ceremony_id", ceremony.Id,
		"proposer", msg.Proposer,
	)

	return &types.MsgProposeResumeResponse{ProposalId: ceremony.Id}, nil
}

// VoteResume handles a Guardian's vote on a resume ceremony.
func (k msgServer) VoteResume(goCtx context.Context, msg *types.MsgVoteResume) (*types.MsgVoteResumeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGuardian(ctx, msg.Voter) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Voter)
	}

	ceremony, found := k.GetCeremony(ctx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if ceremony.Type != string(types.CeremonyResume) {
		return nil, fmt.Errorf("%w: ceremony %s is not a resume ceremony", types.ErrInvalidPhase, msg.ProposalId)
	}

	// Phase-based prevote/precommit determination
	if ceremony.Phase == string(types.PhasePrecommit) {
		err := k.AddPrecommit(ctx, msg.ProposalId, types.EmergencyPrecommit{
			Voter: msg.Voter,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditResumePrecommit, msg.Voter, msg.ProposalId)
	} else {
		err := k.AddPrevote(ctx, msg.ProposalId, types.EmergencyVote{
			Voter:   msg.Voter,
			Approve: msg.Approve,
		})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(ctx, types.AuditResumePrevote, msg.Voter, msg.ProposalId)
	}

	finalized, _ := k.CheckCeremonyProgress(ctx, msg.ProposalId)
	ceremony, _ = k.GetCeremony(ctx, msg.ProposalId)

	if finalized {
		k.HandleCeremonyFinalization(ctx, msg.ProposalId)
	} else if ceremony.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(ctx, msg.ProposalId)
	}

	return &types.MsgVoteResumeResponse{
		QuorumReached: ceremony.Phase == string(types.PhasePrecommit) || ceremony.Phase == string(types.PhaseFinalized),
		ChainResumed:  finalized,
	}, nil
}

// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"emergency_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// --- Helpers ---

func (k msgServer) addVoteAudit(ctx sdk.Context, action types.AuditAction, voter, ceremonyId string) {
	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: uint64(ctx.BlockHeight()),
		Action:      string(action),
		Actor:       voter,
		CeremonyId:  ceremonyId,
		Details:     "",
	})
}
