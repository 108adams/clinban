---
title: Product Overview
kind: overview
scope: project
summary: Describes Clinban's product purpose, users, boundaries, and ecosystem role.
updated: 2026-05-21
links:
  - cli
  - ticket-schema
  - validation
  - architecture
---

# Product Overview

Clinban is a terminal-native kanban system for software projects that keep work tracking close to the code. A ticket is a Markdown file with YAML frontmatter, stored in the repository alongside the implementation it describes.

The product is designed around a simple premise: tickets should be readable, writable, versionable files. Humans can manage them through a CLI, while automata can read and write the same schema directly.

## Users

Clinban serves two first-class user groups.

**Human developers** use the CLI to create, view, list, edit, transition, archive, register, and lint tickets.

**Automata** include AI coding agents, CI/CD pipelines, scripts, and test infrastructure. They use the ticket file schema as the contract. They are not constrained by CLI state-machine enforcement, so lint is the integrity layer for machine-written files.

## Product Boundaries

Clinban intentionally avoids external system complexity:

- No external service.
- No network dependency.
- No authentication or multi-user access control.
- No web UI.
- No time tracking, estimates, burndown, or velocity metrics.
- No built-in git integration beyond storing files in a repository.
- One board per repository.

## Core Capabilities

The stable product surface is:

- YAML-frontmatter ticket schema.
- CRUD-oriented CLI commands.
- State-machine enforcement for CLI moves.
- Active and archived ticket directories.
- Linting for schema integrity.
- Search and filtering through list flags.
- A Claude Code `/tickets` skill for LLM-native board operations.

## LLM Interface

Clinban ships a Claude Code skill at `.claude/skills/tickets/SKILL.md`. It is a co-deliverable with the CLI: when CLI behavior changes, the skill is updated in the same commit.

The skill provides the LLM-native board interface — ticket discovery, creation, status transitions, body editing, and the close-after-commit sequence — on top of `SCHEMA.md`, which it delegates to for field-level detail. Invoke it with `/tickets` in Claude Code.

## Ecosystem Role

Clinban is one layer in a Markdown-native project management stack:

```text
Knowledge base (Markdown wiki)
Architecture decisions (Markdown ADRs)
Work registry (Clinban tickets)
Code (git repository)
```

The long-term value comes from shared plain-text conventions. Humans, LLM agents, scripts, and CI jobs can all inspect and update the same artifacts without an external issue tracker.
