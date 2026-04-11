package app

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/tui"
	"github.com/mattn/go-runewidth"
)

const (
	compactWidthStandard = 28
	compactWidthWide     = 32
	adaptiveLabelWidth   = 8
	homeLabelWidth       = 14
)

var loadingSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func loadingSpinnerFrameCount() int {
	return len(loadingSpinnerFrames)
}

type layoutVariant int

const (
	layoutAdaptive layoutVariant = iota
	layoutNarrow
)

type layoutSpec struct {
	minWidth int
	title    string
	modal    bool
}

func (m RootModel) View() tea.View {
	body := ""
	if m.width <= 0 {
		body = m.renderLegacyScreen()
	} else {
		spec, variant, _ := m.currentLayout()
		if variant == layoutNarrow {
			body = m.renderNarrowWidthMessage(spec)
		} else {
			body = m.renderScreen()
		}
	}

	if loading := m.renderLoadingFooter(); loading != "" {
		body += "\n\n" + loading
	}
	body += "\n" + m.renderStatusLine()
	view := tea.NewView(body)
	if m.screen == ScreenKeymap {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
}

func (m RootModel) renderLegacyScreen() string {
	switch m.screen {
	case ScreenHome:
		switch {
		case m.settingsOpen:
			return m.renderHomeWithSettingsOverlay()
		case m.homeConfirm != nil:
			return m.renderHomeWithConfirmOverlay()
		default:
			return m.renderHome()
		}
	case ScreenQuiz:
		return m.renderQuiz()
	case ScreenFeedback:
		return m.renderFeedback()
	case ScreenResults:
		return m.renderResults()
	case ScreenStats:
		return m.renderStats()
	case ScreenHelp:
		return m.renderHelp()
	case ScreenKeymap:
		return m.renderKeymapEditor()
	default:
		return ""
	}
}

func (m RootModel) renderScreen() string {
	switch m.screen {
	case ScreenHome:
		return m.renderHomeScreen()
	case ScreenQuiz:
		return m.renderQuizScreen()
	case ScreenFeedback:
		return m.renderFeedbackScreen()
	case ScreenResults:
		return m.renderResultsCompact()
	case ScreenStats:
		return m.renderStatsCompact()
	case ScreenHelp:
		return m.renderHelpCompact()
	case ScreenKeymap:
		return m.renderKeymapEditorCompact()
	default:
		return ""
	}
}

func (m RootModel) renderHomeScreen() string {
	switch {
	case m.settingsOpen:
		return m.renderSettingsOverlayCompact()
	case m.homeConfirm != nil:
		return m.renderHomeConfirmOverlayCompact()
	default:
		return m.renderHomeCompact()
	}
}

func (m RootModel) renderQuizScreen() string {
	return m.renderQuizCompact()
}

func (m RootModel) renderFeedbackScreen() string {
	return m.renderFeedbackCompact()
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
		m.renderSettingsRow(settingsRowAudioVoice, i18n.T(i18n.SettingsAudioVoice), m.settingsAudioVoiceLabel()),
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
	if m.homeConfirm == nil {
		return m.renderHome()
	}
	if m.homeConfirm.Kind == homeConfirmDiscard && m.home.ActiveSession == nil {
		return m.renderHome()
	}

	request := m.homeConfirm.Request
	lines := []string{
		m.styles.Title.Render(m.homeConfirmTitle()),
		"",
		m.homeConfirmBody(),
	}
	if active := m.home.ActiveSession; active != nil {
		lines = append(lines,
			"",
			fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.HomeConfirmCurrent), 14), i18n.Tf(i18n.HomeActiveDetail, active.AnsweredQuestions, active.TotalQuestions, sessionModeLabel(active.Mode), answerModeLabel(active.AnswerMode))),
		)
	}
	lines = append(lines,
		"",
		fmt.Sprintf("%s: %s", tui.AlignLabel(i18n.T(i18n.HomeConfirmTarget), 14), m.homeConfirmTarget(request)),
		"",
		m.styles.Muted.Render(m.renderInlineGuides(
			keymap.ContextHomeConfirm,
			keymap.ActionConfirm,
			keymap.ActionBack,
		)),
	)
	return m.styles.ModalPanel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderHomeCompact() string {
	style := m.compactPanelStyle(false)
	width := m.panelContentWidth(style)
	title := tui.Logo
	if width > 0 && width < 36 {
		title = "eitango"
	}

	lines := []string{
		m.styles.Title.Render(title),
		m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.HomeSubtitle), style)),
		"",
		m.renderCompactStyledField(style, i18n.T(i18n.HomeAnswerMode), m.renderAnswerModeTabs(), m.renderAnswerModeTabsPlain(), homeLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.HomeDue), fmt.Sprintf("%d", m.home.DueCount), homeLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.HomeNew), fmt.Sprintf("%d", m.home.NewCount), homeLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.HomeStreak), fmt.Sprintf("%d", m.home.StreakDays), homeLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.HomeWait), fmt.Sprintf("%.1f min", m.stats.Today.WaitMinutes), homeLabelWidth),
	}
	if m.home.ActiveSession != nil {
		lines = append(lines,
			"",
			m.styles.Subtitle.Render(i18n.T(i18n.HomeActive)),
			m.wrapToPanelWidth(i18n.Tf(i18n.HomeActiveDetail, m.home.ActiveSession.AnsweredQuestions, m.home.ActiveSession.TotalQuestions, sessionModeLabel(m.home.ActiveSession.Mode), answerModeLabel(m.home.ActiveSession.AnswerMode)), style),
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
			m.wrapToPanelWidth(i18n.Tf(i18n.HomeUpdateDetail, m.updateLatestTag, currentVersion), style),
			m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.HomeUpdateHint), style)),
		)
	}
	lines = append(lines,
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextHome,
			keymap.ActionToggleAnswerMode,
			keymap.ActionConfirm,
			keymap.ActionNewSession,
			keymap.ActionReview,
			keymap.ActionStats,
			keymap.ActionSettings,
			keymap.ActionQuit,
		)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderSettingsOverlayCompact() string {
	style := m.compactPanelStyle(true)
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.SettingsTitle)),
		"",
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowQuestionCount, i18n.T(i18n.SettingsQuestions), m.settingsQuestionDisplay()),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowWriteDifficulty, i18n.T(i18n.SettingsWriteDifficulty), m.settingsWriteDifficultyLabel()),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowAudioEnabled, i18n.T(i18n.SettingsAudioEnabled), audioStateLabel(m.settingsAudioEnabled)),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowAudioVoice, i18n.T(i18n.SettingsAudioVoice), m.settingsAudioVoiceLabel()),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowAudioAutoplay, i18n.T(i18n.SettingsAudioAutoplay), audioStateLabel(m.settingsAudioAutoplay && m.settingsAudioAvailable())),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowLanguage, i18n.T(i18n.SettingsLanguage), m.settingsLanguageLabel()),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowTheme, i18n.T(i18n.SettingsTheme), m.settingsThemeModeLabel()),
		m.renderCompactSelectable(style, m.settingsCursor == settingsRowKeymap, i18n.T(i18n.SettingsKeymap), i18n.T(i18n.SettingsKeymapOpen)),
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextSettingsOverlay,
			keymap.ActionUp,
			keymap.ActionDown,
			keymap.ActionLeft,
			keymap.ActionRight,
			keymap.ActionConfirm,
			keymap.ActionBack,
		)),
	}
	if m.settings.FocusModeDefault {
		lines = append(lines, "", m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.SettingsFocusNote), style)))
	}
	if config.NormalizeThemeMode(m.settingsThemeMode) == config.ThemeModeCustom {
		lines = append(lines, "", m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.SettingsThemeCustomNote), style)))
	}
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderHomeConfirmOverlayCompact() string {
	if m.homeConfirm == nil {
		return m.renderHomeCompact()
	}
	if m.homeConfirm.Kind == homeConfirmDiscard && m.home.ActiveSession == nil {
		return m.renderHomeCompact()
	}

	style := m.compactPanelStyle(true)
	request := m.homeConfirm.Request
	lines := []string{
		m.styles.Title.Render(m.homeConfirmTitle()),
		"",
		m.wrapToPanelWidth(m.homeConfirmBody(), style),
	}
	if active := m.home.ActiveSession; active != nil {
		lines = append(lines,
			"",
			m.renderCompactField(style, i18n.T(i18n.HomeConfirmCurrent), i18n.Tf(i18n.HomeActiveDetail, active.AnsweredQuestions, active.TotalQuestions, sessionModeLabel(active.Mode), answerModeLabel(active.AnswerMode))),
		)
	}
	lines = append(lines,
		"",
		m.renderCompactField(style, i18n.T(i18n.HomeConfirmTarget), m.homeConfirmTarget(request)),
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextHomeConfirm,
			keymap.ActionConfirm,
			keymap.ActionBack,
		)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
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

func (m RootModel) renderQuizCompact() string {
	if m.currentQ == nil {
		return m.renderConstrainedPanel(m.compactPanelStyle(false), i18n.T(i18n.QuizNoQuestion))
	}
	if m.currentQ.AnswerMode == store.AnswerModeWrite {
		return m.renderWriteQuizCompact()
	}
	return m.renderChoiceQuizCompact()
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

func (m RootModel) renderChoiceQuizCompact() string {
	style := m.compactPanelStyle(false)
	meta1, meta2 := m.compactQuizMeta()
	lines := []string{
		m.styles.Title.Render(m.truncateToPanelWidth(m.currentQ.Word.Lemma, style)),
		"",
		m.styles.QuizMeta.Render(m.truncateToPanelWidth(meta1, style)),
		m.styles.QuizMeta.Render(m.truncateToPanelWidth(meta2, style)),
		"",
	}
	for i, choice := range m.currentQ.Choices {
		prefix := fmt.Sprintf("  %d. ", i+1)
		styleForChoice := m.styles.Choice
		if i == m.cursor {
			prefix = fmt.Sprintf("▸ %d. ", i+1)
			styleForChoice = m.styles.ChoiceSelected
		}
		choiceWidth := m.panelContentWidth(style) - styleForChoice.GetHorizontalFrameSize()
		lines = append(lines, styleForChoice.Render(renderPrefixedWrap(prefix, choice.Meaning, choiceWidth)))
	}
	lines = append(lines,
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.QuizAudio), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextQuizChoice,
			keymap.ActionUp,
			keymap.ActionDown,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
			keymap.ActionHelp,
			keymap.ActionQuit,
		)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
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

func (m RootModel) renderWriteQuizCompact() string {
	style := m.compactPanelStyle(false)
	meta1, meta2 := m.compactQuizMeta()
	lines := []string{
		m.styles.Title.Render(m.truncateToPanelWidth(i18n.T(i18n.AnswerModeWrite), style)),
		"",
		m.styles.QuizMeta.Render(m.truncateToPanelWidth(meta1, style)),
		m.styles.QuizMeta.Render(m.truncateToPanelWidth(meta2, style)),
		"",
		m.renderCompactAlignedField(style, i18n.T(i18n.QuizMeaning), m.currentQ.Word.MeaningJA, adaptiveLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.QuizWord), renderSlots(m.currentQ.Word.Lemma, m.writeHintIndices), adaptiveLabelWidth),
		m.renderCompactAlignedField(style, i18n.T(i18n.QuizInput), renderSpacedInput(m.writeInput), adaptiveLabelWidth),
		m.renderCompactAlignedFieldEllipsis(style, i18n.T(i18n.QuizHints), formatHintCount(m.writeHintCount), adaptiveLabelWidth),
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextQuizWrite,
			keymap.ActionHint,
			keymap.ActionSkip,
			keymap.ActionConfirm,
			keymap.ActionWriteQuit,
		)),
	}
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
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

