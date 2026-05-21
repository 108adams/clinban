---
id: "0008"
status: done
type: task
title: code review
tags: []
created: 2026-05-21T10:33:57+02:00
updated: 2026-05-21T10:36:59.579999895+02:00
---
Code review input for 

0d745e6 2026-05-21T10:15:19+02:00 feat(template): add inline type hint comment to new-ticket template (ticket 0003)
88f3bf4 2026-05-21T10:14:13+02:00 fix(init): show missing artifacts alongside existing ones on partial init
1e27d08 2026-05-21T09:54:08+02:00 feat(new): pre-fill type from default_type config setting (ticket 0002)
9aa4134 2026-05-21T09:30:54+02:00 feat(init): emit SCHEMA.md as fourth init artifact for LLM agents

  1. cmd/clinban/schema.md:157 tells agents to run clinban new --title ... --type ..., but new only consumes those flags when --no-interactive is set; otherwise it opens $EDITOR (cmd/clinban/new.go:36). This will make scripted/agent creation hang or behave interactively from a
     freshly initialized project.
  2. cmd/clinban/schema.md:170 documents clinban edit <id> --title/--type/--tags, but edit has no such flags and only opens $EDITOR (cmd/clinban/edit.go:20). The generated SCHEMA.md is now an init artifact, so this bad instruction gets copied into every new project.
  3. cmd/clinban/schema.md:123 says only five transitions are valid and omits backlog -> blocked. The actual FSM allows it (internal/fsm/fsm.go:12) and the maintained validation doc lists it (docs/validation.md:51). Agents following SCHEMA.md may reject a legal move.
  4. The default_type feature is implemented in config and new (internal/config/config.go:25, cmd/clinban/new.go:198), but the maintained docs still omit it. docs/configuration.md:19 only shows tickets_dir and archive_dir, and docs/cli.md:53 still says --type is required. This
     violates the repo rule that behavior changes keep the wiki aligned.
  5. cmd/clinban/new.go:71 passes raw cfg.DefaultType into the interactive template, and internal/template/new.md:4 renders it directly inside YAML quotes. Non-interactive mode validates the default first, but interactive mode does not; an invalid or quote-containing config value can
     prefill an invalid template and push the user into parse/lint recovery. Prefer passing the default only when ticket.Type(cfg.DefaultType).Valid().
