# Implementation Design
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: ticket 0005 — extend clinban new with optional body args_

## Scope note

Single-package change. Only `cmd/clinban/new.go` and `cmd/clinban/new_test.go` are
modified. No internal package changes required.

---

## Module Structure

### cmd/clinban — new.go

**Changes:**

1. `newCmd`: add `Args: cobra.ArbitraryArgs` so Cobra does not reject positional arguments.

2. `runNew`: change signature to use args; join them as body:
   ```go
   func runNew(cmd *cobra.Command, args []string) error {
       if newFlagValues.noInteractive {
           return runNewNonInteractive(newFlagValues)
       }
       return runNewInteractive(strings.Join(args, " "))
   }
   ```

3. `runNewInteractive(body string) error`: after writing template bytes to the temp
   file, if `body != ""` append `"\n" + body + "\n"` before closing the file:
   ```go
   if body != "" {
       if _, err := tmpFile.Write([]byte("\n" + body + "\n")); err != nil { ... }
   }
   ```
   All subsequent logic (editor open, parse, discard-check, lint loop) is unchanged.

**Interface contract:**
- `runNewInteractive("")` — identical to current behaviour
- `runNewInteractive("some text")` — temp file contains frontmatter + blank line + body; editor opens with body pre-filled; user edits title; rest of flow unchanged
- Positional args with `--no-interactive` are silently ignored (non-interactive uses `--body` flag for body content)

---

## Test Strategy

**New tests in `new_test.go`:**

1. `TestNewInteractiveWithBodyArgs`: run `clinban new "body text here"` with a fake editor
   script that only sets the title (body already present → leave it). After creation, read
   the ticket file and assert body contains `"body text here"`.

2. `TestNewInteractiveNoArgsUnchanged`: run `clinban new` with no args using existing
   `makeEditorScript` → asserts existing happy-path still works (regression).

**Critical path:** `TestNewInteractiveWithBodyArgs` — body text must survive the
editor round-trip and appear in the written ticket file.

**Existing tests must pass unchanged.**

---

## Resolved Design Questions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Multi-word body: join with space or newline? | Space (`strings.Join(args, " ")`) | Args come from shell word-splitting; single space join is natural |
| Args + `--no-interactive`? | Silently ignored | `--body` flag is the designated non-interactive body mechanism; no conflict |
| Append body before or after editor open? | Before — write to temp file, then open editor | User sees body pre-filled in editor, consistent with ticket description |
