//go:build openclaw

package devnet_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/legible-money/legible/tests/openclaw/devnet"
)

// buildBinary compiles legbled to a temp dir and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "legbled")

	// Find module root
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root")
		}
		dir = parent
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/legbled")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("building binary: %v", err)
	}
	return binaryPath
}

func TestHarnessSmokeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping harness smoke test in short mode")
	}

	binary := buildBinary(t)
	h := devnet.NewHarness(t, binary)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Chain should be running and producing blocks
	if err := h.WaitForBlock(ctx, 3); err != nil {
		t.Fatalf("WaitForBlock(3) failed: %v", err)
	}

	// Validator key should have an address
	addr := h.KeyAddress("validator")
	if addr == "" {
		t.Fatal("validator address is empty")
	}
	if len(addr) < 5 || addr[:3] != "lgm" {
		t.Fatalf("unexpected validator address format: %s", addr)
	}

	t.Logf("Smoke test passed: chain_id=%s, validator=%s", h.ChainID, addr)
}