func (m RootModel) renderFeedbackCompact() string {
	if m.feedback == nil {
		return m.renderConstrainedPanel(m.compactPanelStyle(false), i18n.T(i18n.FbNoFeedback))
	}
	if m.feedback.Question.AnswerMode == store.AnswerModeWrite {
		return m.renderWriteFeedbackCompact()
	}
	return m.renderChoiceFeedbackCompact()
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
		m.styles.Muted.Render(m.renderChoiceFeedbackGuide()),
	)
	return panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderChoiceFeedbackCompact() string {
	style := m.compactFeedbackPanelStyle()
	lines := []string{m.compactFeedbackHeadline()}

	if m.feedback.Correct && m.correctStreak >= 3 {
		lines = append(lines, m.styles.Correct.Render(m.wrapToPanelWidth(i18n.Tf(i18n.FbStreak, m.correctStreak), style)))
	}

	lines = append(lines,
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.FbWord), m.feedback.Question.Word.Lemma),
		m.renderCompactField(style, i18n.T(i18n.FbCorrectAnswer), m.feedback.Question.CorrectChoice().Meaning),
	)

	if !m.feedback.Correct {
		selected := m.feedback.Question.Choices[m.feedback.SelectedIndex].Meaning
		lines = append(lines, m.renderCompactField(style, i18n.T(i18n.FbYourAnswer), selected))
	}

	lines = append(lines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.FbResponseTime), fmt.Sprintf("%d ms", m.feedback.ResponseMS)))
	lines = append(lines, m.renderCompactFeedbackExamples(style)...)
	lines = append(lines,
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.QuizAudio), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderChoiceFeedbackGuideCompact(style)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
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

