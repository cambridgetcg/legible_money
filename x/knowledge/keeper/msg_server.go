package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitClaim handles MsgSubmitClaim — the core PoT transaction.
//
// Workflow:
// 1. Validate claim text length, domain existence, epistemic category
// 2. Check for duplicate content hash
// 3. Deduct stake from submitter
// 4. Create claim with deterministic ID
// 5. Create verification round with selected validators
// 6. Emit events
func (k msgServer) SubmitClaim(goCtx context.Context, msg *types.MsgSubmitClaim) (*types.MsgSubmitClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Validate claim text length
	if uint32(len(msg.ClaimText)) < params.MinClaimTextLength {
		return nil, fmt.Errorf("%w: claim text must be at least %d characters",
			types.ErrInvalidClaim, params.MinClaimTextLength)
	}

	// Validate domain exists
	_, domainFound := k.Keeper.GetDomainFromState(ctx, msg.Domain)
	if !domainFound {
		return nil, fmt.Errorf("%w: domain %q not found", types.ErrDomainNotFound, msg.Domain)
	}

	// Validate epistemic category
	if !isValidEpistemicCategory(msg.EpistemicCategory) {
		return nil, fmt.Errorf("%w: invalid epistemic category %q",
			types.ErrInvalidClaim, msg.EpistemicCategory)
	}

	// Compute content hash and check for duplicates
	contentHash := ComputeClaimContentHash(msg.ClaimText, msg.Domain, msg.EpistemicCategory)
	if _, found := k.GetClaimByContentHash(ctx, contentHash); found {
		return nil, fmt.Errorf("%w: duplicate claim content", types.ErrDuplicateClaim)
	}

	// Deduct stake from submitter
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(msg.StakeAmount, 10); !ok || stakeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}

	submitterAddr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid submitter address: %w", err)
	}

	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, submitterAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Debit from period overlay (FIFO from oldest periods)
	if k.tokensKeeper != nil {
		if err := k.tokensKeeper.DebitBalance(ctx, msg.Submitter, msg.StakeAmount); err != nil {
			k.Logger(ctx).Error("failed to debit period balance for claim stake",
				"submitter", msg.Submitter, "error", err)
		}
	}

	// Generate deterministic IDs
	height := uint64(ctx.BlockHeight())
	claimID := GenerateClaimID(msg.Submitter, contentHash, height)
	roundID := GenerateRoundID(claimID, height)

	// Create claim
	claim := types.Claim{
		Id:                claimID,
		ClaimText:         msg.ClaimText,
		ContentHash:       contentHash,
		Domain:            msg.Domain,
		EpistemicCategory: msg.EpistemicCategory,
		Submitter:         msg.Submitter,
		SubmitterDid:      msg.SubmitterDid,
		StakeAmount:       msg.StakeAmount,
		Status:            "pending",
		SubmittedAtHeight: height,
		References:        msg.References,
		LogicZone:         msg.LogicZone,
		RoundId:           roundID,
		Evidence:          msg.Evidence,
	}
	k.SetClaim(ctx, claim)

	// Create verification round (delegates to rounds.go)
	round, err := k.CreateVerificationRound(ctx, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification round: %w", err)
	}
	roundID = round.Id

	// Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"claim_submitted",
			sdk.NewAttribute("claim_id", claimID),
			sdk.NewAttribute("round_id", roundID),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("category", msg.EpistemicCategory),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("stake", msg.StakeAmount),
			sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
		),
	})

	k.Logger(ctx).Info("claim submitted",
		"claim_id", claimID,
		"round_id", roundID,
		"domain", msg.Domain,
		"submitter", msg.Submitter,
	)

	return &types.MsgSubmitClaimResponse{
		ClaimId: claimID,
		RoundId: roundID,
	}, nil
}

