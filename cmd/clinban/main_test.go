package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testBin is the path to the coverage-instrumented clinban binary built once
// by TestMain. All subprocess tests use this binary so that their execution
// contributes to the coverage report collected in testCoverDir.
var testBin string

// testCoverDir is the directory where the instrumented binary writes its
// per-process coverage counters (GOCOVERDIR).
//
// When "go test -cover" is in use, Go sets GOCOVERDIR automatically for the
// test binary. TestMain reads that value and reuses it for subprocess
// invocations so all runs share the same counter directory. When tests are run
// without -cover (GOCOVERDIR unset), a local temp dir is created so the
// instrumented binary still has somewhere to write (it panics without one).
var testCoverDir string

// TestMain builds the clinban binary with -cover once for the entire test run
// and wires GOCOVERDIR so subprocess executions contribute to coverage.
//
// Graceful degradation: if the build fails the package-level testBin is left
// empty and individual tests fall back to buildBinary(t), which compiles a
// plain (non-instrumented) binary.
func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	// If go test -cover set GOCOVERDIR for us, reuse that directory.
	// Otherwise create our own so the -cover binary has somewhere to write.
	testCoverDir = os.Getenv("GOCOVERDIR")
	if testCoverDir == "" {
		dir, err := os.MkdirTemp("", "clinban-cover-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "TestMain: create cover dir: %v\n", err)
			return m.Run()
		}
		defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup of temp dir
		testCoverDir = dir
	}

	// Locate the module root so "go build" can resolve the package path.
	root, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: %v\n", err)
		return m.Run()
	}

	// Build the instrumented binary into a temp dir. The binary is shared by
	// all tests so it is built once here rather than per-test.
	binDir, err := os.MkdirTemp("", "clinban-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: create bin dir: %v\n", err)
		return m.Run()
	}
	defer os.RemoveAll(binDir) //nolint:errcheck // best-effort cleanup of temp dir

	binPath := filepath.Join(binDir, "clinban")
	buildCmd := exec.Command("go", "build", "-cover", "-o", binPath, "./cmd/clinban/")
	buildCmd.Dir = root
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: build -cover failed: %v\n%s\n", err, out)
		// testBin stays empty; tests fall back to buildBinary(t).
		return m.Run()
	}

	testBin = binPath
	return m.Run()
}

// coverEnv returns os.Environ() augmented with GOCOVERDIR=testCoverDir.
// All runXxx helpers call this so that subprocess coverage counters are
// written to the shared directory and included in the final report.
//
// If testCoverDir is empty (graceful-degradation path) the returned slice is
// just os.Environ() unchanged.
func coverEnv() []string {
	env := os.Environ()
	if testCoverDir != "" {
		env = append(env, "GOCOVERDIR="+testCoverDir)
	}
	return env
}

// findModuleRoot walks up from the current working directory until it finds
// a directory containing go.mod.
func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod starting from %s", wd)
		}
		dir = parent
	}
}
