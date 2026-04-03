package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/tui"
)

func (m RootModel) View() tea.View {
	body := ""
	switch m.screen {
	case ScreenHome:
		body = m.renderHome()
		if m.settingsOpen {
			body = m.renderHomeWithSettingsOverlay()
		} else if m.homeConfirm != nil {
			body = m.renderHomeWithConfirmOverlay()
		}
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
	}

	if m.loading {
		body += "\n\n" + m.styles.Muted.Render(i18n.T(i18n.StatusLoading))
	}
	body += "\n" + m.renderStatusLine()
	return tea.NewView(body)
}

func (m RootModel) renderHome() string {
	lines := []string{
		m.styles.Title.Render(tui.Logo),
		m.styles.Muted.Render(i18n.T(i18n.HomeSubtitle)),
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.HomeAnswerMode), 14), m.renderAnswerModeTabs()),
		fmt.Sprintf("%s: %d", tui.AlignLabel(i18n.T(i18n.HomeDue), 14), m.home.DueCount),
		fmt.Sprintf("%s: %d", tui.AlignLabel(i18n.T(i18n.HomeNew), 14), m.home.NewCount),
		fmt.Sprintf("%s: %d", tui.AlignLabel(i18n.T(i18n.HomeStreak), 14), m.home.StreakDays),
		fmt.Sprintf("%s: %.1f min", tui.AlignLabel(i18n.T(i18n.HomeWait), 14), m.stats.Today.WaitMinutes),
	}
	if m.home.ActiveSession != nil {
		lines = append(lines,
			"",
			m.styles.Subtitle.Render(i18n.T(i18n.HomeActive)),
			i18n.Tf(i18n.HomeActiveDetail, m.home.ActiveSession.AnsweredQuestions, m.home.ActiveSession.TotalQuestions, sessionModeLabel(m.home.ActiveSession.Mode), answerModeLabel(m.home.ActiveSession.AnswerMode)),
		)
	}
	if strings.TrimSpace(m.updateLatestTag) != "" {
		currentVersion := m.currentVersion
		if strings.TrimSpace(currentVersion) == "" {
			currentVersion = "dev"
		}
		lines = append(lines,
			"",
			m.styles.Subtitle.Render(i18n.T(i18n.HomeUpdate)),
			i18n.Tf(i18n.HomeUpdateDetail, m.updateLatestTag, currentVersion),
			m.styles.Muted.Render(i18n.T(i18n.HomeUpdateHint)),
		)
	}
	lines = append(lines,
		"",
		m.styles.Muted.Render(i18n.T(i18n.HomeKeys)),
	)
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderHomeWithSettingsOverlay() string {
	return m.renderSettingsOverlay()
}

func (m RootModel) renderHomeWithConfirmOverlay() string {
	return m.renderHomeConfirmOverlay()
}

func (m RootModel) renderSettingsOverlay() string {
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.SettingsTitle)),
		"",
		m.renderSettingsRow(settingsRowQuestionCount, i18n.T(i18n.SettingsQuestions), m.settingsQuestionDisplay()),
		m.renderSettingsRow(settingsRowWriteDifficulty, i18n.T(i18n.SettingsWriteDifficulty), m.settingsWriteDifficultyLabel()),
		m.renderSettingsRow(settingsRowLanguage, i18n.T(i18n.SettingsLanguage), m.settingsLanguageLabel()),
		"",
		m.styles.Muted.Render(i18n.T(i18n.SettingsKeys)),
	}
	if m.settings.FocusModeDefault {
		lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.SettingsFocusNote)))
	}
	return m.styles.ModalPanel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderHomeConfirmOverlay() string {
	if m.homeConfirm == nil || m.home.ActiveSession == nil {
		return m.renderHome()
	}

	request := m.homeConfirm.Request
	active := m.home.ActiveSession
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.HomeConfirmTitle)),
		"",
		i18n.T(i18n.HomeConfirmBody),
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.HomeConfirmCurrent), 14), i18n.Tf(i18n.HomeActiveDetail, active.AnsweredQuestions, active.TotalQuestions, sessionModeLabel(active.Mode), answerModeLabel(active.AnswerMode))),
		fmt.Sprintf("%s: %s / %s", tui.AlignLabel(i18n.T(i18n.HomeConfirmTarget), 14), sessionModeLabel(request.Mode), answerModeLabel(request.AnswerMode)),
		"",
		m.styles.Muted.Render(i18n.T(i18n.HomeConfirmKeys)),
	}
	return m.styles.ModalPanel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderQuiz() string {
	if m.currentQ == nil {
		return m.styles.Panel.Render(i18n.T(i18n.QuizNoQuestion))
	}
	if m.currentQ.AnswerMode == store.AnswerModeWrite {
		return m.renderWriteQuiz()
	}
	return m.renderChoiceQuiz()
}