// SubmitCommitment handles MsgSubmitCommitment — validator commits a verdict hash.
func (k msgServer) SubmitCommitment(goCtx context.Context, msg *types.MsgSubmitCommitment) (*types.MsgSubmitCommitmentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the verification round
	round, found := k.GetVerificationRound(ctx, msg.RoundId)
	if !found {
		return nil, fmt.Errorf("%w: round %s not found", types.ErrRoundNotFound, msg.RoundId)
	}

	// Check round is in commit phase
	if round.Phase != "commit" {
		return nil, fmt.Errorf("%w: round is in %s phase, expected commit",
			types.ErrWrongPhase, round.Phase)
	}

	// Check commit deadline not passed
	if uint64(ctx.BlockHeight()) > round.CommitDeadlineHeight {
		return nil, fmt.Errorf("%w: commit deadline passed at height %d",
			types.ErrDeadlinePassed, round.CommitDeadlineHeight)
	}

	// Enforce VRF selection: only selected validators may commit
	if len(round.SelectedValidators) > 0 {
		isSelected := false
		for _, v := range round.SelectedValidators {
			if v == msg.Validator {
				isSelected = true
				break
			}
		}
		if !isSelected {
			return nil, fmt.Errorf("%w: validator %s not selected for round %s",
				types.ErrVRFSelectionFailed, msg.Validator, msg.RoundId)
		}
	}

	// Check validator hasn't already committed
	if _, exists := round.Commitments[msg.Validator]; exists {
		return nil, fmt.Errorf("%w: validator %s already committed",
			types.ErrDuplicateCommitment, msg.Validator)
	}

	// Validate commitment hash format (64 hex chars = SHA-256)
	if len(msg.CommitmentHash) != 64 {
		return nil, fmt.Errorf("%w: commitment hash must be 64 hex characters",
			types.ErrInvalidCommitment)
	}

	// Store commitment
	round.Commitments[msg.Validator] = types.VerificationCommitment{
		Validator:         msg.Validator,
		CommitmentHash:    msg.CommitmentHash,
		SubmittedAtHeight: uint64(ctx.BlockHeight()),
	}
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"commitment_submitted",
			sdk.NewAttribute("round_id", msg.RoundId),
			sdk.NewAttribute("validator", msg.Validator),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return &types.MsgSubmitCommitmentResponse{}, nil
}

// SubmitReveal handles MsgSubmitReveal — validator reveals their verdict.
func (k msgServer) SubmitReveal(goCtx context.Context, msg *types.MsgSubmitReveal) (*types.MsgSubmitRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the verification round
	round, found := k.GetVerificationRound(ctx, msg.RoundId)
	if !found {
		return nil, fmt.Errorf("%w: round %s not found", types.ErrRoundNotFound, msg.RoundId)
	}

	// Check round is in reveal phase
	if round.Phase != "reveal" {
		return nil, fmt.Errorf("%w: round is in %s phase, expected reveal",
			types.ErrWrongPhase, round.Phase)
	}

	// Check reveal deadline not passed
	if uint64(ctx.BlockHeight()) > round.RevealDeadlineHeight {
		return nil, fmt.Errorf("%w: reveal deadline passed at height %d",
			types.ErrDeadlinePassed, round.RevealDeadlineHeight)
	}

	// Check validator committed in commit phase
	commitment, hasCommitment := round.Commitments[msg.Validator]
	if !hasCommitment {
		return nil, fmt.Errorf("%w: validator %s did not commit",
			types.ErrNoCommitment, msg.Validator)
	}

	// Verify reveal matches commitment (commitment_hash = sha256(roundID || verdict || confidence || salt))
	expectedHash := ComputeCommitmentHash(msg.RoundId, msg.Verdict, msg.Confidence, msg.Salt)
	if expectedHash != commitment.CommitmentHash {
		return nil, fmt.Errorf("%w: reveal does not match commitment",
			types.ErrRevealMismatch)
	}

	// Check validator hasn't already revealed
	if _, exists := round.Reveals[msg.Validator]; exists {
		return nil, fmt.Errorf("%w: validator %s already revealed",
			types.ErrDuplicateReveal, msg.Validator)
	}

	// Store reveal
	round.Reveals[msg.Validator] = types.VerificationReveal{
		Validator:         msg.Validator,
		Verdict:           msg.Verdict,
		Confidence:        msg.Confidence,
		Justification:     msg.Justification,
		VrfProof:          msg.VrfProof,
		Salt:              msg.Salt,
		SubmittedAtHeight: uint64(ctx.BlockHeight()),
	}

	// Check if we have enough reveals to aggregate immediately
	roundCompleted := false
	params := k.GetParams(ctx)
	if uint32(len(round.Reveals)) >= params.MinVerifiers {
		// Use stake-weighted confidence aggregation engine
		result, aggErr := k.AggregateVerificationResult(ctx, round)
		if aggErr != nil {
			k.Logger(ctx).Error("aggregation failed", "round_id", round.Id, "error", aggErr)
		} else {
			// Apply rewards and slashes
			if applyErr := k.ApplyRewardsAndSlashes(ctx, result); applyErr != nil {
				k.Logger(ctx).Error("failed to apply rewards/slashes",
					"round_id", round.Id, "error", applyErr)
			}
			// Complete the round (creates fact if accepted, updates claim status)
			if completeErr := k.CompleteRound(ctx, round, result); completeErr != nil {
				k.Logger(ctx).Error("failed to complete round",
					"round_id", round.Id, "error", completeErr)
			}
			roundCompleted = true
		}
	}

	if !roundCompleted {
		k.SetVerificationRound(ctx, round)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"reveal_submitted",
			sdk.NewAttribute("round_id", msg.RoundId),
			sdk.NewAttribute("validator", msg.Validator),
			sdk.NewAttribute("verdict", msg.Verdict),
		),
	)

	return &types.MsgSubmitRevealResponse{}, nil
}

