package i18n

// Message keys used across the application.
// Defined as constants to catch typos at compile time.

// Home screen
const (
	HomeSubtitle            = "home.subtitle"
	HomeAnswerMode          = "home.answer_mode"
	HomeDue                 = "home.due"
	HomeNew                 = "home.new"
	HomeStreak              = "home.streak"
	HomeWait                = "home.wait"
	HomeActive              = "home.active"
	HomeActiveDetail        = "home.active_detail"
	HomeConfirmTitle        = "home.confirm_title"
	HomeConfirmBody         = "home.confirm_body"
	HomeConfirmCurrent      = "home.confirm_current"
	HomeConfirmTarget       = "home.confirm_target"
	HomeConfirmKeys         = "home.confirm_keys"
	HomeReviewFallbackTitle = "home.review_fallback_title"
	HomeReviewFallbackBody  = "home.review_fallback_body"
	HomeReviewFallbackPool  = "home.review_fallback_pool"
	HomeUpdate              = "home.update"
	HomeUpdateDetail        = "home.update_detail"
	HomeUpdateHint          = "home.update_hint"
	HomeKeys                = "home.keys"
)

// Home settings overlay
const (
	SettingsTitle                = "settings.title"
	SettingsQuestions            = "settings.questions"
	SettingsWriteDifficulty      = "settings.write_difficulty"
	SettingsWriteDifficultyBasic = "settings.write_difficulty_basic"
	SettingsWriteDifficultyHard  = "settings.write_difficulty_hard"
	SettingsAudioEnabled         = "settings.audio_enabled"
	SettingsAudioAutoplay        = "settings.audio_autoplay"
	SettingsLanguage             = "settings.language"
	SettingsLanguageJA           = "settings.language_ja"
	SettingsLanguageEN           = "settings.language_en"
	SettingsTheme                = "settings.theme"
	SettingsThemeDefault         = "settings.theme_default"
	SettingsThemeNoColor         = "settings.theme_no_color"
	SettingsThemeNeon            = "settings.theme_neon"
	SettingsThemeCustom          = "settings.theme_custom"
	SettingsThemeCustomNote      = "settings.theme_custom_note"
	SettingsKeymap               = "settings.keymap"
	SettingsKeymapOpen           = "settings.keymap_open"
	SettingsKeys                 = "settings.keys"
	SettingsFocusNote            = "settings.focus_note"
)

// Start setup screen
const (
	StartTitle              = "start.title"
	StartMode               = "start.mode"
	StartQuestions          = "start.questions"
	StartModeLearn          = "start.mode_learn"
	StartModeReview         = "start.mode_review"
	StartModeReviewPractice = "start.mode_review_practice"
	StartNote               = "start.note"
	StartKeys               = "start.keys"
)

// Quiz screen
const (
	QuizNoQuestion = "quiz.no_question"
	QuizMeaning    = "quiz.meaning"
	QuizWord       = "quiz.word"
	QuizInput      = "quiz.input"
	QuizHints      = "quiz.hints"
	QuizHintNone   = "quiz.hint_none"
	QuizAudio      = "quiz.audio"
	QuizKeysChoice = "quiz.keys_choice"
	QuizKeysWrite  = "quiz.keys_write"
)

// Audio labels
const (
	AudioStateOn  = "audio.on"
	AudioStateOff = "audio.off"
)

// Kind labels
const (
	KindReview = "kind.review"
	KindRetry  = "kind.retry"
	KindNew    = "kind.new"
)

// Answer mode
const (
	AnswerModeChoice = "answer_mode.choice"
	AnswerModeWrite  = "answer_mode.write"
)

