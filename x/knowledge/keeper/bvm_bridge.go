package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// BVM bridge methods implement the x/bvm/types.KnowledgeKeeper write interface.
// These allow BVM smart contracts to interact with the knowledge module pipeline.

// SubmitClaimFromBVM creates a knowledge claim from a BVM contract execution.
// It follows the same pipeline as MsgSubmitClaim but accepts primitive types
// that the BVM host functions can provide.
func (k Keeper) SubmitClaimFromBVM(
	goCtx context.Context,
	submitterDID, domain, contentHash, evidenceHash, logicZone string,
	stakeAmount uint64,
) (string, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate domain exists
	_, domainFound := k.GetDomainFromState(ctx, domain)
	if !domainFound {
		return "", fmt.Errorf("%w: domain %q not found", types.ErrDomainNotFound, domain)
	}

	// Check for duplicate content hash
	if _, found := k.GetClaimByContentHash(ctx, contentHash); found {
		return "", fmt.Errorf("%w: duplicate claim content", types.ErrDuplicateClaim)
	}

	// Determine stake: use provided amount or default minimum
	stake := new(big.Int).SetUint64(stakeAmount)
	if stake.Sign() <= 0 {
		stake = big.NewInt(1000000000000000) // 0.001 LGM default
	}

	// Resolve DID to bech32 address for bank operations
	submitterAddr, err := k.resolveSubmitterAddress(ctx, submitterDID)
	if err != nil {
		return "", fmt.Errorf("cannot resolve submitter DID %s: %w", submitterDID, err)
	}

	// Deduct stake
	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stake))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, submitterAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return "", fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Generate IDs
	height := uint64(ctx.BlockHeight())
	claimID := GenerateClaimID(submitterDID, contentHash, height)
	roundID := GenerateRoundID(claimID, height)

	// Default logic zone
	if logicZone == "" {
		logicZone = "empirical"
	}

	// Create claim
	claim := types.Claim{
		Id:                claimID,
		ClaimText:         fmt.Sprintf("BVM claim: %s", contentHash),
		ContentHash:       contentHash,
		Domain:            domain,
		EpistemicCategory: "empirical",
		Submitter:         submitterAddr.String(),
		SubmitterDid:      submitterDID,
		StakeAmount:       stake.String(),
		Status:            "pending",
		SubmittedAtHeight: height,
		LogicZone:         logicZone,
		RoundId:           roundID,
		Evidence: []*types.ClaimEvidence{
			{EvidenceType: "bvm_hash", ContentHash: evidenceHash},
		},
	}
	k.SetClaim(ctx, &claim)

	// Create verification round
	round, err := k.CreateVerificationRound(ctx, &claim)
	if err != nil {
		return "", fmt.Errorf("failed to create verification round: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"claim_submitted",
		sdk.NewAttribute("claim_id", claimID),
		sdk.NewAttribute("round_id", round.Id),
		sdk.NewAttribute("source", "bvm"),
		sdk.NewAttribute("submitter_did", submitterDID),
	))

	return claimID, nil
}

// CommitVoteFromBVM submits a verification commitment from a BVM contract.
func (k Keeper) CommitVoteFromBVM(
	goCtx context.Context,
	validatorDID, roundId, commitmentHash string,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	round, found := k.GetVerificationRound(ctx, roundId)
	if !found {
		return fmt.Errorf("%w: round %s not found", types.ErrRoundNotFound, roundId)
	}

	if round.Phase != "commit" {
		return fmt.Errorf("%w: round is in %s phase, expected commit", types.ErrWrongPhase, round.Phase)
	}

	if uint64(ctx.BlockHeight()) > round.CommitDeadlineHeight {
		return fmt.Errorf("%w: commit deadline passed", types.ErrDeadlinePassed)
	}

	// Check validator hasn't already committed
	if _, exists := round.Commitments[validatorDID]; exists {
		return fmt.Errorf("%w: validator already committed", types.ErrDuplicateCommitment)
	}

	// Validate commitment hash format
	if len(commitmentHash) != 64 {
		return fmt.Errorf("%w: commitment hash must be 64 hex characters", types.ErrInvalidCommitment)
	}

	round.Commitments[validatorDID] = &types.VerificationCommitment{
		Validator:         validatorDID,
		CommitmentHash:    commitmentHash,
		SubmittedAtHeight: uint64(ctx.BlockHeight()),
	}
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"commitment_submitted",
		sdk.NewAttribute("round_id", roundId),
		sdk.NewAttribute("validator", validatorDID),
		sdk.NewAttribute("source", "bvm"),
	))

	return nil
}

