# Implementation Design
_Produced by: techlead-agent_
_Date: 2026-05-22_
_Status: draft_
_Input: tech lead design session — ticket 0017 (`#` as title separator)_

## Scope

Two items:

1. **Bug (documentation):** Unquoted `#` in bash/zsh is a comment character; the shell strips everything from the first whitespace-preceded `#` before Go receives argv. Fix is a help-text note — no code change to argument handling.
2. **Feature:** When `cfg.SplitRawNew` is true (default), split the joined positional args string on the first `#` and pre-fill the frontmatter `title` field from the left-hand side, leaving the right-hand side as the body text.

---

## Module Structure

### cmd/clinban/new.go

**Files:**
- `cmd/clinban/new.go` — `new` command wiring, interactive flow, and the `splitRawBody` pure function

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `splitRawBody` | `(raw string) (title, body string)` | Split joined args string on first `#`; return left side as title and right side as body; both trimmed of whitespace |
| `runNewInteractive` | `(raw string) error` | Interactive ticket-creation path; consults `cfg.SplitRawNew` and calls `splitRawBody` when enabled |
| `newCmd.Long` | — | Help text for the `new` command; must document the `\#` escaping requirement |

**Interface contract — `splitRawBody`:**
- Accepts: any string (the result of `strings.Join(args, " ")`)
- Returns:
  - `("", "")` when `raw == ""`
  - `("", raw)` when no `#` is present
  - `(left, right)` split on the first `#` only; both sides trimmed
  - `("", right)` when `#` is the first non-space character
  - `(left, "")` when `#` is the last character
- Errors: none (pure function, no error return)

**Interface contract — `runNewInteractive`:**
- Accepts: raw string (joined positional args); reads `cfg.SplitRawNew` from package-level `cfg`
- Behaviour change: when `cfg.SplitRawNew` is true and `raw` is non-empty, calls `splitRawBody(raw)` and passes the resulting `title` to `template.New`; passes `body` as the file-append body text (existing behaviour)
- When `cfg.SplitRawNew` is false, passes empty string as title to `template.New` and the full `raw` string as body (identical to pre-feature behaviour)

---

### internal/config/config.go

**Files:**
- `internal/config/config.go` — `Config` struct, `Load`, `SetKey`, `Entries`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `Config.SplitRawNew` | `bool` TOML `split_raw_new` | Enables `#`-based title/body split in the `new` command; default `true` |
| `Load` | `(projectRoot string) (*Config, error)` | Parse `.clinban`; apply default `true` for `SplitRawNew` when key is absent |
| `SetKey` | `(root, key, value string) error` | Accept `split_raw_new` with values `"true"` / `"false"`; reject all other values with `ErrInvalidValue` |
| `Entries` | `(root string) ([]Entry, error)` | Include `split_raw_new` entry with default shown as `"true"` |

**Interface contract — `Config.SplitRawNew`:**
- TOML key: `split_raw_new`
- Default: `true` (applied by `Load` when key is absent from `.clinban`)
- Valid values via `SetKey`: `"true"`, `"false"` — any other value returns `ErrInvalidValue`
- `Entries` default string: `"true"`

**Note on `Load`:** The existing `Load` function uses an anonymous inline struct for raw parsing. That struct must gain a `SplitRawNew` field (TOML `split_raw_new`) so the value can be read. Because TOML bool zero-value is `false`, the default-`true` requirement means the code must detect absence explicitly. The simplest approach: read the raw value as a `*bool` (pointer) so nil means "not set", then default to `true` when nil. Alternatively keep `bool` and track presence with a separate flag, mirroring the existing `ticketsDirSet` pattern already used in `Entries`.

---

### internal/template/template.go + new.md

**Files:**
- `internal/template/template.go` — renders the new-ticket template
- `internal/template/new.md` — embedded Go template for ticket frontmatter

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `templateData` | struct | Data bag passed to the template; gains `Title string` field |
| `New` | `(now time.Time, defaultType, title string) ([]byte, error)` | New third param; populates `templateData.Title`; empty string preserves current output (title field renders as `""`) |

**Interface contract — `New`:**
- Accepts: `now` (any time), `defaultType` (validated upstream; empty is allowed), `title` (pre-fill value or empty string)
- Returns: rendered template bytes, or error wrapping a parse/execute failure
- Callers must pass `""` for `title` when no pre-fill is needed — this is the backwards-compatible case
- `new.md` template change: `title: ""` becomes `title: "{{.Title}}"` — when `Title` is `""` the rendered output is `title: ""` (identical to current)

