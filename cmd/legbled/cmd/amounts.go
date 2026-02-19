package cmd

import (
	"fmt"
	"math/big"
	"strings"
)

const (
	// lgmToUlgm is the conversion factor: 1 LGM = 1,000,000 ulgm.
	lgmToUlgm = 1_000_000
)

// ParseHumanAmount converts human-readable amounts to ulgm strings.
//
// Accepted formats:
//   - "0.1LGM"    → "100000"
//   - "1LGM"      → "1000000"
//   - "1.5lgm"    → "1500000"
//   - "100000ulgm" → "100000"  (passthrough)
//   - "1000000"   → "1000000"  (bare number, assumed ulgm)
func ParseHumanAmount(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty amount")
	}

	lower := strings.ToLower(s)

	// Case 1: explicit ulgm suffix — passthrough
	if strings.HasSuffix(lower, "ulgm") {
		numStr := strings.TrimSuffix(lower, "ulgm")
		numStr = strings.TrimSpace(numStr)
		n := new(big.Int)
		if _, ok := n.SetString(numStr, 10); !ok || n.Sign() < 0 {
			return "", fmt.Errorf("invalid ulgm amount: %s", s)
		}
		return n.String(), nil
	}

	// Case 2: LGM suffix — convert to ulgm
	if strings.HasSuffix(lower, "lgm") {
		numStr := strings.TrimSuffix(lower, "lgm")
		numStr = strings.TrimSpace(numStr)
		return lgmToUlgmStr(numStr, s)
	}

	// Case 3: bare number — assumed ulgm
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return "", fmt.Errorf("invalid amount: %s (use e.g. 0.1LGM, 100000ulgm, or 100000)", s)
	}
	if n.Sign() < 0 {
		return "", fmt.Errorf("amount cannot be negative: %s", s)
	}
	return n.String(), nil
}

// lgmToUlgmStr converts a decimal LGM string to ulgm.
// Handles up to 6 decimal places (the full precision of ulgm).
func lgmToUlgmStr(numStr string, original string) (string, error) {
	if numStr == "" {
		return "", fmt.Errorf("missing number before LGM: %s", original)
	}

	parts := strings.SplitN(numStr, ".", 2)
	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}

	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	// Validate whole part
	whole := new(big.Int)
	if _, ok := whole.SetString(wholePart, 10); !ok || whole.Sign() < 0 {
		return "", fmt.Errorf("invalid LGM amount: %s", original)
	}

	// Pad or truncate fractional part to 6 digits
	if len(fracPart) > 6 {
		return "", fmt.Errorf("LGM amount has too many decimal places (max 6): %s", original)
	}
	for len(fracPart) < 6 {
		fracPart += "0"
	}

	frac := new(big.Int)
	if _, ok := frac.SetString(fracPart, 10); !ok {
		return "", fmt.Errorf("invalid decimal in LGM amount: %s", original)
	}

	// result = whole * 1_000_000 + frac
	result := new(big.Int).Mul(whole, big.NewInt(lgmToUlgm))
	result.Add(result, frac)

	return result.String(), nil
}

// FormatUlgmToLGM formats a ulgm amount string as human-readable LGM.
// E.g., "1500000" → "1.500000 LGM", "100000" → "0.100000 LGM".
func FormatUlgmToLGM(ulgmStr string) string {
	n := new(big.Int)
	if _, ok := n.SetString(ulgmStr, 10); !ok {
		return ulgmStr + " ulgm"
	}

	divisor := big.NewInt(lgmToUlgm)
	whole := new(big.Int)
	frac := new(big.Int)
	whole.DivMod(n, divisor, frac)

	if frac.Sign() == 0 {
		return fmt.Sprintf("%s LGM", whole.String())
	}
	return fmt.Sprintf("%s.%06s LGM", whole.String(), frac.String())
}

// ParseHumanCoin converts a human-readable amount to an sdk.Coin-style string.
// Returns amount as "Nulgm" suitable for --amount flags.
func ParseHumanCoin(s string) (string, error) {
	ulgm, err := ParseHumanAmount(s)
	if err != nil {
		return "", err
	}
	return ulgm + "ulgm", nil
}
