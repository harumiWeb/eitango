package app

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
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
	case ScreenKeymap:
		body = m.renderKeymapEditor()
	}

	if m.loading {
		body += "\n\n" + m.styles.Muted.Render(i18n.T(i18n.StatusLoading))
	}
	body += "\n" + m.renderStatusLine()
	view := tea.NewView(body)
	if m.screen == ScreenKeymap {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
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
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextHome,
			keymap.ActionToggleAnswerMode,
			keymap.ActionConfirm,
			keymap.ActionNewSession,
			keymap.ActionReview,
			keymap.ActionStats,
			keymap.ActionSettings,
			keymap.ActionQuit,
		)),
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
		m.renderSettingsRow(settingsRowAudioEnabled, i18n.T(i18n.SettingsAudioEnabled), audioStateLabel(m.settingsAudioEnabled)),
		m.renderSettingsRow(settingsRowAudioAutoplay, i18n.T(i18n.SettingsAudioAutoplay), audioStateLabel(m.settingsAudioAutoplay && m.settingsAudioAvailable())),
		m.renderSettingsRow(settingsRowLanguage, i18n.T(i18n.SettingsLanguage), m.settingsLanguageLabel()),
		m.renderSettingsRow(settingsRowTheme, i18n.T(i18n.SettingsTheme), m.settingsThemeModeLabel()),
		m.renderSettingsRow(settingsRowKeymap, i18n.T(i18n.SettingsKeymap), i18n.T(i18n.SettingsKeymapOpen)),
		"",
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextSettingsOverlay,
			keymap.ActionUp,
			keymap.ActionDown,
			keymap.ActionLeft,
			keymap.ActionRight,
			keymap.ActionConfirm,
			keymap.ActionBack,
		)),
	}
	if m.settings.FocusModeDefault {
		lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.SettingsFocusNote)))
	}
	if config.NormalizeThemeMode(m.settingsThemeMode) == config.ThemeModeCustom {
		lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.SettingsThemeCustomNote)))
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
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextHomeConfirm,
			keymap.ActionConfirm,
			keymap.ActionBack,
		)),
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
	lines = append(lines,
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizAudio), 14), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextQuizChoice,
			keymap.ActionUp,
			keymap.ActionDown,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
			keymap.ActionHelp,
			keymap.ActionQuit,
		)),
	)
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
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextQuizWrite,
			keymap.ActionHint,
			keymap.ActionSkip,
			keymap.ActionConfirm,
			keymap.ActionWriteQuit,
		)),
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
	lines = append(lines,
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizAudio), 14), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextFeedbackRate,
			keymap.ActionAgain,
			keymap.ActionHard,
			keymap.ActionGood,
			keymap.ActionEasy,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
		)),
	)
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

	lines = append(lines,
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.QuizAudio), 14), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextFeedbackWrite,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
		)),
	)
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
	lines = append(lines, "", m.styles.Muted.Render(m.renderInlineGuides(
		keymap.ContextResults,
		keymap.ActionConfirm,
		keymap.ActionBack,
		keymap.ActionQuit,
	)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderStats() string {
	return m.styles.Panel.Render(stats.RenderText(m.stats) + "\n" + m.renderInlineGuides(
		keymap.ContextStats,
		keymap.ActionConfirm,
		keymap.ActionBack,
		keymap.ActionQuit,
	))
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
		return m.styles.Error.Render("error: " + msg)
	}
	return m.styles.Status.Render("status: " + msg)
}

func (m RootModel) renderKeymapEditor() string {
	if m.keymapEditor == nil {
		return m.styles.Panel.Render(i18n.T(i18n.KeymapTitle))
	}

	editor := m.keymapEditor
	headerLines := []string{
		m.styles.Title.Render(i18n.T(i18n.KeymapTitle)),
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.KeymapContext), 14), m.keymapFilterLabel(editor.filter)),
		"",
	}

	rows := editor.rows()
	detailLines := []string{}
	if row, ok := editor.selectedRow(); ok {
		detailLines = append(detailLines, "")
		detailLines = append(detailLines, m.styles.Subtitle.Render(i18n.T(i18n.KeymapDetails)))
		detailLines = append(detailLines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.KeymapAction), 14), keymap.ActionLabel(row.Action)))
		detailLines = append(detailLines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.KeymapDefault), 14), keymap.FormatKeys(keymap.DefaultKeys(row.Context, row.Action))))
		detailLines = append(detailLines, fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.KeymapCurrent), 14), keymap.FormatKeys(editor.draft.Keys(row.Context, row.Action))))
		if row.Context == keymap.ContextQuizWrite {
			detailLines = append(detailLines, m.styles.Muted.Render(i18n.T(i18n.KeymapWriteNote)))
		}
	}

	recordingLines := []string{}
	if editor.recording {
		recordingLines = append(recordingLines, "")
		recordingLines = append(recordingLines, m.styles.Subtitle.Render(i18n.T(i18n.KeymapRecordingTitle)))
		recordingLines = append(recordingLines, m.styles.Muted.Render(i18n.T(i18n.KeymapRecordingBody)))
	}

	conflictLines := []string{}
	if editor.conflict != nil {
		conflictLines = append(conflictLines, "")
		conflictLines = append(conflictLines, m.styles.Subtitle.Render(i18n.T(i18n.KeymapConflictTitle)))
		conflictLines = append(conflictLines, m.styles.Muted.Render(i18n.Tf(i18n.KeymapConflictBody, keymap.FormatKeys([]string{editor.conflict.Token}), keymap.ActionLabel(editor.conflict.Conflicts[0].Action), keymap.ContextLabel(editor.conflict.Conflicts[0].Context))))
		conflictLines = append(conflictLines, m.styles.Muted.Render(i18n.T(i18n.KeymapConflictKeys)))
	}

	footerLines := []string{"", m.styles.Muted.Render(i18n.T(i18n.KeymapKeys))}

	// Determine which optional sections to render based on available height.
	// Priority: recording > conflict > detail. Each section is only shown when
	// including it still leaves room for at least one action-list row.
	available := m.keymapEditorInnerHeight()
	fixedLines := len(headerLines) + len(footerLines)

	showRecording := len(recordingLines) > 0 && available-fixedLines >= len(recordingLines)+1
	if showRecording {
		fixedLines += len(recordingLines)
	}
	showConflict := len(conflictLines) > 0 && available-fixedLines >= len(conflictLines)+1
	if showConflict {
		fixedLines += len(conflictLines)
	}
	showDetail := len(detailLines) > 0 && available-fixedLines >= len(detailLines)+1
	if showDetail {
		fixedLines += len(detailLines)
	}

	listLines := make([]string, 0, len(rows))
	if len(rows) == 0 {
		listLines = append(listLines, m.styles.Muted.Render(i18n.T(i18n.KeymapEmpty)))
	} else {
		start, end := m.keymapEditorRowWindow(len(rows), editor.cursor, fixedLines)
		scrollbar := m.keymapEditorScrollbar(len(rows), start, end)
		for index := start; index < end; index++ {
			marker := ""
			if len(scrollbar) > 0 {
				marker = scrollbar[index-start]
			}
			listLines = append(listLines, m.renderKeymapEditorRow(editor, index, rows[index], marker))
		}
	}

	lines := append([]string{}, headerLines...)
	lines = append(lines, listLines...)
	if showDetail {
		lines = append(lines, detailLines...)
	}
	if showRecording {
		lines = append(lines, recordingLines...)
	}
	if showConflict {
		lines = append(lines, conflictLines...)
	}
	lines = append(lines, footerLines...)
	// Safety clip: ensure total lines never exceed the available inner height.
	if available > 0 && len(lines) > available {
		lines = lines[:available]
	}
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderKeymapEditorRow(editor *keymapEditorState, index int, row keymapEditorRow, scrollbar string) string {
	current := editor.draft.Keys(row.Context, row.Action)
	value := keymap.FormatKeys(current)
	if value == "" {
		value = i18n.T(i18n.KeymapUnbound)
	}

	status := i18n.T(i18n.KeymapStateDefault)
	if !slices.Equal(current, keymap.DefaultKeys(row.Context, row.Action)) {
		status = i18n.T(i18n.KeymapStateCustom)
	}

	prefix := "  "
	style := m.styles.Choice
	if editor.cursor == index {
		prefix = "> "
		style = m.styles.ChoiceSelected
	}

	line := strings.Join([]string{
		prefix + tui.AlignText(keymap.ContextLabel(row.Context), 18),
		tui.AlignText(keymap.ActionLabel(row.Action), 22),
		tui.AlignText(value, 12),
		status,
	}, " ")
	if scrollbar != "" {
		return style.Render(line) + " " + scrollbar
	}
	return style.Render(line)
}

