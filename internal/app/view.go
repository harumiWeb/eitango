package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/i18n"
	"github.com/yourname/eitango/internal/stats"
	"github.com/yourname/eitango/internal/store"
	"github.com/yourname/eitango/internal/tui"
)

func (m RootModel) View() tea.View {
	body := ""
	switch m.screen {
	case ScreenQuiz:
		body = m.renderQuiz()
	case ScreenFeedback:
		body = m.renderFeedback()
	case ScreenResults:
		body = m.renderResults()
	case ScreenStats:
		body = m.renderStats()
	case ScreenHelp:
		body = m.renderHelp()
	default:
		body = m.renderHome()
	}

	if m.loading {
		body += "\n\n" + m.styles.Muted.Render(i18n.T(i18n.StatusLoading))
	}
	body += "\n\n" + m.renderStatusLine()
	return tea.NewView(body)
}

func (m RootModel) renderHome() string {
	lines := []string{
		m.styles.Title.Render(tui.Logo),
		m.styles.Muted.Render(i18n.T(i18n.HomeSubtitle)),
		"",
		fmt.Sprintf("%-14s: %d", i18n.T(i18n.HomeDue), m.home.DueCount),
		fmt.Sprintf("%-14s: %d", i18n.T(i18n.HomeNew), m.home.NewCount),
		fmt.Sprintf("%-14s: %d", i18n.T(i18n.HomeStreak), m.home.StreakDays),
		fmt.Sprintf("%-14s: %.1f min", i18n.T(i18n.HomeWait), m.stats.Today.WaitMinutes),
	}
	if m.home.ActiveSession != nil {
		lines = append(lines,
			"",
			m.styles.Subtitle.Render(i18n.T(i18n.HomeActive)),
			i18n.Tf(i18n.HomeActiveDetail, m.home.ActiveSession.AnsweredQuestions, m.home.ActiveSession.TotalQuestions, m.home.ActiveSession.Mode),
		)
	}
	lines = append(lines,
		"",
		m.styles.Muted.Render(i18n.T(i18n.HomeKeys)),
	)
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderQuiz() string {
	if m.currentQ == nil {
		return m.styles.Panel.Render(i18n.T(i18n.QuizNoQuestion))
	}

	lines := []string{
		m.styles.Title.Render(m.currentQ.Word.Lemma),
		m.styles.Muted.Render(fmt.Sprintf("%s  •  %s  •  %d/%d", m.currentQ.Word.Pos, kindLabel(m.currentQ.Kind), m.currentQ.Ordinal, m.currentQ.Total)),
	}
	for i, choice := range m.currentQ.Choices {
		text := fmt.Sprintf("%d. %s", i+1, choice.Meaning)
		if i == m.cursor {
			lines = append(lines, m.styles.ChoiceSelected.Render(text))
		} else {
			lines = append(lines, m.styles.Choice.Render(text))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.QuizKeys)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderFeedback() string {
	if m.feedback == nil {
		return m.styles.Panel.Render(i18n.T(i18n.FbNoFeedback))
	}

	panel := m.styles.WrongPanel
	result := m.styles.Wrong.Render("✗ " + i18n.T(i18n.FbIncorrect))
	if m.feedback.Correct {
		panel = m.styles.CorrectPanel
		result = m.styles.Correct.Render("✓ " + i18n.T(i18n.FbCorrect))
	}

	lines := []string{result}

	if m.feedback.Correct && m.correctStreak >= 3 {
		lines = append(lines, m.styles.Correct.Render(i18n.Tf(i18n.FbStreak, m.correctStreak)))
	}

	lines = append(lines,
		"",
		fmt.Sprintf("%-14s: %s", i18n.T(i18n.FbWord), m.feedback.Question.Word.Lemma),
		fmt.Sprintf("%-14s: %s", i18n.T(i18n.FbCorrectAnswer), m.feedback.Question.CorrectChoice().Meaning),
	)

	if !m.feedback.Correct {
		selected := m.feedback.Question.Choices[m.feedback.SelectedIndex].Meaning
		lines = append(lines, fmt.Sprintf("%-14s: %s", i18n.T(i18n.FbYourAnswer), selected))
	}

	lines = append(lines, fmt.Sprintf("%-14s: %d ms", i18n.T(i18n.FbResponseTime), m.feedback.ResponseMS))

	if m.feedback.Question.Word.ExampleEN != "" || m.feedback.Question.Word.ExampleJA != "" {
		lines = append(lines, "")
		if m.feedback.Question.Word.ExampleEN != "" {
			lines = append(lines, fmt.Sprintf("%-14s: %s", i18n.T(i18n.FbExampleEN), m.feedback.Question.Word.ExampleEN))
		}
		if m.feedback.Question.Word.ExampleJA != "" {
			lines = append(lines, fmt.Sprintf("%-14s: %s", i18n.T(i18n.FbExampleJA), m.feedback.Question.Word.ExampleJA))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.FbKeys)))
	return panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderResults() string {
	if m.summary == nil {
		return m.styles.Panel.Render(i18n.T(i18n.ResultsNoSummary))
	}

	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.ResultsTitle)),
		fmt.Sprintf("%-10s: %.1f%%", i18n.T(i18n.ResultsAccuracy), m.summary.Accuracy),
		fmt.Sprintf("%-10s: %d/%d", i18n.T(i18n.ResultsCorrect), m.summary.CorrectAnswers, m.summary.TotalQuestions),
		fmt.Sprintf("%-10s: %s", i18n.T(i18n.ResultsMix), i18n.Tf(i18n.ResultsMixDetail, m.summary.NewCount, m.summary.ReviewCount, m.summary.RetryCount)),
	}
	if len(m.summary.HardWords) > 0 {
		lines = append(lines, "", m.styles.Subtitle.Render(i18n.T(i18n.ResultsHardWords)))
		for _, word := range m.summary.HardWords {
			lines = append(lines, fmt.Sprintf("- %s: %s", word.Lemma, word.MeaningJA))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.ResultsKeys)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStats() string {
	return m.styles.Panel.Render(stats.RenderText(m.stats) + "\n" + i18n.T(i18n.StatsKeys))
}

func (m RootModel) renderHelp() string {
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.HelpTitle)),
		m.styles.Muted.Render(screenHelpTitle(m.helpReturn)),
	}

	for _, section := range m.helpSections(m.helpReturn) {
		lines = append(lines, "", m.styles.Subtitle.Render(section.title))
		lines = append(lines, section.lines...)
	}

	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.HelpBack)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStatusLine() string {
	if m.err != nil {
		return m.styles.Error.Render(m.status)
	}
	if m.status == "" {
		return m.styles.Status.Render(i18n.T(i18n.StatusReady))
	}
	return m.styles.Status.Render(m.status)
}

