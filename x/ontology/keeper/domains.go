package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/types"
)

// IncrementClaimCount atomically increments a domain's claim count.
// Called by the knowledge module when a new claim is submitted.
func (k Keeper) IncrementClaimCount(ctx sdk.Context, domainName string) error {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}
	domain.ClaimCount++
	domain.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetDomain(ctx, domain)
	return nil
}

// IncrementFactCount atomically increments a domain's verified fact count.
// Called by the knowledge module when a claim is accepted.
func (k Keeper) IncrementFactCount(ctx sdk.Context, domainName string) error {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}
	domain.FactCount++
	domain.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetDomain(ctx, domain)
	return nil
}

// ValidateDomainForClaim checks whether a domain is valid for claim submission.
// Returns an error if the domain does not exist or is not active.
func (k Keeper) ValidateDomainForClaim(ctx sdk.Context, domainName string) error {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}
	if domain.Status != "active" {
		return fmt.Errorf("%w: domain %s has status %s", types.ErrDomainInactive, domainName, domain.Status)
	}
	return nil
}

// GetConfidenceCeiling returns the maximum confidence value allowed for a domain
// based on the stratum it belongs to. Also returns the decay rate.
func (k Keeper) GetConfidenceCeiling(ctx sdk.Context, domainName string) (maxConfidence uint64, decayRate uint64, err error) {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return 0, 0, fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}

	stratum, found := k.GetStratum(ctx, types.Stratum(domain.Stratum))
	if !found {
		return 0, 0, fmt.Errorf("%w: stratum %d for domain %s",
			types.ErrInvalidStratum, domain.Stratum, domainName)
	}

	return stratum.MaxConfidence, stratum.DecayRate, nil
}

// GetStratumForDomain returns the stratum properties for a given domain.
func (k Keeper) GetStratumForDomain(ctx sdk.Context, domainName string) (types.StratumProperties, error) {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return types.StratumProperties{}, fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}

	stratum, found := k.GetStratum(ctx, types.Stratum(domain.Stratum))
	if !found {
		return types.StratumProperties{}, fmt.Errorf("%w: stratum %d for domain %s",
			types.ErrInvalidStratum, domain.Stratum, domainName)
	}

	return stratum, nil
}

// IsGoedelConstrained returns whether claims in this domain are subject to
// Goedel's incompleteness theorems, which means the system acknowledges that
// some true statements may be unprovable.
func (k Keeper) IsGoedelConstrained(ctx sdk.Context, domainName string) (bool, error) {
	stratum, err := k.GetStratumForDomain(ctx, domainName)
	if err != nil {
		return false, err
	}
	return stratum.GoedelApplies, nil
}

// CountDomainsInStratum returns the number of domains belonging to a stratum.
func (k Keeper) CountDomainsInStratum(ctx sdk.Context, stratum types.Stratum) uint32 {
	domains := k.GetDomainsByStratum(ctx, stratum)
	return uint32(len(domains))
}

// DeprecateDomain marks a domain as deprecated. Deprecated domains cannot
// accept new claims but existing facts remain queryable.
func (k Keeper) DeprecateDomain(ctx sdk.Context, domainName string) error {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}
	domain.Status = "deprecated"
	domain.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetDomain(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_deprecated",
			sdk.NewAttribute("domain", domainName),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return nil
}

// ActivateDomain transitions a proposed domain to active status.
func (k Keeper) ActivateDomain(ctx sdk.Context, domainName string) error {
	domain, found := k.GetDomain(ctx, domainName)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDomainNotFound, domainName)
	}
	domain.Status = "active"
	domain.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetDomain(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_activated",
			sdk.NewAttribute("domain", domainName),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return nil
}