func (m RootModel) renderChoiceQuiz() string {
	lines := []string{
		m.styles.Title.Render(m.currentQ.Word.Lemma),
		"",
		m.styles.QuizMeta.Render(fmt.Sprintf("%s  •  %s  •  %s  •  %d/%d", answerModeLabel(m.currentQ.AnswerMode), m.currentQ.Word.Pos, kindLabel(m.currentQ.Kind), m.currentQ.Ordinal, m.currentQ.Total)),
		"",
	}
	for i, choice := range m.currentQ.Choices {
		if i == m.cursor {
			text := fmt.Sprintf("▸ %d. %s", i+1, choice.Meaning)
			lines = append(lines, m.styles.ChoiceSelected.Render(text))
		} else {
			text := fmt.Sprintf("  %d. %s", i+1, choice.Meaning)
			lines = append(lines, m.styles.Choice.Render(text))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.QuizKeysChoice)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderWriteQuiz() string {
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.AnswerModeWrite)),
		"",
		m.styles.QuizMeta.Render(fmt.Sprintf("%s  •  %s  •  %s  •  %d/%d", answerModeLabel(m.currentQ.AnswerMode), m.currentQ.Word.Pos, kindLabel(m.currentQ.Kind), m.currentQ.Ordinal, m.currentQ.Total)),
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizMeaning), 14), m.currentQ.Word.MeaningJA),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizWord), 14), renderSlots(m.currentQ.Word.Lemma, m.writeHintIndices)),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizInput), 14), renderSpacedInput(m.writeInput)),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizHints), 14), formatHintCount(m.writeHintCount)),
		"",
		m.styles.Muted.Render(i18n.T(i18n.QuizKeysWrite)),
	}
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderFeedback() string {
	if m.feedback == nil {
		return m.styles.Panel.Render(i18n.T(i18n.FbNoFeedback))
	}
	if m.feedback.Question.AnswerMode == store.AnswerModeWrite {
		return m.renderWriteFeedback()
	}
	return m.renderChoiceFeedback()
}

func (m RootModel) renderChoiceFeedback() string {
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
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbWord), 14), m.feedback.Question.Word.Lemma),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbCorrectAnswer), 14), m.feedback.Question.CorrectChoice().Meaning),
	)

	if !m.feedback.Correct {
		selected := m.feedback.Question.Choices[m.feedback.SelectedIndex].Meaning
		lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbYourAnswer), 14), selected))
	}

	lines = append(lines, fmt.Sprintf("%s: %d ms", tui.AlignLabel(i18n.T(i18n.FbResponseTime), 14), m.feedback.ResponseMS))

	if m.feedback.Question.Word.ExampleEN != "" || m.feedback.Question.Word.ExampleJA != "" {
		lines = append(lines, "")
		if m.feedback.Question.Word.ExampleEN != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbExampleEN), 14), m.feedback.Question.Word.ExampleEN))
		}
		if m.feedback.Question.Word.ExampleJA != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbExampleJA), 14), m.feedback.Question.Word.ExampleJA))
		}
	}
	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.FbKeys)))
	return panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderWriteFeedback() string {
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
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbWord), 14), m.feedback.Question.Word.Lemma),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbMeaning), 14), m.feedback.Question.Word.MeaningJA),
	)

	if !m.feedback.Correct || m.feedback.Skipped {
		answer := m.feedback.SelectedText
		if m.feedback.Skipped {
			answer = i18n.T(i18n.FbSkipped)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbYourAnswer), 14), answer))
	}
	if m.feedback.HintCount > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", tui.AlignLabel(i18n.T(i18n.FbHints), 14), m.feedback.HintCount))
	}
	lines = append(lines, fmt.Sprintf("%s: %d ms", tui.AlignLabel(i18n.T(i18n.FbResponseTime), 14), m.feedback.ResponseMS))

	if m.feedback.Question.Word.ExampleEN != "" || m.feedback.Question.Word.ExampleJA != "" {
		lines = append(lines, "")
		if m.feedback.Question.Word.ExampleEN != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbExampleEN), 14), m.feedback.Question.Word.ExampleEN))
		}
		if m.feedback.Question.Word.ExampleJA != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.FbExampleJA), 14), m.feedback.Question.Word.ExampleJA))
		}
	}

	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.FbKeysWrite)))
	return panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderResults() string {
	if m.summary == nil {
		return m.styles.Panel.Render(i18n.T(i18n.ResultsNoSummary))
	}

	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.ResultsTitle)),
		fmt.Sprintf("%s: %.1f%%", tui.AlignLabel(i18n.T(i18n.ResultsAccuracy), 14), m.summary.Accuracy),
		fmt.Sprintf("%s: %d/%d", tui.AlignLabel(i18n.T(i18n.ResultsCorrect), 14), m.summary.CorrectAnswers, m.summary.TotalQuestions),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.ResultsMix), 14), i18n.Tf(i18n.ResultsMixDetail, m.summary.NewCount, m.summary.ReviewCount, m.summary.RetryCount)),
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
		m.styles.Muted.Render(m.helpScreenTitle()),
	}

	for _, section := range m.helpSections(m.helpReturn) {
		lines = append(lines, "", m.styles.Subtitle.Render(section.title))
		lines = append(lines, section.lines...)
	}

	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.HelpBack)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStatusLine() string {
	msg := m.status
	if msg == "" {
		msg = i18n.T(i18n.StatusReady)
	}
	if m.err != nil {
		return m.styles.Error.Render("status: " + msg)
	}
	return m.styles.Status.Render("status: " + msg)
}