func kindLabel(kind string) string {
	switch kind {
	case store.ItemKindReview:
		return i18n.T(i18n.KindReview)
	case store.ItemKindRetry:
		return i18n.T(i18n.KindRetry)
	default:
		return i18n.T(i18n.KindNew)
	}
}

type helpSection struct {
	title string
	lines []string
}

func (m RootModel) helpSections(screen Screen) []helpSection {
	switch screen {
	case ScreenQuiz:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
				helpLine(m.keymap.Select1),
				helpLine(m.keymap.Select2),
				helpLine(m.keymap.Select3),
				helpLine(m.keymap.Select4),
				helpLine(m.keymap.Confirm),
			}},
			{title: i18n.T(i18n.HelpSectionMove), lines: []string{
				helpLine(m.keymap.Up),
				helpLine(m.keymap.Down),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	case ScreenFeedback:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionRate), lines: []string{
				helpLine(m.keymap.Again),
				helpLine(m.keymap.Hard),
				helpLine(m.keymap.Good),
				helpLine(m.keymap.Easy),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				fmt.Sprintf("%-10s %s", "q", i18n.T(i18n.HelpQuitDisabled)),
			}},
		}
	case ScreenResults:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.Back),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	case ScreenStats:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.Back),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	default:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionSessions), lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.NewSession),
				helpLine(m.keymap.Review),
				helpLine(m.keymap.Stats),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	}
}

func helpLine(binding key.Binding) string {
	help := binding.Help()
	return fmt.Sprintf("%-10s %s", help.Key, help.Desc)
}

func screenHelpTitle(screen Screen) string {
	switch screen {
	case ScreenQuiz:
		return i18n.T(i18n.HelpScreenQuiz)
	case ScreenFeedback:
		return i18n.T(i18n.HelpScreenFeedback)
	case ScreenResults:
		return i18n.T(i18n.HelpScreenResults)
	case ScreenStats:
		return i18n.T(i18n.HelpScreenStats)
	default:
		return i18n.T(i18n.HelpScreenHome)
	}
}
