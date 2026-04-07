package app

import (
	"slices"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
)

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case homeLoadedMsg:
		m.home = msg.Home
		m.homeConfirm = nil
		if msg.Home.ActiveSession != nil {
			m.selectedAnswerMode = store.NormalizeAnswerMode(msg.Home.ActiveSession.AnswerMode)
		}
		m.stats = msg.Stats
		m.screen = ScreenHome
		m.loading = false
		m.err = nil
		m.status = m.homeStatus()
		return m, nil
	case homeReloadedErrMsg:
		m.home = msg.Home
		if msg.Stats != nil {
			m.stats = *msg.Stats
		}
		m.runtime = nil
		m.currentQ = nil
		m.feedback = nil
		m.summary = nil
		m.screen = ScreenHome
		m.loading = false
		m.homeConfirm = nil
		m.err = msg.err
		if msg.err != nil {
			m.status = msg.err.Error()
		}
		return m, nil
	case statsLoadedMsg:
		m.stats = msg.Snapshot
		m.screen = ScreenStats
		m.loading = false
		m.err = nil
		m.status = i18n.T(i18n.StatusStatsLoaded)
		return m, nil
	case settingsSavedMsg:
		updated, err := m.applySettings(msg.Settings)
		if err != nil {
			m.loading = false
			m.err = err
			m.status = err.Error()
			return m, nil
		}
		m = updated
		m.settingsOpen = false
		m.homeConfirm = nil
		if msg.FocusModeDisabled {
			m.status = i18n.T(i18n.StatusSettingsSavedFocus)
		} else {
			m.status = i18n.T(i18n.StatusSettingsSaved)
		}
		return m, nil
	case keymapSavedMsg:
		updated, err := m.applySettings(msg.Settings)
		if err != nil {
			m.loading = false
			m.err = err
			m.status = err.Error()
			return m, nil
		}
		m = updated
		m.keymapEditor = nil
		m.screen = ScreenHome
		m.settingsOpen = true
		if msg.FocusModeDisabled {
			m.status = i18n.T(i18n.StatusKeymapSavedFocus)
		} else {
			m.status = i18n.T(i18n.StatusKeymapSaved)
		}
		return m, nil
	case updateCheckedMsg:
		if msg.Result.ShouldNotify {
			m.updateLatestTag = msg.Result.Latest.TagName
		} else {
			m.updateLatestTag = ""
		}
		return m, nil
	case sessionLoadedMsg:
		m.runtime = msg.Runtime
		m.selectedAnswerMode = store.NormalizeAnswerMode(msg.Runtime.Session.AnswerMode)
		m.currentQ = &msg.Question
		m.feedback = nil
		m.summary = nil
		m.cursor = 0
		m = m.resetWriteState()
		m.loading = false
		m.err = nil
		m.homeConfirm = nil
		m.screen = ScreenQuiz
		m.status = i18n.T(i18n.StatusSessionStarted)
		m.questionStarted = time.Now().UTC()
		m.recentDistracts = appendRecent(m.recentDistracts, msg.Question.DistractorIDs()...)
		m.autoplayEnabled = m.settings.AudioAutoplay && m.speakerAvailable()
		return m, m.autoplayCmd()
	case answerSavedMsg:
		m.runtime = msg.Runtime
		if msg.Runtime != nil {
			m.selectedAnswerMode = store.NormalizeAnswerMode(msg.Runtime.Session.AnswerMode)
		}
		m.loading = false
		m.err = nil
		m.homeConfirm = nil
		m.status = msg.Status
		if msg.Summary != nil {
			m.summary = msg.Summary
			m.currentQ = nil
			m.feedback = nil
			m.screen = ScreenResults
			return m, nil
		}
		if msg.NextQuestion != nil {
			m.currentQ = msg.NextQuestion
			m.feedback = nil
			m.cursor = 0
			m = m.resetWriteState()
			m.screen = ScreenQuiz
			m.questionStarted = time.Now().UTC()
			m.recentDistracts = appendRecent(m.recentDistracts, msg.NextQuestion.DistractorIDs()...)
			return m, m.autoplayCmd()
		}
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			m.status = msg.err.Error()
		}
		return m, nil
	case audioErrMsg:
		if msg.fromAutoplay {
			m.autoplayEnabled = false
		}
		m.err = msg.err
		m.status = i18n.T(i18n.StatusAudioFailed)
		return m, nil
	case tea.MouseWheelMsg:
		if m.screen == ScreenKeymap {
			return m.updateKeymapEditorWheel(msg)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.screen == ScreenKeymap {
			return m.updateKeymapEditor(msg)
		}
		if m.screen == ScreenQuiz && m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
			switch {
			case m.keymap.Match(keymap.ContextQuizWrite, keymap.ActionWriteQuit, msg):
				return m, tea.Quit
			case m.keymap.Match(keymap.ContextQuizWrite, keymap.ActionQuit, msg):
				return m, tea.Quit
			}
		} else {
			switch {
			case m.matchesQuit(msg):
				switch m.screen {
				case ScreenFeedback:
					if m.feedback != nil && m.feedback.Question.AnswerMode == store.AnswerModeWrite {
						m.status = i18n.T(i18n.StatusWriteContinue)
					} else {
						m.status = i18n.T(i18n.StatusSelectRating)
					}
					return m, nil
				case ScreenHelp:
					if m.helpReturn == ScreenFeedback {
						if m.isWriteFeedback() {
							m.status = i18n.T(i18n.StatusWriteContinue)
						} else {
							m.status = i18n.T(i18n.StatusEscThenRate)
						}
					} else {
						m.status = i18n.T(i18n.StatusEscToReturn)
					}
					return m, nil
				}
				return m, tea.Quit
			}
		}

		switch m.screen {
		case ScreenHome:
			return m.updateHome(msg)
		case ScreenQuiz:
			return m.updateQuiz(msg)
		case ScreenFeedback:
			return m.updateFeedback(msg)
		case ScreenResults:
			return m.updateResults(msg)
		case ScreenStats:
			return m.updateStats(msg)
		case ScreenHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m RootModel) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(m.homeContext(), keymap.ActionHelp, msg):
		return m.openHelp(), nil
	}

	if m.loading {
		return m, nil
	}

	if m.settingsOpen {
		return m.updateSettingsOverlay(msg)
	}
	if m.homeConfirm != nil {
		return m.updateHomeConfirm(msg)
	}

	switch {
	case m.keymap.Match(keymap.ContextHome, keymap.ActionToggleAnswerMode, msg):
		m.selectedAnswerMode = nextAnswerMode(m.selectedAnswerMode)
		m.status = m.homeStatus()
		return m, nil
	case m.keymap.Match(keymap.ContextHome, keymap.ActionStats, msg):
		m.loading = true
		m.status = i18n.T(i18n.StatusLoadingStats)
		return m, loadStatsCmd(m.store)
	case m.keymap.Match(keymap.ContextHome, keymap.ActionSettings, msg):
		return m.openSettingsOverlay(), nil
	case m.keymap.Match(keymap.ContextHome, keymap.ActionReview, msg):
		if m.home.ActiveSession != nil {
			return m.openHomeConfirm(m.sessionRequest(store.ModeReview, true), i18n.StatusStartingReview), nil
		}
		return m.startHomeRequest(m.sessionRequest(store.ModeReview, false), i18n.StatusStartingReview)
	case m.keymap.Match(keymap.ContextHome, keymap.ActionNewSession, msg):
		if m.home.ActiveSession != nil {
			return m.openHomeConfirm(m.sessionRequest(store.ModeLearn, true), i18n.StatusStartingNew), nil
		}
		return m.startHomeRequest(m.sessionRequest(store.ModeLearn, false), i18n.StatusStartingNew)
	case m.keymap.Match(keymap.ContextHome, keymap.ActionConfirm, msg):
		if m.home.ActiveSession != nil {
			if store.NormalizeAnswerMode(m.home.ActiveSession.AnswerMode) != store.NormalizeAnswerMode(m.selectedAnswerMode) {
				return m.openHomeConfirm(m.sessionRequest(store.ModeLearn, true), i18n.StatusStartingLearn), nil
			}
			return m.startHomeRequest(m.sessionRequest(store.ModeLearn, false), i18n.StatusResuming)
		}
		return m.startHomeRequest(m.sessionRequest(store.ModeLearn, false), i18n.StatusStartingLearn)
	}

	return m, nil
}

