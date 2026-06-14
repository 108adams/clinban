# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-06-14_
_Status: draft_
_Input: pipeline/0020/03_design.md_

## Task List

### TASK-001: Write `.github/workflows/release.yml`

- **Description:** Create the GitHub Actions release workflow. Trigger on `push: tags: v*`. Single job `release` with a 3-entry platform matrix (linux/amd64, darwin/arm64, windows/amd64 — all on `ubuntu-latest`). Steps: checkout (default depth — `github.ref_name` provides version, no `git describe` needed), setup-go (version from `go.mod`), cross-compile with `CGO_ENABLED=0` and `-ldflags "-X main.version=${{ github.ref_name }}"`, rename binary to `clinban-{goos}-{goarch}[.exe]`, generate per-binary checksum via `sha256sum <binary> > <binary>.sha256` (avoids parallel-upload race), upload binary + `.sha256` file via `softprops/action-gh-release@v2` with `generate_release_notes: true`. Set `permissions: contents: write` at workflow level. Note: `--version` flag (Cobra built-in) is the implementation of the ticket's "version / -v command" requirement — `-v` shorthand is explicitly out of scope.
- **Module(s):** `.github/workflows/release.yml` (new file)
- **Done criteria:**
  - [ ] File exists at `.github/workflows/release.yml`
  - [ ] YAML is valid (passes `yamllint` or equivalent)
  - [ ] Trigger is `push: tags: ['v*']`
  - [ ] `permissions: contents: write` present
  - [ ] Matrix contains exactly 3 entries covering linux/amd64, darwin/arm64, windows/amd64
  - [ ] Build step includes `CGO_ENABLED=0`, `GOOS`, `GOARCH`, and `-ldflags "-X main.version=${{ github.ref_name }}"`
  - [ ] Rename step produces asset name matching `clinban-{goos}-{goarch}[.exe]` convention
  - [ ] Checksum step produces `<binary>.sha256` (one file per matrix job, not shared `checksums.txt`)
  - [ ] Upload step uses `softprops/action-gh-release@v2` with `generate_release_notes: true`, uploads binary + its `.sha256` file
  - [ ] Windows matrix entry has `suffix: ".exe"`; linux and darwin entries have `suffix: ""`
  - [ ] `clinban --version` on a release binary prints exactly `v<tag>` (verified in TASK-002; design must make it possible)
- **Depends on:** none
- **Notes:** Version variable is `main.version` in package `main` (`cmd/clinban/root.go:18`). `github.ref_name` on a tag push equals the tag string (e.g. `v0.1.0`) — no stripping needed. Per-binary `.sha256` files avoid the parallel-upload race that a shared `checksums.txt` would create. Each release asset: `clinban-linux-amd64`, `clinban-linux-amd64.sha256`, `clinban-darwin-arm64`, `clinban-darwin-arm64.sha256`, `clinban-windows-amd64.exe`, `clinban-windows-amd64.exe.sha256`.

---

### TASK-002: Smoke test — push tag and verify release

- **Description:** After TASK-001 is merged to main, push a real `v*` tag and manually verify the resulting GitHub Release. This is a manual verification step, not automated.
- **Module(s):** No code changes — verification only
- **Done criteria:**
  - [ ] Tag `v<next-version>` pushed to origin (e.g. `git tag v0.2.0 && git push origin v0.2.0`)
  - [ ] GitHub Release page shows 3 binary assets: `clinban-linux-amd64`, `clinban-darwin-arm64`, `clinban-windows-amd64.exe`
  - [ ] `checksums.txt` present as a release asset (one per platform, or confirmed acceptable)
  - [ ] `clinban-linux-amd64 --version` prints exactly `v<tag>` (e.g. `v0.1.0`) — not `"dev"`, not distance format
  - [ ] `sha256sum -c clinban-linux-amd64.sha256` passes for the downloaded linux binary
  - [ ] Release notes are auto-generated (not blank)
- **Depends on:** TASK-001
- **Notes:** Use a real semver tag — the workflow trigger requires `v*`. If the workflow fails, check `GITHUB_TOKEN` permissions in repo settings (Settings > Actions > General > Workflow permissions must allow read/write). Do not use a draft release tag for this smoke test.

---

---

### TASK-003: Update `docs/cli.md` and `docs/log.md`

- **Description:** Add a `--version` flag section to `docs/cli.md` documenting the flag, its output format (`v<tag>` on release builds, `dev` locally without `make build`), and how to compare against GitHub Releases. Append a change entry to `docs/log.md`. Must land in the same commit as TASK-001.
- **Module(s):** `docs/cli.md`, `docs/log.md`
- **Done criteria:**
  - [ ] `docs/cli.md` has a `--version` section covering: flag name, example output, local vs release behaviour
  - [ ] `docs/log.md` has a new entry for ticket 0020
- **Depends on:** TASK-001 (same commit)

---

## Dependency Order

```
TASK-001: Write release.yml  ──┐
TASK-003: Update docs          ├── same commit
                               │
TASK-002: Smoke test (manual, after TASK-001+003 merged + tag pushed)
```
