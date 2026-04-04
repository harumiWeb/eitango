package app

import (
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/tui"
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
		if err := i18n.Load(msg.Settings.Language); err != nil {
			m.loading = false
			m.err = err
			m.status = err.Error()
			return m, nil
		}
		speaker := m.speakerFactory(audioConfigFromSettings(msg.Settings))
		settings := normalizeAutoplaySetting(msg.Settings, speaker)
		m.loading = false
		m.err = nil
		m.settings = settings
		m.keymap = tui.NewKeyMap()
		m.planOptions = planOptionsFromSettings(settings)
		m.settingsOpen = false
		m.homeConfirm = nil
		m.settingsEditing = false
		m.settingsInput = strconv.Itoa(settings.SessionSize)
		m.settingsWriteDifficulty = config.NormalizeWriteModeDifficulty(settings.WriteModeDifficulty)
		m.settingsAudioEnabled = settings.AudioEnabled
		m.settingsAudioAutoplay = settings.AudioAutoplay
		m.settingsAudioAvailableCached = m.probeSettingsAudioAvailable()
		m.settingsLanguage = settings.Language
		m.speaker = speaker
		if !m.speakerAvailable() {
			m.autoplayEnabled = false
		}
		if msg.FocusModeDisabled {
			m.status = i18n.T(i18n.StatusSettingsSavedFocus)
		} else {
			m.status = i18n.T(i18n.StatusSettingsSaved)
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
	case tea.KeyPressMsg:
		if m.screen == ScreenQuiz && m.currentQ != nil && m.currentQ.AnswerMode == store.AnswerModeWrite {
			switch {
			case key.Matches(msg, m.keymap.WriteQuit):
				return m, tea.Quit
			case (len(msg.Text) != 1 || (msg.Text != "q" && msg.Text != "Q")) && key.Matches(msg, m.keymap.Quit):
				return m, tea.Quit
			}
		} else {
			switch {
			case key.Matches(msg, m.keymap.Quit):
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
	case key.Matches(msg, m.keymap.Help):
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
	case key.Matches(msg, m.keymap.ToggleAnswerMode):
		m.selectedAnswerMode = nextAnswerMode(m.selectedAnswerMode)
		m.status = m.homeStatus()
		return m, nil
	case key.Matches(msg, m.keymap.Stats):
		m.loading = true
		m.status = i18n.T(i18n.StatusLoadingStats)
		return m, loadStatsCmd(m.store)
	case key.Matches(msg, m.keymap.Settings):
		return m.openSettingsOverlay(), nil
	case key.Matches(msg, m.keymap.Review):
		if m.home.ActiveSession != nil {
			return m.openHomeConfirm(m.sessionRequest(store.ModeReview, true), i18n.StatusStartingReview), nil
		}
		return m.startHomeRequest(m.sessionRequest(store.ModeReview, false), i18n.StatusStartingReview)
	case key.Matches(msg, m.keymap.NewSession):
		if m.home.ActiveSession != nil {
			return m.openHomeConfirm(m.sessionRequest(store.ModeLearn, true), i18n.StatusStartingNew), nil
		}
		return m.startHomeRequest(m.sessionRequest(store.ModeLearn, false), i18n.StatusStartingNew)
	case key.Matches(msg, m.keymap.Confirm):
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
	case key.Matches(msg, m.keymap.Back):
		return m.closeHomeConfirm(), nil
	case key.Matches(msg, m.keymap.Confirm):
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
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	case key.Matches(msg, m.keymap.Back), key.Matches(msg, m.keymap.Settings):
		return m.closeSettingsOverlay(), nil
	case key.Matches(msg, m.keymap.Up):
		if m.settingsCursor > settingsRowQuestionCount {
			m.settingsCursor--
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Down):
		if m.settingsCursor < settingsRowCount-1 {
			m.settingsCursor++
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Left):
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
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Right):
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
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Confirm):
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
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	}

	if m.currentQ == nil {
		return m, nil
	}
	if m.currentQ.AnswerMode == store.AnswerModeWrite {
		return m.updateWriteQuiz(msg)
	}
	switch {
	case key.Matches(msg, m.keymap.ToggleAutoplay):
		return m.toggleAutoplay(), nil
	case key.Matches(msg, m.keymap.Speak):
		return m.maybeSpeakCurrentWord()
	}
	if len(m.currentQ.Choices) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keymap.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case key.Matches(msg, m.keymap.Down):
		if m.cursor < len(m.currentQ.Choices)-1 {
			m.cursor++
		}
		return m, nil
	case key.Matches(msg, m.keymap.Select1):
		return m.showChoiceFeedback(0), nil
	case key.Matches(msg, m.keymap.Select2):
		return m.showChoiceFeedback(1), nil
	case key.Matches(msg, m.keymap.Select3):
		return m.showChoiceFeedback(2), nil
	case key.Matches(msg, m.keymap.Select4):
		return m.showChoiceFeedback(3), nil
	case key.Matches(msg, m.keymap.Confirm):
		return m.showChoiceFeedback(m.cursor), nil
	}

	return m, nil
}

func (m RootModel) updateFeedback(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	case key.Matches(msg, m.keymap.ToggleAutoplay):
		return m.toggleAutoplay(), nil
	case key.Matches(msg, m.keymap.Speak):
		return m.maybeSpeakCurrentWord()
	}

	if m.feedback == nil || m.runtime == nil {
		return m, nil
	}

	var rating srs.Rating
	if m.feedback.Question.AnswerMode == store.AnswerModeWrite {
		switch {
		case key.Matches(msg, m.keymap.Confirm):
			m.loading = true
			m.status = i18n.T(i18n.StatusSaving)
			return m, submitAnswerCmd(m.store, m.quiz, m.runtime, *m.feedback, m.feedback.Rating, m.recentDistracts)
		default:
			return m, nil
		}
	}
	switch {
	case key.Matches(msg, m.keymap.Again):
		rating = srs.Again
	case key.Matches(msg, m.keymap.Hard):
		rating = srs.Hard
	case key.Matches(msg, m.keymap.Good):
		rating = srs.Good
	case key.Matches(msg, m.keymap.Easy):
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
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	case key.Matches(msg, m.keymap.Confirm), key.Matches(msg, m.keymap.Back):
		m.loading = true
		m.status = i18n.T(i18n.StatusReturningHome)
		return m, loadHomeCmd(m.store)
	}
	return m, nil
}

func (m RootModel) updateStats(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	case key.Matches(msg, m.keymap.Back), key.Matches(msg, m.keymap.Confirm):
		m.screen = ScreenHome
		m.status = i18n.T(i18n.StatusBackHome)
		return m, nil
	}
	return m, nil
}

func (m RootModel) updateHelp(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Back):
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
	case key.Matches(msg, m.keymap.Hint):
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
	case key.Matches(msg, m.keymap.Skip):
		m = m.showWriteFeedback(true, false)
		return m, m.autoplayCmd()
	case key.Matches(msg, m.keymap.Confirm):
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
