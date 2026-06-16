package tui

import "charm.land/bubbles/v2/key"

// keyMap holds the board TUI key bindings. ScrollDown/ScrollUp (T5) and Advance
// (T6) are landed; the editor key (e) arrives in T7.
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	ScrollDown key.Binding
	ScrollUp   key.Binding
	Edit       key.Binding
	Advance    key.Binding
	Reload     key.Binding
	Help       key.Binding
	Quit       key.Binding
}

// defaultKeyMap returns the keyMap with its default bindings.
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "move down"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "scroll preview down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "scroll preview up"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit in $EDITOR"),
		),
		Advance: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "advance status"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns the short help bindings (for the collapsed help bar).
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.ScrollDown, k.ScrollUp, k.Edit, k.Advance, k.Reload, k.Help, k.Quit}
}

// FullHelp returns the full help bindings grouped by column.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.ScrollDown, k.ScrollUp},
		{k.Edit, k.Advance, k.Reload, k.Help, k.Quit},
	}
}