func (m RootModel) updateHomeConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextHomeConfirm, keymap.ActionBack, msg):
		return m.closeHomeConfirm(), nil
	case m.keymap.Match(keymap.ContextHomeConfirm, keymap.ActionConfirm, msg):
		return m.startHomeRequest(m.homeConfirm.Request, m.homeConfirm.StartStatus)
	}
	return m, nil
}

func (m RootModel) startHomeRequest(request sessionRequest, statusKey string) (tea.Model, tea.Cmd) {
	m.loading = true
	m.homeConfirm = nil
	m.status = i18n.T(statusKey)
	return m, sessionCmd(m.store, m.quiz, request, m.recentDistracts)
}

func (m RootModel) updateSettingsOverlay(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionHelp, msg):
		return m.openHelp(), nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionBack, msg), m.keymap.Match(keymap.ContextHome, keymap.ActionSettings, msg):
		return m.closeSettingsOverlay(), nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionUp, msg):
		if m.settingsCursor > settingsRowQuestionCount {
			m.settingsCursor--
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionDown, msg):
		if m.settingsCursor < settingsRowCount-1 {
			m.settingsCursor++
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionLeft, msg):
		switch m.settingsCursor {
		case settingsRowQuestionCount:
			count, ok := m.settingsQuestionCount()
			if !ok || count <= 1 {
				count = 1
			} else {
				count--
			}
			m.settingsInput = strconv.Itoa(count)
		case settingsRowWriteDifficulty:
			m.settingsWriteDifficulty = config.WriteModeDifficultyBasic
		case settingsRowAudioEnabled:
			m.settingsAudioEnabled = false
			m.settingsAudioAutoplay = false
		case settingsRowAudioAutoplay:
			m.settingsAudioAutoplay = false
		case settingsRowLanguage:
			m.settingsLanguage = i18n.LangJA
		case settingsRowTheme:
			m.settingsThemeMode = previousThemeMode(m.settingsThemeMode)
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionRight, msg):
		switch m.settingsCursor {
		case settingsRowQuestionCount:
			count, ok := m.settingsQuestionCount()
			if !ok {
				count = 0
			}
			count++
			m.settingsInput = strconv.Itoa(count)
		case settingsRowWriteDifficulty:
			m.settingsWriteDifficulty = config.WriteModeDifficultyHard
		case settingsRowAudioEnabled:
			m.settingsAudioEnabled = true
		case settingsRowAudioAutoplay:
			if !m.settingsAudioEnabled {
				m.settingsAudioAutoplay = false
				m.settingsEditing = false
				m.status = m.audioBlockedStatus(false)
				return m, nil
			}
			if !m.settingsAudioAvailable() {
				m.settingsAudioAutoplay = false
				m.settingsEditing = false
				m.status = m.audioBlockedStatus(true)
				return m, nil
			}
			m.settingsAudioAutoplay = true
		case settingsRowLanguage:
			m.settingsLanguage = i18n.LangEN
		case settingsRowTheme:
			m.settingsThemeMode = nextThemeMode(m.settingsThemeMode)
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case m.keymap.Match(keymap.ContextSettingsOverlay, keymap.ActionConfirm, msg):
		if m.settingsCursor == settingsRowKeymap {
			return m.openKeymapEditor(), nil
		}
		settings, ok, focusModeDisabled := m.settingsDraft()
		if !ok {
			m.status = i18n.T(i18n.StatusInvalidCount)
			return m, nil
		}
		m.loading = true
		m.status = i18n.T(i18n.StatusSavingSettings)
		return m, saveSettingsCmd(m.configPath, settings, focusModeDisabled)
	}

	if m.settingsCursor == settingsRowQuestionCount {
		switch msg.Code {
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.settingsInput) > 0 {
				m.settingsInput = m.settingsInput[:len(m.settingsInput)-1]
			}
			m.settingsEditing = true
			m.status = i18n.T(i18n.StatusConfiguringSettings)
			return m, nil
		}
	}

	if m.settingsCursor == settingsRowQuestionCount && len(msg.Text) == 1 && msg.Text[0] >= '0' && msg.Text[0] <= '9' {
		if m.settingsEditing {
			m.settingsInput += msg.Text
		} else {
			m.settingsInput = msg.Text
			m.settingsEditing = true
		}
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	}

	return m, nil
}

