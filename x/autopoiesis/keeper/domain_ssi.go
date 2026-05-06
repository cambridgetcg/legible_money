package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// domainSSIKey returns the KV store key for a domain's SSI value.
func domainSSIKey(domain string) []byte {
	return append(types.DomainSSIKeyPrefix, []byte(domain)...)
}

// SetDomainSSI stores a per-domain SSI value.
func (k Keeper) SetDomainSSI(ctx sdk.Context, domain string, score uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	val := types.DomainSSIValue{
		Domain: domain,
		Value:  score,
	}
	bz, err := json.Marshal(val)
	if err != nil {
		return
	}
	_ = kvStore.Set(domainSSIKey(domain), bz)
}

// GetDomainSSI returns the stored per-domain SSI, or 0 if not set.
func (k Keeper) GetDomainSSI(ctx sdk.Context, domain string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(domainSSIKey(domain))
	if err != nil || bz == nil {
		return 0
	}
	var val types.DomainSSIValue
	if err := json.Unmarshal(bz, &val); err != nil {
		return 0
	}
	return val.Value
}

// ComputeDomainSSI computes and stores the per-domain SSI for a given domain observation.
// Returns the computed score in basis points.
func (k Keeper) ComputeDomainSSI(ctx sdk.Context, dom types.DomainObservation) uint64 {
	score := types.ComputeDomainSSI(dom)
	k.SetDomainSSI(ctx, dom.Domain, score)
	return score
}

// ComputeAllDomainSSI computes per-domain SSI for all domains in the observation
// and stores the results. Returns all computed domain SSI values.
func (k Keeper) ComputeAllDomainSSI(ctx sdk.Context, obs types.EpochObservation) []types.DomainSSIValue {
	var results []types.DomainSSIValue
	for _, dom := range obs.Domains {
		score := k.ComputeDomainSSI(ctx, dom)
		results = append(results, types.DomainSSIValue{
			Domain: dom.Domain,
			Value:  score,
		})
	}
	return results
}

// CheckDomainCapture scans all domain SSI values against the global SSI and returns
// domains where domain SSI > DomainCaptureMultiplier * global SSI.
// A captured domain indicates potential validator collusion or systemic failure
// localized to that knowledge domain.
func (k Keeper) CheckDomainCapture(ctx sdk.Context, globalSSI uint64, domainSSIs []types.DomainSSIValue) []types.CapturedDomain {
	var captured []types.CapturedDomain

	// Threshold: domain SSI must exceed 2x global SSI.
	// When global SSI is very low (healthy system), use a minimum threshold
	// to avoid false positives from minor domain fluctuations.
	threshold := globalSSI * types.DomainCaptureMultiplier
	if threshold < types.SSILowModerate {
		threshold = types.SSILowModerate // minimum 0.2 to flag
	}

	for _, ds := range domainSSIs {
		if ds.Value > threshold {
			ratio := uint64(0)
			if globalSSI > 0 {
				ratio = (ds.Value * types.BasisPoints) / globalSSI
			} else {
				ratio = types.BasisPoints * 10 // effectively infinite ratio
			}
			captured = append(captured, types.CapturedDomain{
				Domain:    ds.Domain,
				DomainSSI: ds.Value,
				GlobalSSI: globalSSI,
				Ratio:     ratio,
			})
		}
	}

	return captured
}

// emitDomainCaptureEvents emits SDK events for each captured domain.
func (k Keeper) emitDomainCaptureEvents(ctx sdk.Context, epoch uint64, captured []types.CapturedDomain) {
	for _, c := range captured {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("domain_capture_detected",
				sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
				sdk.NewAttribute("domain", c.Domain),
				sdk.NewAttribute("domain_ssi", fmt.Sprintf("%d", c.DomainSSI)),
				sdk.NewAttribute("global_ssi", fmt.Sprintf("%d", c.GlobalSSI)),
				sdk.NewAttribute("ratio", fmt.Sprintf("%d", c.Ratio)),
			),
		)
		k.Logger(ctx).Warn("domain capture detected",
			"epoch", epoch,
			"domain", c.Domain,
			"domain_ssi", c.DomainSSI,
			"global_ssi", c.GlobalSSI,
			"ratio", c.Ratio,
		)
	}
}
