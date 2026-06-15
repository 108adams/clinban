---
title: Choose terminal UI foundation
status: in-progress
type: spike
tags: [tui, adr, discovery]
created: 2026-06-10T18:19:55+02:00
updated: 2026-06-15T09:38:05.721840509+02:00
---

# ADR: Terminal UI foundation

## Status

Proposed.

This ticket is the discovery artifact for phase two. It records the first foundation decision for a 100% terminal UI. It is not an implementation ticket.

## Context

Clinban is currently a Go CLI around Markdown ticket files with YAML frontmatter. The next phase should add an interactive terminal UI without changing the product's core storage model:

- Tickets remain Markdown files on disk.
- Existing CLI commands remain scriptable and stable.
- The TUI is an additional human interface over the same internal packages.
- The first screen is a two-pane board:
  - left pane: ticket list, navigable with arrow keys and Vim keys;
  - right pane: selected ticket preview;
  - command keys for edit, status change, refresh, filtering, and quit.

Non-goal for this ticket: designing final package boundaries, command names, full keymap, or screen flow.

## Decision

Use the Charm stack as the default TUI foundation:

- `github.com/charmbracelet/bubbletea/v2` or current canonical Bubble Tea v2 import path at implementation time for the application runtime and update loop.
- `github.com/charmbracelet/bubbles/v2` or current canonical Bubbles v2 import path for list, viewport, help, key binding, and text input components where they fit.
- `github.com/charmbracelet/lipgloss/v2` or current canonical Lip Gloss v2 import path for terminal layout and styling.
- Consider `github.com/charmbracelet/glamour/v2` or current canonical Glamour v2 import path for Markdown rendering in the ticket preview pane, but keep raw Markdown preview acceptable for the first spike if Glamour creates too much coupling or visual noise.

Reject `github.com/jroimartin/gocui` as the default foundation.

## Rationale

The hard requirement is not "draw two boxes." The hard requirement is keeping the TUI testable and maintainable as it grows from two panes into real workflows: filtering, editing, moving tickets, handling invalid transitions, resizing terminals, showing validation failures, and possibly later supporting modal commands.

Bubble Tea's model-update-view style is a better fit for this than callback-oriented view mutation. Clinban already has small domain packages and command handlers that coordinate behavior. A Bubble Tea model can sit at the UI boundary and translate key messages into calls against existing config, store, lint, fsm, ticket, and editor behavior.

Bubbles already contains the boring primitives this project needs: list navigation, viewport scrolling, help rendering, key bindings, text input, and table-like components. Reusing those is preferable to building a local widget layer on top of raw terminal cells.

Lip Gloss keeps layout and styling declarative enough that the first implementation can stay boring: left list, vertical divider, right preview, status/help line. It also keeps terminal styling separate from ticket/domain logic.

Glamour is attractive because the right pane displays Markdown. It should be treated as optional until proven useful. The preview pane must not become a second Markdown product with surprising rendering behavior. Raw ticket body plus frontmatter summary is acceptable for the first spike.

## Alternatives Considered

### `jroimartin/gocui`

Pros:

- Small API.
- Direct mental model: views, keybindings, layout function.
- Good enough for a simple split-pane prototype.

Cons:

- Upstream maintenance signal is weak. The latest tag is `v0.5.0` from 2021, and recent activity is sparse.
- Its core model encourages mutable views and callback wiring. That is fine for tiny tools and starts hurting when workflows, validation messages, async editor handoff, and testability matter.
- It does not give Clinban much beyond terminal drawing and keybindings.

Verdict: do not choose it for phase two. It solves the first demo and creates avoidable maintenance debt for the real tool.

### `awesome-gocui/gocui`

Pros:

- Community fork of gocui.
- Tcell-backed.
- Improves wide character handling, cursor handling, one-line views, frame colors, and Docker/container behavior compared with the original.

Cons:

- Latest visible release is from 2022.
- Smaller ecosystem than Charm.
- Still keeps the gocui programming model.

