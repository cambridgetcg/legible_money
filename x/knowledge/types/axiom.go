package types

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// GenesisAxiom represents a foundational truth injected at genesis.
// Axioms are converted to Fact objects with appropriate confidence and "verified" status.
// The layered type system supports 7 claim types with varying confidence models.
type GenesisAxiom struct {
	AxiomID           string   `json:"axiom_id"`                        // "MATH-001", "THEO-001", etc.
	Statement         string   `json:"statement"`                       // Natural language claim text
	FormalExpression  string   `json:"formal_expression,omitempty"`     // Optional symbolic/formal representation
	ClaimType         string   `json:"type"`                            // axiom|empirical_axiom|definition|regime_declaration|derived_claim|measurement_fact|meta
	Domain            string   `json:"domain"`                          // Must match a genesis domain name
	EpistemicCategory string   `json:"epistemic_category"`              // analytic|formal|empirical|protocol
	Regime            string   `json:"regime,omitempty"`                // classical|SR|GR|QM|QFT|thermo|statmech|cosmology|SM
	Confidence        float64  `json:"confidence"`                      // 0.0-1.0 (converted to BPS for on-chain storage)
	ApproxTags        []string `json:"approximation_tags,omitempty"`    // e.g., ["isolated_system", "continuum"]
	DerivationRef     string   `json:"derivation_ref,omitempty"`        // Compact ref to proof/derivation chain
	EvidenceRef       string   `json:"evidence_ref,omitempty"`          // Compact ref to experimental confirmation
	Dependencies      []string `json:"dependencies"`                    // Axiom IDs this builds on (DAG-validated)
	SourceTradition   string   `json:"source_tradition,omitempty"`      // Historical attribution
	References        []string `json:"references,omitempty"`            // Axiom IDs this cites (informational)
	Notes             string   `json:"notes,omitempty"`                 // Optional context
	TierRequired      int      `json:"tier_required,omitempty"`         // Minimum agent tier for challenge rights
}

// AxiomConfidence is the confidence value for genesis axioms (100% = 1,000,000 BPS).
const AxiomConfidence = uint64(1_000_000)

// AxiomSubmitter is the submitter address for genesis axioms.
const AxiomSubmitter = "genesis"

// AxiomSubmitterDID is the DID for genesis axioms.
const AxiomSubmitterDID = "did:lgm:genesis"

// ValidAxiomCategories lists the epistemic categories allowed for genesis axioms.
// analytic and formal have HalfLifeBlocks: 0 (eternal, no decay).
// empirical and protocol are also valid for physics/measurement axioms.
var ValidAxiomCategories = map[string]bool{
	"analytic": true,
	"formal":   true,
	"empirical": true,
	"protocol":  true,
}

// ValidClaimTypes lists the 7 claim types in the layered genesis axiom schema.
var ValidClaimTypes = map[string]bool{
	"axiom":              true,
	"empirical_axiom":    true,
	"definition":         true,
	"regime_declaration": true,
	"derived_claim":      true,
	"measurement_fact":   true,
	"meta":               true,
}

// ValidRegimes lists the physics regime tags for scoping empirical claims.
// Compound regimes (e.g., "QM/QFT") indicate boundary applicability.
var ValidRegimes = map[string]bool{
	"classical":              true,
	"classical_field_theory": true,
	"classical_gravity":      true,
	"SR":                     true,
	"GR":                     true,
	"QM":                     true,
	"QFT":                    true,
	"QM/QFT":                 true,
	"QM/statmech":            true,
	"SR/QFT":                 true,
	"EM":                     true,
	"EM/QM":                  true,
	"EM/QFT":                 true,
	"thermo":                 true,
	"statmech":               true,
	"cosmology":              true,
	"SM":                     true,
	"particle_physics":       true,
	"modern_physics":         true,
}

// ClaimTypeToCategory maps the 7 genesis claim types to 4 epistemic categories.
// Used by AxiomsToFacts() to auto-derive category when not explicitly set.
var ClaimTypeToCategory = map[string]string{
	"axiom":              "analytic",  // mathematically necessary
	"definition":         "analytic",  // true by convention
	"regime_declaration": "formal",    // scope declaration
	"derived_claim":      "formal",    // follows from premises
	"empirical_axiom":    "empirical", // experimentally constrained
	"measurement_fact":   "empirical", // observed parameter
	"meta":               "empirical", // statement about theories
}

