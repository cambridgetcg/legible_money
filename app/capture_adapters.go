package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	capturedefensetypes "github.com/legible-money/legible/x/capture_defense/types"
	ontologykeeper "github.com/legible-money/legible/x/ontology/keeper"
)

// CaptureDefenseOntologyAdapter wraps the ontology keeper to satisfy
// the capture_defense module's OntologyKeeper interface.
type CaptureDefenseOntologyAdapter struct {
	keeper ontologykeeper.Keeper
}

// NewCaptureDefenseOntologyAdapter creates a new adapter.
func NewCaptureDefenseOntologyAdapter(keeper ontologykeeper.Keeper) *CaptureDefenseOntologyAdapter {
	return &CaptureDefenseOntologyAdapter{keeper: keeper}
}

var _ capturedefensetypes.OntologyKeeper = (*CaptureDefenseOntologyAdapter)(nil)

// GetDomainDepth returns the depth of a domain (1 = leaf, higher = deeper).
// Since the ontology module doesn't track depth, default to 1 (leaf domain).
func (a *CaptureDefenseOntologyAdapter) GetDomainDepth(ctx sdk.Context, domain string) (uint32, bool) {
	_, found := a.keeper.GetDomain(ctx, domain)
	if !found {
		return 0, false
	}
	// Default depth 1 — all domains treated as leaf until hierarchy is wired
	return 1, true
}

// GetDomainStratum returns the stratum for a domain.
func (a *CaptureDefenseOntologyAdapter) GetDomainStratum(ctx sdk.Context, domain string) (uint32, bool) {
	d, found := a.keeper.GetDomain(ctx, domain)
	if !found {
		return 0, false
	}
	return uint32(d.Stratum), true
}

// DomainExists returns whether a domain exists.
func (a *CaptureDefenseOntologyAdapter) DomainExists(ctx sdk.Context, domain string) bool {
	_, found := a.keeper.GetDomain(ctx, domain)
	return found
}

// GetParentDomain returns the parent domain. Currently not tracked in ontology.
func (a *CaptureDefenseOntologyAdapter) GetParentDomain(ctx sdk.Context, domain string) (string, bool) {
	// Domain hierarchy not yet tracked — return not found
	return "", false
}

// GetStratumName returns the name for a stratum.
func (a *CaptureDefenseOntologyAdapter) GetStratumName(ctx sdk.Context, stratum uint32) string {
	// Map stratum IDs to names (matching ontology stratum constants)
	names := map[uint32]string{
		0: "axiomatic",
		1: "formal",
		2: "protocol",
		3: "computational",
		4: "empirical",
		5: "historical",
		6: "testimonial",
	}
	if name, ok := names[stratum]; ok {
		return name
	}
	return "unknown"
}
