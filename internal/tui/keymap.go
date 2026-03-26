package tui

import (
	"charm.land/bubbles/v2/key"
	"github.com/yourname/eitango/internal/i18n"
)

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
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
	Settings   key.Binding
	Back       key.Binding
}

func NewKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", i18n.T(i18n.KeyUp))),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", i18n.T(i18n.KeyDown))),
		Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", i18n.T(i18n.KeyLeft))),
		Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", i18n.T(i18n.KeyRight))),
		Select1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", i18n.T(i18n.KeyChoice1))),
		Select2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", i18n.T(i18n.KeyChoice2))),
		Select3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", i18n.T(i18n.KeyChoice3))),
		Select4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", i18n.T(i18n.KeyChoice4))),
		Confirm:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", i18n.T(i18n.KeyConfirm))),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", i18n.T(i18n.KeyQuit))),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", i18n.T(i18n.KeyHelp))),
		Again:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", i18n.T(i18n.KeyAgain))),
		Hard:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", i18n.T(i18n.KeyHard))),
		Good:       key.NewBinding(key.WithKeys("g"), key.WithHelp("g", i18n.T(i18n.KeyGood))),
		Easy:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", i18n.T(i18n.KeyEasy))),
		NewSession: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", i18n.T(i18n.KeyNewSession))),
		Review:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", i18n.T(i18n.KeyReview))),
		Stats:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", i18n.T(i18n.KeyStats))),
		Settings:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", i18n.T(i18n.KeySettings))),
		Back:       key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", i18n.T(i18n.KeyBack))),
	}
}
