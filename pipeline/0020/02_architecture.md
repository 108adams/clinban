# Architecture: Version Command and Binary Distribution (0020)

_Created: 2026-06-14_

## Existing Components (verified)

| Component | File:line | Responsibility |
|-----------|-----------|----------------|
| Root command | `cmd/clinban/root.go:27` | Cobra root; `.Version` field added — surfaces `--version` flag |
| `version` variable | `cmd/clinban/root.go:18` | Package-level var, defaults to `"dev"`, injected by `-ldflags` at build time |
| Makefile build target | `Makefile:17` | `go build` with `-ldflags "-X main.version=$(git describe ...)"` |
| CI workflow | `.github/workflows/ci.yml:1` | Test-only; no release step |

## Proposed Changes

| Change | Replaces/extends | Rationale |
|--------|-----------------|-----------|
| Add GitHub Actions release workflow | Nothing (new file) | Trigger on `v*` tag push; build matrix; upload assets to GitHub Release |
| Tag discipline (SemVer `v*`) | Ad-hoc tags | Release workflow key; `git describe` output format depends on tag format |

> `--version` flag and ldflags injection are **already implemented** as part of this session. Remaining work is the release workflow only.

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|------------|---------|--------|-------------|-------|
| GitHub Releases API | HTTPS (via `gh` / Actions `softprops/action-gh-release`) | Binary assets + SHA256 checksums | Upload fails → job fails, release draft left incomplete; re-run safe | CI/CD |
| `git describe` | Local CLI | `v<tag>-<n>-g<hash>[-dirty]` | No tags → bare hash; acceptable fallback | Developer |

## NFRs

| Category | Requirement | Target | Status |
|----------|------------|--------|--------|
| Operability | Version visible without running a command | `--version` flag (Cobra built-in) | Met |
| Operability | Verify installed vs latest | Compare `clinban --version` against GitHub Releases latest tag | Met via release workflow |
| Security | No secrets in version string | Only git metadata | Met |
| Scalability | Adding platforms | Extend matrix in release workflow | Open question |
| Observability | Build provenance | `git describe` embeds tag + commit hash | Met |
| Distribution | Platforms | linux/amd64, darwin/arm64, windows/amd64 | Defined |

## ADRs

### ADR-1: Version injection via `-ldflags`

**Status:** `accepted`
**Decision:** Inject version string at build time using `go build -ldflags "-X main.version=<value>"` where value comes from `git describe --tags --always --dirty`.
**Context:** The binary must report its own version so users can compare against the latest GitHub Release. Go offers two mechanisms: ldflags injection and `runtime/debug.ReadBuildInfo()`. The user's primary goal is comparing a recognizable version label against a GitHub Release tag.
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| `runtime/debug.ReadBuildInfo()` VCS stamping | Returns commit hash only; `Main.Version` is `(devel)` for all local builds — not comparable to a semver release tag without extra tooling |
| Embedded `VERSION` file | Must be manually updated; drift risk |

**Rationale:** ldflags gives a clean semver string (`v0.1.0`) on a tagged build and a meaningful fallback (`v0.1.0-51-ge68a3e0`) otherwise. Local `go build` (no ldflags) falls back to `"dev"` — unambiguous non-release signal.
**Consequences:**
- `+` Human-readable version on every release binary
- `+` `git describe` output includes tag distance and hash — full provenance in one string
- `-` Plain `go build` (without Makefile) produces `"dev"` — developers must use `make build` for a version-stamped local binary
- `!` Release workflow must pass the ldflags explicitly; missing it silently ships `"dev"`

**Locks:** Release workflow (ADR-2) must pass `-ldflags "-X main.version=..."` at build time. The version variable name is `main.version` (`cmd/clinban/root.go:18`).

---

### ADR-2: Plain GitHub Actions release workflow (no GoReleaser)

**Status:** `accepted`
**Decision:** Use a dedicated GitHub Actions workflow triggered on `push: tags: v*` to build, cross-compile, and upload binaries as GitHub Release assets.
**Context:** Binary distribution requires building for linux/amd64, darwin/arm64, and windows/amd64, then making artifacts downloadable from GitHub. GoReleaser is the standard tool but adds a dependency and config surface area. The project currently has a single test workflow.
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| GoReleaser | Adds tool dependency and config file; current distribution scope doesn't justify it — captured as memo ticket 0026 for future consideration |
| Manual local cross-compilation + upload | Not reproducible; not automated |

**Rationale:** A plain Actions matrix (3 targets, `GOOS`/`GOARCH` env vars, `softprops/action-gh-release` or `gh release upload`) is auditable, zero-dependency, and sufficient for current scope. GoReleaser can replace it later without changing any Go code.
**Consequences:**
- `+` No new tool dependency
- `+` Release pipeline is a single YAML file, fully auditable
- `-` Checksums and archive packaging must be added manually if needed later
- `!` Adding platforms requires editing the workflow matrix (acceptable)

**Locks:** Tag format must be `v*` (e.g. `v0.1.0`) for the workflow trigger and for `git describe` to produce clean semver output. Irregular tag names will break both.

## Open Questions

| Question | Owner | Blocking? |
|----------|-------|-----------|
| SHA256 checksum file alongside binaries? | Tech Lead | No — add to release workflow if desired |
| Binary naming convention (`clinban-linux-amd64` vs `clinban_linux_amd64`)? | Tech Lead | No |
| Windows binary suffix (`.exe`)? | Tech Lead | Yes — `GOOS=windows` requires it; Tech Lead must handle in build matrix |
| `install` script / `curl \| sh` convenience? | Deferred | No |
