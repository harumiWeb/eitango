package i18n

// Message keys used across the application.
// Defined as constants to catch typos at compile time.

// Home screen
const (
	HomeSubtitle     = "home.subtitle"
	HomeDue          = "home.due"
	HomeNew          = "home.new"
	HomeStreak       = "home.streak"
	HomeWait         = "home.wait"
	HomeActive       = "home.active"
	HomeActiveDetail = "home.active_detail"
	HomeKeys         = "home.keys"
)

// Quiz screen
const (
	QuizNoQuestion = "quiz.no_question"
	QuizKeys       = "quiz.keys"
)

// Kind labels
const (
	KindReview = "kind.review"
	KindRetry  = "kind.retry"
	KindNew    = "kind.new"
)

// Feedback screen
const (
	FbNoFeedback    = "fb.no_feedback"
	FbCorrect       = "fb.correct"
	FbIncorrect     = "fb.incorrect"
	FbWord          = "fb.word"
	FbCorrectAnswer = "fb.correct_answer"
	FbYourAnswer    = "fb.your_answer"
	FbResponseTime  = "fb.response_time"
	FbExampleEN     = "fb.example_en"
	FbExampleJA     = "fb.example_ja"
	FbKeys          = "fb.keys"
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
	HelpTitle        = "help.title"
	HelpBack         = "help.back"
	HelpQuitDisabled = "help.quit_disabled"
)

// Help section titles
const (
	HelpSectionAnswer   = "help.section.answer"
	HelpSectionMove     = "help.section.move"
	HelpSectionGeneral  = "help.section.general"
	HelpSectionRate     = "help.section.rate"
	HelpSectionNav      = "help.section.nav"
	HelpSectionSessions = "help.section.sessions"
)

// Help screen titles
const (
	HelpScreenQuiz     = "help.screen.quiz"
	HelpScreenFeedback = "help.screen.feedback"
	HelpScreenResults  = "help.screen.results"
	HelpScreenStats    = "help.screen.stats"
	HelpScreenHome     = "help.screen.home"
)

// Keymap help descriptions
const (
	KeyUp         = "key.up"
	KeyDown       = "key.down"
	KeyChoice1    = "key.choice1"
	KeyChoice2    = "key.choice2"
	KeyChoice3    = "key.choice3"
	KeyChoice4    = "key.choice4"
	KeyConfirm    = "key.confirm"
	KeyQuit       = "key.quit"
	KeyHelp       = "key.help"
	KeyAgain      = "key.again"
	KeyHard       = "key.hard"
	KeyGood       = "key.good"
	KeyEasy       = "key.easy"
	KeyNewSession = "key.new_session"
	KeyReview     = "key.review"
	KeyStats      = "key.stats"
	KeyBack       = "key.back"
)

// Status messages
const (
	StatusReady          = "status.ready"
	StatusLoading        = "status.loading"
	StatusResumeFound    = "status.resume_found"
	StatusStatsLoaded    = "status.stats_loaded"
	StatusSessionStarted = "status.session_started"
	StatusSaved          = "status.saved"
	StatusCheckRate      = "status.check_rate"
	StatusCorrect        = "status.correct"
	StatusSelectRating   = "status.select_rating"
	StatusEscThenRate    = "status.esc_then_rate"
	StatusEscToReturn    = "status.esc_to_return"
	StatusLoadingStats   = "status.loading_stats"
	StatusActiveFound    = "status.active_found"
	StatusStartingReview = "status.starting_review"
	StatusStartingNew    = "status.starting_new"
	StatusStartingLearn  = "status.starting_learn"
	StatusResuming       = "status.resuming"
	StatusSaving         = "status.saving"
	StatusReturningHome  = "status.returning_home"
	StatusBackHome       = "status.back_home"
	StatusHelp           = "status.help"
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
