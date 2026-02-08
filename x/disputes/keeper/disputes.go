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
func (k Keeper) OpenDispute(ctx sdk.Context, factId, challenger, reason, bond string) (types.Dispute, error) {
	params := k.GetParams(ctx)

	// Verify fact exists and is disputable
	fact, found := k.knowledgeKeeper.GetFact(ctx, factId)
	if !found {
		return types.Dispute{}, fmt.Errorf("%w: %s", types.ErrFactNotFound, factId)
	}
	if fact.Status != "verified" && fact.Status != "contested" {
		return types.Dispute{}, fmt.Errorf("%w: fact status is %s", types.ErrFactNotDisputable, fact.Status)
	}

	// Check active dispute limit for this fact
	activeCount := k.GetActiveDisputeCountForFact(ctx, factId)
	if activeCount >= params.MaxActiveDisputes {
		return types.Dispute{}, fmt.Errorf("%w: %d active disputes for fact %s",
			types.ErrMaxActiveDisputes, activeCount, factId)
	}

	// Get tier 1 config
	tierConfig, err := k.GetTierConfig(params, types.TierBasic)
	if err != nil {
		return types.Dispute{}, err
	}

	// Validate bond amount
	bondAmount := new(big.Int)
	if _, ok := bondAmount.SetString(bond, 10); !ok || bondAmount.Sign() <= 0 {
		return types.Dispute{}, fmt.Errorf("%w: invalid amount", types.ErrInvalidBond)
	}

	minBond := new(big.Int)
	if _, ok := minBond.SetString(tierConfig.MinBond, 10); !ok {
		return types.Dispute{}, fmt.Errorf("%w: invalid min bond config", types.ErrTierConfigNotFound)
	}
	if bondAmount.Cmp(minBond) < 0 {
		return types.Dispute{}, fmt.Errorf("%w: need at least %s, got %s",
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
		return types.Dispute{}, err
	}

	dispute := types.Dispute{
		Id:               disputeId,
		FactId:           factId,
		ClaimId:          "", // Will be set if claim info is available
		Challenger:       challenger,
		Defender:         fact.Submitter,
		Reason:           reason,
		Bond:             bond,
		Tier:             types.TierBasic,
		Status:           types.DisputeStatusEvidence,
		Outcome:          "",
		CreatedAt:        height,
		EvidenceDeadline: evidenceDeadline,
		VotingDeadline:   votingDeadline,
		Arbiters:         arbiters,
		EvidenceCount:    0,
	}

	k.SetDispute(ctx, dispute)

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
func (k Keeper) TransitionToVoting(ctx sdk.Context, dispute types.Dispute) error {
	if dispute.Status != types.DisputeStatusEvidence {
		return fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInEvidencePhase, dispute.Status)
	}

	dispute.Status = types.DisputeStatusVoting
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

// SelectArbiters selects eligible arbiters from the active validator set.
// Excludes the challenger and defender from the arbiter pool.
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

	// Sort by stake descending for deterministic selection.
	// In production, VRF-based random selection would be used.
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].Stake != eligible[j].Stake {
			return eligible[i].Stake > eligible[j].Stake
		}
		return eligible[i].Address < eligible[j].Address
	})

	arbiters := make([]string, 0, count)
	for i := uint32(0); i < count && i < uint32(len(eligible)); i++ {
		arbiters = append(arbiters, eligible[i].Address)
	}

	return arbiters, nil
}

// GetTierConfig returns the config for the specified tier from params.
func (k Keeper) GetTierConfig(params types.Params, tier types.EscalationTier) (types.TierConfig, error) {
	for _, tc := range params.TierConfigs {
		if tc.Tier == tier {
			return tc, nil
		}
	}
	return types.TierConfig{}, fmt.Errorf("%w: tier %d", types.ErrTierConfigNotFound, tier)
}

// ProcessExpiredDisputes checks for disputes that have passed their deadlines
// and transitions or expires them accordingly.
func (k Keeper) ProcessExpiredDisputes(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())
	activeDisputes := k.GetActiveDisputes(ctx)

	for _, dispute := range activeDisputes {
		switch dispute.Status {
		case types.DisputeStatusEvidence:
			// Evidence deadline passed — transition to voting
			if currentHeight > dispute.EvidenceDeadline {
				if err := k.TransitionToVoting(ctx, dispute); err != nil {
					k.Logger(ctx).Error("failed to transition dispute to voting",
						"dispute_id", dispute.Id, "error", err)
				}
			}

		case types.DisputeStatusVoting:
			// Voting deadline passed — tally and resolve
			if currentHeight > dispute.VotingDeadline {
				outcome, err := k.TallyVotes(ctx, dispute)
				if err != nil {
					k.Logger(ctx).Error("failed to tally votes",
						"dispute_id", dispute.Id, "error", err)
					// Mark as expired if tally fails
					dispute.Status = types.DisputeStatusExpired
					k.SetDispute(ctx, dispute)
					continue
				}
				if err := k.ResolveDispute(ctx, dispute, outcome); err != nil {
					k.Logger(ctx).Error("failed to resolve dispute",
						"dispute_id", dispute.Id, "error", err)
				}
			}

		case types.DisputeStatusOpen:
			// Open disputes that haven't progressed — expire after evidence deadline
			if currentHeight > dispute.EvidenceDeadline {
				dispute.Status = types.DisputeStatusExpired
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
