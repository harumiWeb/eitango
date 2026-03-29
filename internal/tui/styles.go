package tui

import lipgloss "charm.land/lipgloss/v2"

type Styles struct {
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Panel          lipgloss.Style
	ModalPanel     lipgloss.Style
	CorrectPanel   lipgloss.Style
	WrongPanel     lipgloss.Style
	Choice         lipgloss.Style
	ChoiceSelected lipgloss.Style
	Correct        lipgloss.Style
	Wrong          lipgloss.Style
	Status         lipgloss.Style
	Error          lipgloss.Style
	Muted          lipgloss.Style
	QuizMeta       lipgloss.Style
	Accent         lipgloss.Style
}

func NewStyles() Styles {
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	return Styles{
		Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		Subtitle:       lipgloss.NewStyle().Bold(true),
		Panel:          panel,
		ModalPanel:     lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("81")),
		CorrectPanel:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("42")),
		WrongPanel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("203")),
		Choice:         lipgloss.NewStyle().PaddingLeft(1),
		ChoiceSelected: lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(lipgloss.Color("86")),
		Correct:        lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		Wrong:          lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		Status:         lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		Error:          lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		Muted:          lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		QuizMeta:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Accent:         lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
}
