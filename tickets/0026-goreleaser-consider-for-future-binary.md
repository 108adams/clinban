---
title: 'GoReleaser: consider for future binary distribution'
status: blocked
type: task
tags: []
created: 2026-06-14T11:43:28.948028144+02:00
updated: 2026-06-14T11:43:35.284988942+02:00
---
GoReleaser (goreleaser.com) is the standard Go tool for multi-platform binary releases. Handles: cross-compilation matrix, archive naming conventions, SHA checksums, GitHub Release asset upload, optional Homebrew tap generation, optional changelog injection.

Currently replaced by a plain GitHub Actions release workflow (see ticket 0020 architecture). Consider GoReleaser if distribution requirements grow: multiple package registries, automated Homebrew formula, or richer changelog automation.

Blocked until plain Actions approach proves insufficient.