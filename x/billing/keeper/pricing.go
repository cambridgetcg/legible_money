package keeper

import (
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/types"
)

// bpsBasis is 1000000 (basis points denominator, 1M = 100%).
var bpsBasis = big.NewInt(1000000)

// CalculateQueryPrice computes the price for querying a set of facts.
//
// Per-fact formula: Marginal Information Value pricing.
// Price ∝ how much querying this fact improves agent output beyond what the
// LLM already knows from training data.
//
// Three factors:
//   - Confidence (bell curve): medium-high (threshold–850k) = peak value.
//     Below threshold = uncertainty surcharge. Above 850k = already known, discount.
//   - Novelty (inverted fundamentality): few citations = novel = premium.
//     Many citations = well-known = discount.
//   - Freshness (inverted): recent = NOT in training data = premium.
//     Old = probably in training data = no adjustment.
func (k Keeper) CalculateQueryPrice(ctx sdk.Context, factIds []string, currentBlock uint64) (*big.Int, []*types.FactPriceBreakdown) {
	params := k.GetParams(ctx)

	baseCost := new(big.Int)
	baseCost.SetString(params.BaseCostPerFact, 10)

	totalPrice := new(big.Int)
	var breakdown []*types.FactPriceBreakdown

	for _, factId := range factIds {
		factPrice := new(big.Int).Set(baseCost)
		confidencePremium := new(big.Int)
		fundamentalityBonus := new(big.Int)
		freshnessDiscount := new(big.Int)

		// 1. Confidence (bell curve): marginal information value peaks at
		//    medium-high confidence where facts are verified but not yet
		//    common knowledge (already in LLM training data).
		//    - Below threshold (500k): uncertainty surcharge
		//    - Threshold to 850k: peak value zone (base rate)
		//    - Above 850k: already known, discount
		confidence, found := k.knowledgeKeeper.GetFactConfidence(ctx, factId)
		if found {
			adj := new(big.Int).Mul(baseCost, big.NewInt(int64(params.ConfidencePremiumBps)))
			adj.Div(adj, bpsBasis)
			if confidence < params.ConfidenceThreshold {
				// Low confidence: surcharge (uncertain facts carry risk)
				confidencePremium.Set(adj)
				factPrice.Add(factPrice, confidencePremium)
			} else if confidence > 850000 {
				// High confidence: discount (already in LLM training data)
				confidencePremium.Neg(adj)
				factPrice.Sub(factPrice, adj)
				if factPrice.Sign() < 0 {
					factPrice.SetInt64(0)
				}
			}
			// Threshold to 850k: no adjustment (peak value zone)
		}

		// 2. Novelty (inverted fundamentality): highly cited facts are well-known —
		//    agents already have this in training data, low marginal value.
		//    Novel facts (few citations) are where the knowledge tree genuinely
		//    reduces hallucination.
		citations, citFound := k.knowledgeKeeper.GetFactCitationCount(ctx, factId)
		if citFound && citations > 0 {
			// Discount = baseCost * log2(citations+1) / 10 (capped at 50%)
			log2Val := uint64(math.Log2(float64(citations + 1)))
			if log2Val > 0 {
				fundamentalityBonus.Mul(baseCost, big.NewInt(int64(log2Val)))
				fundamentalityBonus.Div(fundamentalityBonus, big.NewInt(10))
				// Cap at 50% of base
				halfBase := new(big.Int).Div(baseCost, big.NewInt(2))
				if fundamentalityBonus.Cmp(halfBase) > 0 {
					fundamentalityBonus.Set(halfBase)
				}
				fundamentalityBonus.Neg(fundamentalityBonus) // negative = discount
				factPrice.Sub(factPrice, new(big.Int).Neg(fundamentalityBonus))
				if factPrice.Sign() < 0 {
					factPrice.SetInt64(0)
				}
			}
		}

		// 3. Freshness premium (inverted): recently verified facts are most
		//    likely to contain information NOT in LLM training data.
		createdBlock, blockFound := k.knowledgeKeeper.GetFactCreatedBlock(ctx, factId)
		if blockFound && currentBlock > createdBlock {
			age := currentBlock - createdBlock
			if age < params.FreshnessWindowBlocks {
				// Premium = baseCost * discountBps / 1000000 (added, not subtracted)
				freshnessDiscount.Mul(baseCost, big.NewInt(int64(params.FreshnessDiscountBps)))
				freshnessDiscount.Div(freshnessDiscount, bpsBasis)
				freshnessDiscount.Neg(freshnessDiscount) // negative = premium (added via -discount)
				factPrice.Add(factPrice, new(big.Int).Neg(freshnessDiscount))
			}
		}

		totalPrice.Add(totalPrice, factPrice)

		breakdown = append(breakdown, &types.FactPriceBreakdown{
			FactId:              factId,
			BaseCost:            baseCost.String(),
			ConfidencePremium:   confidencePremium.String(),
			FundamentalityBonus: fundamentalityBonus.String(),
			FreshnessDiscount:   freshnessDiscount.String(),
			TotalPrice:          factPrice.String(),
		})
	}

	return totalPrice, breakdown
}

// GenerateQuote creates an APIQuote with pricing breakdown.
func (k Keeper) GenerateQuote(ctx sdk.Context, provider string, caller string, factIds []string, currentBlock uint64) *types.APIQuote {
	params := k.GetParams(ctx)
	totalPrice, breakdown := k.CalculateQueryPrice(ctx, factIds, currentBlock)

	quoteId := k.GetNextQuoteId(ctx)

	quote := &types.APIQuote{
		QuoteId:    quoteId,
		Provider:   provider,
		Caller:     caller,
		FactIds:    factIds,
		Breakdown:  breakdown,
		TotalPrice: totalPrice.String(),
		ValidUntil: currentBlock + params.QuoteValidityBlocks,
		CreatedAt:  currentBlock,
		Executed:   false,
	}

	k.SetQuote(ctx, quote)
	return quote
}