// Feedback screen
const (
	FbNoFeedback    = "fb.no_feedback"
	FbCorrect       = "fb.correct"
	FbIncorrect     = "fb.incorrect"
	FbWord          = "fb.word"
	FbMeaning       = "fb.meaning"
	FbCorrectAnswer = "fb.correct_answer"
	FbYourAnswer    = "fb.your_answer"
	FbSkipped       = "fb.skipped"
	FbHints         = "fb.hints"
	FbResponseTime  = "fb.response_time"
	FbExampleEN     = "fb.example_en"
	FbExampleJA     = "fb.example_ja"
	FbKeys          = "fb.keys"
	FbKeysWrite     = "fb.keys_write"
	FbStreak        = "fb.streak"
)

// Results screen
const (
	ResultsNoSummary = "results.no_summary"
	ResultsTitle     = "results.title"
	ResultsAccuracy  = "results.accuracy"
	ResultsCorrect   = "results.correct"
	ResultsMix       = "results.mix"
	ResultsMixDetail = "results.mix_detail"
	ResultsHardWords = "results.hard_words"
	ResultsKeys      = "results.keys"
)

// Stats screen
const (
	StatsTitle    = "stats.title"
	StatsKeys     = "stats.keys"
	StatsDue      = "stats.due"
	StatsNew      = "stats.new"
	StatsStreak   = "stats.streak"
	StatsReviews  = "stats.reviews"
	StatsCorrect  = "stats.correct"
	StatsAccuracy = "stats.accuracy"
	StatsWait     = "stats.wait"
)

// Help screen
const (
	KeymapTitle          = "keymap.title"
	KeymapContext        = "keymap.context"
	KeymapFilterAll      = "keymap.filter_all"
	KeymapEmpty          = "keymap.empty"
	KeymapUnbound        = "keymap.unbound"
	KeymapStateDefault   = "keymap.state_default"
	KeymapStateCustom    = "keymap.state_custom"
	KeymapDetails        = "keymap.details"
	KeymapAction         = "keymap.action"
	KeymapDefault        = "keymap.default"
	KeymapCurrent        = "keymap.current"
	KeymapWriteNote      = "keymap.write_note"
	KeymapRecordingTitle = "keymap.recording_title"
	KeymapRecordingBody  = "keymap.recording_body"
	KeymapConflictTitle  = "keymap.conflict_title"
	KeymapConflictBody   = "keymap.conflict_body"
	KeymapConflictKeys   = "keymap.conflict_keys"
	KeymapKeys           = "keymap.keys"
)

// Keymap editor
const (
	HelpTitle             = "help.title"
	HelpBack              = "help.back"
	HelpQuitDisabled      = "help.quit_disabled"
	HelpQuitDisabledWrite = "help.quit_disabled_write"
	HelpStartDigits       = "help.start_digits"
	HelpSettingsDigits    = "help.settings_digits"
)

const (
	NarrowWidthTitle = "narrow.title"
	NarrowWidthBody  = "narrow.body"
	NarrowWidthHint  = "narrow.hint"
)

// Help section titles
const (
	HelpSectionAnswer   = "help.section.answer"
	HelpSectionMove     = "help.section.move"
	HelpSectionGeneral  = "help.section.general"
	HelpSectionInput    = "help.section.input"
	HelpSectionRate     = "help.section.rate"
	HelpSectionNav      = "help.section.nav"
	HelpSectionSessions = "help.section.sessions"
)

// Help screen titles
const (
	HelpScreenStart    = "help.screen.start"
	HelpScreenSettings = "help.screen.settings"
	HelpScreenQuiz     = "help.screen.quiz"
	HelpScreenFeedback = "help.screen.feedback"
	HelpScreenResults  = "help.screen.results"
	HelpScreenStats    = "help.screen.stats"
	HelpScreenHome     = "help.screen.home"
	HelpScreenKeymap   = "help.screen.keymap"
)

