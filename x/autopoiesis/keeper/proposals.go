package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ============ Auto-Governance Proposals (Loop 4 — Structural Evolution) ============
//
// CheckProposalTriggers evaluates whether sustained hormonal signals indicate
// structural issues that multiplier adjustments alone cannot fix. When conditions
// are met, it drafts a governance proposal suggestion and emits it as an event
// for off-chain agents to pick up and submit via MsgSubmitProposal.

// CheckProposalTriggers evaluates all auto-proposal conditions.
// Called at end of CollectAndAdapt(), after all adaptations are written.
func (k Keeper) CheckProposalTriggers(ctx sdk.Context, epoch uint64, hormones types.HormoneValues) {
	// Rate limiting: max 1 auto-proposal per 100 epochs
	proposalState := k.GetAutoProposalState(ctx)
	if proposalState.LastProposalEpoch > 0 && epoch-proposalState.LastProposalEpoch < types.AutoProposalCooldownEpochs {
		return
	}

	// Check each trigger type
	if trigger := k.checkBoundHitTrigger(ctx, epoch); trigger != nil {
		k.emitProposalSuggestion(ctx, epoch, trigger, &proposalState)
		return // only 1 proposal per check
	}

	if trigger := k.checkSustainedStressTrigger(ctx, epoch, hormones); trigger != nil {
		k.emitProposalSuggestion(ctx, epoch, trigger, &proposalState)
		return
	}

	if trigger := k.checkDomainStressTrigger(ctx, epoch, hormones); trigger != nil {
		k.emitProposalSuggestion(ctx, epoch, trigger, &proposalState)
		return
	}

	if trigger := k.checkOpenClawCriticalTrigger(ctx); trigger != nil {
		k.emitProposalSuggestion(ctx, epoch, trigger, &proposalState)
		return
	}
}

// checkBoundHitTrigger checks if any multiplier is stuck at its bound for 20+ epochs.
func (k Keeper) checkBoundHitTrigger(ctx sdk.Context, epoch uint64) *types.ProposalTrigger {
	// Check system-wide paths against their bounds
	systemPaths := []struct {
		path   string
		bounds types.MultiplierBounds
	}{
		{types.PathSlashSystem, types.BoundsSlashMultiplier},
		{types.PathBlockRewardSystem, types.BoundsBlockRewardMultiplier},
		{types.PathMaxTxSystem, types.BoundsMaxTxMultiplier},
		{types.PathGasSystem, types.BoundsGasCostMultiplier},
	}

	for _, sp := range systemPaths {
		entry := k.GetMultiplierEntry(ctx, sp.path)
		if entry == nil {
			continue
		}

		atBound := entry.Value <= sp.bounds.Min || entry.Value >= sp.bounds.Max
		if !atBound {
			// Reset tracking for this path
			k.resetTriggerState(ctx, types.TriggerBoundHit, sp.path, "")
			continue
		}

		// Track consecutive epochs at bound
		state := k.GetTriggerState(ctx, types.TriggerBoundHit, sp.path, "")
		if state == nil {
			state = &types.ProposalTriggerState{
				Type: types.TriggerBoundHit,
				Path: sp.path,
			}
		}

		if state.LastCheckedEpoch > 0 && epoch-state.LastCheckedEpoch <= uint64(types.LearningIntervalEpochs)+1 {
			state.ConsecutiveEpochs++
		} else {
			state.ConsecutiveEpochs = 1
		}
		state.LastCheckedEpoch = epoch
		k.SetTriggerState(ctx, *state)

		if state.ConsecutiveEpochs >= types.BoundHitThresholdEpochs {
			return &types.ProposalTrigger{
				Type:              types.TriggerBoundHit,
				Path:              sp.path,
				ConsecutiveEpochs: state.ConsecutiveEpochs,
				Threshold:         types.BoundHitThresholdEpochs,
				CurrentValue:      entry.Value,
			}
		}
	}

	return nil
}

