package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runInit executes the clinban binary with "init [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runInit(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"init"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = coverEnv()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// TestInitFreshDirectory verifies that "clinban init" in a clean directory
// creates all five artifacts and exits 0.
func TestInitFreshDirectory(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, exitCode := runInit(t, bin, dir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/", stdout)
	}
	if !strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/archive/", stdout)
	}
	if !strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout does not contain %q: %q", "created: .clinban", stdout)
	}
	if !strings.Contains(stdout, "created: SCHEMA.md") {
		t.Errorf("stdout does not contain %q: %q", "created: SCHEMA.md", stdout)
	}
	if !strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout does not contain %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}

	// Verify artifacts exist on disk.
	if _, err := os.Stat(filepath.Join(dir, "tickets")); err != nil {
		t.Errorf("tickets/ not found on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tickets", "archive")); err != nil {
		t.Errorf("tickets/archive/ not found on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".clinban")); err != nil {
		t.Errorf(".clinban not found on disk: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "SCHEMA.md"))
	if err != nil {
		t.Errorf("SCHEMA.md not found on disk: %v", err)
	} else if fi.Size() == 0 {
		t.Errorf("SCHEMA.md exists but is empty")
	}
	skillPath := filepath.Join(dir, ".claude", "skills", "tickets", "SKILL.md")
	fiSkill, err := os.Stat(skillPath)
	if err != nil {
		t.Errorf(".claude/skills/tickets/SKILL.md not found on disk: %v", err)
	} else if fiSkill.Size() == 0 {
		t.Errorf(".claude/skills/tickets/SKILL.md exists but is empty")
	}
}

// TestInitAlreadyExists_NoForce verifies that "clinban init" without --force
// exits 1 and reports each existing artifact on stderr when all three already exist.
func TestInitAlreadyExists_NoForce(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create all three artifacts.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "already exists: tickets/") {
		t.Errorf("stderr does not mention existing tickets/: %q", stderr)
	}
	if !strings.Contains(stderr, "already exists: tickets/archive/") {
		t.Errorf("stderr does not mention existing tickets/archive/: %q", stderr)
	}
	if !strings.Contains(stderr, "already exists: .clinban") {
		t.Errorf("stderr does not mention existing .clinban: %q", stderr)
	}
	if !strings.Contains(stderr, "missing: SCHEMA.md") {
		t.Errorf("stderr does not list missing SCHEMA.md: %q", stderr)
	}
}

// TestInitAlreadyExists_WithForce verifies that "clinban init --force" exits 0
// (creates the skill file) when the original four artifacts already exist but
// SKILL.md does not — the fully-initialized guard requires all five.
func TestInitAlreadyExists_WithForce(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create all four original artifacts (no SKILL.md).
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SCHEMA.md"), []byte("# schema"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	// With four of five artifacts present, --force should create the missing one and exit 0.
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout does not contain %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}
	_ = stderr
}

// TestInitPartial_DirsExist_NoConfig_Force verifies that "clinban init --force"
// creates only the missing .clinban, SCHEMA.md, and SKILL.md when tickets/ and tickets/archive/ already exist.
func TestInitPartial_DirsExist_NoConfig_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create directories only, no .clinban, SCHEMA.md, or SKILL.md.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout does not contain %q: %q", "created: .clinban", stdout)
	}
	if !strings.Contains(stdout, "created: SCHEMA.md") {
		t.Errorf("stdout does not contain %q: %q", "created: SCHEMA.md", stdout)
	}
	if !strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout does not contain %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}
	if strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/", stdout)
	}
	if strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/archive/", stdout)
	}
}

// TestInitPartial_ConfigExists_NoDirs_Force verifies that "clinban init --force"
// creates tickets/, tickets/archive/, SCHEMA.md, and SKILL.md when only .clinban already exists.
func TestInitPartial_ConfigExists_NoDirs_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create .clinban only, no directories, SCHEMA.md, or SKILL.md.
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/", stdout)
	}
	if !strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/archive/", stdout)
	}
	if !strings.Contains(stdout, "created: SCHEMA.md") {
		t.Errorf("stdout does not contain %q: %q", "created: SCHEMA.md", stdout)
	}
	if !strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout does not contain %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}
	if strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: .clinban", stdout)
	}
}

// TestInitPartial_SchemaOnly_Force verifies that "clinban init --force" creates
// SCHEMA.md and SKILL.md when tickets/, tickets/archive/, and .clinban already exist.
func TestInitPartial_SchemaOnly_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create three artifacts; do NOT pre-create SCHEMA.md or SKILL.md.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: SCHEMA.md") {
		t.Errorf("stdout does not contain %q: %q", "created: SCHEMA.md", stdout)
	}
	if !strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout does not contain %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}
	if strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/", stdout)
	}
	if strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/archive/", stdout)
	}
	if strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: .clinban", stdout)
	}
}

// TestInitSkillFileExists_NoForce verifies that "clinban init" without --force
// exits 1 and reports the SKILL.md as already existing when it is pre-created.
func TestInitSkillFileExists_NoForce(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create only the SKILL.md file.
	skillPath := filepath.Join(dir, ".claude", "skills", "tickets", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("# skill"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, stderr, exitCode := runInit(t, bin, dir)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "already exists: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stderr does not contain %q: %q", "already exists: .claude/skills/tickets/SKILL.md", stderr)
	}
}

// TestInitSkillFileExists_Force verifies that "clinban init --force" with a
// pre-existing SKILL.md still creates the other missing artifacts and does not
// re-report SKILL.md as created.
func TestInitSkillFileExists_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create only the SKILL.md file.
	skillPath := filepath.Join(dir, ".claude", "skills", "tickets", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("# skill"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	// The other four artifacts should be created.
	if !strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/", stdout)
	}
	if !strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/archive/", stdout)
	}
	if !strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout does not contain %q: %q", "created: .clinban", stdout)
	}
	if !strings.Contains(stdout, "created: SCHEMA.md") {
		t.Errorf("stdout does not contain %q: %q", "created: SCHEMA.md", stdout)
	}
	// SKILL.md was already present — must NOT be reported as created.
	if strings.Contains(stdout, "created: .claude/skills/tickets/SKILL.md") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: .claude/skills/tickets/SKILL.md", stdout)
	}
}

// TestInitAllFiveExist_Force verifies that "clinban init --force" exits 1 with
// "already fully initialized" on stderr when all five artifacts already exist.
func TestInitAllFiveExist_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create all five artifacts.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SCHEMA.md"), []byte("# schema"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	skillPath := filepath.Join(dir, ".claude", "skills", "tickets", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("# skill"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "already fully initialized") {
		t.Errorf("stderr does not contain %q: %q", "already fully initialized", stderr)
	}
}