func (m RootModel) renderWriteFeedbackCompact() string {
	style := m.compactFeedbackPanelStyle()
	lines := []string{m.compactFeedbackHeadline()}
	if m.feedback.Correct && m.correctStreak >= 3 {
		lines = append(lines, m.styles.Correct.Render(m.wrapToPanelWidth(i18n.Tf(i18n.FbStreak, m.correctStreak), style)))
	}

	lines = append(lines,
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.FbWord), m.feedback.Question.Word.Lemma),
		m.renderCompactField(style, i18n.T(i18n.FbMeaning), m.feedback.Question.Word.MeaningJA),
	)

	if !m.feedback.Correct || m.feedback.Skipped {
		answer := m.feedback.SelectedText
		if m.feedback.Skipped {
			answer = i18n.T(i18n.FbSkipped)
		}
		lines = append(lines, m.renderCompactField(style, i18n.T(i18n.FbYourAnswer), answer))
	}
	if m.feedback.HintCount > 0 {
		lines = append(lines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.FbHints), fmt.Sprintf("%d", m.feedback.HintCount)))
	}
	lines = append(lines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.FbResponseTime), fmt.Sprintf("%d ms", m.feedback.ResponseMS)))
	lines = append(lines, m.renderCompactFeedbackExamples(style)...)

	lines = append(lines,
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.QuizAudio), audioStateLabel(m.autoplayActive())),
		m.styles.Muted.Render(m.renderCompactInlineGuides(style, keymap.ContextFeedbackWrite,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
		)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
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

func (m RootModel) renderResultsCompact() string {
	style := m.compactPanelStyle(false)
	if m.summary == nil {
		return m.renderConstrainedPanel(style, m.truncateToPanelWidth(i18n.T(i18n.ResultsNoSummary), style))
	}

	lines := []string{
		m.styles.Title.Render(m.truncateToPanelWidth(i18n.T(i18n.ResultsTitle), style)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.ResultsAccuracy), fmt.Sprintf("%.1f%%", m.summary.Accuracy)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.ResultsCorrect), fmt.Sprintf("%d/%d", m.summary.CorrectAnswers, m.summary.TotalQuestions)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.ResultsMix), i18n.Tf(i18n.ResultsMixDetail, m.summary.NewCount, m.summary.ReviewCount, m.summary.RetryCount)),
	}
	if len(m.summary.HardWords) > 0 {
		lines = append(lines, "", m.styles.Subtitle.Render(m.truncateToPanelWidth(i18n.T(i18n.ResultsHardWords), style)))
		for _, word := range m.summary.HardWords {
			lines = append(lines, m.renderCompactPrefixedWrap(style, "- ", word.Lemma+": "+word.MeaningJA))
		}
	}
	lines = append(lines,
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style,
			keymap.ContextResults,
			keymap.ActionConfirm,
			keymap.ActionBack,
			keymap.ActionQuit,
		)),
	)
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderStats() string {
	return m.styles.Panel.Render(stats.RenderText(m.stats) + "\n" + m.renderInlineGuides(
		keymap.ContextStats,
		keymap.ActionConfirm,
		keymap.ActionBack,
		keymap.ActionQuit,
	))
}

