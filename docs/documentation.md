---
title: Documentation Workflow
kind: workflow
scope: docs
summary: Explains how Go package docs and Markdown project docs are maintained.
updated: 2026-05-19
links:
  - schema
  - index
  - log
---

# Documentation

Clinban uses two complementary documentation layers:

1. Go package documentation generated from doc comments.
2. Markdown project documentation for workflows, architecture, and operating notes.

## Go Package Documentation

Package and exported-symbol documentation lives next to the Go code. Each
package has a `doc.go` file that describes its role and boundaries. Exported
types, functions, variables, constants, and fields are documented where they are
declared.

View local reference docs with:

```bash
go doc ./...
```

For a browsable local pkgsite, install and run:

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest
pkgsite
```

Then open the local pkgsite URL printed by the command.

## Writing Style

Go documentation should describe contracts, not restate implementation.

Good comments explain:

- What a package owns.
- What an exported symbol represents.
- Which side effects a function has.
- Which invariants a caller may rely on.
- Which responsibilities are deliberately handled by another package.

Avoid comments that merely repeat a function name. Prefer:

```go
// WriteTicket serialises t and writes it to path using a same-directory
// temporary file followed by rename.
```

over:

```go
// WriteTicket writes a ticket.
```

## Markdown Project Docs

Use Markdown under `docs/` for material that is larger than API reference:

- CLI usage and examples.
- Ticket schema guide for humans and automata.
- Architecture and package boundaries.
- Security model and filesystem assumptions.
- ADRs and maintenance workflows.

Historical planning documents are source material, not maintained reference
documentation. Long-lived operational knowledge belongs in `docs/`.