func (m RootModel) updateQuiz(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(m.quizContext(), keymap.ActionHelp, msg):
		return m.openHelp(), nil
	}

	if m.currentQ == nil {
		return m, nil
	}
	if m.currentQ.AnswerMode == store.AnswerModeWrite {
		return m.updateWriteQuiz(msg)
	}
	switch {
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionToggleAutoplay, msg):
		return m.toggleAutoplay(), nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionSpeak, msg):
		return m.maybeSpeakCurrentWord()
	}
	if len(m.currentQ.Choices) == 0 {
		return m, nil
	}

	switch {
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionUp, msg):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionDown, msg):
		if m.cursor < len(m.currentQ.Choices)-1 {
			m.cursor++
		}
		return m, nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionSelect1, msg):
		return m.showChoiceFeedback(0), nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionSelect2, msg):
		return m.showChoiceFeedback(1), nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionSelect3, msg):
		return m.showChoiceFeedback(2), nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionSelect4, msg):
		return m.showChoiceFeedback(3), nil
	case m.keymap.Match(keymap.ContextQuizChoice, keymap.ActionConfirm, msg):
		return m.showChoiceFeedback(m.cursor), nil
	}

	return m, nil
}

func (m RootModel) updateFeedback(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(m.feedbackContext(), keymap.ActionHelp, msg):
		return m.openHelp(), nil
	case m.keymap.Match(m.feedbackContext(), keymap.ActionToggleAutoplay, msg):
		return m.toggleAutoplay(), nil
	case m.keymap.Match(m.feedbackContext(), keymap.ActionSpeak, msg):
		return m.maybeSpeakCurrentWord()
	}

	if m.feedback == nil || m.runtime == nil {
		return m, nil
	}

	var rating srs.Rating
	if m.feedback.Question.AnswerMode == store.AnswerModeWrite {
		switch {
		case m.keymap.Match(keymap.ContextFeedbackWrite, keymap.ActionConfirm, msg):
			m.loading = true
			m.status = i18n.T(i18n.StatusSaving)
			return m, submitAnswerCmd(m.store, m.quiz, m.runtime, *m.feedback, m.feedback.Rating, m.recentDistracts)
		default:
			return m, nil
		}
	}
	switch {
	case m.keymap.Match(keymap.ContextFeedbackRate, keymap.ActionAgain, msg):
		rating = srs.Again
	case m.keymap.Match(keymap.ContextFeedbackRate, keymap.ActionHard, msg):
		rating = srs.Hard
	case m.keymap.Match(keymap.ContextFeedbackRate, keymap.ActionGood, msg):
		rating = srs.Good
	case m.keymap.Match(keymap.ContextFeedbackRate, keymap.ActionEasy, msg):
		rating = srs.Easy
	default:
		return m, nil
	}

	m.loading = true
	m.status = i18n.T(i18n.StatusSaving)
	return m, submitAnswerCmd(m.store, m.quiz, m.runtime, *m.feedback, rating, m.recentDistracts)
}

