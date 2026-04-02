package app

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/tui"
	"github.com/harumiWeb/eitango/internal/updatecheck"
)

type Screen int

const (
	ScreenHome Screen = iota
	ScreenQuiz
	ScreenFeedback
	ScreenResults
	ScreenStats
	ScreenHelp
)

const (
	settingsRowQuestionCount = iota
	settingsRowWriteDifficulty
	settingsRowLanguage
	settingsRowCount
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

type settingsSavedMsg struct {
	Settings          config.Settings
	FocusModeDisabled bool
}

type updateCheckedMsg struct {
	Result updatecheck.Result
}

type errMsg struct {
	err error
}

type StartupRequest struct {
	Mode          string
	AnswerMode    string
	ReplaceActive bool
}

type Options struct {
	Plan           session.PlanOptions
	Startup        *StartupRequest
	Settings       config.Settings
	ConfigPath     string
	CurrentVersion string
	UpdateService  updatecheck.Service
}

type RootModel struct {
	store                   *store.Store
	quiz                    *quiz.Service
	planOptions             session.PlanOptions
	startup                 *StartupRequest
	settings                config.Settings
	configPath              string
	currentVersion          string
	updateService           updatecheck.Service
	updateLatestTag         string
	selectedAnswerMode      string
	screen                  Screen
	keymap                  tui.KeyMap
	styles                  tui.Styles
	home                    store.HomeSnapshot
	stats                   stats.Snapshot
	runtime                 *session.Runtime
	currentQ                *quiz.Question
	feedback                *quiz.Feedback
	summary                 *store.SessionSummary
	cursor                  int
	writeInput              string
	writeHintIndices        []int
	writeHintCount          int
	status                  string
	err                     error
	loading                 bool
	settingsOpen            bool
	settingsCursor          int
	settingsInput           string
	settingsEditing         bool
	settingsWriteDifficulty string
	settingsLanguage        string
	helpReturn              Screen
	helpStatus              string
	width                   int
	height                  int
	questionStarted         time.Time
	recentDistracts         []int64
	correctStreak           int
}

func NewModel(store *store.Store, options Options) RootModel {
	settings := options.Settings
	if settings == (config.Settings{}) {
		settings = config.DefaultSettings()
	}

	planOptions := options.Plan.Normalize()
	if options.Plan == (session.PlanOptions{}) {
		planOptions = planOptionsFromSettings(settings)
	}

	return RootModel{
		store:              store,
		quiz:               quiz.NewService(store),
		planOptions:        planOptions,
		startup:            options.Startup,
		settings:           settings,
		configPath:         options.ConfigPath,
		currentVersion:     strings.TrimSpace(options.CurrentVersion),
		updateService:      options.UpdateService,
		selectedAnswerMode: startupAnswerMode(options.Startup),
		screen:             ScreenHome,
		keymap:             tui.NewKeyMap(),
		styles:             tui.NewStyles(),
		loading:            true,
		status:             i18n.T(i18n.StatusLoading),
	}
}

func (m RootModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)
	if cmd := updateCheckCmd(m.updateService, m.currentVersion); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if m.startup != nil {
		cmds = append(cmds, tea.Sequence(
			loadHomeCmd(m.store),
			sessionCmd(m.store, m.quiz, sessionRequest{
				Mode:                m.startup.Mode,
				AnswerMode:          m.startup.AnswerMode,
				WriteModeDifficulty: m.settings.WriteModeDifficulty,
				ReplaceActive:       m.startup.ReplaceActive,
				Plan:                m.planOptions,
			}, m.recentDistracts),
		))
		return tea.Batch(cmds...)
	}
	cmds = append(cmds, loadHomeCmd(m.store))
	return tea.Batch(cmds...)
}

func (m RootModel) sessionRequest(mode string, replaceActive bool) sessionRequest {
	return sessionRequest{
		Mode:                mode,
		AnswerMode:          m.selectedAnswerMode,
		WriteModeDifficulty: m.settings.WriteModeDifficulty,
		ReplaceActive:       replaceActive,
		Plan:                m.planOptions,
	}
}

func (m RootModel) openSettingsOverlay() RootModel {
	m.settingsOpen = true
	m.settingsCursor = settingsRowQuestionCount
	m.settingsInput = strconv.Itoa(m.settings.SessionSize)
	m.settingsEditing = false
	m.settingsWriteDifficulty = config.NormalizeWriteModeDifficulty(m.settings.WriteModeDifficulty)
	m.settingsLanguage = m.settings.Language
	m.err = nil
	m.status = i18n.T(i18n.StatusConfiguringSettings)
	return m
}

func (m RootModel) closeSettingsOverlay() RootModel {
	m.settingsOpen = false
	m.settingsEditing = false
	m.status = m.homeStatus()
	return m
}

func (m RootModel) homeStatus() string {
	if m.home.ActiveSession != nil {
		return i18n.T(i18n.StatusResumeFound)
	}
	return i18n.T(i18n.StatusReady)
}

func (m RootModel) settingsQuestionCount() (int, bool) {
	if m.settingsInput == "" {
		return 0, false
	}
	count, err := strconv.Atoi(m.settingsInput)
	if err != nil || count <= 0 {
		return 0, false
	}
	return count, true
}

func (m RootModel) settingsQuestionDisplay() string {
	if m.settingsInput == "" {
		return "..."
	}
	if m.settingsCursor == settingsRowQuestionCount && m.settingsEditing {
		return m.settingsInput + "_"
	}
	return m.settingsInput
}

func (m RootModel) settingsLanguageLabel() string {
	if m.settingsLanguage == i18n.LangEN {
		return i18n.T(i18n.SettingsLanguageEN)
	}
	return i18n.T(i18n.SettingsLanguageJA)
}

func (m RootModel) settingsWriteDifficultyLabel() string {
	if config.NormalizeWriteModeDifficulty(m.settingsWriteDifficulty) == config.WriteModeDifficultyHard {
		return i18n.T(i18n.SettingsWriteDifficultyHard)
	}
	return i18n.T(i18n.SettingsWriteDifficultyBasic)
}

func (m RootModel) settingsDraft() (config.Settings, bool, bool) {
	count, ok := m.settingsQuestionCount()
	if !ok {
		return config.Settings{}, false, false
	}

	draft := m.settings
	draft.SessionSize = count
	draft.WriteModeDifficulty = config.NormalizeWriteModeDifficulty(m.settingsWriteDifficulty)
	draft.Language = m.settingsLanguage

	focusModeDisabled := draft.FocusModeDefault && draft.SessionSize != m.settings.SessionSize
	if focusModeDisabled {
		draft.FocusModeDefault = false
	}
	return draft, true, focusModeDisabled
}

func planOptionsFromSettings(settings config.Settings) session.PlanOptions {
	options := session.PlanOptions{
		QuestionCount: settings.SessionSize,
		ReviewRatio:   settings.ReviewRatio,
	}
	if settings.FocusModeDefault {
		options.QuestionCount = session.FocusQuestionCount
	}
	return options.Normalize()
}

func startupAnswerMode(startup *StartupRequest) string {
	if startup == nil {
		return store.AnswerModeChoice
	}
	return store.NormalizeAnswerMode(startup.AnswerMode)
}