func (m RootModel) renderStatsCompact() string {
	style := m.compactPanelStyle(false)
	lines := []string{
		m.styles.Title.Render(m.truncateToPanelWidth(i18n.T(i18n.StatsTitle), style)),
		m.renderCompactFieldEllipsis(style, m.stats.Today.Label, m.compactStatsWindowValue(m.stats.Today)),
		m.renderCompactFieldEllipsis(style, m.stats.SevenDays.Label, m.compactStatsWindowValue(m.stats.SevenDays)),
		m.renderCompactFieldEllipsis(style, m.stats.ThirtyDays.Label, m.compactStatsWindowValue(m.stats.ThirtyDays)),
		m.renderCompactFieldEllipsis(style, m.stats.Total.Label, m.compactStatsWindowValue(m.stats.Total)),
		"",
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.StatsDue), fmt.Sprintf("%d", m.stats.DueCount)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.StatsNew), fmt.Sprintf("%d", m.stats.NewCount)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.StatsStreak), fmt.Sprintf("%d", m.stats.StreakDays)),
		"",
		m.styles.Muted.Render(m.renderCompactInlineGuides(style,
			keymap.ContextStats,
			keymap.ActionConfirm,
			keymap.ActionBack,
			keymap.ActionQuit,
		)),
	}
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderHelp() string {
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.HelpTitle)),
		m.styles.Muted.Render(m.helpScreenTitle()),
	}

	for _, section := range m.helpSections(m.helpReturn, false) {
		lines = append(lines, "", m.styles.Subtitle.Render(section.title))
		lines = append(lines, section.lines...)
	}

	lines = append(lines, "", m.styles.Muted.Render(i18n.T(i18n.HelpBack)))
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderHelpCompact() string {
	style := m.compactPanelStyle(false)
	lines := []string{
		m.styles.Title.Render(i18n.T(i18n.HelpTitle)),
		m.styles.Muted.Render(m.wrapToPanelWidth(m.helpScreenTitle(), style)),
	}

	for _, section := range m.helpSections(m.helpReturn, true) {
		lines = append(lines, "", m.styles.Subtitle.Render(m.wrapToPanelWidth(section.title, style)))
		for _, line := range section.lines {
			lines = append(lines, m.wrapToPanelWidth(line, style))
		}
	}

	lines = append(lines, "", m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.HelpBack), style)))
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderStatusLine() string {
	msg := m.status
	if msg == "" {
		msg = i18n.T(i18n.StatusReady)
	}
	prefix := ""
	if m.settingsLoading {
		prefix = loadingSpinnerFrames[m.loadingFrame%loadingSpinnerFrameCount()] + " "
	}
	if m.err != nil {
		return m.styles.Error.Render(m.wrapToWindow(prefix + "error: " + msg))
	}
	return m.styles.Status.Render(m.wrapToWindow(prefix + "status: " + msg))
}

func (m RootModel) renderLoadingFooter() string {
	if m.settingsLoading {
		return ""
	}
	if !m.loading {
		return ""
	}
	return m.styles.Muted.Render(m.wrapToWindow(i18n.T(i18n.StatusLoading)))
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
	if available > 0 && len(lines) > available {
		lines = lines[:available]
	}
	return m.styles.Panel.Render(strings.Join(lines, "\n"))
}

