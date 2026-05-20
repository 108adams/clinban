---
title: Documentation Log
kind: log
scope: docs
summary: Records chronological maintenance activity for the Clinban documentation wiki.
updated: 2026-05-20
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

## [2026-05-20] update | GitHub migration — module path and CI

- Source: migration commit `6a0cd17`
- Updated: `docs/development.md`, `docs/log.md`
- Notes: Added CI section documenting GitHub Actions workflow; no other wiki pages had stale GitLab or old module path references.