// ClaimTypeDefaultConfidence returns the default confidence for a claim type
// when no explicit confidence is provided (confidence == 0).
var ClaimTypeDefaultConfidence = map[string]float64{
	"axiom":              1.0,
	"definition":         1.0,
	"regime_declaration": 1.0,
	"empirical_axiom":    0.97,
	"derived_claim":      0.95,
	"measurement_fact":   0.90,
	"meta":               0.85,
}

// DomainPrefixMap maps domain names to their axiom ID prefixes.
var DomainPrefixMap = map[string]string{
	"theology":           "THEO",
	"philosophy":         "PHIL",
	"mathematics":        "MATH",
	"logic":              "LOGIC",
	"physics":            "PHYS",
	"chemistry":          "CHEM",
	"biology":            "BIO",
	"computer_science":   "CS",
	"economics":          "ECON",
	"linguistics":        "LING",
	"psychology":         "PSYCH",
	"sociology":          "SOC",
	"cosmology":          "COSM",
	"information_theory": "INFO",
	"ethics":             "ETHIC",
	"agent_rights":       "AGRT",
}

// PrefixToDomainMap is the reverse mapping from prefix to domain.
var PrefixToDomainMap map[string]string

func init() {
	PrefixToDomainMap = make(map[string]string, len(DomainPrefixMap))
	for domain, prefix := range DomainPrefixMap {
		PrefixToDomainMap[prefix] = domain
	}
}

// axiomIDRegex validates axiom IDs: PREFIX-NNN with optional lowercase suffixes
// (e.g., "MATH-001", "AGRT-004a", "AGRT-004a-ii").
var axiomIDRegex = regexp.MustCompile(`^[A-Z]+-\d{3}[a-z0-9\-]*$`)

// ValidateAxiomID checks that an axiom ID has the correct PREFIX-NNN format
// and that the prefix maps to a known domain.
func ValidateAxiomID(id string) error {
	if !axiomIDRegex.MatchString(id) {
		return fmt.Errorf("axiom ID %q does not match format PREFIX-NNN (e.g., MATH-001)", id)
	}
	prefix := id[:strings.Index(id, "-")]
	if _, ok := PrefixToDomainMap[prefix]; !ok {
		return fmt.Errorf("axiom ID %q has unknown prefix %q", id, prefix)
	}
	return nil
}

// LoadAxiomsFromFile reads a JSON array of GenesisAxiom from a file.
func LoadAxiomsFromFile(path string) ([]*GenesisAxiom, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read axiom file %s: %w", path, err)
	}
	return ParseAxioms(data)
}

// ParseAxioms parses a JSON array of GenesisAxiom from raw bytes.
func ParseAxioms(data []byte) ([]*GenesisAxiom, error) {
	var axioms []*GenesisAxiom
	if err := json.Unmarshal(data, &axioms); err != nil {
		return nil, fmt.Errorf("failed to unmarshal axioms: %w", err)
	}
	return axioms, nil
}