func (m RootModel) renderKeymapEditorCompact() string {
	style := m.compactPanelStyle(false)
	if m.keymapEditor == nil {
		return m.renderConstrainedPanel(style, m.truncateToPanelWidth(i18n.T(i18n.KeymapTitle), style))
	}

	editor := m.keymapEditor
	headerLines := []string{
		m.styles.Title.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapTitle), style)),
		m.renderCompactFieldEllipsis(style, i18n.T(i18n.KeymapContext), m.keymapFilterLabel(editor.filter)),
		"",
	}

	rows := editor.rows()
	detailLines := []string{}
	if row, ok := editor.selectedRow(); ok {
		detailLines = append(detailLines, "")
		detailLines = append(detailLines, m.styles.Subtitle.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapDetails), style)))
		detailLines = append(detailLines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.KeymapAction), keymap.ActionLabel(row.Action)))
		detailLines = append(detailLines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.KeymapDefault), keymap.FormatKeys(keymap.DefaultKeys(row.Context, row.Action))))
		detailLines = append(detailLines, m.renderCompactFieldEllipsis(style, i18n.T(i18n.KeymapCurrent), keymap.FormatKeys(editor.draft.Keys(row.Context, row.Action))))
		if row.Context == keymap.ContextQuizWrite {
			detailLines = append(detailLines, m.styles.Muted.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapWriteNote), style)))
		}
	}

	recordingLines := []string{}
	if editor.recording {
		recordingLines = append(recordingLines, "")
		recordingLines = append(recordingLines, m.styles.Subtitle.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapRecordingTitle), style)))
		recordingLines = append(recordingLines, m.styles.Muted.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapRecordingBody), style)))
	}

	conflictLines := []string{}
	if editor.conflict != nil {
		conflictLines = append(conflictLines, "")
		conflictLines = append(conflictLines, m.styles.Subtitle.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapConflictTitle), style)))
		conflictLines = append(conflictLines, m.styles.Muted.Render(m.truncateToPanelWidth(i18n.Tf(i18n.KeymapConflictBody, keymap.FormatKeys([]string{editor.conflict.Token}), keymap.ActionLabel(editor.conflict.Conflicts[0].Action), keymap.ContextLabel(editor.conflict.Conflicts[0].Context)), style)))
		conflictLines = append(conflictLines, m.styles.Muted.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapConflictKeys), style)))
	}

	footerLines := []string{"", m.styles.Muted.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapKeys), style))}

	available := m.keymapEditorInnerHeightForStyle(style)
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
		listLines = append(listLines, m.styles.Muted.Render(m.truncateToPanelWidth(i18n.T(i18n.KeymapEmpty), style)))
	} else {
		start, end := m.keymapEditorRowWindowForStyle(style, len(rows), editor.cursor, fixedLines)
		scrollbar := m.keymapEditorScrollbar(len(rows), start, end)
		for index := start; index < end; index++ {
			marker := ""
			if len(scrollbar) > 0 {
				marker = scrollbar[index-start]
			}
			listLines = append(listLines, m.renderKeymapEditorRowCompact(style, editor, index, rows[index], marker))
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
	if available > 0 && len(lines) > available {
		lines = lines[:available]
	}
	return m.renderConstrainedPanel(style, strings.Join(lines, "\n"))
}

func (m RootModel) renderKeymapEditorRowCompact(style lipgloss.Style, editor *keymapEditorState, index int, row keymapEditorRow, scrollbar string) string {
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
	rowStyle := m.styles.Choice
	if editor.cursor == index {
		prefix = "> "
		rowStyle = m.styles.ChoiceSelected
	}

	contentWidth := m.panelContentWidth(style)
	contentWidth -= rowStyle.GetHorizontalFrameSize()
	if scrollbar != "" {
		contentWidth -= 2
	}
	left := fmt.Sprintf("%s/%s: %s", keymap.ContextLabel(row.Context), keymap.ActionLabel(row.Action), value)
	line := fitCompactKeymapRow(prefix, left, status, contentWidth)
	if scrollbar != "" {
		return rowStyle.Render(line) + " " + scrollbar
	}
	return rowStyle.Render(line)
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

func (m RootModel) keymapEditorRowWindowForStyle(style lipgloss.Style, totalRows, cursor, fixedLines int) (int, int) {
	if totalRows <= 0 {
		return 0, 0
	}

	available := m.keymapEditorInnerHeightForStyle(style)
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

func (m RootModel) keymapEditorInnerHeightForStyle(style lipgloss.Style) int {
	if m.height <= 0 {
		return 0
	}

	reserved := m.viewFooterHeight()
	available := m.height - reserved - style.GetVerticalFrameSize()
	if available < 1 {
		return 1
	}
	return available
}

func (m RootModel) keymapEditorInnerHeight() int {
	if m.height <= 0 {
		return 0
	}

	reserved := m.viewFooterHeight()
	available := m.height - reserved - m.styles.Panel.GetVerticalFrameSize()
	if available < 1 {
		return 1
	}
	return available
}

func (m RootModel) viewFooterHeight() int {
	height := lipgloss.Height(m.renderStatusLine())
	if loading := m.renderLoadingFooter(); loading != "" {
		height += 2 + lipgloss.Height(loading)
	}
	return height
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

func (m RootModel) renderAnswerModeTabsPlain() string {
	choice := answerModeLabel(store.AnswerModeChoice)
	write := answerModeLabel(store.AnswerModeWrite)
	if store.NormalizeAnswerMode(m.selectedAnswerMode) == store.AnswerModeWrite {
		return choice + "  [" + write + "]"
	}
	return "[" + choice + "]  " + write
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

func (m RootModel) helpSections(screen Screen, compact bool) []helpSection {
	renderBinding := helpLine
	renderDisabled := disabledHelpLine
	if compact {
		renderBinding = compactHelpLine
		renderDisabled = compactDisabledHelpLine
	}

	if screen == ScreenHome && m.settingsOpen {
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionUp)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionDown)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionLeft)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionRight)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionInput), lines: []string{
				i18n.T(i18n.HelpSettingsDigits),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextSettingsOverlay, keymap.ActionQuit)),
			}},
		}
	}
	if screen == ScreenHome && m.homeConfirm != nil {
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				renderBinding(m.binding(keymap.ContextHomeConfirm, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextHomeConfirm, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextHomeConfirm, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextHomeConfirm, keymap.ActionQuit)),
			}},
		}
	}

	switch screen {
	case ScreenQuiz:
		if m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
					renderBinding(m.binding(keymap.ContextQuizWrite, keymap.ActionConfirm)),
					renderBinding(m.binding(keymap.ContextQuizWrite, keymap.ActionHint)),
					renderBinding(m.binding(keymap.ContextQuizWrite, keymap.ActionSkip)),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					renderBinding(m.binding(keymap.ContextQuizWrite, keymap.ActionHelp)),
					renderBinding(m.binding(keymap.ContextQuizWrite, keymap.ActionWriteQuit)),
				}},
			}
		}
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionAnswer), lines: []string{
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect1)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect2)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect3)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionSelect4)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionSpeak)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionToggleAutoplay)),
			}},
			{title: i18n.T(i18n.HelpSectionMove), lines: []string{
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionUp)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionDown)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextQuizChoice, keymap.ActionQuit)),
			}},
		}
	case ScreenFeedback:
		if m.feedbackUsesEnterOnly() {
			return []helpSection{
				{title: i18n.T(i18n.HelpSectionNav), lines: []string{
					renderBinding(m.binding(keymap.ContextFeedbackWrite, keymap.ActionConfirm)),
					renderBinding(m.binding(keymap.ContextFeedbackWrite, keymap.ActionSpeak)),
					renderBinding(m.binding(keymap.ContextFeedbackWrite, keymap.ActionToggleAutoplay)),
				}},
				{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
					renderBinding(m.binding(keymap.ContextFeedbackWrite, keymap.ActionHelp)),
					renderDisabled(m.binding(keymap.ContextFeedbackWrite, keymap.ActionQuit), i18n.T(i18n.HelpQuitDisabledWrite)),
				}},
			}
		}
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionRate), lines: []string{
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionAgain)),
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionHard)),
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionGood)),
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionEasy)),
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionSpeak)),
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionToggleAutoplay)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextFeedbackRate, keymap.ActionHelp)),
				renderDisabled(m.binding(keymap.ContextFeedbackRate, keymap.ActionQuit), i18n.T(i18n.HelpQuitDisabled)),
			}},
		}
	case ScreenResults:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				renderBinding(m.binding(keymap.ContextResults, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextResults, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextResults, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextResults, keymap.ActionQuit)),
			}},
		}
	case ScreenStats:
		return []helpSection{
			{title: i18n.T(i18n.HelpSectionNav), lines: []string{
				renderBinding(m.binding(keymap.ContextStats, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextStats, keymap.ActionBack)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextStats, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextStats, keymap.ActionQuit)),
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
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionConfirm)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionNewSession)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionReview)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionToggleAnswerMode)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionStats)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionSettings)),
			}},
			{title: i18n.T(i18n.HelpSectionGeneral), lines: []string{
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionHelp)),
				renderBinding(m.binding(keymap.ContextHome, keymap.ActionQuit)),
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

func compactHelpLine(binding key.Binding) string {
	help := binding.Help()
	if help.Key == "" {
		help.Key = i18n.T(i18n.KeymapUnbound)
	}
	return help.Key + ": " + help.Desc
}

