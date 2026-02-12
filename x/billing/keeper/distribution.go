package keeper

import (
	"fmt"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/types"
)

// Treasury module account names for the 5-way research split.
const (
	ResearchTreasury   = "treasury_research"
	FoundationTreasury = "treasury_foundation"
	CommunityTreasury  = "treasury_community"
	DevelopersTreasury = "treasury_developers"
	ReserveTreasury    = "treasury_reserve"
	ProtocolTreasury   = "treasury_protocol"
)

// GetEffectiveResearchShareBps computes the progressive research share rate.
//
// Matches TS prototype's getEffectiveResearchShareBps():
//   - At block 0: returns ResearchMinBps (5% default)
//   - At ResearchRampBlocks: returns ResearchShareBps (22% default)
//   - After ramp: returns ResearchShareBps
//   - If ResearchRampBlocks is 0: returns ResearchShareBps immediately (backward compatible)
func GetEffectiveResearchShareBps(blockHeight int64, params types.Params) uint64 {
	rampBlocks := params.ResearchRampBlocks
	maxBps := params.ResearchShareBps
	minBps := params.ResearchMinBps

	// No ramp configured — use full rate immediately (backward compatible)
	if rampBlocks == 0 {
		return maxBps
	}

	// Past ramp period — full rate
	if uint64(blockHeight) >= rampBlocks {
		return maxBps
	}

	// Linear interpolation: minBps + (maxBps - minBps) * blockHeight / rampBlocks
	rangeBps := maxBps - minBps
	return minBps + (rangeBps * uint64(blockHeight) / rampBlocks)
}

// CalculateDistribution computes the payment distribution for a billing payment.
//
// Distribution flow (all bigint math):
//  1. Research share FIRST: totalPayment * effectiveResearchBps / 1000000 (ramps from 5% → 22%)
//  2. Research 5-way split: 77% research, 7% foundation, 7% community, 7% developers, 2% reserve
//  3. Remaining: 60% provider, 30% knowledge pool, 10% protocol
//  4. Protocol 10% split: 50% burn, 50% treasury
func (k Keeper) CalculateDistribution(ctx sdk.Context, totalPayment *big.Int, factIds []string) types.PaymentDistribution {
	params := k.GetParams(ctx)

	// Step 1: Research share with progressive ramp
	effectiveBps := GetEffectiveResearchShareBps(ctx.BlockHeight(), params)
	researchShare := new(big.Int).Mul(totalPayment, big.NewInt(int64(effectiveBps)))
	researchShare.Div(researchShare, bpsBasis)

	// Step 2: 5-way research split (77/7/7/7/2 = 100 parts)
	researchFund := new(big.Int).Mul(researchShare, big.NewInt(77))
	researchFund.Div(researchFund, big.NewInt(100))

	foundationFund := new(big.Int).Mul(researchShare, big.NewInt(7))
	foundationFund.Div(foundationFund, big.NewInt(100))

	communityFund := new(big.Int).Mul(researchShare, big.NewInt(7))
	communityFund.Div(communityFund, big.NewInt(100))

	developersFund := new(big.Int).Mul(researchShare, big.NewInt(7))
	developersFund.Div(developersFund, big.NewInt(100))

	// Reserve gets the remainder to avoid rounding loss
	reserveFund := new(big.Int).Set(researchShare)
	reserveFund.Sub(reserveFund, researchFund)
	reserveFund.Sub(reserveFund, foundationFund)
	reserveFund.Sub(reserveFund, communityFund)
	reserveFund.Sub(reserveFund, developersFund)

	// Step 3: Remaining amount after research share
	remaining := new(big.Int).Sub(totalPayment, researchShare)

	// Provider: 60% of remaining
	providerShare := new(big.Int).Mul(remaining, big.NewInt(600000))
	providerShare.Div(providerShare, bpsBasis)

	// Knowledge pool: 30% of remaining
	knowledgePoolTotal := new(big.Int).Mul(remaining, big.NewInt(300000))
	knowledgePoolTotal.Div(knowledgePoolTotal, bpsBasis)

	// Protocol: 10% of remaining (remainder to avoid rounding loss)
	protocolTotal := new(big.Int).Set(remaining)
	protocolTotal.Sub(protocolTotal, providerShare)
	protocolTotal.Sub(protocolTotal, knowledgePoolTotal)

	// Protocol split: 50% burn, 50% treasury
	protocolBurn := new(big.Int).Div(protocolTotal, big.NewInt(2))
	protocolTreasury := new(big.Int).Sub(protocolTotal, protocolBurn)

	// Step 4: Knowledge pool weighted by confidence * log2(citations+1)
	knowledgePool := k.calculateKnowledgePoolDistribution(ctx, knowledgePoolTotal, factIds)

	return types.PaymentDistribution{
		TotalPayment: totalPayment.String(),
		ResearchShare: researchShare.String(),
		ResearchSplit: types.ResearchFundSplit{
			Research:   researchFund.String(),
			Foundation: foundationFund.String(),
			Community:  communityFund.String(),
			Developers: developersFund.String(),
			Reserve:    reserveFund.String(),
		},
		ProviderShare:    providerShare.String(),
		KnowledgePool:    knowledgePool,
		ProtocolBurn:     protocolBurn.String(),
		ProtocolTreasury: protocolTreasury.String(),
	}
}