func (m RootModel) keymapEditorRowWindow(totalRows, cursor, fixedLines int) (int, int) {
	if totalRows <= 0 {
		return 0, 0
	}

	available := m.keymapEditorInnerHeight()
	if available <= 0 {
		return 0, totalRows
	}

	visible := totalRows
	if available <= fixedLines {
		visible = 1
	} else if maxVisible := available - fixedLines; maxVisible < visible {
		visible = maxVisible
	}
	if visible < 1 {
		visible = 1
	}
	if visible >= totalRows {
		return 0, totalRows
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= totalRows {
		cursor = totalRows - 1
	}

	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	maxStart := totalRows - visible
	if start > maxStart {
		start = maxStart
	}
	return start, start + visible
}

func (m RootModel) keymapEditorInnerHeight() int {
	if m.height <= 0 {
		return 0
	}

	reserved := 1 // status line
	if m.loading {
		reserved += 2
	}
	available := m.height - reserved - lipgloss.Height(m.styles.Panel.Render(""))
	if available < 1 {
		return 1
	}
	return available
}

func (m RootModel) keymapEditorScrollbar(totalRows, start, end int) []string {
	visible := end - start
	if totalRows <= visible || visible <= 0 {
		return nil
	}

	markers := make([]string, visible)
	for i := range markers {
		markers[i] = m.styles.Muted.Render("│")
	}

	thumbSize := (visible*visible + totalRows - 1) / totalRows
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > visible {
		thumbSize = visible
	}

	thumbStart := 0
	maxThumbStart := visible - thumbSize
	if maxThumbStart > 0 && totalRows > visible {
		thumbStart = (start * maxThumbStart) / (totalRows - visible)
	}
	for i := thumbStart; i < thumbStart+thumbSize && i < visible; i++ {
		markers[i] = m.styles.Accent.Render("█")
	}
	return markers
}

func (m RootModel) renderAnswerModeTabs() string {
	choice := answerModeLabel(store.AnswerModeChoice)
	write := answerModeLabel(store.AnswerModeWrite)
	if store.NormalizeAnswerMode(m.selectedAnswerMode) == store.AnswerModeWrite {
		return m.styles.Choice.Render(choice) + "  " + m.styles.ChoiceSelected.Render("["+write+"]")
	}
	return m.styles.ChoiceSelected.Render("["+choice+"]") + "  " + m.styles.Choice.Render(write)
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
	prefix := "  "
	if m.settingsCursor == index {
		prefix = "> "
	}
	text := prefix + fmt.Sprintf("%s: %s", tui.AlignLabel(label, 18), value)
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
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionUp)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionDown)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionLeft)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionRight)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionInput), lines: []string{
				i18n.T(i18n.HelpSettingsDigits),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextSettingsOverlay, keymap.ActionQuit)),
			}},
		}
	}
	if screen == ScreenHome && m.homeConfirm != nil {
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.binding(keymap.ContextHomeConfirm, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextHomeConfirm, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextHomeConfirm, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextHomeConfirm, keymap.ActionQuit)),
			}},
		}
	}

	switch screen {
	case ScreenQuiz:
		if m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
					helpLine(m.binding(keymap.ContextQuizWrite, keymap.ActionConfirm)),
					helpLine(m.binding(keymap.ContextQuizWrite, keymap.ActionHint)),
					helpLine(m.binding(keymap.ContextQuizWrite, keymap.ActionSkip)),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					helpLine(m.binding(keymap.ContextQuizWrite, keymap.ActionHelp)),
					helpLine(m.binding(keymap.ContextQuizWrite, keymap.ActionWriteQuit)),
				}},
			}
		}
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect1)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect2)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect3)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect4)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionSpeak)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionToggleAutoplay)),
			}},
			{title: i18n.T(i18n.HelpSectionMove), lines: []string{
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionUp)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionDown)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextQuizChoice, keymap.ActionQuit)),
			}},
		}
	case ScreenFeedback:
		if m.isWriteFeedback() {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionNav), lines: []string{
					helpLine(m.binding(keymap.ContextFeedbackWrite, keymap.ActionConfirm)),
					helpLine(m.binding(keymap.ContextFeedbackWrite, keymap.ActionSpeak)),
					helpLine(m.binding(keymap.ContextFeedbackWrite, keymap.ActionToggleAutoplay)),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					helpLine(m.binding(keymap.ContextFeedbackWrite, keymap.ActionHelp)),
					disabledHelpLine(m.binding(keymap.ContextFeedbackWrite, keymap.ActionQuit), i18n.T(i18n.HelpQuitDisabledWrite)),
				}},
			}
		}
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionRate), lines: []string{
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionAgain)),
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionHard)),
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionGood)),
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionEasy)),
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionSpeak)),
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionToggleAutoplay)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionHelp)),
				disabledHelpLine(m.binding(keymap.ContextFeedbackRate, keymap.ActionQuit), i18n.T(i18n.HelpQuitDisabled)),
			}},
		}
	case ScreenResults:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.binding(keymap.ContextResults, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextResults, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextResults, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextResults, keymap.ActionQuit)),
			}},
		}
	case ScreenStats:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				helpLine(m.binding(keymap.ContextStats, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextStats, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextStats, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextStats, keymap.ActionQuit)),
			}},
		}
	case ScreenKeymap:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				i18n.T(i18n.KeymapKeys),
			}},
		}
	default:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionSessions), lines: []string{
				helpLine(m.binding(keymap.ContextHome, keymap.ActionConfirm)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionNewSession)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionReview)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionToggleAnswerMode)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionStats)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionSettings)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				helpLine(m.binding(keymap.ContextHome, keymap.ActionHelp)),
				helpLine(m.binding(keymap.ContextHome, keymap.ActionQuit)),
			}},
		}
	}
}