func compactDisabledHelpLine(binding key.Binding, desc string) string {
	help := binding.Help()
	if help.Key == "" {
		help.Key = i18n.T(i18n.KeymapUnbound)
	}
	return help.Key + ": " + desc
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

func (m RootModel) renderChoiceFeedbackGuide() string {
	if m.isInfiniteReviewRuntime() {
		return m.renderInlineGuides(
			keymap.ContextFeedbackWrite,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
		)
	}
	return m.renderInlineGuides(
		keymap.ContextFeedbackRate,
		keymap.ActionAgain,
		keymap.ActionHard,
		keymap.ActionGood,
		keymap.ActionEasy,
		keymap.ActionSpeak,
		keymap.ActionToggleAutoplay,
	)
}

func (m RootModel) renderChoiceFeedbackGuideCompact(style lipgloss.Style) string {
	if m.isInfiniteReviewRuntime() {
		return m.renderCompactInlineGuides(style,
			keymap.ContextFeedbackWrite,
			keymap.ActionConfirm,
			keymap.ActionSpeak,
			keymap.ActionToggleAutoplay,
		)
	}
	return m.renderCompactInlineGuides(style,
		keymap.ContextFeedbackRate,
		keymap.ActionAgain,
		keymap.ActionHard,
		keymap.ActionGood,
		keymap.ActionEasy,
		keymap.ActionSpeak,
		keymap.ActionToggleAutoplay,
	)
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

func (m RootModel) renderCompactInlineGuides(style lipgloss.Style, ctx keymap.Context, actions ...keymap.Action) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		help := m.binding(ctx, action).Help()
		if help.Key == "" || help.Desc == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", help.Key, help.Desc))
	}
	return packTextPartsWidth(parts, m.panelContentWidth(style), "  ")
}

func (m RootModel) renderCompactField(style lipgloss.Style, label, value string) string {
	width := m.panelContentWidth(style)
	return renderStackedField(label, value, width)
}

func (m RootModel) renderCompactAlignedField(style lipgloss.Style, label, value string, labelWidth int) string {
	if m.panelContentWidth(style) < labelWidth+10 {
		return m.renderCompactField(style, label, value)
	}
	return renderPrefixedWrap(fmt.Sprintf("%s: ", tui.AlignLabel(label, labelWidth)), value, m.panelContentWidth(style))
}

func (m RootModel) renderCompactFieldEllipsis(style lipgloss.Style, label, value string) string {
	return renderSingleLineField(label, value, m.panelContentWidth(style))
}

func (m RootModel) renderCompactAlignedFieldEllipsis(style lipgloss.Style, label, value string, labelWidth int) string {
	if m.panelContentWidth(style) < labelWidth+10 {
		return m.renderCompactFieldEllipsis(style, label, value)
	}
	return renderSingleLinePrefixed(fmt.Sprintf("%s: ", tui.AlignLabel(label, labelWidth)), value, m.panelContentWidth(style))
}

func (m RootModel) renderCompactStyledField(style lipgloss.Style, label, styledValue, plainValue string, labelWidth int) string {
	width := m.panelContentWidth(style)
	if width < labelWidth+8 {
		return renderSingleLineField(label, plainValue, width)
	}
	prefix := fmt.Sprintf("%s: ", tui.AlignLabel(label, labelWidth))
	if runewidth.StringWidth(prefix)+lipgloss.Width(styledValue) <= width {
		return prefix + styledValue
	}
	return renderSingleLinePrefixed(prefix, plainValue, width)
}

func (m RootModel) renderCompactPrefixedWrap(style lipgloss.Style, prefix, value string) string {
	return renderPrefixedWrap(prefix, value, m.panelContentWidth(style))
}

func (m RootModel) renderCompactSelectable(style lipgloss.Style, selected bool, label, value string) string {
	prefix := "  "
	rowStyle := m.styles.Choice
	if selected {
		prefix = "> "
		rowStyle = m.styles.ChoiceSelected
	}
	return rowStyle.Render(renderSingleLinePrefixed(prefix+label+": ", value, m.panelContentWidth(style)))
}

func (m RootModel) compactQuizMeta() (string, string) {
	return fmt.Sprintf("%s • %s • %s", answerModeLabel(m.currentQ.AnswerMode), m.currentQ.Word.Pos, kindLabel(m.currentQ.Kind)),
		fmt.Sprintf("%d/%d", m.currentQ.Ordinal, m.currentQ.Total)
}

func (m RootModel) compactFeedbackHeadline() string {
	if m.feedback.Correct {
		return m.styles.Correct.Render("✓ " + i18n.T(i18n.FbCorrect))
	}
	return m.styles.Wrong.Render("✗ " + i18n.T(i18n.FbIncorrect))
}

func (m RootModel) compactStatsWindowValue(window stats.Window) string {
	return strings.Join([]string{
		fmt.Sprintf("%s=%d", i18n.T(i18n.StatsReviews), window.Reviews),
		fmt.Sprintf("%s=%d", i18n.T(i18n.StatsCorrect), window.Correct),
		fmt.Sprintf("%s=%.1f%%", i18n.T(i18n.StatsAccuracy), window.Accuracy()),
		fmt.Sprintf("%s=%.1fm", i18n.T(i18n.StatsWait), window.WaitMinutes),
	}, "  ")
}

func (m RootModel) renderCompactFeedbackExamples(style lipgloss.Style) []string {
	lines := []string{}
	if m.feedback.Question.Word.ExampleEN != "" || m.feedback.Question.Word.ExampleJA != "" {
		lines = append(lines, "")
		if m.feedback.Question.Word.ExampleEN != "" {
			lines = append(lines, m.renderCompactField(style, i18n.T(i18n.FbExampleEN), m.feedback.Question.Word.ExampleEN))
		}
		if m.feedback.Question.Word.ExampleJA != "" {
			lines = append(lines, m.renderCompactField(style, i18n.T(i18n.FbExampleJA), m.feedback.Question.Word.ExampleJA))
		}
	}
	return lines
}

func (m RootModel) keymapFilterLabel(filter keymap.Context) string {
	if filter == "" {
		return i18n.T(i18n.KeymapFilterAll)
	}
	return keymap.ContextLabel(filter)
}

