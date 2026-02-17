package bugreport

// Severity classifies the impact of a reported bug.
type Severity string

const (
	SeverityCritical Severity = "critical" // Consensus-breaking, fund loss
	SeverityHigh     Severity = "high"     // Incorrect state transitions
	SeverityMedium   Severity = "medium"   // Edge-case failures
	SeverityLow      Severity = "low"      // Cosmetic, non-functional
)

// BountyMultiplier returns the payout multiplier for a severity level.
// Critical bugs pay 4x base, high 2x, medium 1x, low 0.5x.
func (s Severity) BountyMultiplier() uint32 {
	switch s {
	case SeverityCritical:
		return 400 // 4.00x in BPS/100
	case SeverityHigh:
		return 200
	case SeverityMedium:
		return 100
	case SeverityLow:
		return 50
	default:
		return 100
	}
}

// Valid returns true if the severity is a recognized value.
func (s Severity) Valid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	default:
		return false
	}
}