func helpLine(binding key.Binding) string {
	help := binding.Help()
	return fmt.Sprintf("%-10s %s", help.Key, help.Desc)
}

func disabledHelpLine(binding key.Binding, desc string) string {
	help := binding.Help()
	if help.Key == "" {
		help.Key = i18n.T(i18n.KeymapUnbound)
	}
	return fmt.Sprintf("%-10s %s", help.Key, desc)
}

func audioStateLabel(enabled bool) string {
	if enabled {
		return i18n.T(i18n.AudioStateOn)
	}
	return i18n.T(i18n.AudioStateOff)
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
	case ScreenKeymap:
		return i18n.T(i18n.HelpScreenKeymap)
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

func (m RootModel) binding(ctx keymap.Context, action keymap.Action) key.Binding {
	return m.keymap.Binding(ctx, action)
}

func (m RootModel) renderInlineGuides(ctx keymap.Context, actions ...keymap.Action) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		help := m.binding(ctx, action).Help()
		if help.Key == "" || help.Desc == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", help.Key, help.Desc))
	}
	return strings.Join(parts, "  ")
}

func (m RootModel) keymapFilterLabel(filter keymap.Context) string {
	if filter == "" {
		return i18n.T(i18n.KeymapFilterAll)
	}
	return keymap.ContextLabel(filter)
}