func (m RootModel) renderAnswerModeTabs() string {
	choice := answerModeLabel(store.AnswerModeChoice)
	write := answerModeLabel(store.AnswerModeWrite)
	if store.NormalizeAnswerMode(m.selectedAnswerMode) == store.AnswerModeWrite {
		return m.styles.Choice.Render(choice) + "  " + m.styles.ChoiceSelected.Render(write)
	}
	return m.styles.ChoiceSelected.Render(choice) + "  " + m.styles.Choice.Render(write)
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

func (m RootModel) renderSettingsRow(index int, label, value string) string {
	text := fmt.Sprintf("%s: %s", tui.AlignLabel(label, 18), value)
	if m.settingsCursor == index {
		return m.styles.ChoiceSelected.Render(text)
	}
	return m.styles.Choice.Render(text)
}

type helpSection struct {
	title string
	lines []string
}

func (m RootModel) helpSections(screen Screen) []helpSection {
	if screen == ScreenHome && m.settingsOpen {
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.keymap.Up),
				helpLine(m.keymap.Down),
				helpLine(m.keymap.Left),
				helpLine(m.keymap.Right),
				helpLine(m.keymap.Confirm),
				helpLine(m.keymap.Back),
			}},
			{title: i18n.T(i18n.HelpSectionInput), lines: []string{
				fmt.Sprintf("%-10s %s", "0-9", i18n.T(i18n.HelpSettingsDigits)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.keymap.Help),
				helpLine(m.keymap.Quit),
			}},
		}
	}
	if screen == ScreenHome && m.homeConfirm != nil {
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
	}

	switch screen {
	case ScreenQuiz:
		if m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
					helpLine(m.keymap.Confirm),
					helpLine(m.keymap.Hint),
					helpLine(m.keymap.Skip),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					helpLine(m.keymap.Help),
					helpLine(m.keymap.WriteQuit),
				}},
			}
		}
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
		if m.isWriteFeedback() {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionNav), lines: []string{
					helpLine(m.keymap.Confirm),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					helpLine(m.keymap.Help),
					fmt.Sprintf("%-10s %s", "q", i18n.T(i18n.HelpQuitDisabledWrite)),
				}},
			}
		}
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
				helpLine(m.keymap.ToggleAnswerMode),
				helpLine(m.keymap.Stats),
				helpLine(m.keymap.Settings),
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

func (m RootModel) isWriteFeedback() bool {
	return m.feedback != nil && m.feedback.Question.AnswerMode == store.AnswerModeWrite
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

func (m RootModel) helpScreenTitle() string {
	if m.helpReturn == ScreenHome && m.settingsOpen {
		return i18n.T(i18n.HelpScreenSettings)
	}
	return screenHelpTitle(m.helpReturn)
}
