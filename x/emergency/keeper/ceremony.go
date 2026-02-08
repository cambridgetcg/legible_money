package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/types"
)

// CreateHaltCeremony creates a new halt ceremony.
func (k Keeper) CreateHaltCeremony(ctx sdk.Context, proposal types.EmergencyHaltProposal) (types.EmergencyCeremony, error) {
	params := k.GetParams(ctx)
	startBlock := uint64(ctx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              types.CeremonyHalt,
		Phase:             types.PhasePrevote,
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.HaltPrevoteBlocks,
		PrecommitDeadline: startBlock + params.HaltPrevoteBlocks + params.HaltPrecommitBlocks,
		TimeoutDeadline:   startBlock + params.HaltTimeoutBlocks,
		Prevotes:          make(map[string]types.EmergencyVote),
		Precommits:        make(map[string]types.EmergencyPrecommit),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}

	// Marshal proposal into ProposalData
	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return types.EmergencyCeremony{}, fmt.Errorf("failed to marshal halt proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	k.SetCeremony(ctx, ceremony)
	return ceremony, nil
}

// CreateRevertCeremony creates a new revert ceremony.
func (k Keeper) CreateRevertCeremony(ctx sdk.Context, proposal types.EmergencyRevertProposal) (types.EmergencyCeremony, error) {
	params := k.GetParams(ctx)
	startBlock := uint64(ctx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              types.CeremonyRevert,
		Phase:             types.PhasePrevote,
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.RevertPrevoteBlocks,
		PrecommitDeadline: startBlock + params.RevertPrevoteBlocks + params.RevertPrecommitBlocks,
		TimeoutDeadline:   startBlock + params.RevertTimeoutBlocks,
		Prevotes:          make(map[string]types.EmergencyVote),
		Precommits:        make(map[string]types.EmergencyPrecommit),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}

	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return types.EmergencyCeremony{}, fmt.Errorf("failed to marshal revert proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	k.SetCeremony(ctx, ceremony)
	return ceremony, nil
}

// CreateResumeCeremony creates a new resume ceremony.
func (k Keeper) CreateResumeCeremony(ctx sdk.Context, proposal types.EmergencyResumeProposal) (types.EmergencyCeremony, error) {
	params := k.GetParams(ctx)
	startBlock := uint64(ctx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              types.CeremonyResume,
		Phase:             types.PhasePrevote,
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.ResumePrevoteBlocks,
		PrecommitDeadline: startBlock + params.ResumePrevoteBlocks + params.ResumePrecommitBlocks,
		TimeoutDeadline:   startBlock + params.ResumeTimeoutBlocks,
		Prevotes:          make(map[string]types.EmergencyVote),
		Precommits:        make(map[string]types.EmergencyPrecommit),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}

	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return types.EmergencyCeremony{}, fmt.Errorf("failed to marshal resume proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	k.SetCeremony(ctx, ceremony)
	return ceremony, nil
}

// AddPrevote adds a prevote to a ceremony.
func (k Keeper) AddPrevote(ctx sdk.Context, ceremonyId string, vote types.EmergencyVote) error {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	if ceremony.Phase != types.PhasePrevote {
		return fmt.Errorf("%w: expected prevote, got %s", types.ErrInvalidPhase, ceremony.Phase)
	}

	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > ceremony.PrevoteDeadline {
		return fmt.Errorf("%w: prevote deadline passed", types.ErrCeremonyTimedOut)
	}

	if !k.IsGuardian(ctx, vote.Voter) {
		return fmt.Errorf("%w: %s", types.ErrNotGuardian, vote.Voter)
	}

	if _, exists := ceremony.Prevotes[vote.Voter]; exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateVote, vote.Voter)
	}

	// Record vote and update stake tallies
	ceremony.Prevotes[vote.Voter] = vote
	voterStake := k.GetGuardianEffectiveStake(ctx, vote.Voter)

	if vote.Approve {
		yesStake, _ := new(big.Int).SetString(ceremony.YesPrevoteStake, 10)
		if yesStake == nil {
			yesStake = new(big.Int)
		}
		yesStake.Add(yesStake, voterStake)
		ceremony.YesPrevoteStake = yesStake.String()
	} else {
		noStake, _ := new(big.Int).SetString(ceremony.NoPrevoteStake, 10)
		if noStake == nil {
			noStake = new(big.Int)
		}
		noStake.Add(noStake, voterStake)
		ceremony.NoPrevoteStake = noStake.String()
	}

	k.SetCeremony(ctx, ceremony)
	return nil
}

// AddPrecommit adds a precommit to a ceremony.
func (k Keeper) AddPrecommit(ctx sdk.Context, ceremonyId string, precommit types.EmergencyPrecommit) error {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	if ceremony.Phase != types.PhasePrecommit {
		return fmt.Errorf("%w: expected precommit, got %s", types.ErrInvalidPhase, ceremony.Phase)
	}

	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > ceremony.PrecommitDeadline {
		return fmt.Errorf("%w: precommit deadline passed", types.ErrCeremonyTimedOut)
	}

	if !k.IsGuardian(ctx, precommit.Voter) {
		return fmt.Errorf("%w: %s", types.ErrNotGuardian, precommit.Voter)
	}

	// Must have prevoted "yes"
	prevote, hasPrevoted := ceremony.Prevotes[precommit.Voter]
	if !hasPrevoted || !prevote.Approve {
		return fmt.Errorf("%w: %s", types.ErrPrevoteRequired, precommit.Voter)
	}

	if _, exists := ceremony.Precommits[precommit.Voter]; exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateVote, precommit.Voter)
	}

	// Record precommit and update stake tally
	ceremony.Precommits[precommit.Voter] = precommit
	voterStake := k.GetGuardianEffectiveStake(ctx, precommit.Voter)

	precommitStake, _ := new(big.Int).SetString(ceremony.PrecommitStake, 10)
	if precommitStake == nil {
		precommitStake = new(big.Int)
	}
	precommitStake.Add(precommitStake, voterStake)
	ceremony.PrecommitStake = precommitStake.String()

	k.SetCeremony(ctx, ceremony)
	return nil
}

