package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/quiz"
	"github.com/yourname/eitango/internal/session"
	"github.com/yourname/eitango/internal/stats"
	"github.com/yourname/eitango/internal/store"
	"github.com/yourname/eitango/internal/tui"
)

type Screen int

const (
	ScreenHome Screen = iota
	ScreenQuiz
	ScreenFeedback
	ScreenResults
	ScreenStats
)

type homeLoadedMsg struct {
	Home  store.HomeSnapshot
	Stats stats.Snapshot
}

type statsLoadedMsg struct {
	Snapshot stats.Snapshot
}

type sessionLoadedMsg struct {
	Runtime  *session.Runtime
	Question quiz.Question
}

type answerSavedMsg struct {
	Runtime      *session.Runtime
	NextQuestion *quiz.Question
	Summary      *store.SessionSummary
	Status       string
}

type errMsg struct {
	err error
}

type StartupRequest struct {
	Mode          string
	ReplaceActive bool
}

type Options struct {
	Plan    session.PlanOptions
	Startup *StartupRequest
}

type RootModel struct {
	store           *store.Store
	quiz            *quiz.Service
	planOptions     session.PlanOptions
	startup         *StartupRequest
	screen          Screen
	keymap          tui.KeyMap
	styles          tui.Styles
	home            store.HomeSnapshot
	stats           stats.Snapshot
	runtime         *session.Runtime
	currentQ        *quiz.Question
	feedback        *quiz.Feedback
	summary         *store.SessionSummary
	cursor          int
	status          string
	err             error
	loading         bool
	width           int
	height          int
	questionStarted time.Time
	recentDistracts []int64
}

func NewModel(store *store.Store, options Options) RootModel {
	return RootModel{
		store:       store,
		quiz:        quiz.NewService(store),
		planOptions: options.Plan.Normalize(),
		startup:     options.Startup,
		screen:      ScreenHome,
		keymap:      tui.NewKeyMap(),
		styles:      tui.NewStyles(),
		loading:     true,
		status:      "Loading...",
	}
}

func (m RootModel) Init() tea.Cmd {
	if m.startup != nil {
		return tea.Sequence(
			loadHomeCmd(m.store),
			sessionCmd(m.store, m.quiz, sessionRequest{
				Mode:          m.startup.Mode,
				ReplaceActive: m.startup.ReplaceActive,
				Plan:          m.planOptions,
			}, m.recentDistracts),
		)
	}
	return loadHomeCmd(m.store)
}

func (m RootModel) sessionRequest(mode string, replaceActive bool) sessionRequest {
	return sessionRequest{
		Mode:          mode,
		ReplaceActive: replaceActive,
		Plan:          m.planOptions,
	}
}