// RevealVoteFromBVM submits a verification reveal from a BVM contract.
func (k Keeper) RevealVoteFromBVM(
	goCtx context.Context,
	validatorDID, roundId, verdict string,
	confidence uint64, salt string,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	round, found := k.GetVerificationRound(ctx, roundId)
	if !found {
		return fmt.Errorf("%w: round %s not found", types.ErrRoundNotFound, roundId)
	}

	if round.Phase != "reveal" {
		return fmt.Errorf("%w: round is in %s phase, expected reveal", types.ErrWrongPhase, round.Phase)
	}

	if uint64(ctx.BlockHeight()) > round.RevealDeadlineHeight {
		return fmt.Errorf("%w: reveal deadline passed", types.ErrDeadlinePassed)
	}

	// Check validator committed
	commitment, hasCommitment := round.Commitments[validatorDID]
	if !hasCommitment {
		return fmt.Errorf("%w: validator did not commit", types.ErrNoCommitment)
	}

	// Verify reveal matches commitment
	expectedHash := ComputeCommitmentHash(roundId, verdict, confidence, salt)
	if expectedHash != commitment.CommitmentHash {
		return fmt.Errorf("%w: reveal does not match commitment", types.ErrRevealMismatch)
	}

	// Check not already revealed
	if _, exists := round.Reveals[validatorDID]; exists {
		return fmt.Errorf("%w: validator already revealed", types.ErrDuplicateReveal)
	}

	round.Reveals[validatorDID] = &types.VerificationReveal{
		Validator:         validatorDID,
		Verdict:           verdict,
		Confidence:        confidence,
		Salt:              salt,
		SubmittedAtHeight: uint64(ctx.BlockHeight()),
	}
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"reveal_submitted",
		sdk.NewAttribute("round_id", roundId),
		sdk.NewAttribute("validator", validatorDID),
		sdk.NewAttribute("verdict", verdict),
		sdk.NewAttribute("source", "bvm"),
	))

	return nil
}

// ChallengeFactFromBVM initiates a fact challenge from a BVM contract.
func (k Keeper) ChallengeFactFromBVM(
	goCtx context.Context,
	challengerDID, factId, evidenceHash string,
	stakeAmount uint64,
) (string, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fact, found := k.GetFact(ctx, factId)
	if !found {
		return "", fmt.Errorf("%w: fact %s not found", types.ErrFactNotFound, factId)
	}

	if fact.Status != "verified" {
		return "", fmt.Errorf("%w: fact is %s, only verified facts can be challenged",
			types.ErrInvalidChallenge, fact.Status)
	}

	if k.HasActiveRoundForClaim(ctx, factId) {
		return "", fmt.Errorf("%w: fact already has an active verification round",
			types.ErrDuplicateChallenge)
	}

	stake := new(big.Int).SetUint64(stakeAmount)
	if stake.Sign() <= 0 {
		return "", fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}

	// Resolve DID to address
	challengerAddr, err := k.resolveSubmitterAddress(ctx, challengerDID)
	if err != nil {
		return "", fmt.Errorf("cannot resolve challenger DID %s: %w", challengerDID, err)
	}

	// Deduct stake
	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stake))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, challengerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return "", fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Mark fact as contested
	fact.Status = "contested"
	k.SetFact(ctx, fact)

	// Create verification round
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)
	roundID := GenerateRoundID(factId+":bvm-challenge", height)

	round := types.VerificationRound{
		Id:                   roundID,
		ClaimId:              factId,
		Phase:                "commit",
		StartHeight:          height,
		CommitDeadlineHeight: height + params.CommitPhaseDuration,
		RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
		SelectedValidators:   []string{},
		Commitments:          make(map[string]*types.VerificationCommitment),
		Reveals:              make(map[string]*types.VerificationReveal),
	}
	k.SetVerificationRound(ctx, &round)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"fact_challenged",
		sdk.NewAttribute("fact_id", factId),
		sdk.NewAttribute("round_id", roundID),
		sdk.NewAttribute("challenger_did", challengerDID),
		sdk.NewAttribute("stake", stake.String()),
		sdk.NewAttribute("source", "bvm"),
	))

	return roundID, nil
}

// resolveSubmitterAddress converts a DID or address string to an sdk.AccAddress.
// Supports:
//   - bech32 addresses directly (e.g. "lgm1abc...")
//   - did:lgm:<bech32> format (extracts the suffix as bech32)
//   - did:lgm:<hex> format (uses hex bytes as address)
func (k Keeper) resolveSubmitterAddress(_ sdk.Context, did string) (sdk.AccAddress, error) {
	// Try direct bech32 first
	if addr, err := sdk.AccAddressFromBech32(did); err == nil {
		return addr, nil
	}

	// Handle did:lgm: prefix
	if len(did) > 8 && did[:8] == "did:lgm:" {
		suffix := did[8:]
		// Try suffix as bech32
		if addr, err := sdk.AccAddressFromBech32(suffix); err == nil {
			return addr, nil
		}
		// Try suffix as hex address bytes
		if len(suffix) >= 40 {
			addrBytes := make([]byte, 20)
			for i := 0; i < 20; i++ {
				var b byte
				if _, err := fmt.Sscanf(suffix[i*2:i*2+2], "%02x", &b); err != nil {
					return nil, fmt.Errorf("cannot parse DID hex %s: %w", did, err)
				}
				addrBytes[i] = b
			}
			return sdk.AccAddress(addrBytes), nil
		}
	}
	return nil, fmt.Errorf("cannot resolve DID to address: %s", did)
}