// ValidateAxioms performs comprehensive validation on a set of genesis axioms.
// Checks: non-empty, unique IDs, valid ID format, valid domains, valid categories,
// prefix-domain consistency, non-empty statements, and DAG acyclicity.
func ValidateAxioms(axioms []*GenesisAxiom, validDomains []string) error {
	if len(axioms) == 0 {
		return fmt.Errorf("axiom set is empty")
	}

	domainSet := make(map[string]bool, len(validDomains))
	for _, d := range validDomains {
		domainSet[d] = true
	}

	idSet := make(map[string]bool, len(axioms))

	for i, a := range axioms {
		if a == nil {
			return fmt.Errorf("axiom at index %d is nil", i)
		}

		// Validate ID format
		if err := ValidateAxiomID(a.AxiomID); err != nil {
			return fmt.Errorf("axiom index %d: %w", i, err)
		}

		// Unique ID
		if idSet[a.AxiomID] {
			return fmt.Errorf("duplicate axiom ID: %s", a.AxiomID)
		}
		idSet[a.AxiomID] = true

		// Domain must exist
		if !domainSet[a.Domain] {
			return fmt.Errorf("axiom %s references unknown domain %q", a.AxiomID, a.Domain)
		}

		// Prefix must match domain
		prefix := a.AxiomID[:strings.Index(a.AxiomID, "-")]
		expectedDomain := PrefixToDomainMap[prefix]
		if expectedDomain != a.Domain {
			return fmt.Errorf("axiom %s prefix %q implies domain %q but domain is %q",
				a.AxiomID, prefix, expectedDomain, a.Domain)
		}

		// Valid claim type
		if a.ClaimType != "" && !ValidClaimTypes[a.ClaimType] {
			return fmt.Errorf("axiom %s has invalid claim type %q", a.AxiomID, a.ClaimType)
		}

		// Valid epistemic category
		if !ValidAxiomCategories[a.EpistemicCategory] {
			return fmt.Errorf("axiom %s has invalid epistemic category %q (must be analytic, formal, empirical, or protocol)",
				a.AxiomID, a.EpistemicCategory)
		}

		// Valid regime (if set)
		if a.Regime != "" && !ValidRegimes[a.Regime] {
			return fmt.Errorf("axiom %s has invalid regime %q", a.AxiomID, a.Regime)
		}

		// Confidence must be in [0, 1]
		if a.Confidence < 0 || a.Confidence > 1.0 {
			return fmt.Errorf("axiom %s has confidence %f outside [0, 1]", a.AxiomID, a.Confidence)
		}

		// derived_claim must have at least one dependency
		if a.ClaimType == "derived_claim" && len(a.Dependencies) == 0 {
			return fmt.Errorf("axiom %s is a derived_claim but has no dependencies", a.AxiomID)
		}

		// Non-empty statement
		if strings.TrimSpace(a.Statement) == "" {
			return fmt.Errorf("axiom %s has empty statement", a.AxiomID)
		}
	}

	// Validate references point to existing axioms
	for _, a := range axioms {
		for _, ref := range a.References {
			if !idSet[ref] {
				return fmt.Errorf("axiom %s references unknown axiom %q", a.AxiomID, ref)
			}
		}
	}

	// Validate dependencies form an acyclic DAG
	if err := ValidateAxiomDAG(axioms); err != nil {
		return err
	}

	// Validate stratum consistency (axioms/definitions must not depend on derived_claims)
	if err := ValidateStratumConsistency(axioms); err != nil {
		return err
	}

	// Validate derived_claim confidence <= min(dependency confidences)
	if err := ValidateDerivedConfidence(axioms); err != nil {
		return err
	}

	return nil
}

// ValidateAxiomDAG checks that axiom dependencies form a directed acyclic graph.
// Uses Kahn's algorithm (topological sort) to detect cycles.
func ValidateAxiomDAG(axioms []*GenesisAxiom) error {
	idSet := make(map[string]bool, len(axioms))
	for _, a := range axioms {
		idSet[a.AxiomID] = true
	}

	// Check all dependencies reference existing axioms
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			if !idSet[dep] {
				// Enhanced error: identify expected domain from prefix
				if dashIdx := strings.Index(dep, "-"); dashIdx > 0 {
					prefix := dep[:dashIdx]
					if domain, ok := PrefixToDomainMap[prefix]; ok {
						return fmt.Errorf("axiom %s depends on unknown axiom %q (expected in domain %q based on prefix %q)",
							a.AxiomID, dep, domain, prefix)
					}
				}
				return fmt.Errorf("axiom %s depends on unknown axiom %q", a.AxiomID, dep)
			}
			if dep == a.AxiomID {
				return fmt.Errorf("axiom %s has self-dependency", a.AxiomID)
			}
		}
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int, len(axioms))
	dependents := make(map[string][]string, len(axioms)) // dep -> axioms that depend on it

	for _, a := range axioms {
		if _, ok := inDegree[a.AxiomID]; !ok {
			inDegree[a.AxiomID] = 0
		}
		for _, dep := range a.Dependencies {
			inDegree[a.AxiomID]++
			dependents[dep] = append(dependents[dep], a.AxiomID)
		}
	}

	// Queue of nodes with no incoming edges
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	sorted := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted++

		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if sorted != len(axioms) {
		// Find a cycle for error reporting
		var cycleMembers []string
		for id, degree := range inDegree {
			if degree > 0 {
				cycleMembers = append(cycleMembers, id)
			}
		}
		return fmt.Errorf("dependency cycle detected among axioms: %v", cycleMembers)
	}

	return nil
}

