# Implementation Design
_Produced by: techlead-agent_
_Date: 2026-06-14_
_Status: draft_
_Input: pipeline/0020/02_architecture.md_

## Module Structure

### Release Workflow

**Files:**
- `.github/workflows/release.yml` — GitHub Actions workflow that cross-compiles clinban for 3 platforms and uploads binaries + checksums to a GitHub Release on every `v*` tag push

**Key jobs / steps:**

| Name | Location | Responsibility |
|------|----------|----------------|
| `release` (matrix) | `jobs.release` | Runs once per platform; builds binary, generates checksum, uploads both as release assets |
| checkout | step 1 | `actions/checkout@v4` — default depth sufficient; `github.ref_name` supplies version, no `git describe` needed |
| setup-go | step 2 | `actions/setup-go@v5` with `go-version-file: go.mod` — pins Go version from module |
| build | step 3 | Cross-compile with `CGO_ENABLED=0`, inject version via `-ldflags "-X main.version=${{ github.ref_name }}"` |
| rename | step 4 | Move binary to canonical asset name `clinban-{goos}-{goarch}[.exe]` |
| checksum | step 5 | `sha256sum <binary> > <binary>.sha256` — per-binary file, no parallel race |
| upload | step 6 | `softprops/action-gh-release@v2` — upload binary + `.sha256` file, `generate_release_notes: true` |

**Scope note — `--version` vs `-v`:**
The ticket requests "version / -v command." The architecture (ADR-1) chose Cobra's built-in `Version` field, which exposes `--version` only. Adding a `-v` shorthand would require a separate `BoolP` flag that conflicts with Cobra internals and is out of scope for this ticket. `--version` is the canonical implementation; `-v` is explicitly deferred.

**Interface contract:**

- Accepts: git tag push matching `refs/tags/v*`; `github.ref_name` becomes the version string
- Returns:
  - GitHub Release named after the tag
  - 3 binary assets: `clinban-linux-amd64`, `clinban-darwin-arm64`, `clinban-windows-amd64.exe`
  - Per-binary checksum files: `clinban-linux-amd64.sha256`, `clinban-darwin-arm64.sha256`, `clinban-windows-amd64.exe.sha256` (one file per matrix job — avoids parallel-upload race)
  - Version string format: `v<tag>` exactly (e.g. `v0.1.0`) — not `dev`, not `git describe` distance format
- Raises:
  - Build step failure if Go code does not compile for target platform
  - Upload failure if `GITHUB_TOKEN` lacks `contents: write` permission
  - Silent wrong version if `-ldflags` step is omitted (binary reports `"dev"`) — guard via smoke test

**Build matrix:**

| goos | goarch | os runner | suffix |
|------|--------|-----------|--------|
| linux | amd64 | ubuntu-latest | (none) |
| darwin | arm64 | ubuntu-latest | (none) |
| windows | amd64 | ubuntu-latest | `.exe` |

All three run on `ubuntu-latest` — Go cross-compilation is clean with `CGO_ENABLED=0`, no macOS runner needed.

**Version variable lock:**
- Variable: `main.version` at `cmd/clinban/root.go:18`
- Build flag: `-X main.version=${{ github.ref_name }}`
- Local fallback default: `"dev"`

---

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| Git tag push (`v*`) | `release.yml` trigger | GitHub Actions webhook | `github.ref_name` = tag string |
| `release.yml` build step | Go toolchain | `go build` subprocess | `GOOS`, `GOARCH`, `-ldflags` |
| `release.yml` upload step | GitHub Releases API | `softprops/action-gh-release@v2` | binary file path, `checksums.txt`, `generate_release_notes: true` |
| User shell | Downloaded binary | `./clinban --version` | stdout: version string matching tag |

---

## Test Strategy

**Unit tests (per module):**
- No new Go code — existing tests unchanged; `go test ./...` must pass before tagging

**Critical paths (must be tested before first ship):**
1. Push `v*` tag → GitHub Release created with exactly 3 binary assets and 3 per-binary `.sha256` files
2. Downloaded binary `clinban --version` returns exactly `v<tag>` (e.g. `v0.1.0`) — not `"dev"`, not distance format
3. `sha256sum -c clinban-linux-amd64.sha256` passes — file integrity confirmed per binary

**Integration tests (manual smoke):**
- Download all 3 binaries, run `--version` on linux and darwin, verify `.exe` suffix on windows asset
- Confirm release notes are auto-generated (not blank)
- Re-push same tag is a no-op / idempotent (GitHub rejects duplicate; verify graceful failure)

---

## Resolved Architecture Questions

| Question (from 02_architecture.md) | Decision | Rationale |
|------------------------------------|----------|-----------|
| SHA256 checksum file alongside binaries? | Yes — `sha256sum` step, upload `checksums.txt` | Allows users to verify download integrity without extra tooling |
| Binary naming convention (dash vs underscore)? | Dash: `clinban-linux-amd64` | Consistent with most Go project conventions; readable |
| Windows binary suffix (`.exe`)? | Yes — matrix `suffix` variable set to `.exe` for windows/amd64 | Required by Windows; conditional via matrix field |
| Platforms to target? | linux/amd64, darwin/arm64, windows/amd64 (3 targets) | Covers primary user bases; no darwin/amd64 or linux/arm64 in initial scope |
| macOS runner vs ubuntu cross-compile for darwin? | ubuntu-latest for all 3 | CGO disabled — cross-compile is clean; ubuntu runners are cheaper and faster |
