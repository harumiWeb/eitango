package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/stats"
	"github.com/yourname/eitango/internal/store"
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
		body += "\n\n" + m.styles.Muted.Render("Loading...")
	}
	body += "\n\n" + m.renderStatusLine()
	return tea.NewView(body)
}

func (m RootModel) renderHome() string {
	lines := []string{
		m.styles.Title.Render("eitango"),
		m.styles.Muted.Render("AI waiting time -> 1-3 minute vocab loop"),
		"",
		fmt.Sprintf("Due now      : %d", m.home.DueCount),
		fmt.Sprintf("New available: %d", m.home.NewCount),
		fmt.Sprintf("Streak days  : %d", m.home.StreakDays),
		fmt.Sprintf("Wait today   : %.1f min", m.stats.Today.WaitMinutes),
	}
	if m.home.ActiveSession != nil {
		lines = append(lines,
			"",
			m.styles.Subtitle.Render("Active session"),
			fmt.Sprintf("%d/%d answered (%s)", m.home.ActiveSession.AnsweredQuestions, m.home.ActiveSession.TotalQuestions, m.home.ActiveSession.Mode),
		)
	}
	lines = append(lines,
		"",
		m.styles.Muted.Render("Enter=start/resume  n=new  r=review  s=stats  q=quit"),
	)
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderQuiz() string {
	if m.currentQ == nil {
		return m.styles.Panel.Render("No question loaded")
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
	lines = append(lines, "", m.styles.Muted.Render("1-4=answer  j/k=move  enter=confirm  q=save and quit"))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderFeedback() string {
	if m.feedback == nil {
		return m.styles.Panel.Render("No feedback available")
	}

	result := m.styles.Wrong.Render("Incorrect")
	if m.feedback.Correct {
		result = m.styles.Correct.Render("Correct")
	}

	lines := []string{
		result,
		fmt.Sprintf("Word         : %s", m.feedback.Question.Word.Lemma),
		fmt.Sprintf("Correct      : %s", m.feedback.Question.CorrectChoice().Meaning),
		fmt.Sprintf("Response time: %d ms", m.feedback.ResponseMS),
	}
	if m.feedback.Question.Word.ExampleEN != "" || m.feedback.Question.Word.ExampleJA != "" {
		lines = append(lines, "")
		if m.feedback.Question.Word.ExampleEN != "" {
			lines = append(lines, fmt.Sprintf("Example EN   : %s", m.feedback.Question.Word.ExampleEN))
		}
		if m.feedback.Question.Word.ExampleJA != "" {
			lines = append(lines, fmt.Sprintf("Example JA   : %s", m.feedback.Question.Word.ExampleJA))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render("Rate it: a=again  h=hard  g=good  e=easy"))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderResults() string {
	if m.summary == nil {
		return m.styles.Panel.Render("No session summary available")
	}

	lines := []string{
		m.styles.Title.Render("Session results"),
		fmt.Sprintf("Accuracy : %.1f%%", m.summary.Accuracy),
		fmt.Sprintf("Correct  : %d/%d", m.summary.CorrectAnswers, m.summary.TotalQuestions),
		fmt.Sprintf("Mix      : new=%d review=%d retry=%d", m.summary.NewCount, m.summary.ReviewCount, m.summary.RetryCount),
	}
	if len(m.summary.HardWords) > 0 {
		lines = append(lines, "", m.styles.Subtitle.Render("Hard words"))
		for _, word := range m.summary.HardWords {
			lines = append(lines, fmt.Sprintf("- %s: %s", word.Lemma, word.MeaningJA))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render("Enter/Esc=home  q=quit"))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStats() string {
	return m.styles.Panel.Render(stats.RenderText(m.stats) + "\nEsc/Enter=back")
}

func (m RootModel) renderHelp() string {
	lines := []string{
		m.styles.Title.Render("Help"),
		m.styles.Muted.Render(screenHelpTitle(m.helpReturn)),
	}

	for _, section := range m.helpSections(m.helpReturn) {
		lines = append(lines, "", m.styles.Subtitle.Render(section.title))
		lines = append(lines, section.lines...)
	}

	lines = append(lines, "", m.styles.Muted.Render("Esc=back"))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStatusLine() string {
	if m.err != nil {
		return m.styles.Error.Render(m.status)
	}
	if m.status == "" {
		return m.styles.Status.Render("Ready")
	}
	return m.styles.Status.Render(m.status)
}

func kindLabel(kind string) string {
	switch kind {
	case store.ItemKindReview:
		return "review"
	case store.ItemKindRetry:
		return "retry"
	default:
		return "new"
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
			{title: "Answer", lines: []string{
				helpLine(m.keymap.Select1),
				helpLine(m.keymap.Select2),
				helpLine(m.keymap.Select3),
				helpLine(m.keymap.Select4),
				helpLine(m.keymap.Confirm),
			}},
			{title: "Move", lines: []string{
				helpLine(m.keymap.Up),
				helpLine(m.keymap.Down),
			}},
			{title: "General", lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	case ScreenFeedback:
		return []helpSection{
			{title: "Rate recall", lines: []string{
				helpLine(m.keymap.Again),
				helpLine(m.keymap.Hard),
				helpLine(m.keymap.Good),
				helpLine(m.keymap.Easy),
			}},
			{title: "General", lines: []string{
				helpLine(m.keymap.Help),
				fmt.Sprintf("%-10s %s", "q", "disabled until you rate"),
			}},
		}
	case ScreenResults:
		return []helpSection{
			{title: "Navigation", lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.Back),
			}},
			{title: "General", lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	case ScreenStats:
		return []helpSection{
			{title: "Navigation", lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.Back),
			}},
			{title: "General", lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	default:
		return []helpSection{
			{title: "Sessions", lines: []string{
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.NewSession),
				helpLine(m.keymap.Review),
				helpLine(m.keymap.Stats),
			}},
			{title: "General", lines: []string{
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
		return "Quiz controls"
	case ScreenFeedback:
		return "Feedback controls"
	case ScreenResults:
		return "Results controls"
	case ScreenStats:
		return "Stats controls"
	default:
		return "Home controls"
	}
}
