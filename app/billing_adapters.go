package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	billingtypes "github.com/legible-money/legible/x/billing/types"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
)

// --- Billing Knowledge Adapter ---

// BillingKnowledgeAdapter wraps x/knowledge keeper to satisfy billing types.KnowledgeKeeper.
type BillingKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

func NewBillingKnowledgeAdapter(keeper knowledgekeeper.Keeper) *BillingKnowledgeAdapter {
	return &BillingKnowledgeAdapter{keeper: keeper}
}

var _ billingtypes.KnowledgeKeeper = (*BillingKnowledgeAdapter)(nil)

func (a *BillingKnowledgeAdapter) GetFactConfidence(ctx sdk.Context, factId string) (uint64, bool) {
	fact, found := a.keeper.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.Confidence, true
}

func (a *BillingKnowledgeAdapter) GetFactCitationCount(ctx sdk.Context, factId string) (uint64, bool) {
	fact, found := a.keeper.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.CitationCount, true
}

func (a *BillingKnowledgeAdapter) GetFactSubmitter(ctx sdk.Context, factId string) (string, bool) {
	fact, found := a.keeper.GetFact(ctx, factId)
	if !found {
		return "", false
	}
	return fact.Submitter, true
}

func (a *BillingKnowledgeAdapter) GetFactCreatedBlock(ctx sdk.Context, factId string) (uint64, bool) {
	fact, found := a.keeper.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.CreatedAtHeight, true
}

func (a *BillingKnowledgeAdapter) IncrementCitationCount(ctx sdk.Context, factId string) error {
	return a.keeper.IncrementCitationCount(ctx, factId)
}
