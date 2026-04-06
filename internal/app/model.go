package app

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/audio"
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
	settingsRowAudioEnabled
	settingsRowAudioAutoplay
	settingsRowLanguage
	settingsRowTheme
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

type homeReloadedErrMsg struct {
	Home  store.HomeSnapshot
	Stats *stats.Snapshot
	err   error
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

type audioErrMsg struct {
	fromAutoplay bool
	err          error
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
	SpeakerFactory func(audio.Config) audio.Speaker
}

type homeConfirmState struct {
	Request     sessionRequest
	StartStatus string
}

type RootModel struct {
	store                        *store.Store
	quiz                         *quiz.Service
	planOptions                  session.PlanOptions
	startup                      *StartupRequest
	settings                     config.Settings
	configPath                   string
	currentVersion               string
	updateService                updatecheck.Service
	speaker                      audio.Speaker
	speakerFactory               func(audio.Config) audio.Speaker
	updateLatestTag              string
	selectedAnswerMode           string
	screen                       Screen
	keymap                       tui.KeyMap
	styles                       tui.Styles
	home                         store.HomeSnapshot
	stats                        stats.Snapshot
	runtime                      *session.Runtime
	currentQ                     *quiz.Question
	feedback                     *quiz.Feedback
	summary                      *store.SessionSummary
	cursor                       int
	writeInput                   string
	writeHintIndices             []int
	writeHintCount               int
	status                       string
	err                          error
	loading                      bool
	settingsOpen                 bool
	homeConfirm                  *homeConfirmState
	settingsCursor               int
	settingsInput                string
	settingsEditing              bool
	settingsWriteDifficulty      string
	settingsAudioEnabled         bool
	settingsAudioAutoplay        bool
	settingsAudioAvailableCached bool
	settingsLanguage             string
	settingsThemeMode            string
	helpReturn                   Screen
	helpStatus                   string
	width                        int
	height                       int
	questionStarted              time.Time
	recentDistracts              []int64
	correctStreak                int
	autoplayEnabled              bool
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
	speakerFactory := options.SpeakerFactory
	if speakerFactory == nil {
		speakerFactory = audio.NewSpeaker
	}
	speaker := speakerFactory(audioConfigFromSettings(settings))
	settings = normalizeAutoplaySetting(settings, speaker)
	settings.ThemeMode = config.NormalizeThemeMode(settings.ThemeMode)

	return RootModel{
		store:              store,
		quiz:               quiz.NewService(store),
		planOptions:        planOptions,
		startup:            options.Startup,
		settings:           settings,
		configPath:         options.ConfigPath,
		currentVersion:     strings.TrimSpace(options.CurrentVersion),
		updateService:      options.UpdateService,
		speaker:            speaker,
		speakerFactory:     speakerFactory,
		selectedAnswerMode: startupAnswerMode(options.Startup),
		screen:             ScreenHome,
		keymap:             tui.NewKeyMap(),
		styles:             tui.NewStyles(themeFromSettings(settings)),
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
	m.homeConfirm = nil
	m.settingsCursor = settingsRowQuestionCount
	m.settingsInput = strconv.Itoa(m.settings.SessionSize)
	m.settingsEditing = false
	m.settingsWriteDifficulty = config.NormalizeWriteModeDifficulty(m.settings.WriteModeDifficulty)
	m.settingsAudioEnabled = m.settings.AudioEnabled
	m.settingsAudioAutoplay = m.settings.AudioAutoplay
	m.settingsAudioAvailableCached = m.probeSettingsAudioAvailable()
	if !m.settingsAudioAvailable() {
		m.settingsAudioAutoplay = false
	}
	m.settingsLanguage = m.settings.Language
	m.settingsThemeMode = config.NormalizeThemeMode(m.settings.ThemeMode)
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

func (m RootModel) openHomeConfirm(request sessionRequest, startStatus string) RootModel {
	m.homeConfirm = &homeConfirmState{
		Request:     request,
		StartStatus: startStatus,
	}
	m.err = nil
	m.status = i18n.T(i18n.StatusConfirmDiscard)
	return m
}

func (m RootModel) closeHomeConfirm() RootModel {
	m.homeConfirm = nil
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

func (m RootModel) settingsThemeModeLabel() string {
	switch m.settingsThemeMode {
	case config.ThemeModeNoColor:
		return i18n.T(i18n.SettingsThemeNoColor)
	case config.ThemeModeNeon:
		return i18n.T(i18n.SettingsThemeNeon)
	case config.ThemeModeCustom:
		return i18n.T(i18n.SettingsThemeCustom)
	default:
		return i18n.T(i18n.SettingsThemeDefault)
	}
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
	draft.AudioEnabled = m.settingsAudioEnabled
	draft.AudioAutoplay = m.settingsAudioAutoplay && m.settingsAudioAvailable()
	draft.Language = m.settingsLanguage
	draft.ThemeMode = config.NormalizeThemeMode(m.settingsThemeMode)

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

func audioConfigFromSettings(settings config.Settings) audio.Config {
	return audio.Config{Enabled: settings.AudioEnabled}
}

func themeFromSettings(settings config.Settings) tui.Theme {
	return tui.Theme{
		Mode: config.NormalizeThemeMode(settings.ThemeMode),
		Palette: tui.ThemePalette{
			Accent:  settings.ThemePalette.Accent,
			Success: settings.ThemePalette.Success,
			Danger:  settings.ThemePalette.Danger,
			Muted:   settings.ThemePalette.Muted,
			Border:  settings.ThemePalette.Border,
		},
	}
}

func normalizeAutoplaySetting(settings config.Settings, speaker audio.Speaker) config.Settings {
	if speaker == nil || !speaker.Enabled() {
		settings.AudioAutoplay = false
	}
	return settings
}

func (m RootModel) probeSettingsAudioAvailable() bool {
	settings := m.settings
	settings.AudioEnabled = true
	cfg := audioConfigFromSettings(settings)
	speaker := m.speakerFactory(cfg)
	return speaker != nil && speaker.Enabled()
}

func (m RootModel) speakerAvailable() bool {
	return m.speaker != nil && m.speaker.Enabled()
}

func (m RootModel) settingsAudioAvailable() bool {
	return m.settingsAudioEnabled && m.settingsAudioAvailableCached
}

func (m RootModel) autoplayActive() bool {
	return m.autoplayEnabled && m.speakerAvailable()
}
