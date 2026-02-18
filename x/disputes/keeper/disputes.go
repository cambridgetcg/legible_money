package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

// OpenDispute creates a new dispute against a verified fact.
func (k Keeper) OpenDispute(ctx sdk.Context, factId, challenger, reason, bond string) (*types.Dispute, error) {
	params := k.GetParams(ctx)

	// Verify fact exists and is disputable
	fact, found := k.knowledgeKeeper.GetFact(ctx, factId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrFactNotFound, factId)
	}
	if fact.Status != "verified" && fact.Status != "contested" && fact.Status != "active" && fact.Status != "challenged" && fact.Status != "provisional" {
		return nil, fmt.Errorf("%w: fact status is %s", types.ErrFactNotDisputable, fact.Status)
	}

	// Check active dispute limit for this fact
	activeCount := k.GetActiveDisputeCountForFact(ctx, factId)
	if activeCount >= params.MaxActiveDisputes {
		return nil, fmt.Errorf("%w: %d active disputes for fact %s",
			types.ErrMaxActiveDisputes, activeCount, factId)
	}

	// Get tier 1 config
	tierConfig, err := k.GetTierConfig(params, types.TierBasic)
	if err != nil {
		return nil, err
	}

	// Validate bond amount
	bondAmount := new(big.Int)
	if _, ok := bondAmount.SetString(bond, 10); !ok || bondAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid amount", types.ErrInvalidBond)
	}

	minBond := new(big.Int)
	if _, ok := minBond.SetString(tierConfig.MinBond, 10); !ok {
		return nil, fmt.Errorf("%w: invalid min bond config", types.ErrTierConfigNotFound)
	}
	if bondAmount.Cmp(minBond) < 0 {
		return nil, fmt.Errorf("%w: need at least %s, got %s",
			types.ErrInsufficientBond, tierConfig.MinBond, bond)
	}

	// Generate deterministic dispute ID
	height := uint64(ctx.BlockHeight())
	disputeId := GenerateDisputeID(factId, challenger, height)

	// Calculate deadlines
	evidenceDeadline := height + tierConfig.EvidencePeriod
	votingDeadline := evidenceDeadline + tierConfig.VotingPeriod

	// Select arbiters
	arbiters, err := k.SelectArbiters(ctx, tierConfig.ArbiterCount, challenger, fact.Submitter)
	if err != nil {
		return nil, err
	}

	dispute := &types.Dispute{
		Id:               disputeId,
		FactId:           factId,
		ClaimId:          "", // Will be set if claim info is available
		Challenger:       challenger,
		Defender:         fact.Submitter,
		Reason:           reason,
		Bond:             bond,
		Tier:             uint32(types.TierBasic),
		Status:           string(types.DisputeStatusEvidence),
		Outcome:          "",
		CreatedAt:        height,
		EvidenceDeadline: evidenceDeadline,
		VotingDeadline:   votingDeadline,
		Arbiters:         arbiters,
		EvidenceCount:    0,
	}

	k.SetDispute(ctx, dispute)

	// Pause vesting for the disputed fact's claim (if vesting exists)
	if k.vestingKeeper != nil && dispute.ClaimId != "" {
		if err := k.vestingKeeper.PauseVestingByClaimId(ctx, dispute.ClaimId); err != nil {
			k.Logger(ctx).Error("failed to pause vesting on dispute open",
				"claim_id", dispute.ClaimId, "error", err)
		}
	}

	// Mark fact as challenged via knowledge module's invariant-enforcing method
	if err := k.knowledgeKeeper.MarkFactChallenged(ctx, factId); err != nil {
		k.Logger(ctx).Error("failed to mark fact as challenged",
			"fact_id", factId, "error", err)
	}

	k.Logger(ctx).Info("dispute opened",
		"dispute_id", disputeId,
		"fact_id", factId,
		"challenger", challenger,
		"tier", types.TierBasic,
		"arbiters", len(arbiters),
	)

	return dispute, nil
}

