---
title: 'GitHub release cycle: tag, release notes, and artefacts'
status: backlog
type: task
tags: [ci, release, github]
created: 2026-06-14T11:53:36.817332345+02:00
updated: 2026-06-14T11:53:36.817332345+02:00
---
Offspring of ticket 0020 (version command + ldflags injection).

## Goal

Establish the standard GitHub-native release cycle for this Go project: tag a commit, trigger a build pipeline, generate release notes, and publish binary artefacts — using only what GitHub provides out of the box.

## Scope

- GitHub Actions release workflow triggered on `v*` tag push
- Cross-compile for linux/amd64, darwin/arm64, windows/amd64 (CGO_ENABLED=0)
- Binary naming: `clinban-{os}-{arch}[.exe]`
- SHA256 checksums file (`checksums.txt`) uploaded alongside binaries
- Release notes: auto-generated from commit history via GitHub's built-in release notes feature
- Tag discipline: SemVer `v*` (e.g. `v0.1.0`)

## Out of scope

- GoReleaser (captured in 0026)
- Install script / curl | sh
- Homebrew tap or package manager integration

## Acceptance criteria

- [ ] `.github/workflows/release.yml` exists and triggers on `push: tags: v*`
- [ ] Build matrix produces 3 binaries per release
- [ ] Each binary reports correct version via `clinban --version` (matches tag)
- [ ] `checksums.txt` uploaded to release
- [ ] Release notes auto-populated (GitHub built-in)
- [ ] Re-running a failed release job is safe (idempotent upload)