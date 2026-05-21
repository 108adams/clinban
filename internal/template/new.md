---
id: "{{printf "%04d" .ID}}"
status: "backlog"
type: "{{.Type}}"
# types: bug, task, feature, spike
title: ""
tags: []
created: "{{.Now.Format "2006-01-02T15:04:05Z07:00"}}"
updated: "{{.Now.Format "2006-01-02T15:04:05Z07:00"}}"
---
