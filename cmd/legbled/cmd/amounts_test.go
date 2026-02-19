package cmd

import "testing"

func TestParseHumanAmount(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		// LGM amounts
		{"0.1LGM", "100000", false},
		{"1LGM", "1000000", false},
		{"1.5lgm", "1500000", false},
		{"0.000001LGM", "1", false},
		{"10LGM", "10000000", false},
		{"100.123456LGM", "100123456", false},

		// ulgm passthrough
		{"100000ulgm", "100000", false},
		{"1000000ulgm", "1000000", false},
		{"0ulgm", "0", false},

		// Bare numbers (assumed ulgm)
		{"100000", "100000", false},
		{"1000000", "1000000", false},
		{"0", "0", false},

		// Case insensitive
		{"0.1LGM", "100000", false},
		{"0.1lgm", "100000", false},
		{"0.1Lgm", "100000", false},
		{"100ULGM", "100", false},

		// Whitespace
		{"  1LGM  ", "1000000", false},
		{" 100000 ulgm", "100000", false},

		// Errors
		{"", "", true},
		{"abc", "", true},
		{"-1LGM", "", true},
		{"-100ulgm", "", true},
		{"-100", "", true},
		{"1.1234567LGM", "", true}, // too many decimals
		{"LGM", "", true},          // missing number
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseHumanAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHumanAmount(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseHumanAmount(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatUlgmToLGM(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1000000", "1 LGM"},
		{"1500000", "1.500000 LGM"},
		{"100000", "0.100000 LGM"},
		{"0", "0 LGM"},
		{"10000000", "10 LGM"},
		{"1", "0.000001 LGM"},
		{"invalid", "invalid ulgm"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatUlgmToLGM(tt.input)
			if got != tt.want {
				t.Errorf("FormatUlgmToLGM(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseHumanCoin(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"0.1LGM", "100000ulgm", false},
		{"1LGM", "1000000ulgm", false},
		{"100000ulgm", "100000ulgm", false},
		{"100000", "100000ulgm", false},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseHumanCoin(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHumanCoin(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseHumanCoin(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
