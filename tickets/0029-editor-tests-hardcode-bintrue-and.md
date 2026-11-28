---
title: Editor tests hardcode /bin/true and /bin/false, fail on macOS
status: backlog
type: bug
tags: [tests, portability]
created: 2026-11-28T17:59:05.085119+01:00
updated: 2026-11-28T17:59:05.085119+01:00
---
## Problem

`TestEditorSuccess` and `TestEditorFailure` set `EDITOR` to absolute paths that do not exist on macOS
(`true`/`false` live in `/usr/bin` there):

- `internal/editor/editor_test.go:110` — `t.Setenv("EDITOR", "/bin/true")`
- `internal/editor/editor_test.go:118` — `t.Setenv("EDITOR", "/bin/false")`

Observed on macOS 27.2 (arm64), go1.26.5, at `e1fbc2a`:

```
--- FAIL: TestEditorSuccess
    editor_test.go:113: Open returned unexpected error: editor "true" exited with error: fork/exec /bin/true: no such file or directory
--- FAIL: TestEditorFailure
    editor_test.go:125: error "editor \"false\" exited with error: fork/exec /bin/false: no such file or directory" does not contain 'exit status'
```

`make check` therefore fails on macOS; all other packages pass.

## Fix

Resolve the binaries at test time with `exec.LookPath("true")` / `exec.LookPath("false")` and
`t.Skip` if not found, instead of hardcoding paths.

## Acceptance

- [ ] `make check` passes on macOS and Linux
- [ ] No absolute `/bin/true` or `/bin/false` left in `internal/editor/editor_test.go`