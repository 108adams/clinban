# Clinban

Clinban is a small terminal kanban tool for software projects.

It stores every ticket as a Markdown file with YAML frontmatter. There is no server, no database, no web UI, and no external service. Your work items live in the same repository as your code.

## What Problem It Solves

Small software projects often track work in many scattered places:

- notes in Markdown files
- TODO lists
- chat messages
- issue trackers
- prompts for AI coding agents

This creates a problem: the work is not always close to the code, and machines cannot reliably understand the state of the work.

Clinban solves this by making tickets simple files with a stable schema. Humans can read and edit them. Scripts, CI jobs, and LLM agents can also read and write them.

## How It Works

A ticket is a Markdown file like this:

```yaml
---
id: "0042"
status: "in-progress"
type: "bug"
title: "Fix login timeout"
tags: ["auth", "backend"]
created: "2026-05-18T14:30:00Z"
updated: "2026-05-18T15:00:00Z"
---

Notes about the work go here.
```

Active tickets stay in the configured ticket directory. Done tickets can be moved to an archive directory.

The CLI can create, list, show, edit, move, archive, register, and lint tickets.

## Main Ideas

- Tickets are plain Markdown files.
- YAML frontmatter is the schema.
- The filesystem is the storage layer.
- Git can version the tickets, but Clinban does not require git integration.
- Humans and automata use the same ticket format.
- `clinban lint` checks that ticket files follow the schema.

## Technology

Clinban is written in Go.

The project uses:

- Go standard library for filesystem and process work
- Cobra for CLI commands
- YAML and TOML libraries for ticket and config parsing
- Markdown files as the data format

Target platforms are Linux and macOS.

## Who It Is For

Clinban is designed for:

- individual developers
- very small teams
- projects that already live in a code repository
- workflows where LLM agents, scripts, or CI tools need to understand work items

It is intentionally not a large project management system.

## Installation

### Prerequisites

- Go 1.25 or later. Check with:

```bash
go version
```

If Go is not installed, follow the instructions at [go.dev/doc/install](https://go.dev/doc/install).

### Clone and install

```bash
git clone https://gitlab.com/108adams/clinban.git
cd clinban
go install ./cmd/clinban
```

`go install` compiles the binary and places it in your Go binary directory, which is `~/go/bin` by default (or `$GOPATH/bin` if you have `$GOPATH` set).

### Make sure the binary is on your PATH

Check whether the directory is already on your PATH:

```bash
echo $PATH | tr ':' '\n' | grep go
```

If nothing appears, add it. For bash or zsh, add this line to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then reload your shell:

```bash
source ~/.bashrc   # or ~/.zshrc
```

### Verify

```bash
clinban --help
```

You should see the Clinban command list. If the shell says `command not found`, double-check that the output of `go env GOPATH` resolves to a directory that is on your `$PATH`.

## Documentation

Start here:

- [Documentation Index](docs/index.md)
- [CLI Reference](docs/cli.md) - explains how to use each `clinban` command, including creation, listing, editing, moving, archiving, linting, and shell completion.
- [Ticket Schema](docs/ticket-schema.md)
- [Architecture](docs/architecture.md)

The `docs/` directory is maintained as a lightweight project wiki for both humans and LLMs.

## Development

Run tests:

```bash
go test ./...
```

Run vet:

```bash
go vet ./...
```

If the Go build cache is read-only in a sandbox, use:

```bash
GOCACHE=/tmp/go-trello-gocache go test ./...
GOCACHE=/tmp/go-trello-gocache go vet ./...
```

## Created With Human and LLM Collaboration

Clinban was developed by Adam <108adams@gmail.com> together with LLM coding agents.

Claude and Codex worked in sync during the project: planning, reviewing, implementing, documenting, and improving the codebase.

The project itself reflects that collaboration model. Its files are meant to be understandable not only by humans, but also by future LLM agents that may help maintain and extend it.
