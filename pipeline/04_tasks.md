# Developer Tasks
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Accept body args in clinban new interactive path

- **Description:** Allow `clinban new [body...]` to accept optional positional arguments.
  Join them as body text and pre-fill the temp file below the frontmatter before opening
  the editor.
- **Module(s):** `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- **Done criteria:**
  - [ ] `newCmd` has `Args: cobra.ArbitraryArgs`
  - [ ] `runNew` signature is `func runNew(cmd *cobra.Command, args []string) error`; passes `strings.Join(args, " ")` to `runNewInteractive`
  - [ ] `runNewInteractive` signature is `func runNewInteractive(body string) error`
  - [ ] When `body != ""`, `"\n" + body + "\n"` is written to the temp file after the template bytes, before the file is closed
  - [ ] `TestNewInteractiveWithBodyArgs`: runs `clinban new "body text here"` with a fake editor that sets a title; asserts exit 0 and the created ticket file body contains `"body text here"`
  - [ ] `TestNewInteractiveNoArgsUnchanged`: runs `clinban new` (no args) using existing editor script; asserts existing happy-path behaviour still works
  - [ ] All existing `new` tests pass
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** none
- **Notes:**
  - Use the existing `makeEditorScript` / `runNewInteractive` test helpers — read the test file before writing new tests.
  - The fake editor for `TestNewInteractiveWithBodyArgs` only needs to set the title (body already present in the file); it must NOT overwrite the body. The existing `makeEditorScript` uses sed to set `title:` and `type:` lines — it leaves the body untouched, so it can be reused directly.
  - Positional args with `--no-interactive` are silently ignored — no test needed for that path.
  - `strings` is already imported in `new.go`.

---

## Dependency Order

Single task. No dependencies.