func (m RootModel) updateResults(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextResults, keymap.ActionHelp, msg):
		return m.openHelp(), nil
	case m.keymap.Match(keymap.ContextResults, keymap.ActionConfirm, msg), m.keymap.Match(keymap.ContextResults, keymap.ActionBack, msg):
		m.loading = true
		m.status = i18n.T(i18n.StatusReturningHome)
		return m, loadHomeCmd(m.store)
	}
	return m, nil
}

func (m RootModel) updateStats(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextStats, keymap.ActionHelp, msg):
		return m.openHelp(), nil
	case m.keymap.Match(keymap.ContextStats, keymap.ActionBack, msg), m.keymap.Match(keymap.ContextStats, keymap.ActionConfirm, msg):
		m.screen = ScreenHome
		m.status = i18n.T(i18n.StatusBackHome)
		return m, nil
	}
	return m, nil
}

func (m RootModel) updateHelp(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextHelp, keymap.ActionBack, msg):
		return m.closeHelp(), nil
	}
	return m, nil
}

func (m RootModel) showChoiceFeedback(index int) RootModel {
	if m.currentQ == nil || index < 0 || index >= len(m.currentQ.Choices) {
		return m
	}

	responseMS := time.Since(m.questionStarted).Milliseconds()
	feedback := quiz.BuildFeedback(*m.currentQ, index, responseMS)
	m.feedback = &feedback
	m.screen = ScreenFeedback
	if feedback.Correct {
		m.correctStreak++
		m.status = i18n.T(i18n.StatusCorrect)
	} else {
		m.correctStreak = 0
		m.status = i18n.T(i18n.StatusCheckRate)
	}
	return m
}

