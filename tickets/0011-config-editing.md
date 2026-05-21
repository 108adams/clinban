---
id: "0011"
status: "backlog"
type: "feature"
# types: bug, task, feature, spike
title: "config editing"
tags: []
created: "2026-05-21T11:13:32+02:00"
updated: "2026-05-21T11:13:32+02:00"
---
add `config` command

1. if invoked with no args: list all possible config keys. If they are set in .clinban - show value. If not set explicitely - show default value and note like (not set in .clinban, default)
2. if invoked with args: if valid key, value, update .clinban with key, value, if error - exit 0 with error message
Example: `clinban config default_type="bug"` adds or changes the key in .clinban
