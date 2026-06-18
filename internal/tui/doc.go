// Package tui implements the Bubble Tea model for the clinban board TUI.
//
// The entry point is [New], which constructs a [Model] from a store. The model
// is a pure consumer: it reads from the store via commands but never calls
// os.Link, os.Remove, or os.Rename on managed tickets. All mutation goes through
// store.WriteTicket and internal/fsm.
//
// File layout:
//
//   - model.go    — Model struct, New, Init, Update, View
//   - keys.go     — keyMap (key.Binding set), ShortHelp/FullHelp
//   - commands.go — tea.Cmd factories (the only code that touches the store)
//   - messages.go — message types returned by commands
package tui