// checkSustainedStressTrigger checks if SSI has been above moderate for 10+ epochs.
func (k Keeper) checkSustainedStressTrigger(ctx sdk.Context, epoch uint64, hormones types.HormoneValues) *types.ProposalTrigger {
	state := k.GetTriggerState(ctx, types.TriggerSustainedStress, "", "")
	if state == nil {
		state = &types.ProposalTriggerState{
			Type: types.TriggerSustainedStress,
		}
	}

	if hormones.SSI >= types.SSIModerate {
		if state.LastCheckedEpoch > 0 && epoch-state.LastCheckedEpoch <= uint64(types.LearningIntervalEpochs)+1 {
			state.ConsecutiveEpochs++
		} else {
			state.ConsecutiveEpochs = 1
		}
	} else {
		state.ConsecutiveEpochs = 0
	}
	state.LastCheckedEpoch = epoch
	k.SetTriggerState(ctx, *state)

	if state.ConsecutiveEpochs >= types.SustainedStressThresholdEpochs {
		return &types.ProposalTrigger{
			Type:              types.TriggerSustainedStress,
			ConsecutiveEpochs: state.ConsecutiveEpochs,
			Threshold:         types.SustainedStressThresholdEpochs,
			CurrentValue:      hormones.SSI,
		}
	}
	return nil
}

// checkDomainStressTrigger checks if any domain DVI has been negative for 10+ epochs.
func (k Keeper) checkDomainStressTrigger(ctx sdk.Context, epoch uint64, hormones types.HormoneValues) *types.ProposalTrigger {
	for _, dvi := range hormones.DVI {
		if dvi.ConsecutiveNegativeEpochs >= types.DomainStressThresholdEpochs {
			return &types.ProposalTrigger{
				Type:              types.TriggerDomainStress,
				Domain:            dvi.Domain,
				ConsecutiveEpochs: dvi.ConsecutiveNegativeEpochs,
				Threshold:         types.DomainStressThresholdEpochs,
			}
		}
	}
	return nil
}

// checkOpenClawCriticalTrigger checks if 3+ verified protocol_observation facts
// have accumulated since the last time this trigger fired. Unlike the epoch-based
// triggers, this is count-based: it queries the knowledge module for facts verified
// after the last acknowledged block height.
func (k Keeper) checkOpenClawCriticalTrigger(ctx sdk.Context) *types.ProposalTrigger {
	if k.knowledgeKeeper == nil {
		return nil
	}

	// Load trigger state to find the block since which to count new facts.
	state := k.GetTriggerState(ctx, types.TriggerOpenClawCritical, "", "")
	var sinceBlock uint64
	if state != nil {
		sinceBlock = state.AcknowledgedAtBlock
	}

	count := k.knowledgeKeeper.GetVerifiedProtocolObservationCount(ctx, sinceBlock)
	if count < types.OpenClawCriticalThreshold {
		return nil
	}

	// Threshold met — update acknowledged block so the same facts don't re-trigger.
	blockHeight := uint64(ctx.BlockHeight())
	if state == nil {
		state = &types.ProposalTriggerState{
			Type: types.TriggerOpenClawCritical,
		}
	}
	state.AcknowledgedAtBlock = blockHeight
	state.LastCheckedEpoch = blockHeight / k.GetParams(ctx).EpochLengthBlocks
	k.SetTriggerState(ctx, *state)

	return &types.ProposalTrigger{
		Type:         types.TriggerOpenClawCritical,
		Threshold:    types.OpenClawCriticalThreshold,
		CurrentValue: count,
	}
}

// emitProposalSuggestion drafts a proposal and emits an event.
func (k Keeper) emitProposalSuggestion(ctx sdk.Context, epoch uint64, trigger *types.ProposalTrigger, state *types.AutoProposalState) {
	proposal := k.draftAutoProposal(epoch, trigger)

	// Emit event for off-chain agents to observe
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("autopoiesis_proposal_suggested",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
			sdk.NewAttribute("trigger_type", string(trigger.Type)),
			sdk.NewAttribute("title", proposal.Title),
			sdk.NewAttribute("description", proposal.Description),
			sdk.NewAttribute("param_path", proposal.ParamPath),
			sdk.NewAttribute("suggested_value", fmt.Sprintf("%d", proposal.SuggestedValue)),
		),
	)

	// Update rate limiting state
	state.LastProposalEpoch = epoch
	state.ProposalsInWindow++
	k.SetAutoProposalState(ctx, *state)

	k.Logger(ctx).Info("auto-proposal suggested",
		"epoch", epoch,
		"trigger", trigger.Type,
		"title", proposal.Title,
	)
}