func (m RootModel) renderNarrowWidthMessage(spec layoutSpec) string {
	lines := []string{
		m.styles.Title.Render(m.wrapToPanelWidth(i18n.T(i18n.NarrowWidthTitle), m.narrowPanelStyle(spec.modal))),
		m.styles.Muted.Render(m.wrapToPanelWidth(spec.title, m.narrowPanelStyle(spec.modal))),
		"",
		m.wrapToPanelWidth(i18n.Tf(i18n.NarrowWidthBody, m.width, spec.minWidth), m.narrowPanelStyle(spec.modal)),
		m.styles.Muted.Render(m.wrapToPanelWidth(i18n.T(i18n.NarrowWidthHint), m.narrowPanelStyle(spec.modal))),
	}
	return m.renderConstrainedPanel(m.narrowPanelStyle(spec.modal), strings.Join(lines, "\n"))
}

func (m RootModel) currentLayout() (layoutSpec, layoutVariant, bool) {
	spec, ok := m.layoutSpec()
	if !ok {
		return layoutSpec{}, layoutAdaptive, false
	}
	if m.width <= 0 {
		return spec, layoutAdaptive, true
	}
	if m.width < spec.minWidth {
		return spec, layoutNarrow, true
	}
	return spec, layoutAdaptive, true
}

func (m RootModel) layoutSpec() (layoutSpec, bool) {
	switch {
	case m.screen == ScreenHome && m.settingsOpen:
		return layoutSpec{minWidth: compactWidthWide, title: i18n.T(i18n.SettingsTitle), modal: true}, true
	case m.screen == ScreenHome && m.homeConfirm != nil:
		return layoutSpec{minWidth: compactWidthWide, title: m.homeConfirmTitle(), modal: true}, true
	case m.screen == ScreenQuiz && m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite:
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.AnswerModeWrite)}, true
	case m.screen == ScreenQuiz:
		return layoutSpec{minWidth: compactWidthWide, title: i18n.T(i18n.HelpScreenQuiz)}, true
	case m.screen == ScreenFeedback && m.isWriteFeedback():
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.HelpScreenFeedback)}, true
	case m.screen == ScreenFeedback && m.feedbackUsesEnterOnly():
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.HelpScreenFeedback)}, true
	case m.screen == ScreenFeedback:
		return layoutSpec{minWidth: compactWidthWide, title: i18n.T(i18n.HelpScreenFeedback)}, true
	case m.screen == ScreenResults:
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.ResultsTitle)}, true
	case m.screen == ScreenStats:
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.StatsTitle)}, true
	case m.screen == ScreenHelp:
		return layoutSpec{minWidth: compactWidthWide, title: i18n.T(i18n.HelpTitle)}, true
	case m.screen == ScreenKeymap:
		return layoutSpec{minWidth: compactWidthWide, title: i18n.T(i18n.KeymapTitle)}, true
	case m.screen == ScreenHome:
		return layoutSpec{minWidth: compactWidthStandard, title: i18n.T(i18n.HelpScreenHome)}, true
	default:
		return layoutSpec{}, false
	}
}

func (m RootModel) homeConfirmTitle() string {
	if m.homeConfirm != nil && m.homeConfirm.Kind == homeConfirmReviewFallback {
		return i18n.T(i18n.HomeReviewFallbackTitle)
	}
	return i18n.T(i18n.HomeConfirmTitle)
}

func (m RootModel) homeConfirmBody() string {
	if m.homeConfirm != nil && m.homeConfirm.Kind == homeConfirmReviewFallback {
		return i18n.T(i18n.HomeReviewFallbackBody)
	}
	return i18n.T(i18n.HomeConfirmBody)
}

func (m RootModel) homeConfirmTarget(request sessionRequest) string {
	target := fmt.Sprintf("%s / %s", sessionModeLabel(request.Mode), answerModeLabel(request.AnswerMode))
	if m.homeConfirm != nil && m.homeConfirm.Kind == homeConfirmReviewFallback {
		return fmt.Sprintf("%s / %s", target, i18n.T(i18n.HomeReviewFallbackPool))
	}
	return target
}

func (m RootModel) narrowPanelStyle(modal bool) lipgloss.Style {
	if modal {
		return m.styles.ModalPanel
	}
	return m.styles.Panel
}

func (m RootModel) compactPanelStyle(modal bool) lipgloss.Style {
	return m.compactPanelBase(modal)
}

func (m RootModel) compactPanelBase(modal bool) lipgloss.Style {
	if modal {
		return m.styles.ModalPanel.Padding(1, 2).Margin(0)
	}
	return m.styles.Panel.Padding(1, 2).Margin(0)
}

func (m RootModel) compactFeedbackPanelStyle() lipgloss.Style {
	if m.feedback != nil && m.feedback.Correct {
		return m.styles.CorrectPanel.Padding(1, 2).Margin(0)
	}
	return m.styles.WrongPanel.Padding(1, 2).Margin(0)
}

func (m RootModel) wrapToWindow(text string) string {
	if m.width <= 0 {
		return text
	}
	return wrapTextWidth(text, m.width)
}

func (m RootModel) wrapToPanelWidth(text string, style lipgloss.Style) string {
	return wrapTextWidth(text, m.panelContentWidth(style))
}

func (m RootModel) truncateToPanelWidth(text string, style lipgloss.Style) string {
	return truncateWithEllipsis(text, m.panelContentWidth(style))
}

func (m RootModel) panelContentWidth(style lipgloss.Style) int {
	if m.width <= 0 {
		return 0
	}
	frameWidth := lipgloss.Width(style.Render(""))
	width := m.width - frameWidth
	if width < 1 {
		return 1
	}
	return width
}

func (m RootModel) renderConstrainedPanel(style lipgloss.Style, text string) string {
	if m.width <= 0 {
		return style.Render(text)
	}
	contentWidth := m.panelContentWidth(style)
	if contentWidth <= 1 || m.width <= style.GetHorizontalFrameSize() {
		return text
	}
	return style.Render(text)
}

func renderStackedField(label, value string, width int) string {
	if value == "" {
		return wrapTextWidth(label, width)
	}
	prefix := label + ": "
	return renderPrefixedWrap(prefix, value, width)
}

func renderSingleLineField(label, value string, width int) string {
	if value == "" {
		return truncateWithEllipsis(label, width)
	}
	return renderSingleLinePrefixed(label+": ", value, width)
}

