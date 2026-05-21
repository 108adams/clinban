---
title: ""
id: "{{printf "%04d" .ID}}"
status: "backlog"
# states: backlog, in-progress, blocked, done
type: "{{.Type}}"
# types: bug, task, feature, spike
tags: []
created: "{{.Now.Format "2006-01-02T15:04:05Z07:00"}}"
updated: "{{.Now.Format "2006-01-02T15:04:05Z07:00"}}"
---