// draftAutoProposal creates proposal content from a trigger.
// Content is deterministic — derived entirely from on-chain state.
func (k Keeper) draftAutoProposal(epoch uint64, trigger *types.ProposalTrigger) types.AutoProposal {
	proposal := types.AutoProposal{
		Trigger: *trigger,
		Epoch:   epoch,
	}

	switch trigger.Type {
	case types.TriggerBoundHit:
		proposal.Title = fmt.Sprintf("LIP: Adjust base value for %s (bound-hit)", trigger.Path)
		proposal.Description = fmt.Sprintf(
			"The multiplier at path '%s' has been at its bound for %d consecutive epochs. "+
				"Current value: %d BPS. The autopoiesis hormonal system cannot adjust further. "+
				"Governance should consider updating the base parameter value to give the "+
				"adaptive system more room to operate.",
			trigger.Path, trigger.ConsecutiveEpochs, trigger.CurrentValue,
		)
		proposal.ParamPath = trigger.Path
		// Suggest shifting base 10% in the direction the multiplier is pressing
		if trigger.CurrentValue >= types.DefaultMultiplier {
			proposal.SuggestedValue = trigger.CurrentValue + (trigger.CurrentValue * 100_000 / types.BasisPoints)
		} else {
			proposal.SuggestedValue = trigger.CurrentValue - (trigger.CurrentValue * 100_000 / types.BasisPoints)
		}

	case types.TriggerSustainedStress:
		proposal.Title = "LIP: Protocol under sustained stress — reduce verification load"
		proposal.Description = fmt.Sprintf(
			"System Stress Index (SSI) has been above moderate (%d BPS) for %d consecutive epochs. "+
				"Current SSI: %d BPS. Consider reducing challenge windows or increasing minimum verifier count "+
				"to alleviate pressure on the verification system.",
			types.SSIModerate, trigger.ConsecutiveEpochs, trigger.CurrentValue,
		)

	case types.TriggerDomainStress:
		proposal.Title = fmt.Sprintf("LIP: Domain '%s' under sustained vitality stress", trigger.Domain)
		proposal.Description = fmt.Sprintf(
			"Domain '%s' has had negative Domain Vitality Index (DVI) for %d consecutive epochs. "+
				"Consider increasing the domain reward weight or archiving the domain if it is no longer "+
				"producing valuable knowledge.",
			trigger.Domain, trigger.ConsecutiveEpochs,
		)

	case types.TriggerOpenClawCritical:
		proposal.Title = "LIP: Multiple verified protocol observation findings"
		proposal.Description = fmt.Sprintf(
			"%d or more verified protocol_observation facts flag the same structural issue. "+
				"An emergency review is recommended.",
			trigger.Threshold,
		)
	}

	return proposal
}

// ============ Trigger State Store Methods ============

func triggerStateKey(triggerType types.ProposalTriggerType, path, domain string) []byte {
	key := types.TriggerStateKey(triggerType, path, domain)
	return append(types.ProposalTriggerKeyPrefix, []byte(key)...)
}

// SetTriggerState stores a proposal trigger tracking state.
func (k Keeper) SetTriggerState(ctx sdk.Context, state types.ProposalTriggerState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = kvStore.Set(triggerStateKey(state.Type, state.Path, state.Domain), bz)
}

// GetTriggerState returns the trigger tracking state, or nil.
func (k Keeper) GetTriggerState(ctx sdk.Context, triggerType types.ProposalTriggerType, path, domain string) *types.ProposalTriggerState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(triggerStateKey(triggerType, path, domain))
	if err != nil || bz == nil {
		return nil
	}
	var state types.ProposalTriggerState
	if err := json.Unmarshal(bz, &state); err != nil {
		return nil
	}
	return &state
}

// resetTriggerState clears the consecutive epoch count for a trigger.
func (k Keeper) resetTriggerState(ctx sdk.Context, triggerType types.ProposalTriggerType, path, domain string) {
	k.SetTriggerState(ctx, types.ProposalTriggerState{
		Type:   triggerType,
		Path:   path,
		Domain: domain,
	})
}

// SetAutoProposalState stores the auto-proposal rate limiting state.
func (k Keeper) SetAutoProposalState(ctx sdk.Context, state types.AutoProposalState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.AutoProposalStateKey, bz)
}

// GetAutoProposalState returns the auto-proposal rate limiting state.
func (k Keeper) GetAutoProposalState(ctx sdk.Context) types.AutoProposalState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AutoProposalStateKey)
	if err != nil || bz == nil {
		return types.AutoProposalState{}
	}
	var state types.AutoProposalState
	if err := json.Unmarshal(bz, &state); err != nil {
		return types.AutoProposalState{}
	}
	return state
}
