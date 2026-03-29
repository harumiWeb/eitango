package app

import (
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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
		m.stats = msg.Stats
		m.screen = ScreenHome
		m.loading = false
		m.err = nil
		m.status = m.homeStatus()
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
		m.loading = false
		m.err = nil
		m.settings = msg.Settings
		m.keymap = tui.NewKeyMap()
		m.planOptions = planOptionsFromSettings(msg.Settings)
		m.settingsOpen = false
		m.settingsEditing = false
		m.settingsInput = strconv.Itoa(msg.Settings.SessionSize)
		m.settingsLanguage = msg.Settings.Language
		if msg.FocusModeDisabled {
			m.status = i18n.T(i18n.StatusSettingsSavedFocus)
		} else {
			m.status = i18n.T(i18n.StatusSettingsSaved)
		}
		return m, nil
	case updateCheckedMsg:
		if msg.Result.ShouldNotify {
			m.updateLatestTag = msg.Result.Latest.TagName
		} else if !msg.Result.UpdateAvailable {
			m.updateLatestTag = ""
		}
		return m, nil
	case sessionLoadedMsg:
		m.runtime = msg.Runtime
		m.currentQ = &msg.Question
		m.feedback = nil
		m.summary = nil
		m.cursor = 0
		m.loading = false
		m.err = nil
		m.screen = ScreenQuiz
		m.status = i18n.T(i18n.StatusSessionStarted)
		m.questionStarted = time.Now().UTC()
		m.recentDistracts = appendRecent(m.recentDistracts, msg.Question.DistractorIDs()...)
		return m, nil
	case answerSavedMsg:
		m.runtime = msg.Runtime
		m.loading = false
		m.err = nil
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
			m.screen = ScreenQuiz
			m.questionStarted = time.Now().UTC()
			m.recentDistracts = appendRecent(m.recentDistracts, msg.NextQuestion.DistractorIDs()...)
		}
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			m.status = msg.err.Error()
		}
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			switch m.screen {
			case ScreenFeedback:
				m.status = i18n.T(i18n.StatusSelectRating)
				return m, nil
			case ScreenHelp:
				if m.helpReturn == ScreenFeedback {
					m.status = i18n.T(i18n.StatusEscThenRate)
				} else {
					m.status = i18n.T(i18n.StatusEscToReturn)
				}
				return m, nil
			}
			return m, tea.Quit
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

	switch {
	case key.Matches(msg, m.keymap.Stats):
		m.loading = true
		m.status = i18n.T(i18n.StatusLoadingStats)
		return m, loadStatsCmd(m.store)
	case key.Matches(msg, m.keymap.Settings):
		return m.openSettingsOverlay(), nil
	case key.Matches(msg, m.keymap.Review):
		if m.home.ActiveSession != nil {
			m.status = i18n.T(i18n.StatusActiveFound)
			return m, nil
		}
		m.loading = true
		m.status = i18n.T(i18n.StatusStartingReview)
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeReview, false), m.recentDistracts)
	case key.Matches(msg, m.keymap.NewSession):
		m.loading = true
		m.status = i18n.T(i18n.StatusStartingNew)
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeLearn, true), m.recentDistracts)
	case key.Matches(msg, m.keymap.Confirm):
		if m.home.ActiveSession != nil {
			m.loading = true
			m.status = i18n.T(i18n.StatusResuming)
			return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeLearn, false), m.recentDistracts)
		}
		m.loading = true
		m.status = i18n.T(i18n.StatusStartingLearn)
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeLearn, false), m.recentDistracts)
	}

	return m, nil
}

func (m RootModel) updateSettingsOverlay(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	case key.Matches(msg, m.keymap.Back), key.Matches(msg, m.keymap.Settings):
		return m.closeSettingsOverlay(), nil
	case key.Matches(msg, m.keymap.Up):
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Down):
		if m.settingsCursor < 1 {
			m.settingsCursor++
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Left):
		switch m.settingsCursor {
		case 0:
			count, ok := m.settingsQuestionCount()
			if !ok || count <= 1 {
				count = 1
			} else {
				count--
			}
			m.settingsInput = strconv.Itoa(count)
		case 1:
			m.settingsLanguage = i18n.LangJA
		}
		m.settingsEditing = false
		m.status = i18n.T(i18n.StatusConfiguringSettings)
		return m, nil
	case key.Matches(msg, m.keymap.Right):
		switch m.settingsCursor {
		case 0:
			count, ok := m.settingsQuestionCount()
			if !ok {
				count = 0
			}
			count++
			m.settingsInput = strconv.Itoa(count)
		case 1:
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

	if m.settingsCursor == 0 {
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

	if m.settingsCursor == 0 && len(msg.Text) == 1 && msg.Text[0] >= '0' && msg.Text[0] <= '9' {
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

	if m.currentQ == nil || len(m.currentQ.Choices) == 0 {
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
		return m.showFeedback(0), nil
	case key.Matches(msg, m.keymap.Select2):
		return m.showFeedback(1), nil
	case key.Matches(msg, m.keymap.Select3):
		return m.showFeedback(2), nil
	case key.Matches(msg, m.keymap.Select4):
		return m.showFeedback(3), nil
	case key.Matches(msg, m.keymap.Confirm):
		return m.showFeedback(m.cursor), nil
	}

	return m, nil
}

func (m RootModel) updateFeedback(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Help):
		return m.openHelp(), nil
	}

	if m.feedback == nil || m.runtime == nil {
		return m, nil
	}

	var rating srs.Rating
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

func (m RootModel) showFeedback(index int) RootModel {
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

func appendRecent(existing []int64, ids ...int64) []int64 {
	combined := append(append([]int64{}, existing...), ids...)
	if len(combined) <= 12 {
		return combined
	}
	return combined[len(combined)-12:]
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