// ChallengeFact handles MsgChallengeFact — initiates reverification of a fact.
func (k msgServer) ChallengeFact(goCtx context.Context, msg *types.MsgChallengeFact) (*types.MsgChallengeFactResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify fact exists
	fact, found := k.GetFact(ctx, msg.FactId)
	if !found {
		return nil, fmt.Errorf("%w: fact %s not found", types.ErrFactNotFound, msg.FactId)
	}

	// Check fact is in a challengeable state
	if fact.Status != "verified" {
		return nil, fmt.Errorf("%w: fact is %s, only verified facts can be challenged",
			types.ErrInvalidChallenge, fact.Status)
	}

	// Deduct challenge stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(msg.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}

	challengerAddr, err := sdk.AccAddressFromBech32(msg.Challenger)
	if err != nil {
		return nil, fmt.Errorf("invalid challenger address: %w", err)
	}

	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, challengerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Debit from period overlay (FIFO from oldest periods)
	if k.tokensKeeper != nil {
		if err := k.tokensKeeper.DebitBalance(ctx, msg.Challenger, msg.Stake); err != nil {
			k.Logger(ctx).Error("failed to debit period balance for challenge stake",
				"challenger", msg.Challenger, "error", err)
		}
	}

	// Mark fact as contested
	fact.Status = "contested"
	k.SetFact(ctx, fact)

	// Create new verification round for the challenge
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)
	roundID := GenerateRoundID(msg.FactId+":challenge", height)

	round := types.VerificationRound{
		Id:                   roundID,
		ClaimId:              msg.FactId, // Reuse fact ID as claim reference
		Phase:                "commit",
		StartHeight:          height,
		CommitDeadlineHeight: height + params.CommitPhaseDuration,
		RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
		SelectedValidators:   []string{},
		Commitments:          make(map[string]types.VerificationCommitment),
		Reveals:              make(map[string]types.VerificationReveal),
	}
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"fact_challenged",
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("round_id", roundID),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("stake", msg.Stake),
		),
	)

	return &types.MsgChallengeFactResponse{RoundId: roundID}, nil
}

// AddFact handles MsgAddFact — governance-gated fact creation (for genesis/migration).
func (k msgServer) AddFact(goCtx context.Context, msg *types.MsgAddFact) (*types.MsgAddFactResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	height := uint64(ctx.BlockHeight())
	contentHash := ComputeClaimContentHash(msg.ClaimText, msg.Domain, msg.EpistemicCategory)
	factID := GenerateFactID(contentHash, fmt.Sprintf("gov:%d", height))

	fact := types.Fact{
		Id:                   factID,
		ClaimText:            msg.ClaimText,
		Domain:               msg.Domain,
		EpistemicCategory:    msg.EpistemicCategory,
		Confidence:           msg.Confidence,
		Status:               "verified",
		Submitter:            msg.Authority,
		References:           msg.References,
		VerifiedAtHeight:     height,
		VerifierCount:        0, // Governance-added, no verifiers
		LogicZone:            msg.LogicZone,
		CreatedAtHeight:      height,
		LastVerifiedHeight:   height,
		LastConfidenceUpdate: height,
		BaseConfidence:       msg.Confidence,
	}
	k.SetFact(ctx, fact)

	// Set references
	for _, refID := range msg.References {
		k.SetFactReference(ctx, types.FactReference{
			FromFactId:      factID,
			ToFactId:        refID,
			ReferenceType:   "supports",
			Weight:          100,
			CreatedAtHeight: height,
		})
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"fact_added",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgAddFactResponse{FactId: factID}, nil
}

// ProposeDomain handles MsgProposeDomain.
func (k msgServer) ProposeDomain(goCtx context.Context, msg *types.MsgProposeDomain) (*types.MsgProposeDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check domain doesn't already exist
	if _, found := k.Keeper.GetDomainFromState(ctx, msg.Name); found {
		return nil, fmt.Errorf("%w: domain %q already exists", types.ErrDomainExists, msg.Name)
	}

	// Deduct stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(msg.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}

	proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
	if err != nil {
		return nil, fmt.Errorf("invalid proposer address: %w", err)
	}

	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, proposerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Debit from period overlay (FIFO from oldest periods)
	if k.tokensKeeper != nil {
		if err := k.tokensKeeper.DebitBalance(ctx, msg.Proposer, msg.Stake); err != nil {
			k.Logger(ctx).Error("failed to debit period balance for domain proposal stake",
				"proposer", msg.Proposer, "error", err)
		}
	}

	domain := types.OntologicalDomain{
		Name:            msg.Name,
		Description:     msg.Description,
		Proposer:        msg.Proposer,
		Status:          "proposed",
		Endorsements:    0,
		Challenges:      0,
		CreatedAtHeight: uint64(ctx.BlockHeight()),
	}
	k.Keeper.SetDomainInState(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_proposed",
			sdk.NewAttribute("domain", msg.Name),
			sdk.NewAttribute("proposer", msg.Proposer),
		),
	)

	return &types.MsgProposeDomainResponse{DomainName: msg.Name}, nil
}

