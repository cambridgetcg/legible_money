package evaluation

import (
	"testing"
)

func TestEvaluateCategory_KnownCategories(t *testing.T) {
	tests := []struct {
		category string
		expected uint64
	}{
		{"protocol", 800_000},
		{"analytic", 700_000},
		{"formal", 700_000},
		{"computational", 650_000},
		{"replicated", 550_000},
		{"empirical", 500_000},
		{"historical", 450_000},
		{"social", 400_000},
		{"predictive", 350_000},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := evaluateCategory(tt.category)
			if result.Adjustment != int64(tt.expected) {
				t.Errorf("category %s: got %d, want %d", tt.category, result.Adjustment, tt.expected)
			}
			if result.Name != "category_base" {
				t.Errorf("expected stage name 'category_base', got %s", result.Name)
			}
		})
	}
}

func TestEvaluateCategory_UnknownCategory(t *testing.T) {
	result := evaluateCategory("unknown_category")
	if result.Adjustment != int64(defaultCategoryBase) {
		t.Errorf("unknown category: got %d, want %d", result.Adjustment, defaultCategoryBase)
	}
}

func TestEvaluateCategory_EmptyCategory(t *testing.T) {
	result := evaluateCategory("")
	if result.Adjustment != int64(defaultCategoryBase) {
		t.Errorf("empty category: got %d, want %d", result.Adjustment, defaultCategoryBase)
	}
}
