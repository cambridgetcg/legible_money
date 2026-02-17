//go:build openclaw

package openclaw

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binaryPath holds the path to the compiled legbled binary, built once in TestMain.
var binaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "openclaw-bin-")
	if err != nil {
		panic("creating temp dir for binary: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "legbled")

	// Find module root by walking up from cwd looking for go.mod
	root := findModuleRoot()
	if root == "" {
		panic("could not find module root (go.mod)")
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/legbled")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building legbled binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// findModuleRoot walks up from the working directory to find go.mod.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