Verdict: better than original gocui, still not the best foundation.

### `rivo/tview`

Pros:

- Mature widget library.
- Rich components are available immediately.
- Built on tcell and uniseg.
- Backwards compatibility is an explicit project concern.

Cons:

- The framework is widget/application oriented. For Clinban, that can push too much UI policy into widget state instead of keeping state transitions explicit.
- Testability of application behavior is likely weaker than a pure message/update model.
- It is a good choice for forms-heavy terminal apps; Clinban's core interaction is closer to a keyboard-driven browser/editor shell.

Verdict: credible second choice. Use it only if the Bubble Tea spike shows unacceptable complexity or poor terminal behavior.

### `gdamore/tcell`

Pros:

- Low-level terminal foundation.
- Mature, actively released, and used underneath other libraries.
- Maximum control.

Cons:

- Too low-level for phase two.
- Clinban would have to build its own application loop, widgets, focus management, help, scrolling, and text handling.

Verdict: do not use directly unless higher-level libraries fail. It is a substrate, not the right product-level API.

### No TUI library / raw ANSI

Pros:

- Minimal dependencies.
- Full control over output.

Cons:

- False economy. Keyboard input, alternate screen behavior, resizing, focus, scrolling, and testability will become local infrastructure.

Verdict: reject.

## Consequences

Expected benefits:

- The TUI can be modeled as explicit state and messages.
- Navigation behavior can be tested without a real terminal for most logic.
- Layout can remain separate from domain operations.
- The first two-pane screen can grow into filters, status actions, and editor handoff without changing foundations.

Costs:

- Charm libraries add a new dependency family.
- Developers must understand Bubble Tea's architecture rather than writing direct imperative terminal mutations.
- Some Bubbles components may be too opinionated and may need local wrappers or replacement over time.
- Bubble Tea v2 import paths and APIs must be checked at implementation time because the ecosystem moved to v2 recently.

## Implementation Spike Requirements

Before accepting this ADR as final, perform a narrow spike:

- Add a hidden or experimental TUI entrypoint; exact CLI shape is not decided here.
- Load active tickets through existing config/store/ticket/lint paths.
- Render a two-pane screen.
- Support `up/down` and `k/j` ticket selection.
- Support right-pane scrolling.
- Support terminal resize.
- Support `q` and `ctrl+c` exit.
- Show an error state when tickets cannot be loaded or parsed.
- Open `$EDITOR` for the selected ticket using existing editor behavior, then refresh the list after the editor exits.
- Move the selected ticket through at least one valid status transition through existing fsm/store behavior.
- Keep the TUI out of domain packages; the UI is a consumer, not a new source of ticket truth.
- Confirm whether Glamour improves or harms preview readability on realistic ticket bodies.
- Verify behavior in a normal terminal and in CI-testable model/unit tests.

## Open Questions

- Should the TUI be invoked as `clinban tui`, `clinban board`, or a flag on the root command?
- Should archived tickets be visible in the first TUI release or only active tickets?
- Should status changes be single-key actions, a small command palette, or a modal selector?
- Should filters mirror `clinban list` flags exactly, or should the TUI introduce interactive filtering first?
- Should the preview pane render full raw ticket files, parsed frontmatter plus Markdown body, or Markdown body only?

## Source Notes

- `jroimartin/gocui`: https://github.com/jroimartin/gocui
- `awesome-gocui/gocui`: https://github.com/awesome-gocui/gocui
- `charmbracelet/bubbletea`: https://github.com/charmbracelet/bubbletea
- `charmbracelet/bubbles`: https://github.com/charmbracelet/bubbles
- `charmbracelet/lipgloss`: https://github.com/charmbracelet/lipgloss
- `charmbracelet/glamour`: https://github.com/charmbracelet/glamour
- `rivo/tview`: https://github.com/rivo/tview
- `gdamore/tcell`: https://github.com/gdamore/tcell