func (m RootModel) updateWriteQuiz(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.keymap.Match(keymap.ContextQuizWrite, keymap.ActionHint, msg):
		next := nextHintIndices(m.currentQ.Word.Lemma, m.writeHintIndices, m.writeHintCount)
		if len(next) != len(m.writeHintIndices) {
			m.writeHintIndices = next
			m.writeHintCount++
			if len(next) == len([]rune(m.currentQ.Word.Lemma)) {
				m = m.showWriteFeedback(false, true)
				return m, m.autoplayCmd()
			}
		}
		return m, nil
	case m.keymap.Match(keymap.ContextQuizWrite, keymap.ActionSkip, msg):
		m = m.showWriteFeedback(true, false)
		return m, m.autoplayCmd()
	case m.keymap.Match(keymap.ContextQuizWrite, keymap.ActionConfirm, msg):
		m = m.showWriteFeedback(false, false)
		return m, m.autoplayCmd()
	}

	switch msg.Code {
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.writeInput) > 0 {
			m.writeInput = m.writeInput[:len(m.writeInput)-1]
		}
		return m, nil
	}

	if len(msg.Text) == 1 && isASCIIAlpha(msg.Text[0]) {
		m.writeInput += msg.Text
	}
	return m, nil
}

func (m RootModel) showWriteFeedback(skipped bool, forceIncorrect bool) RootModel {
	if m.currentQ == nil {
		return m
	}

	responseMS := time.Since(m.questionStarted).Milliseconds()
	feedback := quiz.BuildWriteFeedback(*m.currentQ, m.writeInput, m.writeHintCount, skipped, forceIncorrect, responseMS)
	m.feedback = &feedback
	m.screen = ScreenFeedback
	if feedback.Correct {
		m.correctStreak++
		m.status = i18n.T(i18n.StatusCorrect)
	} else {
		m.correctStreak = 0
		m.status = i18n.T(i18n.StatusWriteContinue)
	}
	return m
}

func (m RootModel) resetWriteState() RootModel {
	m.writeInput = ""
	m.writeHintIndices = nil
	m.writeHintCount = 0
	return m
}

func nextAnswerMode(current string) string {
	if store.NormalizeAnswerMode(current) == store.AnswerModeWrite {
		return store.AnswerModeChoice
	}
	return store.AnswerModeWrite
}

func isASCIIAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func appendRecent(existing []int64, ids ...int64) []int64 {
	combined := append(append([]int64{}, existing...), ids...)
	if len(combined) <= 12 {
		return combined
	}
	return combined[len(combined)-12:]
}

func (m RootModel) toggleAutoplay() RootModel {
	if m.autoplayEnabled {
		m.autoplayEnabled = false
		m.status = i18n.T(i18n.StatusAutoplayOff)
		return m
	}
	if !m.speakerAvailable() {
		m.status = m.audioBlockedStatus(m.settings.AudioEnabled)
		return m
	}
	m.autoplayEnabled = true
	m.status = i18n.T(i18n.StatusAutoplayOn)
	return m
}

func (m RootModel) maybeSpeakCurrentWord() (tea.Model, tea.Cmd) {
	text := m.currentAudioText()
	if text == "" {
		return m, nil
	}
	if !m.speakerAvailable() {
		m.status = m.audioBlockedStatus(m.settings.AudioEnabled)
		return m, nil
	}
	return m, speakCmd(m.speaker, text, false)
}

func (m RootModel) audioBlockedStatus(audioEnabled bool) string {
	if !audioEnabled {
		return i18n.T(i18n.StatusAudioDisabled)
	}
	return i18n.T(i18n.StatusAudioUnavailable)
}

func nextThemeMode(current string) string {
	switch config.NormalizeThemeMode(current) {
	case config.ThemeModeDefault:
		return config.ThemeModeNoColor
	case config.ThemeModeNoColor:
		return config.ThemeModeNeon
	case config.ThemeModeNeon:
		return config.ThemeModeCustom
	default:
		return config.ThemeModeDefault
	}
}

func previousThemeMode(current string) string {
	switch config.NormalizeThemeMode(current) {
	case config.ThemeModeDefault:
		return config.ThemeModeCustom
	case config.ThemeModeNoColor:
		return config.ThemeModeDefault
	case config.ThemeModeNeon:
		return config.ThemeModeNoColor
	default:
		return config.ThemeModeNeon
	}
}