**Interface contract — `new.md`:**
- `{{.Title}}` interpolated into the title field value; Go `text/template` HTML-escapes nothing here (raw string is fine)

---

### cmd/clinban/config.go

**Files:**
- `cmd/clinban/config.go` — `config` command help text and dispatch

**Change:** Add `split_raw_new` to the `Known keys:` section of `configCmd.Long`. No logic change.

---

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| `cmd/clinban/new.go` (`runNewInteractive`) | `internal/config` (`cfg.SplitRawNew`) | struct field read | `bool` |
| `cmd/clinban/new.go` (`runNewInteractive`) | `splitRawBody` | function call | `raw string` → `(title, body string)` |
| `cmd/clinban/new.go` (`runNewInteractive`) | `internal/template.New` | function call | `(now, defType, title string)` → `([]byte, error)` |
| `internal/template.New` | `internal/template/new.md` | `text/template.Execute` | `templateData{Now, Type, Title}` |
| `cmd/clinban/config.go` | `internal/config.SetKey` | function call | `(root, "split_raw_new", "true"/"false")` |
| `cmd/clinban/config.go` | `internal/config.Entries` | function call | returns `[]Entry` including `split_raw_new` |

---

## Test Strategy

**Unit tests (per module):**

- `cmd/clinban/new_test.go`: test `splitRawBody` for all specified edge cases — empty input, no `#`, `#` at start, `#` at end, single `#`, multiple `#` chars (only first splits)
- `internal/config/config_test.go`: test `Load` default for `SplitRawNew` (absent key → `true`); test `SetKey` accepts `"true"` and `"false"`, rejects other values; test `Entries` lists `split_raw_new` with default `"true"`
- `internal/template/template_test.go` (if it exists) or new: test `New` with non-empty `title` produces correct frontmatter line; test `New` with `title=""` produces `title: ""` (unchanged output)

**Critical paths (must be tested before first ship):**

1. **Split on first `#` only:** `splitRawBody("title # body with # more")` returns `("title", "body with # more")` — any subsequent `#` stays in body
2. **No `#` leaves current behaviour unchanged:** `splitRawBody("just body text")` returns `("", "just body text")`; the full raw string becomes body, title field renders as `""` in template
3. **`split_raw_new=false` disables splitting:** when `cfg.SplitRawNew` is false, `runNewInteractive` passes `""` as title to `template.New` and the full raw string as body, regardless of whether `#` is present

**Integration tests:**

- `runNewInteractiveWithArgs` (or equivalent test helper) with args `["title", "#", "body text"]` and `SplitRawNew=true`: verify the created temp file contains `title: "title"` in frontmatter and `"body text"` in body
- Same args with `SplitRawNew=false`: verify `title: ""` in frontmatter and full raw string `"title # body text"` in body

---

## Resolved Architecture Questions

| Question (from design session) | Decision | Rationale |
|-------------------------------|----------|-----------|
| Should the `#` shell escaping issue be fixed in code or docs? | Documentation only — update `newCmd.Long` help text | Shell comment-stripping happens before Go receives argv; there is nothing for Go to intercept. Users must escape `\#` or quote the full string. |
| Default for `split_raw_new`? | `true` | New behaviour is strictly additive and always-on by default; users who want the old behaviour opt out explicitly. |
| How to detect TOML bool absence (default-`true` problem)? | Use separate presence-tracking bool mirroring the existing `ticketsDirSet` pattern in `Entries`; in `Load` use an anonymous struct with a `*bool` field so nil means unset, default `true` | Consistent with the codebase's existing approach; avoids adding a third-party dependency. |
| Should `splitRawBody` be exported? | No (`splitRawBody`, lowercase) | It is an implementation detail of the `new` command; testing via package-internal test file (`new_test.go` in `package main`) is sufficient. |
| Does `template.New` signature change break `runNewNonInteractive`? | Yes — caller must be updated to pass `""` as the new third argument | `runNewNonInteractive` does not use `template.New` (it builds a `ticket.Ticket` struct directly), so no change is needed there. Only `runNewInteractive` calls `template.New`. |
