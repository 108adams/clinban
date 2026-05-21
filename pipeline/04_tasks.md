# Developer Tasks
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Add DefaultType to internal/config

- **Description:** Add `DefaultType string` to `Config` and parse `default_type` from the `.clinban` TOML file.
- **Module(s):** `internal/config/config.go`, `internal/config/config_test.go`
- **Done criteria:**
  - [ ] `Config` struct has `DefaultType string \`toml:"default_type"\`` field
  - [ ] `raw` struct in `Load` has `DefaultType string \`toml:"default_type"\`` field
  - [ ] `cfg.DefaultType` is set from `raw.DefaultType` (no validation)
  - [ ] Existing config tests still pass
  - [ ] New test: `.clinban` with `default_type = "feature"` → `cfg.DefaultType == "feature"`
  - [ ] New test: `.clinban` without `default_type` → `cfg.DefaultType == ""`
  - [ ] `go test ./internal/config/...` passes
  - [ ] `go vet ./internal/config/...` clean
- **Depends on:** none

---

### TASK-002: Update internal/template to accept and render default type

- **Description:** Add `Type string` to `templateData`, add `defaultType string` as third param to `New`, and update `new.md` to render the type field.
- **Module(s):** `internal/template/template.go`, `internal/template/new.md`, `internal/template/template_test.go`
- **Done criteria:**
  - [ ] `templateData` has `Type string` field
  - [ ] `New` signature is `(id int, now time.Time, defaultType string) ([]byte, error)`
  - [ ] `new.md` has `type: "{{.Type}}"` (not `type: ""`)
  - [ ] `TestNewReturnsParseableTicket` updated to pass `""` as third arg; still passes
  - [ ] `TestNewContainsIDAndTimestamp` updated to pass `""` as third arg; still passes
  - [ ] New test `TestNewWithDefaultType`: `New(1, now, "bug")` → bytes contain `type: "bug"`
  - [ ] `go test ./internal/template/...` passes
  - [ ] `go vet ./internal/template/...` clean
- **Depends on:** none (config and template are independent)
- **Notes:** `new.md` change: `type: ""` → `type: "{{.Type}}"`. When `defaultType` is `""` the rendered output is `type: ""` — identical to current behaviour.

---

### TASK-003: Wire default type into cmd/clinban/new.go

- **Description:** Pass `cfg.DefaultType` to `template.New` in the interactive path, and use it as a fallback for `--type` in the non-interactive path.
- **Module(s):** `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- **Done criteria:**
  - [ ] `runNewInteractive`: `template.New(nextID, now, cfg.DefaultType)` (third arg added)
  - [ ] `runNewNonInteractive`: if `flags.ticketType == ""` and `cfg.DefaultType` is a valid type, set `flags.ticketType = cfg.DefaultType`; otherwise keep existing "error: --type is required" path
  - [ ] New subprocess test: write `.clinban` with `default_type = "task"`, run `clinban new --no-interactive --title "X"` without `--type` → exits 0, ticket file contains `type: task`
  - [ ] New subprocess test: no `default_type` in config, run `clinban new --no-interactive --title "X"` without `--type` → exits 1, stderr contains "required"
  - [ ] New subprocess test: `default_type = "notavalidtype"` in config, run without `--type` → exits 1
  - [ ] Existing `new` tests still pass
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** TASK-001, TASK-002

---

## Dependency Order

```
TASK-001 (internal/config)  ──┐
                               ├──► TASK-003 (cmd/clinban/new.go)
TASK-002 (internal/template) ──┘
```

TASK-001 and TASK-002 are independent and can be done in either order.
TASK-003 depends on both.