func (m RootModel) autoplayCmd() tea.Cmd {
	if !m.autoplayActive() {
		return nil
	}

	text := ""
	switch {
	case m.screen == ScreenFeedback && m.feedback != nil:
		text = m.feedback.Question.Word.Lemma
	case m.currentQ != nil && m.currentQ.AnswerMode != store.AnswerModeWrite:
		text = m.currentQ.Word.Lemma
	}
	if text == "" {
		return nil
	}
	return speakCmd(m.speaker, text, true)
}

func (m RootModel) currentAudioText() string {
	if m.screen == ScreenFeedback && m.feedback != nil {
		return m.feedback.Question.Word.Lemma
	}
	if m.currentQ != nil {
		return m.currentQ.Word.Lemma
	}
	return ""
}

func (m RootModel) openHelp() RootModel {
	if m.screen == ScreenHelp {
		return m
	}
	m.helpReturn = m.screen
	m.helpStatus = m.status
	m.screen = ScreenHelp
	m.status = i18n.T(i18n.StatusHelp)
	return m
}

func (m RootModel) closeHelp() RootModel {
	m.screen = m.helpReturn
	m.status = m.helpStatus
	return m
}

func (m RootModel) updateKeymapEditor(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.keymapEditor == nil {
		return m.closeKeymapEditor(), nil
	}
	editor := *m.keymapEditor
	rows := editor.rows()
	if len(rows) == 0 {
		editor.cursor = 0
	}
	if editor.recording {
		switch msg.String() {
		case "ctrl+g":
			editor.recording = false
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapEditing)
			return m, nil
		}
		token, err := keymap.NormalizeRecordedKey(msg.String())
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		row, ok := editor.selectedRow()
		if !ok {
			editor.recording = false
			m.keymapEditor = &editor
			return m, nil
		}
		conflicts := editor.draft.ConflictsFor(row.Context, row.Action, token)
		if len(conflicts) > 0 {
			editor.recording = false
			editor.conflict = &keymapConflictState{
				Context:   row.Context,
				Action:    row.Action,
				Token:     token,
				Conflicts: conflicts,
			}
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapConflict)
			return m, nil
		}
		keys := editor.draft.Keys(row.Context, row.Action)
		if !slices.Contains(keys, token) {
			keys = append(keys, token)
		}
		if err := editor.draft.SetKeys(row.Context, row.Action, keys); err != nil {
			m.status = err.Error()
			return m, nil
		}
		editor.recording = false
		m.keymapEditor = &editor
		m.status = i18n.T(i18n.StatusKeymapRecorded)
		return m, nil
	}

	if editor.conflict != nil {
		switch msg.Code {
		case tea.KeyEnter:
			if err := editor.draft.ReplaceKey(editor.conflict.Context, editor.conflict.Action, editor.conflict.Token, editor.conflict.Conflicts); err != nil {
				m.status = err.Error()
				return m, nil
			}
			editor.conflict = nil
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapRecorded)
			return m, nil
		case tea.KeyEsc:
			editor.conflict = nil
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapEditing)
			return m, nil
		}
		return m, nil
	}

	switch {
	case msg.String() == "a":
		if _, ok := editor.selectedRow(); ok {
			editor.recording = true
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapRecording)
		}
		return m, nil
	case msg.String() == "d":
		if row, ok := editor.selectedRow(); ok {
			if err := editor.draft.SetKeys(row.Context, row.Action, nil); err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapCleared)
		}
		return m, nil
	case msg.String() == "r":
		if row, ok := editor.selectedRow(); ok {
			if err := editor.draft.SetKeys(row.Context, row.Action, keymap.DefaultKeys(row.Context, row.Action)); err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.keymapEditor = &editor
			m.status = i18n.T(i18n.StatusKeymapReset)
		}
		return m, nil
	case msg.String() == "s":
		settings, ok, focusModeDisabled := m.settingsForKeymapSave()
		if !ok {
			m.status = i18n.T(i18n.StatusInvalidCount)
			return m, nil
		}
		settings.Keymap = editor.draft.ToConfig()
		m.loading = true
		m.status = i18n.T(i18n.StatusSavingSettings)
		return m, saveKeymapCmd(m.configPath, settings, focusModeDisabled)
	case msg.String() == "?":
		m.keymapEditor = &editor
		return m.openHelp(), nil
	}

	switch msg.Code {
	case tea.KeyEsc:
		return m.closeKeymapEditor(), nil
	case tea.KeyUp:
		if editor.cursor > 0 {
			editor.cursor--
		}
		m.keymapEditor = &editor
		return m, nil
	case tea.KeyDown:
		if editor.cursor < len(rows)-1 {
			editor.cursor++
		}
		m.keymapEditor = &editor
		return m, nil
	case tea.KeyLeft:
		editor.shiftFilter(-1)
		m.keymapEditor = &editor
		return m, nil
	case tea.KeyRight:
		editor.shiftFilter(1)
		m.keymapEditor = &editor
		return m, nil
	}

	if m.keymap.Match(keymap.ContextHelp, keymap.ActionBack, msg) {
		return m.closeKeymapEditor(), nil
	}
	return m, nil
}