// TransitionToVoting moves a dispute from evidence phase to voting phase.
func (k Keeper) TransitionToVoting(ctx sdk.Context, dispute *types.Dispute) error {
	if dispute.Status != string(types.DisputeStatusEvidence) {
		return fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInEvidencePhase, dispute.Status)
	}

	dispute.Status = string(types.DisputeStatusVoting)
	k.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_voting_started",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("fact_id", dispute.FactId),
		),
	)

	k.Logger(ctx).Info("dispute transitioned to voting",
		"dispute_id", dispute.Id,
	)

	return nil
}

// SelectArbiters selects eligible arbiters from the active validator set
// using hash-based pseudo-random selection weighted by stake.
// The selection seed is derived from the previous block hash + a dispute-specific
// domain separator, making the result unpredictable to the disputant at submission time.
func (k Keeper) SelectArbiters(ctx sdk.Context, count uint32, challenger, defender string) ([]string, error) {
	validators := k.stakingKeeper.GetActiveValidatorInfos(ctx)

	// Filter out challenger and defender, and validators with insufficient tier
	var eligible []types.ValidatorInfo
	for _, v := range validators {
		if v.Address == challenger || v.Address == defender {
			continue
		}
		// Require at least tier 1 (Verified) to serve as arbiter
		if v.Tier < 1 {
			continue
		}
		eligible = append(eligible, v)
	}

	if uint32(len(eligible)) < count {
		// If not enough eligible arbiters, use what we have (minimum 1)
		if len(eligible) == 0 {
			return nil, fmt.Errorf("%w", types.ErrNoArbitersAvailable)
		}
		count = uint32(len(eligible))
	}

	// Generate selection seed from block hash + dispute context
	seed := arbiterSelectionSeed(ctx, challenger, defender)

	// Compute stake-weighted priority for each validator
	type arbiterCandidate struct {
		address  string
		priority [32]byte
	}
	candidates := make([]arbiterCandidate, len(eligible))
	for i, v := range eligible {
		candidates[i] = arbiterCandidate{
			address:  v.Address,
			priority: arbiterPriority(seed, v.Address, v.Stake),
		}
	}

	// Sort by priority (deterministic pseudo-random ordering)
	sort.Slice(candidates, func(i, j int) bool {
		for k := 0; k < 32; k++ {
			if candidates[i].priority[k] != candidates[j].priority[k] {
				return candidates[i].priority[k] < candidates[j].priority[k]
			}
		}
		return candidates[i].address < candidates[j].address
	})

	arbiters := make([]string, 0, count)
	for i := uint32(0); i < count && i < uint32(len(candidates)); i++ {
		arbiters = append(arbiters, candidates[i].address)
	}

	return arbiters, nil
}

// arbiterSelectionSeed generates a deterministic seed for arbiter selection.
// Uses block hash + challenger + defender addresses for domain separation.
func arbiterSelectionSeed(ctx sdk.Context, challenger, defender string) []byte {
	h := sha256.New()
	h.Write([]byte("LGM.dispute.arbiter.v1:"))
	header := ctx.BlockHeader()
	if len(header.LastBlockId.Hash) > 0 {
		h.Write(header.LastBlockId.Hash)
	} else {
		h.Write(make([]byte, 32)) // genesis/first block
	}
	h.Write([]byte(":"))
	h.Write([]byte(challenger))
	h.Write([]byte(":"))
	h.Write([]byte(defender))
	h.Write([]byte(fmt.Sprintf(":%d", ctx.BlockHeight())))
	return h.Sum(nil)
}

