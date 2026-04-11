package app

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/audio"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
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
	ScreenKeymap
)

const (
	settingsRowQuestionCount = iota
	settingsRowWriteDifficulty
	settingsRowUpdateCheck
	settingsRowAudioEnabled
	settingsRowAudioVoice
	settingsRowAudioAutoplay
	settingsRowLanguage
	settingsRowTheme
	settingsRowKeymap
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

type reviewFallbackPromptMsg struct {
	Request sessionRequest
}

type quitAfterCleanupMsg struct{}

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

type settingsOverlayLoadedMsg struct {
	voices         []audio.Voice
	voicesLoaded   bool
	audioVoice     string
	audioAvailable bool
}

type updateCheckedMsg struct {
	Result updatecheck.Result
}

type keymapSavedMsg struct {
	Settings          config.Settings
	FocusModeDisabled bool
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
	VoiceCatalog   func() ([]audio.Voice, bool)
}

type homeConfirmKind int

const (
	homeConfirmDiscard homeConfirmKind = iota
	homeConfirmReviewFallback
)

type homeConfirmState struct {
	Kind        homeConfirmKind
	Request     sessionRequest
	StartStatus string
}

type keymapEditorState struct {
	filter    keymap.Context
	cursor    int
	draft     keymap.State
	original  keymap.State
	recording bool
	conflict  *keymapConflictState
}

type keymapConflictState struct {
	Context   keymap.Context
	Action    keymap.Action
	Token     string
	Conflicts []keymap.Conflict
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
	voiceCatalog                 func() ([]audio.Voice, bool)
	updateLatestTag              string
	selectedAnswerMode           string
	screen                       Screen
	keymap                       keymap.State
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
	loadingFrame                 int
	settingsOpen                 bool
	settingsLoading              bool
	homeConfirm                  *homeConfirmState
	settingsCursor               int
	settingsInput                string
	settingsEditing              bool
	settingsWriteDifficulty      string
	settingsUpdateCheckEnabled   bool
	settingsAudioEnabled         bool
	settingsAudioVoice           string
	settingsAudioAutoplay        bool
	settingsAudioVoices          []audio.Voice
	settingsAudioVoicesLoaded    bool
	settingsAudioAvailableCached bool
	settingsLanguage             string
	settingsThemeMode            string
	helpReturn                   Screen
	helpStatus                   string
	keymapEditor                 *keymapEditorState
	width                        int
	height                       int
	questionStarted              time.Time
	recentDistracts              []int64
	correctStreak                int
	autoplayEnabled              bool
}

func NewModel(store *store.Store, options Options) RootModel {
	settings := options.Settings
	if settings.IsZero() {
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
	voiceCatalog := options.VoiceCatalog
	if voiceCatalog == nil {
		voiceCatalog = audio.InstalledVoices
	}
	settings.AudioVoice = normalizeAudioVoiceSetting(settings.AudioVoice, voiceCatalog)
	speaker := speakerFactory(audioConfigFromSettings(settings))
	settings = normalizeAutoplaySetting(settings, speaker)
	settings.ThemeMode = config.NormalizeThemeMode(settings.ThemeMode)
	keyState, err := keymap.Resolve(settings.Keymap)
	if err != nil {
		keyState = keymap.DefaultState()
		settings.Keymap = keyState.ToConfig()
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
		speaker:            speaker,
		speakerFactory:     speakerFactory,
		voiceCatalog:       voiceCatalog,
		selectedAnswerMode: startupAnswerMode(options.Startup),
		screen:             ScreenHome,
		keymap:             keyState,
		styles:             tui.NewStyles(themeFromSettings(settings)),
		err:                err,
		loading:            true,
		status:             i18n.T(i18n.StatusLoading),
	}
}

func (m RootModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)
	if cmd := updateCheckCmd(m.updateService, m.currentVersion, m.settings.UpdateCheckEnabled); cmd != nil {
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
		AllowReviewFallback: false,
		ReplaceActive:       replaceActive,
		Plan:                m.planOptions,
	}
}

// openSettingsOverlay synchronously prepares the settings overlay for tests and
// direct in-memory transitions. Interactive UI flows should use
// startSettingsOverlayLoad so the expensive audio checks run asynchronously.
func (m RootModel) openSettingsOverlay() RootModel {
	m = m.prepareSettingsOverlay()
	voices, loaded, audioVoice, audioAvailable := m.loadSettingsOverlayData()
	return m.applySettingsOverlayLoad(voices, loaded, audioVoice, audioAvailable)
}

func (m RootModel) startSettingsOverlayLoad() RootModel {
	m = m.prepareSettingsOverlay()
	m.loading = true
	m.loadingFrame = 0
	m.settingsLoading = true
	m.status = i18n.T(i18n.StatusLoading)
	return m
}

func (m RootModel) prepareSettingsOverlay() RootModel {
	m.settingsOpen = true
	m.homeConfirm = nil
	m.settingsCursor = settingsRowQuestionCount
	m.settingsInput = strconv.Itoa(m.settings.SessionSize)
	m.settingsEditing = false
	m.settingsWriteDifficulty = config.NormalizeWriteModeDifficulty(m.settings.WriteModeDifficulty)
	m.settingsUpdateCheckEnabled = m.settings.UpdateCheckEnabled
	m.settingsAudioEnabled = m.settings.AudioEnabled
	m.settingsAudioVoices = nil
	m.settingsAudioVoicesLoaded = false
	m.settingsAudioVoice = config.NormalizeAudioVoice(m.settings.AudioVoice)
	m.settingsAudioAutoplay = m.settings.AudioAutoplay
	m.settingsAudioAvailableCached = false
	m.settingsLanguage = m.settings.Language
	m.settingsThemeMode = config.NormalizeThemeMode(m.settings.ThemeMode)
	m.err = nil
	m.status = i18n.T(i18n.StatusConfiguringSettings)
	return m
}

func (m RootModel) applySettingsOverlayLoad(voices []audio.Voice, loaded bool, audioVoice string, audioAvailable bool) RootModel {
	m.settingsAudioVoices = voices
	m.settingsAudioVoicesLoaded = loaded
	m.settingsAudioVoice = normalizeAudioVoiceInList(audioVoice, voices, loaded)
	m.settingsAudioAvailableCached = audioAvailable
	if !m.settingsAudioAvailable() {
		m.settingsAudioAutoplay = false
	}
	m.loading = false
	m.settingsLoading = false
	if m.screen == ScreenHelp {
		m.helpStatus = i18n.T(i18n.StatusConfiguringSettings)
		m.status = i18n.T(i18n.StatusHelp)
		return m
	}
	m.status = i18n.T(i18n.StatusConfiguringSettings)
	return m
}

func (m RootModel) closeSettingsOverlay() RootModel {
	m.loading = false
	m.settingsLoading = false
	m.settingsOpen = false
	m.settingsEditing = false
	m.status = m.homeStatus()
	return m
}

func (m RootModel) openHomeConfirm(request sessionRequest, startStatus string) RootModel {
	m.homeConfirm = &homeConfirmState{
		Kind:        homeConfirmDiscard,
		Request:     request,
		StartStatus: startStatus,
	}
	m.err = nil
	m.status = i18n.T(i18n.StatusConfirmDiscard)
	return m
}

func (m RootModel) openReviewFallbackConfirm(request sessionRequest) RootModel {
	m.homeConfirm = &homeConfirmState{
		Kind:        homeConfirmReviewFallback,
		Request:     request,
		StartStatus: i18n.StatusStartingReview,
	}
	m.err = nil
	m.status = i18n.T(i18n.StatusConfirmReviewFallback)
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

func (m RootModel) settingsAudioVoiceLabel() string {
	if !m.settingsAudioVoicesLoaded {
		return i18n.T(i18n.SettingsAudioVoiceUnavailable)
	}
	if m.settingsAudioVoice == "" {
		return i18n.T(i18n.SettingsAudioVoiceAuto)
	}
	for _, voice := range m.settingsAudioVoices {
		if voice.ID == m.settingsAudioVoice {
			return voice.Label
		}
	}
	return i18n.T(i18n.SettingsAudioVoiceAuto)
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
	draft.UpdateCheckEnabled = m.settingsUpdateCheckEnabled
	draft.AudioEnabled = m.settingsAudioEnabled
	draft.AudioVoice = m.settingsAudioVoice
	draft.AudioAutoplay = m.settingsAudioAutoplay && m.settingsAudioAvailable()
	draft.Language = m.settingsLanguage
	draft.ThemeMode = config.NormalizeThemeMode(m.settingsThemeMode)
	draft.Keymap = m.settings.Keymap

	focusModeDisabled := draft.FocusModeDefault && draft.SessionSize != m.settings.SessionSize
	if focusModeDisabled {
		draft.FocusModeDefault = false
	}
	return draft, true, focusModeDisabled
}

func (m RootModel) settingsForKeymapSave() (config.Settings, bool, bool) {
	if !m.settingsOpen {
		return m.settings, true, false
	}
	return m.settingsDraft()
}

func (m RootModel) applySettings(settings config.Settings) (RootModel, error) {
	if err := i18n.Load(settings.Language); err != nil {
		return m, err
	}
	settings.AudioVoice = normalizeAudioVoiceSetting(settings.AudioVoice, m.voiceCatalog)
	speaker := m.speakerFactory(audioConfigFromSettings(settings))
	settings = normalizeAutoplaySetting(settings, speaker)
	settings.ThemeMode = config.NormalizeThemeMode(settings.ThemeMode)
	keyState, err := keymap.Resolve(settings.Keymap)
	if err != nil {
		return m, err
	}
	settings.Keymap = keyState.ToConfig()

	m.loading = false
	m.settingsLoading = false
	m.err = nil
	m.settings = settings
	m.keymap = keyState
	m.styles = tui.NewStyles(themeFromSettings(settings))
	m.planOptions = planOptionsFromSettings(settings)
	m.settingsEditing = false
	m.settingsInput = strconv.Itoa(settings.SessionSize)
	m.settingsWriteDifficulty = config.NormalizeWriteModeDifficulty(settings.WriteModeDifficulty)
	m.settingsUpdateCheckEnabled = settings.UpdateCheckEnabled
	m.settingsAudioEnabled = settings.AudioEnabled
	m.settingsAudioVoices, m.settingsAudioVoicesLoaded = m.loadAudioVoices()
	m.settingsAudioVoice = normalizeAudioVoiceInList(settings.AudioVoice, m.settingsAudioVoices, m.settingsAudioVoicesLoaded)
	m.settingsAudioAutoplay = settings.AudioAutoplay
	m.settingsAudioAvailableCached = m.probeSettingsAudioAvailableFor(m.settingsAudioVoice)
	m.settingsLanguage = settings.Language
	m.settingsThemeMode = config.NormalizeThemeMode(settings.ThemeMode)
	m.speaker = speaker
	if !m.speakerAvailable() {
		m.autoplayEnabled = false
	}
	return m, nil
}

func (m RootModel) openKeymapEditor() RootModel {
	editor := &keymapEditorState{
		draft:    m.keymap.Clone(),
		original: m.keymap.Clone(),
	}
	m.keymapEditor = editor
	m.screen = ScreenKeymap
	m.status = i18n.T(i18n.StatusKeymapEditing)
	return m
}

func (m RootModel) closeKeymapEditor() RootModel {
	m.keymapEditor = nil
	m.screen = ScreenHome
	m.settingsOpen = true
	if m.status == "" || m.status == i18n.T(i18n.StatusKeymapEditing) {
		m.status = i18n.T(i18n.StatusConfiguringSettings)
	}
	return m
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
	return audio.Config{
		Enabled: settings.AudioEnabled,
		Voice:   settings.AudioVoice,
	}
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

func normalizeAudioVoiceSetting(value string, voiceCatalog func() ([]audio.Voice, bool)) string {
	value = config.NormalizeAudioVoice(value)
	if value == "" {
		return ""
	}
	if voiceCatalog == nil {
		return value
	}
	voices, loaded := voiceCatalog()
	return normalizeAudioVoiceInList(value, voices, loaded)
}

func normalizeAudioVoiceInList(value string, voices []audio.Voice, loaded bool) string {
	value = config.NormalizeAudioVoice(value)
	if value == "" {
		return ""
	}
	if !loaded {
		return value
	}
	for _, voice := range voices {
		if voice.ID == value {
			return voice.ID
		}
	}
	return ""
}

func (m RootModel) loadAudioVoices() ([]audio.Voice, bool) {
	if m.voiceCatalog == nil {
		return nil, false
	}
	voices, loaded := m.voiceCatalog()
	if len(voices) == 0 {
		return nil, loaded
	}
	cloned := make([]audio.Voice, len(voices))
	copy(cloned, voices)
	return cloned, loaded
}

func (m RootModel) loadSettingsOverlayCmd() tea.Cmd {
	return func() tea.Msg {
		voices, loaded, audioVoice, audioAvailable := m.loadSettingsOverlayData()
		return settingsOverlayLoadedMsg{
			voices:         voices,
			voicesLoaded:   loaded,
			audioVoice:     audioVoice,
			audioAvailable: audioAvailable,
		}
	}
}

func (m RootModel) loadSettingsOverlayData() ([]audio.Voice, bool, string, bool) {
	voices, loaded := m.loadAudioVoices()
	audioVoice := normalizeAudioVoiceInList(m.settings.AudioVoice, voices, loaded)
	return voices, loaded, audioVoice, m.probeSettingsAudioAvailableFor(audioVoice)
}

func (m RootModel) settingsAudioVoiceChoices() []string {
	choices := []string{""}
	if !m.settingsAudioVoicesLoaded {
		return choices
	}
	for _, voice := range m.settingsAudioVoices {
		choices = append(choices, voice.ID)
	}
	return choices
}

func (m RootModel) cycleSettingsAudioVoice(step int) string {
	choices := m.settingsAudioVoiceChoices()
	if len(choices) == 0 {
		return ""
	}
	if !m.settingsAudioVoicesLoaded {
		return normalizeAudioVoiceInList(m.settingsAudioVoice, nil, false)
	}

	current := normalizeAudioVoiceInList(m.settingsAudioVoice, m.settingsAudioVoices, m.settingsAudioVoicesLoaded)
	index := 0
	for i, choice := range choices {
		if choice == current {
			index = i
			break
		}
	}

	index = (index + step + len(choices)) % len(choices)
	return choices[index]
}

func (m RootModel) updateSettingsAudioVoice(value string) RootModel {
	m.settingsAudioVoice = normalizeAudioVoiceInList(value, m.settingsAudioVoices, m.settingsAudioVoicesLoaded)
	m.settingsAudioAvailableCached = m.probeSettingsAudioAvailableFor(m.settingsAudioVoice)
	if !m.settingsAudioAvailable() {
		m.settingsAudioAutoplay = false
	}
	return m
}

func (m RootModel) probeSettingsAudioAvailableFor(voice string) bool {
	settings := m.settings
	settings.AudioEnabled = true
	settings.AudioVoice = voice
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