// ResolveAxiomCategory determines the epistemic category for a genesis axiom.
// If the axiom has an explicit EpistemicCategory, it is used (override).
// Otherwise, the category is auto-derived from ClaimType via ClaimTypeToCategory.
// Returns the resolved category and a bool indicating if an explicit override
// disagreed with the auto-derived value (for logging warnings).
func ResolveAxiomCategory(a *GenesisAxiom) (category string, overrideDisagrees bool) {
	autoDerived := ""
	if a.ClaimType != "" {
		autoDerived = ClaimTypeToCategory[a.ClaimType]
	}

	if a.EpistemicCategory != "" {
		// Explicit category set — use it, but flag if it disagrees with auto-derived
		if autoDerived != "" && a.EpistemicCategory != autoDerived {
			return a.EpistemicCategory, true
		}
		return a.EpistemicCategory, false
	}

	// No explicit category — use auto-derived
	if autoDerived != "" {
		return autoDerived, false
	}

	// Fallback: no claim type and no category — default to analytic (legacy axioms)
	return "analytic", false
}

// axiomConfidenceBPS converts a genesis axiom's float64 confidence [0,1] to BPS uint64 [0,1_000_000].
// If confidence is 0, uses the default for the claim type.
func axiomConfidenceBPS(a *GenesisAxiom) uint64 {
	conf := a.Confidence
	if conf == 0 {
		if def, ok := ClaimTypeDefaultConfidence[a.ClaimType]; ok {
			conf = def
		} else {
			conf = 1.0 // fallback for legacy axioms without claim type
		}
	}
	return uint64(conf * 1_000_000)
}

// AxiomsToFacts converts genesis axioms to Fact objects ready for InitGenesis.
// Facts are created with per-axiom confidence, "verified" status, "canonical" maturity,
// and block height 0 (genesis). Category is auto-derived from ClaimType unless
// explicitly overridden by EpistemicCategory on the axiom.
func AxiomsToFacts(axioms []*GenesisAxiom) []*Fact {
	facts := make([]*Fact, 0, len(axioms))
	for _, a := range axioms {
		confBPS := axiomConfidenceBPS(a)
		resolvedCategory, _ := ResolveAxiomCategory(a)
		fact := &Fact{
			Id:                a.AxiomID,
			ClaimText:         a.Statement,
			Domain:            a.Domain,
			EpistemicCategory: resolvedCategory,
			Confidence:        confBPS,
			BaseConfidence:    confBPS,
			Status:            FactStatusVerified,
			Submitter:         AxiomSubmitter,
			SubmitterDid:      AxiomSubmitterDID,
			References:        a.Dependencies, // Dependencies become fact references
			VerifiedAtHeight:  0,              // Genesis block
			CreatedAtHeight:   0,
			LastVerifiedHeight:     0,
			LastConfidenceUpdate:   0,
			VerifierCount:          0, // No verifiers needed for axioms
			Maturity:               MaturityCanonical,
			StratumCeiling:         1_000_000, // No stratum limit for axioms
			ExpiresAtBlock:         0,         // Never expires
			IsProvisional:          false,
		}
		facts = append(facts, fact)
	}
	return facts
}

// ValidateDerivedConfidence checks that no derived_claim has confidence exceeding
// the minimum confidence of its dependencies. This enforces the rule:
// derived_claim confidence <= min(dependency confidences).
// Uses effective confidence (with defaults applied for zero values).
func ValidateDerivedConfidence(axioms []*GenesisAxiom) error {
	// Build confidence lookup using effective confidence
	confOf := make(map[string]float64, len(axioms))
	for _, a := range axioms {
		conf := a.Confidence
		if conf == 0 {
			if def, ok := ClaimTypeDefaultConfidence[a.ClaimType]; ok {
				conf = def
			} else {
				conf = 1.0
			}
		}
		confOf[a.AxiomID] = conf
	}

	for _, a := range axioms {
		if a.ClaimType != "derived_claim" || len(a.Dependencies) == 0 {
			continue
		}

		minDepConf := 2.0 // sentinel > 1.0
		for _, dep := range a.Dependencies {
			if c, ok := confOf[dep]; ok && c < minDepConf {
				minDepConf = c
			}
		}

		effectiveConf := confOf[a.AxiomID]
		if effectiveConf > minDepConf {
			return fmt.Errorf("derived_claim %s has confidence %.4f exceeding min dependency confidence %.4f",
				a.AxiomID, effectiveConf, minDepConf)
		}
	}
	return nil
}

