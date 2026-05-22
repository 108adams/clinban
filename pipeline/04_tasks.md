# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-05-22_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Document shell `#` escaping in `new` command help text
- **Description:** Edit `newCmd.Long` in `cmd/clinban/new.go` to add a note that an unquoted `#` preceded by whitespace is treated as a shell comment and stripped before Go receives argv. Users must write `\#` or quote the full string (e.g. `clinban new "title # body"`).
- **Module(s):** `cmd/clinban/new.go` (Long field only)
- **Done criteria:**
  - [ ] `clinban new --help` output contains a sentence explaining the shell `#` comment behaviour and shows `\#` or quoted-string as the remedy
  - [ ] No logic changes; `go vet ./...` and `go test ./...` continue to pass
- **Depends on:** none
- **Notes:** This is a pure documentation change. The existing `strings.Join(args, " ")` in `runNew` already produces the correct string once the shell delivers argv intact; no code change is needed.

---

### TASK-002: Add `split_raw_new` to `config` package
- **Description:** Add `SplitRawNew bool` to the `Config` struct with TOML key `split_raw_new` and default `true`. Update `Load` to default to `true` when the key is absent. Update `SetKey` to accept `split_raw_new` with valid values `"true"` and `"false"`. Update `Entries` to include a `split_raw_new` entry with default string `"true"`. Add `split_raw_new` to the Known keys section of `configCmd.Long` in `cmd/clinban/config.go`.
- **Module(s):** `internal/config/config.go`, `cmd/clinban/config.go`
- **Done criteria:**
  - [ ] `Config.SplitRawNew` is `true` when `.clinban` does not contain `split_raw_new`
  - [ ] `Config.SplitRawNew` is `false` after `clinban config split_raw_new=false`
  - [ ] `clinban config` lists `split_raw_new = true   (not set in .clinban, default: true)` when unset
  - [ ] `config.SetKey(root, "split_raw_new", "maybe")` returns `ErrInvalidValue`
  - [ ] `config.SetKey(root, "split_raw_new", "true")` and `"false"` return nil
  - [ ] Unit tests in `internal/config/config_test.go` cover all four bullets above
  - [ ] `go vet ./...` and `go test ./...` pass
- **Depends on:** none
- **Notes:** The TOML bool zero-value is `false`, which conflicts with the default-`true` requirement. In `Load`, use a `*bool` field on the anonymous raw struct so that nil (absent) can be distinguished from explicit `false`. In `Entries`, mirror the existing `ticketsDirSet` tracking pattern. The `SetKey` marshal block must emit `split_raw_new = true/false` unconditionally when the value is set (use `fmt.Sprintf("split_raw_new = %v\n", raw.SplitRawNew)` or equivalent).

---

### TASK-003: Implement `splitRawBody` pure function and unit tests
- **Description:** Add `func splitRawBody(raw string) (title, body string)` to `cmd/clinban/new.go`. The function splits `raw` on the first `#` character: the trimmed left side becomes `title`, the trimmed right side becomes `body`. Write exhaustive unit tests in `cmd/clinban/new_test.go`.
- **Module(s):** `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- **Done criteria:**
  - [ ] `splitRawBody("")` returns `("", "")`
  - [ ] `splitRawBody("just body")` returns `("", "just body")`
  - [ ] `splitRawBody("title # body")` returns `("title", "body")`
  - [ ] `splitRawBody("title # body with # hashes")` returns `("title", "body with # hashes")`
  - [ ] `splitRawBody("title #")` returns `("title", "")`
  - [ ] `splitRawBody("# body only")` returns `("", "body only")`
  - [ ] All above cases covered by table-driven tests in `new_test.go`
  - [ ] `go test ./cmd/clinban/...` passes
- **Depends on:** none
- **Notes:** Use `strings.Cut(raw, "#")` (available since Go 1.18) for a clean single-line implementation. Both sides must be passed through `strings.TrimSpace` before returning. The function is unexported; tests live in `package main` (same package, `_test.go` file).

---

### TASK-004: Add `Title` to template data and update `template.New` signature
- **Description:** Add `Title string` to `templateData` in `internal/template/template.go`. Change `New` to accept a third `title string` parameter and populate `templateData.Title`. Update `internal/template/new.md` to use `title: "{{.Title}}"` instead of `title: ""`. Update the single call site in `cmd/clinban/new.go` (`runNewInteractive`) to pass `""` as the new third argument (temporary placeholder — wired in TASK-005).
- **Module(s):** `internal/template/template.go`, `internal/template/new.md`, `cmd/clinban/new.go` (call site only)
- **Done criteria:**
  - [ ] `template.New(now, "", "")` returns bytes containing `title: ""`  (identical to pre-change output)
  - [ ] `template.New(now, "", "My Title")` returns bytes containing `title: "My Title"`
  - [ ] Existing tests (if any) in `internal/template/` continue to pass
  - [ ] `go build ./...` succeeds (no broken call sites)
  - [ ] `go test ./...` passes
- **Depends on:** none
- **Notes:** Only `runNewInteractive` calls `template.New`; `runNewNonInteractive` builds a `ticket.Ticket` struct directly and is not affected. The template uses `text/template` — no HTML escaping occurs for string fields, so arbitrary title text is safe.

---

### TASK-005: Wire `splitRawBody` into `runNewInteractive` and add integration tests
- **Description:** Update `runNewInteractive` in `cmd/clinban/new.go` to call `splitRawBody(raw)` when `cfg.SplitRawNew` is true and `raw` is non-empty. Pass the returned `title` to `template.New` and the returned `body` as the file-append body text. When `cfg.SplitRawNew` is false, pass `""` as title and the full `raw` string as body (unchanged behaviour). Add integration tests covering: (a) split enabled with `#` in args, (b) split disabled with `#` in args, (c) no `#` in args with split enabled.
- **Module(s):** `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- **Done criteria:**
  - [ ] Args `["title", "#", "body text"]` with `SplitRawNew=true` produce a temp file whose frontmatter contains `title: "title"` and body section contains `"body text"`
  - [ ] Args `["title", "#", "body text"]` with `SplitRawNew=false` produce `title: ""` in frontmatter and `"title # body text"` in body
  - [ ] Args `["just body"]` with `SplitRawNew=true` produce `title: ""` in frontmatter and `"just body"` in body
  - [ ] Integration tests use a test helper (`runNewInteractiveWithArgs` or equivalent) that bypasses editor invocation
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` passes
- **Depends on:** TASK-002, TASK-003, TASK-004
- **Notes:** The existing `runNewInteractive` already appends `body` to the temp file after the template bytes. The title pre-fill is purely a change to the `template.New` call. The integration test must stub or skip the `editor.Open` call — check whether the existing test suite already has an editor-bypass mechanism (look for `EDITOR` env var override or a test hook in `cmd/clinban/`).

---

## Dependency Order

```
TASK-001 ──── (independent, ship any time)

TASK-002 ─────────────────────────────────┐
                                           ├──► TASK-005
TASK-003 ─────────────────────────────────┤
                                           │
TASK-004 ─────────────────────────────────┘
```

Plain ordered list for a single developer working sequentially:

1. TASK-001 (no deps, ~30 min — ship first as it is a standalone bug fix)
2. TASK-002 (no deps, ~2 h)
3. TASK-003 (no deps, ~45 min — can be done in parallel with TASK-002)
4. TASK-004 (no deps, ~30 min — can be done in parallel with TASK-002/003)
5. TASK-005 (requires TASK-002, TASK-003, TASK-004 complete, ~2 h)
