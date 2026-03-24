package tui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Select1    key.Binding
	Select2    key.Binding
	Select3    key.Binding
	Select4    key.Binding
	Confirm    key.Binding
	Quit       key.Binding
	Help       key.Binding
	Again      key.Binding
	Hard       key.Binding
	Good       key.Binding
	Easy       key.Binding
	NewSession key.Binding
	Review     key.Binding
	Stats      key.Binding
	Back       key.Binding
}

func NewKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		Select1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "choice 1")),
		Select2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "choice 2")),
		Select3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "choice 3")),
		Select4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "choice 4")),
		Confirm:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Again:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "again")),
		Hard:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hard")),
		Good:       key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "good")),
		Easy:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "easy")),
		NewSession: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
		Review:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "review")),
		Stats:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stats")),
		Back:       key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "back")),
	}
}
