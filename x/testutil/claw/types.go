package claw

// Variant classifies the attack surface being probed.
type Variant string

const (
	// OpenClaw probes macro game theory: multi-actor economics, collusion,
	// information asymmetry, strategic behavior.
	OpenClaw Variant = "OC"

	// NanoClaw probes precision arithmetic: integer truncation, BPS boundaries,
	// off-by-one errors, type conversion edge cases.
	NanoClaw Variant = "NC"

	// ZeroClaw probes zero-cost attack vectors: empty work exploits, free
	// information extraction, gas-less exploitation, zero-stake operations.
	ZeroClaw Variant = "ZC"

	// CrossModule probes attacks spanning multiple Cosmos SDK modules
	// (e.g., staking + knowledge race conditions).
	CrossModule Variant = "XM"
)

// Severity classifies the potential impact.
type Severity string

const (
	Critical Severity = "CRITICAL" // Token loss, consensus break
	High     Severity = "HIGH"     // Economic manipulation, unfair advantage
	Medium   Severity = "MEDIUM"   // Degraded incentives, gaming
	Low      Severity = "LOW"      // Information leakage, minor gaming
)

// Status tracks the defense state.
type Status string

const (
	Defended  Status = "DEFENDED"   // Defense implemented and verified
	Loophole  Status = "LOOPHOLE"  // Vulnerability identified, no defense
	Partial   Status = "PARTIAL"   // Defense exists but incomplete
	NotTested Status = "NOT_TESTED" // Module has no claw probes yet
)

// Probe describes a single adversarial test.
type Probe struct {
	ID       string   // e.g. "OC-1", "OC-BVM-3", "OCA-T1"
	Name     string   // Human-readable, e.g. "Whale Domination"
	Variant  Variant
	Severity Severity
	Status   Status
	Module   string   // e.g. "knowledge", "auth", "bvm"
	File     string   // Relative path to test file
	TestFunc string   // Go test function name
	Defense  string   // Brief description of defense mechanism
}
