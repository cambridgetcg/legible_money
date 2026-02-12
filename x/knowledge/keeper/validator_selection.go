package keeper

import (
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vrfcrypto "github.com/legible-money/legible/x/knowledge/crypto"
	"github.com/legible-money/legible/x/knowledge/types"
)

// GetEligibleValidators returns all validators eligible to participate
// in verification rounds. Filters by active status and minimum tier requirements.
func (k Keeper) GetEligibleValidators(ctx sdk.Context) []types.ValidatorInfo {
	return k.stakingKeeper.GetActiveValidatorInfos(ctx)
}

// GetEligibleValidatorsForDomain returns validators eligible to verify claims
// in a specific domain. Filters by domain qualification when the qualification
// keeper is available, falling back to all active validators otherwise.
func (k Keeper) GetEligibleValidatorsForDomain(ctx sdk.Context, domain string) []types.ValidatorInfo {
	all := k.stakingKeeper.GetActiveValidatorInfos(ctx)
	if k.qualificationKeeper == nil {
		return all // graceful nil-safe fallback
	}
	var qualified []types.ValidatorInfo
	for _, v := range all {
		if k.qualificationKeeper.IsQualified(ctx, v.Address, domain) {
			qualified = append(qualified, v)
		}
	}
	if len(qualified) == 0 {
		return all // don't block verification if no one qualified
	}
	return qualified
}

// GetTotalStake returns the total stake of all active validators.
func (k Keeper) GetTotalStake(ctx sdk.Context) uint64 {
	return k.stakingKeeper.GetTotalStake(ctx)
}

// VerifyValidatorVRFSelection verifies that a validator was legitimately
// selected for a verification round via VRF.
//
// Checks:
// 1. VRF proof is cryptographically valid for the validator's public key
// 2. VRF output qualifies the validator for selection (stake-weighted probability)
// 3. Validator is active and eligible
func (k Keeper) VerifyValidatorVRFSelection(
	ctx sdk.Context,
	round types.VerificationRound,
	validatorAddr string,
	vrfOutputHex string,
	vrfProofHex string,
) error {
	// Get validator info
	validatorInfo, found := k.stakingKeeper.GetValidatorInfo(ctx, validatorAddr)
	if !found {
		return fmt.Errorf("%w: validator %s", types.ErrValidatorNotEligible, validatorAddr)
	}

	if len(validatorInfo.PublicKey) == 0 {
		return fmt.Errorf("%w: validator %s has no public key", types.ErrValidatorNotEligible, validatorAddr)
	}

	// Decode VRF proof
	vrfProof, err := hex.DecodeString(vrfProofHex)
	if err != nil {
		return fmt.Errorf("%w: malformed VRF proof hex: %v", types.ErrInvalidVRFProof, err)
	}

	// Decode VRF output for comparison
	vrfOutput, err := hex.DecodeString(vrfOutputHex)
	if err != nil {
		return fmt.Errorf("%w: malformed VRF output hex: %v", types.ErrInvalidVRFProof, err)
	}

	// Generate the VRF seed for this round
	claim, claimFound := k.GetClaim(ctx, round.ClaimId)
	if !claimFound {
		return fmt.Errorf("%w: claim %s for round %s", types.ErrClaimNotFound, round.ClaimId, round.Id)
	}

	prevBlockHash := getPreviousBlockHash(ctx)
	seed := vrfcrypto.GenerateVRFSeed(claim.Id, round.StartHeight, prevBlockHash)

	// Verify VRF proof
	computedOutput, valid := vrfcrypto.VerifyVRF(seed, validatorInfo.PublicKey, vrfProof)
	if !valid {
		return fmt.Errorf("%w: proof verification failed for validator %s", types.ErrInvalidVRFProof, validatorAddr)
	}

	// Verify the claimed output matches what the proof produces
	if hex.EncodeToString(computedOutput) != hex.EncodeToString(vrfOutput) {
		return fmt.Errorf("%w: VRF output mismatch", types.ErrInvalidVRFProof)
	}

	// Get effective stake for selection check
	effectiveStake := k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
	totalStake := k.stakingKeeper.GetTotalStake(ctx)

	params := k.GetParams(ctx)

	// Check if VRF output qualifies for selection
	selected, _ := vrfcrypto.IsValidatorSelected(
		vrfOutput,
		effectiveStake,
		totalStake,
		params.MinVerifiers,
	)

	if !selected {
		return fmt.Errorf("%w: validator %s not selected (stake=%d, total=%d)",
			types.ErrVRFSelectionFailed, validatorAddr, effectiveStake, totalStake)
	}

	return nil
}

// GenerateValidatorVRF generates VRF output and proof for a validator.
// Called locally by validator nodes to determine if they should participate.
func GenerateValidatorVRF(
	privateKey []byte,
	claimID string,
	startHeight uint64,
	prevBlockHash []byte,
) (vrfOutput, vrfProof []byte, err error) {
	seed := vrfcrypto.GenerateVRFSeed(claimID, startHeight, prevBlockHash)
	return vrfcrypto.GenerateVRF(seed, privateKey)
}

// ShouldValidatorParticipate checks if a validator's VRF output qualifies
// them for participation in a verification round.
// Called locally by validator nodes after generating their VRF.
func ShouldValidatorParticipate(
	vrfOutput []byte,
	effectiveStake uint64,
	totalStake uint64,
	targetValidators uint32,
) (shouldParticipate bool, priority uint64) {
	return vrfcrypto.IsValidatorSelected(vrfOutput, effectiveStake, totalStake, targetValidators)
}

// SlashMissedVerification slashes a validator who committed but failed to reveal.
func (k Keeper) SlashMissedVerification(
	ctx sdk.Context,
	validatorAddr string,
	slashAmount string,
) error {
	return k.stakingKeeper.SlashValidator(ctx, validatorAddr, slashAmount, "missed_verification_reveal")
}

// ---- Internal helpers ----

// getPreviousBlockHash extracts the previous block hash from context.
// In ABCI++, this is available from the block header.
func getPreviousBlockHash(ctx sdk.Context) []byte {
	header := ctx.BlockHeader()
	if len(header.LastBlockId.Hash) > 0 {
		return header.LastBlockId.Hash
	}
	// Genesis or first block — use empty hash
	return make([]byte, 32)
}