// calculateKnowledgePoolDistribution distributes the knowledge pool share
// weighted by confidence * log2(citations+1) for each fact.
func (k Keeper) calculateKnowledgePoolDistribution(ctx sdk.Context, poolTotal *big.Int, factIds []string) []types.KnowledgePoolEntry {
	type factWeight struct {
		factId    string
		submitter string
		weight    *big.Int
	}

	var entries []factWeight
	totalWeight := new(big.Int)

	for _, factId := range factIds {
		submitter, found := k.knowledgeKeeper.GetFactSubmitter(ctx, factId)
		if !found {
			continue
		}

		confidence, _ := k.knowledgeKeeper.GetFactConfidence(ctx, factId)
		citations, _ := k.knowledgeKeeper.GetFactCitationCount(ctx, factId)

		// weight = confidence * floor(log2(citations+1) + 1)
		// confidence is in bps (0-10000), log factor ensures minimum weight of 1
		logFactor := uint64(math.Log2(float64(citations+1))) + 1
		weight := new(big.Int).Mul(big.NewInt(int64(confidence)), big.NewInt(int64(logFactor)))

		if weight.Sign() <= 0 {
			weight.SetInt64(1) // minimum weight
		}

		entries = append(entries, factWeight{
			factId:    factId,
			submitter: submitter,
			weight:    weight,
		})
		totalWeight.Add(totalWeight, weight)
	}

	var pool []types.KnowledgePoolEntry

	if totalWeight.Sign() == 0 || len(entries) == 0 {
		// No eligible facts; entire knowledge pool goes to protocol treasury
		pool = append(pool, types.KnowledgePoolEntry{
			FactId:    "",
			Submitter: ProtocolTreasury,
			Amount:    poolTotal.String(),
			Weight:    "0",
		})
		return pool
	}

	distributed := new(big.Int)
	for i, entry := range entries {
		var amount *big.Int
		if i == len(entries)-1 {
			// Last entry gets remainder to avoid rounding loss
			amount = new(big.Int).Sub(poolTotal, distributed)
		} else {
			amount = new(big.Int).Mul(poolTotal, entry.weight)
			amount.Div(amount, totalWeight)
		}
		distributed.Add(distributed, amount)

		pool = append(pool, types.KnowledgePoolEntry{
			FactId:    entry.factId,
			Submitter: entry.submitter,
			Amount:    amount.String(),
			Weight:    entry.weight.String(),
		})
	}

	return pool
}

// ExecuteDistribution performs the atomic transfers for a payment distribution.
func (k Keeper) ExecuteDistribution(ctx sdk.Context, callerAddr sdk.AccAddress, providerAddr sdk.AccAddress, distribution types.PaymentDistribution) error {
	// 1. Send research splits to treasury modules
	researchSplit := distribution.ResearchSplit

	if err := k.sendToModule(ctx, callerAddr, ResearchTreasury, researchSplit.Research); err != nil {
		return fmt.Errorf("failed to send research fund: %w", err)
	}
	if err := k.sendToModule(ctx, callerAddr, FoundationTreasury, researchSplit.Foundation); err != nil {
		return fmt.Errorf("failed to send foundation fund: %w", err)
	}
	if err := k.sendToModule(ctx, callerAddr, CommunityTreasury, researchSplit.Community); err != nil {
		return fmt.Errorf("failed to send community fund: %w", err)
	}
	if err := k.sendToModule(ctx, callerAddr, DevelopersTreasury, researchSplit.Developers); err != nil {
		return fmt.Errorf("failed to send developers fund: %w", err)
	}
	if err := k.sendToModule(ctx, callerAddr, ReserveTreasury, researchSplit.Reserve); err != nil {
		return fmt.Errorf("failed to send reserve fund: %w", err)
	}

	// 2. Send provider share directly
	providerAmt := new(big.Int)
	providerAmt.SetString(distribution.ProviderShare, 10)
	if providerAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(providerAmt)))
		if err := k.bankKeeper.SendCoins(ctx, callerAddr, providerAddr, coins); err != nil {
			return fmt.Errorf("failed to send provider share: %w", err)
		}
	}

	// 3. Send knowledge pool payments to submitters
	for _, entry := range distribution.KnowledgePool {
		amt := new(big.Int)
		amt.SetString(entry.Amount, 10)
		if amt.Sign() <= 0 {
			continue
		}
		if entry.Submitter == ProtocolTreasury {
			// Fallback: send to protocol treasury module
			coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amt)))
			if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, ProtocolTreasury, coins); err != nil {
				return fmt.Errorf("failed to send knowledge pool fallback: %w", err)
			}
			continue
		}
		submitterAddr, err := sdk.AccAddressFromBech32(entry.Submitter)
		if err != nil {
			continue // skip invalid addresses
		}
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amt)))
		if err := k.bankKeeper.SendCoins(ctx, callerAddr, submitterAddr, coins); err != nil {
			return fmt.Errorf("failed to send knowledge pool to %s: %w", entry.Submitter, err)
		}
	}

	// 4. Protocol burn
	burnAmt := new(big.Int)
	burnAmt.SetString(distribution.ProtocolBurn, 10)
	if burnAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(burnAmt)))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, types.ModuleName, coins); err != nil {
			return fmt.Errorf("failed to send to module for burn: %w", err)
		}
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
			return fmt.Errorf("failed to burn protocol share: %w", err)
		}
	}

	// 5. Protocol treasury
	treasuryAmt := new(big.Int)
	treasuryAmt.SetString(distribution.ProtocolTreasury, 10)
	if treasuryAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(treasuryAmt)))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, ProtocolTreasury, coins); err != nil {
			return fmt.Errorf("failed to send protocol treasury share: %w", err)
		}
	}

	return nil
}

// sendToModule sends coins from an account to a module account.
func (k Keeper) sendToModule(ctx sdk.Context, fromAddr sdk.AccAddress, moduleName string, amountStr string) error {
	amt := new(big.Int)
	amt.SetString(amountStr, 10)
	if amt.Sign() <= 0 {
		return nil
	}
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amt)))
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddr, moduleName, coins)
}
