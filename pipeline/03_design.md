# Implementation Design
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: ticket 0002 — default configurable type_

## Scope note

Small, targeted feature. Three packages touched in a linear dependency chain:
`internal/config` → `internal/template` → `cmd/clinban/new.go`. No new packages.

---

## Module Structure

### internal/config — config.go

**Files:**
- `internal/config/config.go` — add `DefaultType string` field; parse `default_type` from TOML

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `Config` | struct | Add `DefaultType string \`toml:"default_type"\`` field |
| `Load` | `(projectRoot string) → (*Config, error)` | Parse `default_type` from raw TOML into `cfg.DefaultType`; no validation — keep config loading pure |

**Interface contract:**
- `Config.DefaultType` is the raw string from `.clinban`; empty string when unset
- No validation at load time; callers validate against `ticket.Type.Valid()`
- `raw` struct in `Load` gains `DefaultType string \`toml:"default_type"\``

---

### internal/template — template.go + new.md

**Files:**
- `internal/template/template.go` — add `Type string` to `templateData`; add `defaultType string` parameter to `New`
- `internal/template/new.md` — change `type: ""` to `type: "{{.Type}}"`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `templateData` | struct | Add `Type string` field |
| `New` | `(id int, now time.Time, defaultType string) → ([]byte, error)` | Pass `defaultType` as `templateData.Type`; no validation — caller is responsible |

**Interface contract:**
- `New` accepts any `defaultType` string and renders it verbatim into the template
- Empty string renders as `type: ""` — same behaviour as before for callers that pass `""`
- Breaking change: all call sites must add the third argument

---

### cmd/clinban — new.go

**Files:**
- `cmd/clinban/new.go` — two sites updated

**Changes:**

1. `runNewInteractive`: pass `cfg.DefaultType` as third arg to `template.New`

2. `runNewNonInteractive`: after the `flags.ticketType == ""` check, fall back to
   `cfg.DefaultType` when it holds a valid type:
   ```go
   if flags.ticketType == "" {
       if cfg.DefaultType != "" && ticket.Type(cfg.DefaultType).Valid() {
           flags.ticketType = cfg.DefaultType
       } else {
           fmt.Fprintln(os.Stderr, "error: --type is required")
           os.Exit(1)
       }
   }
   ```
   The existing `tt.Valid()` check below remains unchanged — it guards the
   user-supplied flag value; the default is already validated above.

---

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| `cmd/clinban/root.go` PersistentPreRun | `config.Load` | function call | projectRoot string |
| `cmd/clinban/new.go runNewInteractive` | `template.New` | function call | `cfg.DefaultType` as third arg |
| `cmd/clinban/new.go runNewNonInteractive` | `ticket.Type.Valid()` | method call | `cfg.DefaultType` validity check |

---

## Test Strategy

**Unit (per module):**
- `internal/config`: add test case for `.clinban` with `default_type = "feature"`; assert `cfg.DefaultType == "feature"`. Test absent field → `cfg.DefaultType == ""`.
- `internal/template`: update `TestNewReturnsParseableTicket` and `TestNewContainsIDAndTimestamp` to pass `""` as third arg (regression). Add `TestNewWithDefaultType`: pass `"bug"` → rendered bytes contain `type: "bug"`.
- `cmd/clinban`: subprocess tests for `--no-interactive` without `--type` but with `default_type = "task"` in `.clinban` → ticket created with `type: task`. Also test missing `--type` with no `default_type` → exits 1.

**Critical paths:**
1. `config.Load` with `default_type = "spike"` in `.clinban` → `cfg.DefaultType == "spike"`
2. `template.New(1, now, "feature")` → rendered bytes contain `type: "feature"`
3. `clinban new --no-interactive --title "X"` with `default_type = "task"` in config → creates ticket with `type: task`

---

## Resolved Design Questions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Validate `default_type` at config load? | No — load raw, validate at use | Keeps config loading pure; invalid value is simply ignored (falls through to the "type required" error) |
| Make `--type` optional for `--no-interactive` when default set? | Yes | "Pre-filled" applies to both paths; non-interactive should be unblocked by a valid default |
| Template signature: new param vs options struct? | New param (breaking) | No other callers in the codebase; options struct is premature |
