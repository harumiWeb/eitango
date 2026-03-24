package app

import (
	"fmt"
	"strings"

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
		"",
		m.styles.Muted.Render("Rate it: a=again  h=hard  g=good  e=easy"),
	}
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
