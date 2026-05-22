---
id: "0012"
status: done
type: feature
title: 'new command: remove'
tags: []
created: 2026-05-21T12:47:56+02:00
updated: 2026-05-21T13:26:46.656462548+02:00
---

implelent new command: `remove id` which just removes task file altogether

if no file to remove (id is non-existent) exit 1 with error message
if multiple files to remove (multiple files share the same id) exit 1 with error message, suggest linting the ticket repository and list colliding files