// AxiomsToFactReferences creates FactReference edges from axiom dependencies.
// Each dependency edge (A depends on B) becomes a "supports" reference:
// from_fact_id=A, to_fact_id=B — meaning "A references B as supporting evidence".
//
// Direction: axiom.Dependencies says "I depend on these", so for each dep B in
// axiom A's dependencies, A→B is "B supports A", stored as from=A, to=B.
func AxiomsToFactReferences(axioms []*GenesisAxiom) []*FactReference {
	var refs []*FactReference
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			ref := &FactReference{
				FromFactId:      a.AxiomID,     // the dependent (referencer)
				ToFactId:        dep,            // the dependency (referenced)
				ReferenceType:   "supports",     // dependency = supported by
				Weight:          1_000_000,      // full weight for genesis
				CreatedAtHeight: 0,              // genesis block
			}
			refs = append(refs, ref)
		}
	}
	return refs
}

// FactReferenceStats computes summary statistics for a set of FactReferences.
type FactReferenceStats struct {
	Total      int
	ByType     map[string]int
	CrossDomain int
	IntraDomain int
}

// ComputeFactReferenceStats computes reference stats given axioms and their references.
func ComputeFactReferenceStats(axioms []*GenesisAxiom, refs []*FactReference) FactReferenceStats {
	stats := FactReferenceStats{
		Total:  len(refs),
		ByType: make(map[string]int),
	}

	// Build axiom domain lookup
	domainOf := make(map[string]string, len(axioms))
	for _, a := range axioms {
		domainOf[a.AxiomID] = a.Domain
	}

	for _, r := range refs {
		stats.ByType[r.ReferenceType]++
		fromDomain := domainOf[r.FromFactId]
		toDomain := domainOf[r.ToFactId]
		if fromDomain != "" && toDomain != "" {
			if fromDomain == toDomain {
				stats.IntraDomain++
			} else {
				stats.CrossDomain++
			}
		}
	}
	return stats
}

// AxiomDomainNames returns the list of domain names that support axioms
// (all domains in DomainPrefixMap).
func AxiomDomainNames() []string {
	names := make([]string, 0, len(DomainPrefixMap))
	for domain := range DomainPrefixMap {
		names = append(names, domain)
	}
	return names
}

// DAGStats holds topological analysis results for the axiom dependency DAG.
type DAGStats struct {
	RootCount      int         // nodes with no dependencies (true axioms)
	MaxDepth       int         // longest path from any root to leaf
	DepthCounts    map[int]int // depth → count of items at that depth
	Orphans        []string    // items with no deps AND nothing depends on them
	MostReferenced []string    // top-N most-depended-on items (hub nodes)
	TotalEdges     int         // total dependency edges
}

// CrossDomainEntry records references between two domains.
type CrossDomainEntry struct {
	FromDomain string
	ToDomain   string
	Count      int
}

// CrossDomainMatrix holds the full inter-domain dependency matrix.
type CrossDomainMatrix struct {
	Entries []CrossDomainEntry
	Domains []string // sorted unique domain names
}