func (m RootModel) updateKeymapEditorWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.keymapEditor == nil {
		return m, nil
	}

	editor := *m.keymapEditor
	if editor.recording || editor.conflict != nil {
		return m, nil
	}

	rows := editor.rows()
	if len(rows) == 0 {
		return m, nil
	}

	switch msg.Mouse().Button {
	case tea.MouseWheelUp:
		if editor.cursor > 0 {
			editor.cursor--
		}
	case tea.MouseWheelDown:
		if editor.cursor < len(rows)-1 {
			editor.cursor++
		}
	default:
		return m, nil
	}

	m.keymapEditor = &editor
	return m, nil
}

type keymapEditorRow struct {
	Context keymap.Context
	Action  keymap.Action
}

func (e *keymapEditorState) rows() []keymapEditorRow {
	rows := make([]keymapEditorRow, 0)
	contexts := keymap.Contexts()
	for _, ctx := range contexts {
		if e.filter != "" && e.filter != ctx {
			continue
		}
		for _, action := range keymap.ActionsForContext(ctx) {
			rows = append(rows, keymapEditorRow{Context: ctx, Action: action})
		}
	}
	return rows
}

func (e *keymapEditorState) selectedRow() (keymapEditorRow, bool) {
	rows := e.rows()
	if len(rows) == 0 {
		return keymapEditorRow{}, false
	}
	if e.cursor < 0 {
		e.cursor = 0
	}
	if e.cursor >= len(rows) {
		e.cursor = len(rows) - 1
	}
	return rows[e.cursor], true
}

func (e *keymapEditorState) shiftFilter(delta int) {
	filters := append([]keymap.Context{""}, keymap.Contexts()...)
	index := slices.Index(filters, e.filter)
	if index < 0 {
		index = 0
	}
	index = (index + delta + len(filters)) % len(filters)
	e.filter = filters[index]
	e.cursor = 0
}

func (m RootModel) homeContext() keymap.Context {
	if m.settingsOpen {
		return keymap.ContextSettingsOverlay
	}
	if m.homeConfirm != nil {
		return keymap.ContextHomeConfirm
	}
	return keymap.ContextHome
}

func (m RootModel) quizContext() keymap.Context {
	if m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
		return keymap.ContextQuizWrite
	}
	return keymap.ContextQuizChoice
}

func (m RootModel) feedbackContext() keymap.Context {
	if m.isWriteFeedback() {
		return keymap.ContextFeedbackWrite
	}
	return keymap.ContextFeedbackRate
}

func (m RootModel) matchesQuit(msg tea.KeyPressMsg) bool {
	switch m.screen {
	case ScreenHome:
		return m.keymap.Match(m.homeContext(), keymap.ActionQuit, msg)
	case ScreenQuiz:
		return m.keymap.Match(m.quizContext(), keymap.ActionQuit, msg)
	case ScreenFeedback:
		return m.keymap.Match(m.feedbackContext(), keymap.ActionQuit, msg)
	case ScreenResults:
		return m.keymap.Match(keymap.ContextResults, keymap.ActionQuit, msg)
	case ScreenStats:
		return m.keymap.Match(keymap.ContextStats, keymap.ActionQuit, msg)
	case ScreenHelp:
		return m.keymap.Match(keymap.ContextHelp, keymap.ActionQuit, msg)
	default:
		return false
	}
}
