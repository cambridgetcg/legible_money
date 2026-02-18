package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// Keeper manages the evidence_mgmt module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	// Dependency keepers
	bankKeeper      types.BankKeeper
	knowledgeKeeper types.KnowledgeKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new evidence_mgmt module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration.
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// SetKnowledgeKeeper sets the knowledge keeper after construction (for init ordering).
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, evidence := range genState.EvidenceItems {
		if evidence != nil {
			k.SetEvidence(ctx, evidence)
		}
	}
	for _, audit := range genState.Audits {
		if audit != nil {
			k.SetAudit(ctx, audit)
		}
	}
	for _, consensus := range genState.ConsensusRecords {
		if consensus != nil {
			k.SetConsensus(ctx, consensus)
		}
	}
	for _, oracle := range genState.Oracles {
		if oracle != nil {
			k.SetOracle(ctx, oracle)
			if oracle.IsActive {
				k.SetActiveOracle(ctx, oracle.Id)
			}
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var evidences []*types.Evidence
	k.IterateEvidence(ctx, func(e *types.Evidence) bool {
		evidences = append(evidences, e)
		return false
	})

	var audits []*types.EvidenceAudit
	k.IterateAudits(ctx, func(a *types.EvidenceAudit) bool {
		audits = append(audits, a)
		return false
	})

	var consensusRecords []*types.AuditConsensus
	k.IterateConsensus(ctx, func(c *types.AuditConsensus) bool {
		consensusRecords = append(consensusRecords, c)
		return false
	})

	var oracles []*types.Oracle
	k.IterateOracles(ctx, func(o *types.Oracle) bool {
		oracles = append(oracles, o)
		return false
	})

	p := k.GetParams(ctx)
	return &types.GenesisState{
		Params:           p,
		EvidenceItems:    evidences,
		Audits:           audits,
		ConsensusRecords: consensusRecords,
		Oracles:          oracles,
	}
}
