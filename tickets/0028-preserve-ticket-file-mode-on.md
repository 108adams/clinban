---
title: Preserve ticket file mode on write; new tickets honour umask
status: done
type: bug
tags: [store, permissions]
created: 2026-10-16T14:06:22.240724896+02:00
updated: 2026-10-16T14:06:34.996582023+02:00
---
Moved from next-lm ticket 0281 (filed there 2026-10-04 because that is where the board lived).

## Root cause
`internal/store/write.go:57` called `tmp.Chmod(0o600)` on every ticket write (move, edit, push, register, TUI). The interactive `clinban new` path used `os.CreateTemp`, which also creates `0600`, and renamed that file into place. In next-lm this left 67 of 86 files in `tickets/` and 60 of 195 in `tickets/archive/` at `600`, unreadable to the web container running under another uid (next-lm 0231; `Makefile:101` re-applies `chmod a+r` before every in-container pytest run as a workaround).

`archive` / `unarchive` use `os.Link` + `os.Remove` (`internal/store/move.go:200`, `:220`, `:243`), which keeps the inode, so they were not affected.

## Fix
- Overwriting an existing ticket keeps its permission bits.
- A new ticket file gets `0666` minus the process umask: `store.CreateTemp` opens with `O_EXCL` and `0666`, so the kernel applies the umask. The interactive `new` path and `WriteTicket` both use it.
- `config.go` still writes `.clinban` at `0600` — out of scope.

## Done criteria
- [x] Rewrite preserves mode (`TestWriteTicketPreservesExistingMode`, `TestMovePreservesTicketMode`)
- [x] New tickets honour umask, both `new` paths (`TestWriteTicketNewFileHonoursUmask`, `TestNewNoInteractiveHonoursUmask`, `TestNewInteractiveHonoursUmask`)
- [x] `go test ./...` green; `go vet` for linux, windows, darwin