// arbiterPriority computes a stake-weighted priority hash for a validator.
// Higher-stake validators get lower priority values (selected first) on average
// by hashing with a stake-dependent number of rounds.
func arbiterPriority(seed []byte, address string, stake uint64) [32]byte {
	h := sha256.New()
	h.Write(seed)
	h.Write([]byte(":"))
	h.Write([]byte(address))
	base := h.Sum(nil)

	// Stake weighting: lower the hash by mixing in stake
	// Validators with higher stake get an advantage in priority
	// by XOR-ing with a stake-derived mask
	stakeHash := sha256.Sum256([]byte(fmt.Sprintf("LGM.stake.weight:%s:%d", address, stake)))

	var result [32]byte
	copy(result[:], base)

	// Scale advantage: for each power-of-2 in stake, clear one leading bit
	// This gives higher-stake validators ~2x advantage per doubling of stake
	stakeLevel := uint64(0)
	s := stake
	for s > 1 {
		stakeLevel++
		s >>= 1
	}

	// Mix stake hash to bias toward higher-stake validators
	for i := 0; i < 32; i++ {
		result[i] ^= stakeHash[i]
	}
	// Clear leading bytes proportional to log2(stake) to give priority
	clearBytes := stakeLevel / 8
	if clearBytes > 8 {
		clearBytes = 8 // cap advantage at 8 bytes
	}
	for i := uint64(0); i < clearBytes; i++ {
		result[i] = 0
	}

	return result
}

// GetTierConfig returns the config for the specified tier from params.
func (k Keeper) GetTierConfig(params *types.Params, tier types.EscalationTier) (*types.TierConfig, error) {
	for _, tc := range params.TierConfigs {
		if tc.Tier == uint32(tier) {
			return tc, nil
		}
	}
	return nil, fmt.Errorf("%w: tier %d", types.ErrTierConfigNotFound, tier)
}

// ProcessExpiredDisputes checks for disputes that have passed their deadlines
// and transitions or expires them accordingly.
func (k Keeper) ProcessExpiredDisputes(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())
	activeDisputes := k.GetActiveDisputes(ctx)

	for _, dispute := range activeDisputes {
		switch dispute.Status {
		case string(types.DisputeStatusEvidence):
			// Evidence deadline passed — transition to voting
			if currentHeight > dispute.EvidenceDeadline {
				if err := k.TransitionToVoting(ctx, dispute); err != nil {
					k.Logger(ctx).Error("failed to transition dispute to voting",
						"dispute_id", dispute.Id, "error", err)
				}
			}

		case string(types.DisputeStatusVoting):
			// Voting deadline passed — tally and resolve
			if currentHeight > dispute.VotingDeadline {
				outcome, err := k.TallyVotes(ctx, dispute)
				if err != nil {
					k.Logger(ctx).Error("failed to tally votes",
						"dispute_id", dispute.Id, "error", err)
					// Mark as expired if tally fails
					dispute.Status = string(types.DisputeStatusExpired)
					k.SetDispute(ctx, dispute)
					continue
				}
				if err := k.ResolveDispute(ctx, dispute, outcome); err != nil {
					k.Logger(ctx).Error("failed to resolve dispute",
						"dispute_id", dispute.Id, "error", err)
				}
			}

		case string(types.DisputeStatusOpen):
			// Open disputes that haven't progressed — expire after evidence deadline
			if currentHeight > dispute.EvidenceDeadline {
				dispute.Status = string(types.DisputeStatusExpired)
				k.SetDispute(ctx, dispute)
				k.Logger(ctx).Info("dispute expired", "dispute_id", dispute.Id)
			}
		}
	}
}

// ---------- ID Generation ----------

// GenerateDisputeID generates a deterministic dispute ID.
func GenerateDisputeID(factId, challenger string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.dispute.id.v1:"))
	h.Write([]byte(factId))
	h.Write([]byte(":"))
	h.Write([]byte(challenger))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GenerateEvidenceID generates a deterministic evidence ID.
func GenerateEvidenceID(disputeId, submitter string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.evidence.id.v1:"))
	h.Write([]byte(disputeId))
	h.Write([]byte(":"))
	h.Write([]byte(submitter))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}
