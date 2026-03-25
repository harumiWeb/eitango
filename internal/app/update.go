package app

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/quiz"
	"github.com/yourname/eitango/internal/srs"
	"github.com/yourname/eitango/internal/store"
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
		if m.home.ActiveSession != nil {
			m.status = "Resume session found"
		} else {
			m.status = "Ready"
		}
		return m, nil
	case statsLoadedMsg:
		m.stats = msg.Snapshot
		m.screen = ScreenStats
		m.loading = false
		m.err = nil
		m.status = "Stats loaded"
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
		m.status = "Session started"
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
			if m.screen == ScreenFeedback {
				m.status = "Select a/h/g/e to continue"
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
		}
	}

	return m, nil
}

func (m RootModel) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keymap.Stats):
		m.loading = true
		m.status = "Loading stats..."
		return m, loadStatsCmd(m.store)
	case key.Matches(msg, m.keymap.Review):
		if m.home.ActiveSession != nil {
			m.status = "Active session found. Press Enter to resume or n to replace it."
			return m, nil
		}
		m.loading = true
		m.status = "Starting review session..."
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeReview, false), m.recentDistracts)
	case key.Matches(msg, m.keymap.NewSession):
		m.loading = true
		m.status = "Starting new session..."
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeLearn, true), m.recentDistracts)
	case key.Matches(msg, m.keymap.Confirm):
		m.loading = true
		if m.home.ActiveSession != nil {
			m.status = "Resuming session..."
		} else {
			m.status = "Starting learn session..."
		}
		return m, sessionCmd(m.store, m.quiz, m.sessionRequest(store.ModeLearn, false), m.recentDistracts)
	case key.Matches(msg, m.keymap.Help):
		m.status = "Enter=start/resume, n=new, r=review, s=stats, q=quit"
		return m, nil
	}

	return m, nil
}

func (m RootModel) updateQuiz(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case key.Matches(msg, m.keymap.Help):
		m.status = "1-4=select, j/k=move, enter=confirm, q=save and quit"
		return m, nil
	}

	return m, nil
}

func (m RootModel) updateFeedback(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	m.status = "Saving..."
	return m, submitAnswerCmd(m.store, m.quiz, m.runtime, *m.feedback, rating, m.recentDistracts)
}

func (m RootModel) updateResults(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Confirm), key.Matches(msg, m.keymap.Back):
		m.loading = true
		m.status = "Returning home..."
		return m, loadHomeCmd(m.store)
	case key.Matches(msg, m.keymap.Help):
		m.status = "Enter or Esc to return home"
		return m, nil
	}
	return m, nil
}

func (m RootModel) updateStats(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Back), key.Matches(msg, m.keymap.Confirm):
		m.screen = ScreenHome
		m.status = "Back to home"
		return m, nil
	case key.Matches(msg, m.keymap.Help):
		m.status = "Esc or Enter to go back"
		return m, nil
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
		m.status = "Correct"
	} else {
		m.status = "Check the answer and rate it"
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
