---
title: 'refactor: extract shared directory-scan helper in store/scan.go'
status: done
type: task
tags: [store, refactor, cleanup]
created: 2026-06-11T10:50:26.683958138+02:00
updated: 2026-06-14T18:06:20.039901424+02:00
---

## Background

`internal/store/scan.go` now contains three functions that share identical
directory-walking boilerplate: `ReadDir` call, `os.IsNotExist` guard, `IsDir`
skip, `idPattern` match, `filepath.Join` path construction:

- `scanDir` (returns `[]int`) — used by `AllIDs`/`NextID`
- `scanDirIDs` (returns `[]string`) — used by `ListActive`/`ListArchive` ID listing
- `managedFilesInDir` (returns `[]ManagedFile`) — added in 0022 for `resolve`

A bug fix to the shared logic (e.g. surfacing `fs.ErrPermission` on individual
entries, or correcting the `IsNotExist` check) must currently be applied to all
three independently. One will be missed.

## Desired Outcome

Extract a single private helper that does the ReadDir + IsNotExist + IsDir +
idPattern loop and yields `(id string, path string)` pairs via a callback or
returned slice. `scanDir`, `scanDirIDs`, and `managedFilesInDir` each call this
helper and project the result into their respective return types.

## Acceptance Criteria

- [ ] One private function owns the ReadDir + idPattern loop
- [ ] `scanDir`, `scanDirIDs`, and `managedFilesInDir` all delegate to it
- [ ] No behaviour change — all existing store tests pass unchanged
- [ ] `go vet` and `gofmt` clean