// Keymap help descriptions
const (
	KeyUp         = "key.up"
	KeyDown       = "key.down"
	KeyLeft       = "key.left"
	KeyRight      = "key.right"
	KeyChoice1    = "key.choice1"
	KeyChoice2    = "key.choice2"
	KeyChoice3    = "key.choice3"
	KeyChoice4    = "key.choice4"
	KeyToggleMode = "key.toggle_mode"
	KeyConfirm    = "key.confirm"
	KeyQuit       = "key.quit"
	KeyHelp       = "key.help"
	KeySpeak      = "key.speak"
	KeyHint       = "key.hint"
	KeySkip       = "key.skip"
	KeyToggleAuto = "key.toggle_autoplay"
	KeyAgain      = "key.again"
	KeyHard       = "key.hard"
	KeyGood       = "key.good"
	KeyEasy       = "key.easy"
	KeyNewSession = "key.new_session"
	KeyReview     = "key.review"
	KeyStats      = "key.stats"
	KeySettings   = "key.settings"
	KeyBack       = "key.back"
)

// Status messages
const (
	StatusReady                  = "status.ready"
	StatusLoading                = "status.loading"
	StatusResumeFound            = "status.resume_found"
	StatusConfiguring            = "status.configuring_session"
	StatusConfiguringSettings    = "status.configuring_settings"
	StatusStatsLoaded            = "status.stats_loaded"
	StatusSessionStarted         = "status.session_started"
	StatusSaved                  = "status.saved"
	StatusCheckRate              = "status.check_rate"
	StatusCorrect                = "status.correct"
	StatusSelectRating           = "status.select_rating"
	StatusEscThenRate            = "status.esc_then_rate"
	StatusEscToReturn            = "status.esc_to_return"
	StatusLoadingStats           = "status.loading_stats"
	StatusActiveFound            = "status.active_found"
	StatusConfirmDiscard         = "status.confirm_discard"
	StatusConfirmReviewFallback  = "status.confirm_review_fallback"
	StatusStartingReview         = "status.starting_review"
	StatusStartingNew            = "status.starting_new"
	StatusStartingLearn          = "status.starting_learn"
	StatusResuming               = "status.resuming"
	StatusSaving                 = "status.saving"
	StatusReturningHome          = "status.returning_home"
	StatusBackHome               = "status.back_home"
	StatusHelp                   = "status.help"
	StatusInvalidCount           = "status.invalid_question_count"
	StatusSavingSettings         = "status.saving_settings"
	StatusSettingsSaved          = "status.settings_saved"
	StatusSettingsSavedFocus     = "status.settings_saved_focus_mode"
	StatusWriteContinue          = "status.write_continue"
	StatusReviewPracticeContinue = "status.review_practice_continue"
	StatusWriteBasicEmpty        = "status.write_basic_empty"
	StatusAudioDisabled          = "status.audio_disabled"
	StatusAudioUnavailable       = "status.audio_unavailable"
	StatusAudioFailed            = "status.audio_failed"
	StatusAutoplayOn             = "status.autoplay_on"
	StatusAutoplayOff            = "status.autoplay_off"
	StatusKeymapEditing          = "status.keymap_editing"
	StatusKeymapRecording        = "status.keymap_recording"
	StatusKeymapRecorded         = "status.keymap_recorded"
	StatusKeymapConflict         = "status.keymap_conflict"
	StatusKeymapCleared          = "status.keymap_cleared"
	StatusKeymapReset            = "status.keymap_reset"
	StatusKeymapSaved            = "status.keymap_saved"
	StatusKeymapSavedFocus       = "status.keymap_saved_focus_mode"
)

// CLI report messages
const (
	CLIRootShort      = "cli.root_short"
	CLIDoctorHeader   = "cli.doctor_header"
	CLIDoctorOK       = "cli.doctor_ok"
	CLIDoctorErrors   = "cli.doctor_errors"
	CLIDoctorWarnings = "cli.doctor_warnings"
	CLIDoctorBoth     = "cli.doctor_both"
	CLIResetHeader    = "cli.reset_header"
	CLIResetCleared   = "cli.reset_cleared"
	CLIResetReseeded  = "cli.reset_reseeded"
)