func renderSingleLinePrefixed(prefix, value string, width int) string {
	if width <= 0 {
		return prefix + value
	}
	prefixWidth := runewidth.StringWidth(prefix)
	if prefixWidth >= width {
		return truncateWithEllipsis(prefix, width)
	}
	return prefix + truncateWithEllipsis(value, width-prefixWidth)
}

func fitCompactKeymapRow(prefix, left, status string, width int) string {
	if width <= 0 {
		return prefix + left + " " + status
	}
	prefixWidth := runewidth.StringWidth(prefix)
	if prefixWidth >= width {
		return truncateWithEllipsis(prefix, width)
	}

	remaining := width - prefixWidth
	status = truncateWithEllipsis(status, remaining)
	if left == "" {
		return prefix + status
	}

	statusWidth := runewidth.StringWidth(status)
	separator := " "
	separatorWidth := runewidth.StringWidth(separator)
	if statusWidth+separatorWidth >= remaining {
		return prefix + truncateWithEllipsis(status, remaining)
	}

	leftWidth := remaining - statusWidth - separatorWidth
	leftText := truncateWithEllipsis(left, leftWidth)
	leftTextWidth := runewidth.StringWidth(leftText)
	if leftTextWidth < leftWidth {
		leftText += strings.Repeat(" ", leftWidth-leftTextWidth)
	}
	return prefix + leftText + separator + status
}

func renderPrefixedWrap(prefix, value string, width int) string {
	if width <= 0 {
		return prefix + value
	}

	prefixWidth := runewidth.StringWidth(prefix)
	if prefixWidth >= width {
		lines := append(wrapSegments(prefix, width), wrapSegments(value, width)...)
		return strings.Join(lines, "\n")
	}

	segments := wrapSegments(value, width-prefixWidth)
	if len(segments) == 0 {
		return strings.TrimSpace(prefix)
	}

	lines := []string{prefix + segments[0]}
	indent := strings.Repeat(" ", prefixWidth)
	for _, segment := range segments[1:] {
		lines = append(lines, indent+segment)
	}
	return strings.Join(lines, "\n")
}

func packTextPartsWidth(parts []string, width int, separator string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	if width <= 0 {
		return strings.Join(filtered, separator)
	}

	lines := []string{truncateWithEllipsis(filtered[0], width)}
	for _, part := range filtered[1:] {
		current := lines[len(lines)-1]
		candidate := current + separator + part
		if runewidth.StringWidth(candidate) <= width {
			lines[len(lines)-1] = candidate
			continue
		}
		part = truncateWithEllipsis(part, width)
		if runewidth.StringWidth(part) <= width {
			lines = append(lines, part)
			continue
		}
		lines = append(lines, truncateWithEllipsis(part, width))
	}
	return strings.Join(lines, "\n")
}

func truncateWithEllipsis(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= width {
		return text
	}
	ellipsis := "..."
	ellipsisWidth := runewidth.StringWidth(ellipsis)
	if width <= ellipsisWidth {
		return strings.Repeat(".", width)
	}

	var b strings.Builder
	currentWidth := 0
	limit := width - ellipsisWidth
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if currentWidth+rw > limit {
			break
		}
		b.WriteRune(r)
		currentWidth += rw
	}
	return b.String() + ellipsis
}

func wrapSegments(text string, width int) []string {
	if width <= 0 || text == "" {
		return []string{text}
	}

	lines := []string{}
	for _, rawLine := range strings.Split(text, "\n") {
		if rawLine == "" {
			lines = append(lines, "")
			continue
		}

		current := strings.Builder{}
		currentWidth := 0
		for _, token := range wrapTokens(rawLine) {
			tokenWidth := runewidth.StringWidth(token)
			if strings.TrimSpace(token) == "" {
				if currentWidth == 0 {
					continue
				}
				if currentWidth+tokenWidth <= width {
					current.WriteString(token)
					currentWidth += tokenWidth
					continue
				}
				lines = append(lines, strings.TrimRight(current.String(), " "))
				current.Reset()
				currentWidth = 0
				continue
			}

			if tokenWidth > width {
				if currentWidth > 0 {
					lines = append(lines, strings.TrimRight(current.String(), " "))
					current.Reset()
					currentWidth = 0
				}
				lines = append(lines, splitTokenWidth(token, width)...)
				continue
			}

			if currentWidth > 0 && currentWidth+tokenWidth > width {
				lines = append(lines, strings.TrimRight(current.String(), " "))
				current.Reset()
				currentWidth = 0
			}
			current.WriteString(token)
			currentWidth += tokenWidth
		}
		if current.Len() > 0 {
			lines = append(lines, strings.TrimRight(current.String(), " "))
		}
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func wrapTokens(text string) []string {
	tokens := make([]string, 0, len(text))
	var current strings.Builder
	currentKind := wrapTokenOther

	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}

	for _, r := range text {
		kind := classifyWrapRune(r)
		if current.Len() > 0 && kind != currentKind {
			flush()
		}
		currentKind = kind
		current.WriteRune(r)
	}
	flush()
	return tokens
}

type wrapTokenKind int

const (
	wrapTokenWhitespace wrapTokenKind = iota
	wrapTokenASCIIWord
	wrapTokenOther
)

func classifyWrapRune(r rune) wrapTokenKind {
	if r == ' ' || r == '\t' {
		return wrapTokenWhitespace
	}
	if r <= unicode.MaxASCII && !unicode.IsSpace(r) {
		return wrapTokenASCIIWord
	}
	return wrapTokenOther
}

func splitTokenWidth(token string, width int) []string {
	if width <= 0 || token == "" {
		return []string{token}
	}

	segments := make([]string, 0, len(token))
	current := strings.Builder{}
	currentWidth := 0
	for _, r := range token {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if currentWidth > 0 && currentWidth+rw > width {
			segments = append(segments, current.String())
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += rw
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	if len(segments) == 0 {
		return []string{""}
	}
	return segments
}

func wrapTextWidth(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, runewidth.Wrap(line, width))
	}
	return strings.Join(wrapped, "\n")
}