// ComputeDAGStats performs topological analysis on the axiom dependency DAG.
// Returns depth histogram, root/orphan counts, hub nodes, and total edges.
func ComputeDAGStats(axioms []*GenesisAxiom) (*DAGStats, error) {
	// Build adjacency and in-degree maps (same as Kahn's in ValidateAxiomDAG)
	inDegree := make(map[string]int, len(axioms))
	dependents := make(map[string][]string, len(axioms)) // dep → items that depend on it
	depCount := make(map[string]int, len(axioms))        // how many items depend on this

	for _, a := range axioms {
		if _, ok := inDegree[a.AxiomID]; !ok {
			inDegree[a.AxiomID] = 0
		}
		for _, dep := range a.Dependencies {
			inDegree[a.AxiomID]++
			dependents[dep] = append(dependents[dep], a.AxiomID)
			depCount[dep]++
		}
	}

	// BFS from roots to compute depth of each node
	depth := make(map[string]int, len(axioms))
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
			depth[id] = 0
		}
	}

	// Process in topological order; each node's depth = max(dep depths) + 1
	processed := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		processed++

		for _, dependent := range dependents[node] {
			candidateDepth := depth[node] + 1
			if candidateDepth > depth[dependent] {
				depth[dependent] = candidateDepth
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if processed != len(axioms) {
		return nil, fmt.Errorf("cycle detected: processed %d of %d nodes", processed, len(axioms))
	}

	// Compute stats
	stats := &DAGStats{
		DepthCounts: make(map[int]int),
	}

	dependedOn := make(map[string]bool)
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			dependedOn[dep] = true
		}
		stats.TotalEdges += len(a.Dependencies)
	}

	maxDepth := 0
	for _, a := range axioms {
		d := depth[a.AxiomID]
		stats.DepthCounts[d]++
		if d > maxDepth {
			maxDepth = d
		}
		if d == 0 {
			stats.RootCount++
		}
		if len(a.Dependencies) == 0 && !dependedOn[a.AxiomID] {
			stats.Orphans = append(stats.Orphans, a.AxiomID)
		}
	}
	stats.MaxDepth = maxDepth

	sort.Strings(stats.Orphans)

	// Top 10 most-referenced items
	type idCount struct {
		id    string
		count int
	}
	var counts []idCount
	for id, c := range depCount {
		counts = append(counts, idCount{id, c})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].count != counts[j].count {
			return counts[i].count > counts[j].count
		}
		return counts[i].id < counts[j].id
	})
	limit := 10
	if len(counts) < limit {
		limit = len(counts)
	}
	stats.MostReferenced = make([]string, limit)
	for i := 0; i < limit; i++ {
		stats.MostReferenced[i] = fmt.Sprintf("%s (%d)", counts[i].id, counts[i].count)
	}

	return stats, nil
}

// ComputeCrossDomainMatrix builds an inter-domain dependency matrix.
// For each axiom, for each dependency in a different domain, increments the from→to count.
func ComputeCrossDomainMatrix(axioms []*GenesisAxiom) *CrossDomainMatrix {
	idToDomain := make(map[string]string, len(axioms))
	domainSet := make(map[string]bool)
	for _, a := range axioms {
		idToDomain[a.AxiomID] = a.Domain
		domainSet[a.Domain] = true
	}

	type pair struct{ from, to string }
	matrix := make(map[pair]int)
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			depDomain := idToDomain[dep]
			if depDomain == "" {
				if dashIdx := strings.Index(dep, "-"); dashIdx > 0 {
					if d, ok := PrefixToDomainMap[dep[:dashIdx]]; ok {
						depDomain = d
					}
				}
			}
			if a.Domain != depDomain && depDomain != "" {
				matrix[pair{a.Domain, depDomain}]++
			}
		}
	}

	var entries []CrossDomainEntry
	for p, count := range matrix {
		entries = append(entries, CrossDomainEntry{
			FromDomain: p.from,
			ToDomain:   p.to,
			Count:      count,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		if entries[i].FromDomain != entries[j].FromDomain {
			return entries[i].FromDomain < entries[j].FromDomain
		}
		return entries[i].ToDomain < entries[j].ToDomain
	})

	var domains []string
	for d := range domainSet {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	return &CrossDomainMatrix{
		Entries: entries,
		Domains: domains,
	}
}

// ValidateStratumConsistency checks that axioms do not depend on derived types.
// Definitions MAY depend on derived claims (e.g., a physics definition referencing
// a mathematical theorem) since cross-domain definitions often build on established
// results. Only axioms (self-evident truths) are strictly foundational.
func ValidateStratumConsistency(axioms []*GenesisAxiom) error {
	idToType := make(map[string]string, len(axioms))
	for _, a := range axioms {
		if a.ClaimType != "" {
			idToType[a.AxiomID] = a.ClaimType
		}
	}

	for _, a := range axioms {
		if a.ClaimType != "axiom" {
			continue
		}
		for _, dep := range a.Dependencies {
			depType := idToType[dep]
			if depType == "derived_claim" || depType == "measurement_fact" {
				return fmt.Errorf("stratum violation: axiom %s depends on %s %s — axioms must not depend on derived or measured claims",
					a.AxiomID, depType, dep)
			}
		}
	}
	return nil
}

