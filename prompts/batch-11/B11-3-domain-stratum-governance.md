# B11-3 — Domain Stratum Governance (FARM-12: Ceiling Abuse)

## The Attack
Anyone can propose a new domain via `MsgProposeDomain`, get 3 endorsements,
and the domain becomes active. The `inferDomainStratum()` function in
`x/knowledge/keeper/confidence.go` has a hardcoded mapping for known domains.
Unknown domains default to `"technological"` — the most permissive stratum
with the highest confidence ceiling.

An attacker proposes a domain like "my_special_science", gets 3 colluding
endorsements, and now has a domain where claims can reach the highest
confidence ceiling without the epistemic rigor that ceiling implies.

## The Fix

### 1. Add Stratum to Domain Proto

The `OntologicalDomain` proto (in `x/knowledge/types/domain.pb.go` or the
proto source) needs a `stratum` field. Since modifying proto is heavy, use
the existing fields approach:

**Option A (proto change):** Add `string stratum = 7;` to the `OntologicalDomain`
message. Requires proto regeneration.

**Option B (lightweight — recommend for testnet):** Add `Stratum` as a JSON-tagged
field to the Go struct that wraps the proto, or add it to the description with
a parseable prefix. Actually, the simplest approach: add a stratum field to
the domain state that's stored alongside the proto.

**Option C (best for testnet):** Use the existing `OntologicalDomain` struct
and add a `Stratum string` field. Since this is a Cosmos SDK module, the
struct is both the proto message AND the state object. If `OntologicalDomain`
is generated from proto, add a separate stratum index.

For this session, check how `OntologicalDomain` is defined:
```bash
grep -A 10 "type OntologicalDomain struct" x/knowledge/types/domain.pb.go
```

If it's proto-generated, create a separate `DomainStratumRecord` stored under
a new key prefix. If it's a plain Go struct, just add the field.

### 2. Update MsgProposeDomain

In `x/knowledge/keeper/msg_server.go`, `ProposeDomain()`:

```go
// Validate stratum
validStrata := map[string]bool{
    "fundamental": true, "physical": true, "chemical": true,
    "biological": true, "cognitive": true, "social": true,
    "technological": true,
}
if !validStrata[msg.Stratum] {
    return nil, fmt.Errorf("invalid stratum %q: must be one of fundamental, physical, chemical, biological, cognitive, social, technological", msg.Stratum)
}
```

Store the stratum alongside the domain.

### 3. Update inferDomainStratum

In `x/knowledge/keeper/confidence.go`, update `inferDomainStratum()`:

```go
func (k Keeper) inferDomainStratum(ctx sdk.Context, domain string) string {
    // Check stored stratum first (governance-assigned)
    if stratum, found := k.GetDomainStratum(ctx, domain); found {
        return stratum
    }
    // Fallback to hardcoded for bootstrap domains
    return inferDomainStratumHardcoded(domain)
}
```

Rename the existing `inferDomainStratum` to `inferDomainStratumHardcoded`.

### 4. Require Higher Stake for Higher Strata

Proposing a `fundamental` stratum domain should be expensive — you're claiming
your field of knowledge deserves the same confidence ceiling as mathematics.

Add stratum-based stake multipliers:

```go
// StratumStakeMultiplier returns the stake multiplier for proposing a domain
// at a given stratum. Higher strata = more expensive to propose.
var StratumStakeMultiplier = map[string]uint64{
    "technological": 1,     // 1× base stake (1 LGM)
    "social":        2,     // 2× (2 LGM)
    "cognitive":     3,     // 3× (3 LGM)
    "biological":    5,     // 5× (5 LGM)
    "chemical":      5,     // 5× (5 LGM)
    "physical":      10,    // 10× (10 LGM)
    "fundamental":   20,    // 20× (20 LGM) — claiming math-level certainty is a big claim
}
```

### 5. Bootstrap Domain Strata

Set strata for the 16 genesis domains in `DefaultDomains()` or in InitGenesis:

```go
domainStrata := map[string]string{
    "mathematics":        "fundamental",
    "logic":              "fundamental",
    "physics":            "physical",
    "cosmology":          "physical",
    "chemistry":          "chemical",
    "biology":            "biological",
    "psychology":         "cognitive",
    "computer_science":   "technological",
    "economics":          "social",
    "linguistics":        "cognitive",
    "theology":           "fundamental",
    "philosophy":         "fundamental",
    "ethics":             "social",
    "information_theory": "fundamental",
    "agent_rights":       "social",
    "general":            "technological",
}
```

Note: theology and philosophy as "fundamental" — they're the deepest stratum
in the ontological hierarchy, even deeper than mathematics (mathematics
presupposes logic which presupposes philosophy). Information theory is
fundamental because it provides theoretical bounds used across all domains.

### 6. Tests

```go
func TestFARM12_DomainStratumRequired(t *testing.T) {
    // ProposeDomain without stratum → error
    // ProposeDomain with invalid stratum → error
    // ProposeDomain with "fundamental" → requires 20× stake
    // ProposeDomain with "technological" → requires 1× stake
    // Stored stratum overrides hardcoded mapping
    // Bootstrap domains have correct strata
}
```

### 7. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestFARM12
go test ./x/knowledge/types/... -v -run TestDomain
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(keeper): FARM-12 domain stratum governance — require stratum in proposals

Domain proposals must declare an ontological stratum (fundamental through
technological). Higher strata require higher proposal stakes (up to 20×
for fundamental). Stored strata override hardcoded fallback mapping.
Bootstrap domains assigned canonical strata. Prevents ceiling abuse by
creating unclassified high-ceiling domains.
```
