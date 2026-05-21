---
title: Documentation Log
kind: log
scope: docs
summary: Records chronological maintenance activity for the Clinban documentation wiki.
updated: 2026-05-21
links:
  - index
  - schema
  - product
  - architecture
---

# Documentation Log

## [2026-05-19] scaffold | Wiki core

- Source: `wiki.md`, user discussion
- Updated: `docs/schema.md`, `docs/index.md`, `docs/log.md`
- Notes: Created the lightweight documentation wiki foundation and frontmatter conventions.

## [2026-05-19] ingest | Product overview

- Source: `pipeline/00_vision.md`
- Updated: `docs/product.md`, `docs/index.md`, `docs/log.md`, `docs/documentation.md`
- Notes: Distilled product-level vision into current wiki form and retired the imported source file.

## [2026-05-19] ingest | Pipeline reference migration

- Source: `pipeline/01_requirements.md`, `pipeline/02_architecture.md`, `pipeline/03_design.md`, `pipeline/04_tasks.md`, `pipeline/05_review.md`
- Updated: `docs/cli.md`, `docs/ticket-schema.md`, `docs/configuration.md`, `docs/validation.md`, `docs/storage.md`, `docs/security.md`, `docs/architecture.md`, `docs/development.md`, `docs/adr/0001-cli-framework.md`, `docs/adr/0002-package-decomposition.md`, `docs/adr/0003-atomic-file-writes.md`, `docs/index.md`, `docs/log.md`
- Notes: Distilled remaining pipeline knowledge into current wiki pages and retired the imported pipeline sources.

## [2026-05-19] lint | Wiki health check

- Source: full wiki scan
- Updated: `docs/index.md`, `docs/schema.md`
- Notes: Fixed orphan `development.md` (missing from index); expanded scope examples in schema to match pages in use.

## [2026-05-19] update | clinban init command and default directory layout

- Source: implementation (`cmd/clinban/init.go`, `internal/config/config.go`)
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`
- Notes: Documented `clinban init`; updated default directory layout from project root to `tickets/` and `tickets/archive/`.

## [2026-05-19] update | Shell completion documentation

- Source: Cobra-generated `clinban completion --help`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Documented shell completion generation for bash, zsh, fish, and powershell.

## [2026-05-21] feature | clinban init emits SCHEMA.md (ticket 0001)

- Source: `cmd/clinban/schema.md`, `cmd/clinban/init.go`, `cmd/clinban/init_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: `clinban init` now creates a fourth artifact, `SCHEMA.md`, at the project root. The file is a static Markdown schema reference for LLM agents and human contributors; it covers ticket format, field constraints, status transitions, and step-by-step agent operations.

## [2026-05-21] update | clinban push, new body args, default_type, init improvements

- Source: commits `9aa4134`–`c41d3fb` (tickets 0001–0003, 0005–0006, 0008)
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`, `CHANGELOG`
- Notes: Documented `clinban push`, body-arg behavior for `clinban new`, `default_type` config field, SCHEMA.md init artifact, partial-init missing-items output, and inline type hint in the ticket template.

## [2026-05-21] update | End-to-end testing strategy

- Source: maintainer discussion
- Updated: `docs/development.md`, `docs/log.md`
- Notes: Added E2E testing guidance for black-box CLI scenarios, fake editors, PTY use, subprocess coverage, and when to consider testscript.

## [2026-05-21] fix | Unknown command shows help (ticket 0007)

- Source: `cmd/clinban/root.go`, `cmd/clinban/root_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: `clinban <unknown>` now prints an "unknown command" error to stderr, displays root help to stdout, and exits 1 instead of silently failing.

## [2026-05-21] fix | Interactive new waits for GUI editors

- Source: `internal/editor/editor.go`, `internal/editor/editor_test.go`, `cmd/clinban/new_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Documented editor command arguments and automatic wait flags for common GUI editors so interactive `new` and `edit` do not read unchanged temp files before save.

## [2026-05-21] fix | Interactive new validates before writing

- Source: `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Empty titles and other lint failures now trigger the re-open prompt before a managed ticket file is created; declining the prompt exits 1 and leaves no invalid ticket behind.

## [2026-05-21] feature | clinban config command

- Source: `internal/config/config.go`, `cmd/clinban/config.go`, `cmd/clinban/config_test.go`
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`
- Notes: Added `clinban config` subcommand. No-args mode lists all three known keys with values, defaults, and set/not-set notes. Single `key=value` arg sets the key in `.clinban`, creating the file if absent, with validation for unknown keys, invalid `default_type` values, and empty path values.

## [2026-05-20] update | GitHub migration — module path and CI

- Source: migration commit `6a0cd17`
- Updated: `docs/development.md`, `docs/log.md`
- Notes: Added CI section documenting GitHub Actions workflow; no other wiki pages had stale GitLab or old module path references.
