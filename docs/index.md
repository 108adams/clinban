---
title: Documentation Index
kind: overview
scope: docs
summary: Maps the maintained Clinban documentation wiki.
updated: 2026-05-19
links:
  - schema
  - log
  - product
  - cli
  - ticket-schema
  - architecture
  - development
---

# Documentation Index

This index is the entry point for the maintained Clinban wiki.

## Wiki Operations

- [Documentation Schema](schema.md) - Defines page frontmatter, wiki structure, navigation, logging, and source-retirement rules.
- [Documentation Log](log.md) - Chronological record of documentation scaffold, ingests, updates, and lint passes.
- [Documentation Workflow](documentation.md) - Explains how Go package docs and Markdown project docs are maintained.

## Project Knowledge

- [Product Overview](product.md) - Describes Clinban's product purpose, users, boundaries, and ecosystem role.

## Reference

- [CLI Reference](cli.md) - Documents commands, expected outputs, and exit-code conventions.
- [Ticket Schema](ticket-schema.md) - Defines ticket files, YAML frontmatter fields, filename convention, and ownership rules.
- [Configuration](configuration.md) - Describes `.clinban`, root discovery, and ticket directory defaults.
- [Validation](validation.md) - Explains parse errors, lint rules, and workflow transition enforcement.
- [Storage](storage.md) - Describes filesystem layout, ticket discovery, ID scanning, writes, and archiving.
- [Security Model](security.md) - Captures the local trust model and filesystem safety assumptions.

## Architecture

- [Architecture](architecture.md) - Describes package boundaries, dependency rules, and major design responsibilities.
- [ADR 0001: CLI Framework](adr/0001-cli-framework.md) - Records the decision to use Cobra.
- [ADR 0002: Package Decomposition](adr/0002-package-decomposition.md) - Records the package separation model.
- [ADR 0003: Atomic File Writes](adr/0003-atomic-file-writes.md) - Records the write-temp-then-rename strategy.

## Development

- [Development](development.md) - Build, test, vet, and documentation commands; test strategy overview.