// CheckCeremonyProgress checks and advances ceremony phase. Returns true if finalized.
func (k Keeper) CheckCeremonyProgress(ctx sdk.Context, ceremonyId string) (bool, error) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return false, fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	// Terminal states — no further processing
	if ceremony.Phase == types.PhaseFinalized || ceremony.Phase == types.PhaseFailed {
		return ceremony.Phase == types.PhaseFinalized, nil
	}

	currentBlock := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	// Overall timeout
	if currentBlock > ceremony.TimeoutDeadline {
		ceremony.Phase = types.PhaseFailed
		ceremony.FailureReason = "ceremony timed out"
		k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	// Guardian stake check
	guardianStake := k.GetGuardianStake(ctx)
	if guardianStake.Sign() == 0 {
		ceremony.Phase = types.PhaseFailed
		ceremony.FailureReason = "no active guardians"
		k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	// Min guardian stake floor (skip if council active — H-5)
	if !k.isCouncilActive(ctx, params) {
		minStake, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if ok && guardianStake.Cmp(minStake) < 0 {
			ceremony.Phase = types.PhaseFailed
			ceremony.FailureReason = "insufficient total guardian stake"
			k.SetCeremony(ctx, ceremony)
			return false, nil
		}
	}

	threshold := getQuorumThreshold(ceremony.Type, params)

	switch ceremony.Phase {
	case types.PhasePrevote:
		return k.checkPrevotePhase(ctx, &ceremony, currentBlock, guardianStake, threshold)
	case types.PhasePrecommit:
		return k.checkPrecommitPhase(ctx, &ceremony, currentBlock, guardianStake, threshold, params)
	}

	return false, nil
}

func (k Keeper) checkPrevotePhase(ctx sdk.Context, ceremony *types.EmergencyCeremony, currentBlock uint64, guardianStake *big.Int, threshold uint64) (bool, error) {
	yesStake, _ := new(big.Int).SetString(ceremony.YesPrevoteStake, 10)
	if yesStake == nil {
		yesStake = new(big.Int)
	}
	noStake, _ := new(big.Int).SetString(ceremony.NoPrevoteStake, 10)
	if noStake == nil {
		noStake = new(big.Int)
	}

	// Check if yes quorum reached → advance to precommit
	// yesRatio = yesStake * 1000000 / guardianStake
	yesRatio := new(big.Int).Mul(yesStake, big.NewInt(1000000))
	yesRatio.Div(yesRatio, guardianStake)

	if yesRatio.Uint64() >= threshold {
		ceremony.Phase = types.PhasePrecommit
		k.SetCeremony(ctx, *ceremony)
		return false, nil
	}

	// Check if quorum is impossible (too many no votes)
	noRatio := new(big.Int).Mul(noStake, big.NewInt(1000000))
	noRatio.Div(noRatio, guardianStake)

	if noRatio.Uint64() > (1000000 - threshold) {
		ceremony.Phase = types.PhaseFailed
		ceremony.FailureReason = "quorum impossible: too many no votes"
		k.SetCeremony(ctx, *ceremony)
		return false, nil
	}

	// Check prevote deadline
	if currentBlock > ceremony.PrevoteDeadline {
		ceremony.Phase = types.PhaseFailed
		ceremony.FailureReason = "prevote quorum not reached before deadline"
		k.SetCeremony(ctx, *ceremony)
		return false, nil
	}

	return false, nil
}

func (k Keeper) checkPrecommitPhase(ctx sdk.Context, ceremony *types.EmergencyCeremony, currentBlock uint64, guardianStake *big.Int, threshold uint64, params types.Params) (bool, error) {
	precommitStake, _ := new(big.Int).SetString(ceremony.PrecommitStake, 10)
	if precommitStake == nil {
		precommitStake = new(big.Int)
	}

	// precommitRatio = precommitStake * 1000000 / guardianStake
	precommitRatio := new(big.Int).Mul(precommitStake, big.NewInt(1000000))
	precommitRatio.Div(precommitRatio, guardianStake)

	distinctVoters := uint64(len(ceremony.Precommits))

	if precommitRatio.Uint64() >= threshold {
		if distinctVoters >= params.MinDistinctVoters {
			// Finalized!
			ceremony.Phase = types.PhaseFinalized
			k.SetCeremony(ctx, *ceremony)
			return true, nil
		}
		// Stake met but not enough distinct voters — wait (don't fail yet)
	}

	// Check precommit deadline
	if currentBlock > ceremony.PrecommitDeadline {
		if precommitRatio.Uint64() >= threshold && distinctVoters < params.MinDistinctVoters {
			ceremony.Phase = types.PhaseFailed
			ceremony.FailureReason = fmt.Sprintf("insufficient distinct voters: %d < %d", distinctVoters, params.MinDistinctVoters)
		} else {
			ceremony.Phase = types.PhaseFailed
			ceremony.FailureReason = "precommit quorum not reached before deadline"
		}
		k.SetCeremony(ctx, *ceremony)
		return false, nil
	}

	return false, nil
}

// HandleCeremonyFinalization processes a finalized ceremony's effects.
func (k Keeper) HandleCeremonyFinalization(ctx sdk.Context, ceremonyId string) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found || ceremony.Phase != types.PhaseFinalized {
		return
	}

	switch ceremony.Type {
	case types.CeremonyHalt:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, ceremony.Id)
		k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
			Timestamp:   ctx.BlockTime().Unix(),
			BlockNumber: uint64(ctx.BlockHeight()),
			Action:      types.AuditHaltExecuted,
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "halt ceremony finalized and executed",
		})

	case types.CeremonyRevert:
		// Revert returns to halted state — resume still required
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
			Timestamp:   ctx.BlockTime().Unix(),
			BlockNumber: uint64(ctx.BlockHeight()),
			Action:      types.AuditRevertExecuted,
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "revert ceremony finalized and executed",
		})

	case types.CeremonyResume:
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		k.SetActiveHaltCeremonyId(ctx, "")
		k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
			Timestamp:   ctx.BlockTime().Unix(),
			BlockNumber: uint64(ctx.BlockHeight()),
			Action:      types.AuditResumeExecuted,
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "resume ceremony finalized and executed",
		})
	}
}

