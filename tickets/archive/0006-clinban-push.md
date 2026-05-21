---
id: "0006"
status: done
type: feature
title: clinban push
tags: []
created: 2026-05-21T09:54:51+02:00
updated: 2026-05-21T10:17:48.933352418+02:00
---
Add a new command `push` which moves given ticket to the next status

for example if ticket id 1 is in "backlog", issuing `clinban push 1` would change its status to "in-progress"
exit 0 with a message "ticket 0001 has been moved to state in-progress" (adjust for proper English)

if ticket is on the final status and there is no next status to push, exit 0 with a proper message