// EndorseDomainProposal handles MsgEndorseDomainProposal.
func (k msgServer) EndorseDomainProposal(goCtx context.Context, msg *types.MsgEndorseDomainProposal) (*types.MsgEndorseDomainProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	domain, found := k.Keeper.GetDomainFromState(ctx, msg.DomainName)
	if !found {
		return nil, fmt.Errorf("%w: domain %q not found", types.ErrDomainNotFound, msg.DomainName)
	}

	if domain.Status != "proposed" {
		return nil, fmt.Errorf("domain %q is %s, not proposed", msg.DomainName, domain.Status)
	}

	domain.Endorsements++

	// Check if enough endorsements to activate
	params := k.GetParams(ctx)
	if domain.Endorsements >= params.MinEndorsements {
		domain.Status = "active"
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("domain_activated", sdk.NewAttribute("domain", msg.DomainName)),
		)
	}

	k.Keeper.SetDomainInState(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_endorsed",
			sdk.NewAttribute("domain", msg.DomainName),
			sdk.NewAttribute("endorser", msg.Endorser),
			sdk.NewAttribute("endorsements", fmt.Sprintf("%d", domain.Endorsements)),
		),
	)

	return &types.MsgEndorseDomainProposalResponse{}, nil
}

// ChallengeDomainProposal handles MsgChallengeDomainProposal.
func (k msgServer) ChallengeDomainProposal(goCtx context.Context, msg *types.MsgChallengeDomainProposal) (*types.MsgChallengeDomainProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	domain, found := k.Keeper.GetDomainFromState(ctx, msg.DomainName)
	if !found {
		return nil, fmt.Errorf("%w: domain %q not found", types.ErrDomainNotFound, msg.DomainName)
	}

	if domain.Status != "proposed" {
		return nil, fmt.Errorf("domain %q is %s, not proposed", msg.DomainName, domain.Status)
	}

	domain.Challenges++
	k.Keeper.SetDomainInState(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_challenged",
			sdk.NewAttribute("domain", msg.DomainName),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgChallengeDomainProposalResponse{}, nil
}

// RegisterStratum handles MsgRegisterStratum.
func (k msgServer) RegisterStratum(goCtx context.Context, msg *types.MsgRegisterStratum) (*types.MsgRegisterStratumResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.Logger(ctx).Info("RegisterStratum", "stratum", msg.StratumName, "registrar", msg.Registrar)
	// Stratum registration is a Phase 2 feature (ontology module)
	return &types.MsgRegisterStratumResponse{}, nil
}

// UpdateParams handles MsgUpdateParams.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)
	return &types.MsgUpdateParamsResponse{}, nil
}

// ---------- Internal Helpers ----------

// isValidEpistemicCategory checks if the category is one of the valid types.
// Matches the TS reference: analytic, formal, protocol, empirical, historical, computational, social, predictive, replicated.
func isValidEpistemicCategory(cat string) bool {
	switch cat {
	case "protocol", "analytic", "formal", "empirical", "historical",
		"computational", "social", "predictive", "replicated":
		return true
	}
	return false
}

// ComputeCommitmentHash generates a commitment hash for verification.
// hash = SHA-256(roundID || verdict || confidence_bytes || salt)
func ComputeCommitmentHash(roundID, verdict string, confidence uint64, salt string) string {
	return ComputeClaimContentHash(roundID+verdict+fmt.Sprintf("%d", confidence), salt, "commitment")
}