// HandleCeremonyFailure processes a failed ceremony's effects.
func (k Keeper) HandleCeremonyFailure(ctx sdk.Context, ceremonyId string) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found || ceremony.Phase != types.PhaseFailed {
		return
	}

	var auditAction types.AuditAction
	switch ceremony.Type {
	case types.CeremonyHalt:
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		auditAction = types.AuditHaltFailed
	case types.CeremonyRevert:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		auditAction = types.AuditRevertFailed
	case types.CeremonyResume:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		auditAction = types.AuditResumeFailed
	}

	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: uint64(ctx.BlockHeight()),
		Action:      auditAction,
		Actor:       "system",
		CeremonyId:  ceremony.Id,
		Details:     fmt.Sprintf("ceremony failed: %s", ceremony.FailureReason),
	})
}

// --- Helpers ---

func getQuorumThreshold(ceremonyType types.CeremonyType, params types.Params) uint64 {
	switch ceremonyType {
	case types.CeremonyHalt:
		return params.HaltQuorum
	case types.CeremonyRevert:
		return params.RevertQuorum
	case types.CeremonyResume:
		return params.ResumeQuorum
	default:
		return 800000 // safe default
	}
}

func marshalProposal(proposal interface{}) ([]byte, error) {
	return json.Marshal(proposal)
}
