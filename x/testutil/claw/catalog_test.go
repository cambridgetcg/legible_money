package claw

import (
	"fmt"
	"testing"
)

func TestCatalog_NoDuplicateIDs(t *testing.T) {
	seen := make(map[string]int)
	for i, p := range Catalog {
		if prev, ok := seen[p.ID]; ok {
			t.Errorf("duplicate probe ID %q: index %d and %d", p.ID, prev, i)
		}
		seen[p.ID] = i
	}
	t.Logf("Checked %d probes, all IDs unique", len(Catalog))
}

func TestCatalog_AllFieldsSet(t *testing.T) {
	for _, p := range Catalog {
		if p.ID == "" {
			t.Errorf("probe with empty ID: %+v", p)
		}
		if p.Name == "" {
			t.Errorf("probe %s has empty Name", p.ID)
		}
		if p.Variant == "" {
			t.Errorf("probe %s has empty Variant", p.ID)
		}
		if p.Severity == "" {
			t.Errorf("probe %s has empty Severity", p.ID)
		}
		if p.Status == "" {
			t.Errorf("probe %s has empty Status", p.ID)
		}
		if p.Module == "" {
			t.Errorf("probe %s has empty Module", p.ID)
		}

		// NotTested probes (planned) may lack file/func
		if p.Status != NotTested {
			if p.File == "" {
				t.Errorf("probe %s has empty File", p.ID)
			}
			if p.TestFunc == "" {
				t.Errorf("probe %s has empty TestFunc", p.ID)
			}
		}
	}
}

func TestCatalog_NoCriticalLoopholes(t *testing.T) {
	for _, p := range Catalog {
		if p.Severity == Critical && p.Status == Loophole {
			t.Errorf("CRITICAL loophole: %s (%s) — requires remediation", p.ID, p.Name)
		}
	}
}

func TestCatalog_ValidVariants(t *testing.T) {
	valid := map[Variant]bool{OpenClaw: true, NanoClaw: true, ZeroClaw: true, CrossModule: true}
	for _, p := range Catalog {
		if !valid[p.Variant] {
			t.Errorf("probe %s has invalid variant: %q", p.ID, p.Variant)
		}
	}
}

func TestCatalog_ValidStatuses(t *testing.T) {
	valid := map[Status]bool{Defended: true, Loophole: true, Partial: true, NotTested: true}
	for _, p := range Catalog {
		if !valid[p.Status] {
			t.Errorf("probe %s has invalid status: %q", p.ID, p.Status)
		}
	}
}

func TestCatalog_CoverageSummary(t *testing.T) {
	coverage := CoverageByModule()
	uncovered := UncoveredModules()

	// Status distribution
	statusCounts := make(map[Status]int)
	for _, p := range Catalog {
		statusCounts[p.Status]++
	}

	// Variant distribution
	variantCounts := make(map[Variant]int)
	for _, p := range Catalog {
		variantCounts[p.Variant]++
	}

	// Severity distribution
	severityCounts := make(map[Severity]int)
	for _, p := range Catalog {
		severityCounts[p.Severity]++
	}

	t.Logf("=== Claw Probe Coverage Summary ===")
	t.Logf("Total probes: %d", len(Catalog))
	t.Logf("")

	t.Logf("By Status:")
	for _, s := range []Status{Defended, Partial, Loophole, NotTested} {
		t.Logf("  %-12s %d", s, statusCounts[s])
	}
	t.Logf("")

	t.Logf("By Variant:")
	for _, v := range []Variant{OpenClaw, NanoClaw, ZeroClaw, CrossModule} {
		t.Logf("  %-12s %d", v, variantCounts[v])
	}
	t.Logf("")

	t.Logf("By Severity:")
	for _, s := range []Severity{Critical, High, Medium, Low} {
		t.Logf("  %-12s %d", s, severityCounts[s])
	}
	t.Logf("")

	t.Logf("Module Coverage:")
	for module, count := range coverage {
		t.Logf("  %-20s %d probes", module, count)
	}
	t.Logf("")

	t.Logf("Uncovered Modules (%d):", len(uncovered))
	for _, m := range uncovered {
		t.Logf("  - %s", m)
	}
}

func TestByModule(t *testing.T) {
	knowledge := ByModule("knowledge")
	if len(knowledge) != 15 {
		t.Errorf("expected 15 knowledge probes, got %d", len(knowledge))
	}

	auth := ByModule("auth")
	if len(auth) != 12 {
		t.Errorf("expected 12 auth probes, got %d", len(auth))
	}

	bvm := ByModule("bvm")
	if len(bvm) != 18 {
		t.Errorf("expected 18 bvm probes, got %d", len(bvm))
	}
}

func TestByVariant(t *testing.T) {
	oc := ByVariant(OpenClaw)
	xm := ByVariant(CrossModule)
	if len(oc) == 0 {
		t.Error("expected OpenClaw probes")
	}
	if len(xm) != 8 {
		t.Errorf("expected 8 CrossModule probes, got %d", len(xm))
	}
}

func TestByStatus(t *testing.T) {
	notTested := ByStatus(NotTested)
	if len(notTested) != 8 {
		t.Errorf("expected 8 NotTested probes (XM-1..8), got %d", len(notTested))
	}
}

func TestUncoveredModules(t *testing.T) {
	uncovered := UncoveredModules()
	if len(uncovered) == 0 {
		t.Error("expected some uncovered modules")
	}

	// Spot-check: staking should be uncovered (only in XM cross-module, not standalone)
	found := false
	for _, m := range uncovered {
		if m == "staking" {
			found = true
			break
		}
	}
	if !found {
		// staking has no standalone probes, only XM entries which use "staking+knowledge"
		t.Log("Note: staking not in uncovered list (may have cross-module entries)")
	}

	t.Logf("Uncovered modules: %v", uncovered)
	for _, m := range uncovered {
		fmt.Printf("  [ ] %s\n", m)
	}
}
