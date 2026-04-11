package i18n_test

import (
	"testing"

	"github.com/harumiWeb/eitango/internal/i18n"
)

func TestLoadJA(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}
	got := i18n.T(i18n.HomeSubtitle)
	if got == "" || got == i18n.HomeSubtitle {
		t.Errorf("T(%s) returned %q; want non-empty translated string", i18n.HomeSubtitle, got)
	}
}

func TestLoadJAAudioVoiceLabels(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}
	if got := i18n.T(i18n.SettingsAudioVoice); got != "ローカル音声" {
		t.Fatalf("T(%s) = %q; want %q", i18n.SettingsAudioVoice, got, "ローカル音声")
	}
	if got := i18n.T(i18n.SettingsAudioVoiceAuto); got != "自動" {
		t.Fatalf("T(%s) = %q; want %q", i18n.SettingsAudioVoiceAuto, got, "自動")
	}
}

func TestLoadEN(t *testing.T) {
	if err := i18n.Load("en"); err != nil {
		t.Fatalf("Load(en): %v", err)
	}
	got := i18n.T(i18n.HomeSubtitle)
	want := "AI waiting time -> 1-3 minute vocab loop"
	if got != want {
		t.Errorf("T(%s) = %q; want %q", i18n.HomeSubtitle, got, want)
	}
}

func TestFallback(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}
	// A key that exists in both should return the primary (ja) version.
	ja := i18n.T(i18n.StatusReady)
	if ja == "Ready" {
		t.Errorf("T(%s) returned English fallback instead of Japanese", i18n.StatusReady)
	}
}

func TestMissingKeyReturnsKey(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}
	got := i18n.T("nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T(nonexistent.key) = %q; want raw key", got)
	}
}

func TestTf(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}
	got := i18n.Tf(i18n.HomeActiveDetail, 5, 10, "learn", "choice")
	want := "5/10 問回答済み (learn / choice)"
	if got != want {
		t.Fatalf("Tf(%s) = %q; want %q", i18n.HomeActiveDetail, got, want)
	}
}

func TestNormaliseLang(t *testing.T) {
	for _, lang := range []string{"ja", "ja_JP", "JA", "ja-jp"} {
		if !i18n.ValidLang(lang) {
			t.Errorf("ValidLang(%q) = false; want true", lang)
		}
	}
	for _, lang := range []string{"en", "en_US", "EN", "en-gb"} {
		if !i18n.ValidLang(lang) {
			t.Errorf("ValidLang(%q) = false; want true", lang)
		}
	}
}

func TestAllJAKeysExistInEN(t *testing.T) {
	if err := i18n.Load("ja"); err != nil {
		t.Fatalf("Load(ja): %v", err)
	}

	// Ensure every defined key constant returns a non-key value for both languages.
	keys := []string{
		i18n.HomeSubtitle, i18n.HomeAnswerMode, i18n.HomeDue, i18n.HomeNew, i18n.HomeStreak,
		i18n.HomeWait, i18n.HomeActive, i18n.HomeActiveDetail, i18n.HomeConfirmTitle,
		i18n.HomeConfirmBody, i18n.HomeConfirmCurrent, i18n.HomeConfirmTarget,
		i18n.HomeConfirmKeys, i18n.HomeReviewFallbackTitle, i18n.HomeReviewFallbackBody, i18n.HomeReviewFallbackPool, i18n.HomeUpdate,
		i18n.HomeUpdateDetail, i18n.HomeUpdateHint, i18n.HomeKeys,
		i18n.SettingsTitle, i18n.SettingsQuestions, i18n.SettingsWriteDifficulty,
		i18n.SettingsWriteDifficultyBasic, i18n.SettingsWriteDifficultyHard,
		i18n.SettingsUpdateCheck,
		i18n.SettingsAudioEnabled, i18n.SettingsAudioVoice, i18n.SettingsAudioVoiceAuto, i18n.SettingsAudioVoiceUnavailable, i18n.SettingsAudioAutoplay,
		i18n.SettingsLanguage, i18n.SettingsLanguageJA, i18n.SettingsLanguageEN,
		i18n.SettingsTheme, i18n.SettingsThemeDefault, i18n.SettingsThemeNoColor,
		i18n.SettingsThemeNeon, i18n.SettingsThemeCustom, i18n.SettingsThemeCustomNote,
		i18n.SettingsKeymap, i18n.SettingsKeymapOpen,
		i18n.SettingsKeys, i18n.SettingsFocusNote,
		i18n.KeymapTitle, i18n.KeymapContext, i18n.KeymapFilterAll, i18n.KeymapEmpty,
		i18n.KeymapUnbound, i18n.KeymapStateDefault, i18n.KeymapStateCustom,
		i18n.KeymapDetails, i18n.KeymapAction, i18n.KeymapDefault, i18n.KeymapCurrent,
		i18n.KeymapWriteNote, i18n.KeymapRecordingTitle, i18n.KeymapRecordingBody,
		i18n.KeymapConflictTitle, i18n.KeymapConflictBody, i18n.KeymapConflictKeys,
		i18n.KeymapKeys,
		i18n.QuizNoQuestion, i18n.QuizMeaning, i18n.QuizWord, i18n.QuizInput,
		i18n.QuizHints, i18n.QuizHintNone, i18n.QuizAudio, i18n.QuizKeysChoice, i18n.QuizKeysWrite,
		i18n.AudioStateOn, i18n.AudioStateOff,
		i18n.AnswerModeChoice, i18n.AnswerModeWrite,
		i18n.KindReview, i18n.KindRetry, i18n.KindNew,
		i18n.FbNoFeedback, i18n.FbCorrect, i18n.FbIncorrect,
		i18n.FbWord, i18n.FbMeaning, i18n.FbCorrectAnswer, i18n.FbYourAnswer,
		i18n.FbSkipped, i18n.FbHints,
		i18n.FbResponseTime, i18n.FbExampleEN, i18n.FbExampleJA,
		i18n.FbKeys, i18n.FbKeysWrite, i18n.FbStreak,
		i18n.ResultsNoSummary, i18n.ResultsTitle, i18n.ResultsAccuracy,
		i18n.ResultsCorrect, i18n.ResultsMix, i18n.ResultsMixDetail,
		i18n.ResultsHardWords, i18n.ResultsKeys,
		i18n.StatsTitle, i18n.StatsKeys, i18n.StatsDue, i18n.StatsNew,
		i18n.StatsStreak, i18n.StatsReviews, i18n.StatsCorrect,
		i18n.StatsAccuracy, i18n.StatsWait,
		i18n.HelpTitle, i18n.HelpBack, i18n.HelpQuitDisabled, i18n.HelpQuitDisabledWrite,
		i18n.NarrowWidthTitle, i18n.NarrowWidthBody, i18n.NarrowWidthHint,
		i18n.HelpSectionAnswer, i18n.HelpSectionMove, i18n.HelpSectionGeneral,
		i18n.HelpSectionRate, i18n.HelpSectionNav, i18n.HelpSectionSessions,
		i18n.StartModeLearn, i18n.StartModeReview, i18n.StartModeReviewPractice,
		i18n.HelpScreenQuiz, i18n.HelpScreenFeedback, i18n.HelpScreenResults,
		i18n.HelpScreenStats, i18n.HelpScreenHome, i18n.HelpScreenKeymap,
		i18n.KeyUp, i18n.KeyDown, i18n.KeyChoice1, i18n.KeyChoice2,
		i18n.KeyChoice3, i18n.KeyChoice4, i18n.KeyToggleMode, i18n.KeyConfirm, i18n.KeyQuit,
		i18n.KeyHelp, i18n.KeySpeak, i18n.KeyHint, i18n.KeySkip, i18n.KeyToggleAuto, i18n.KeyAgain, i18n.KeyHard, i18n.KeyGood,
		i18n.KeyEasy, i18n.KeyNewSession, i18n.KeyReview, i18n.KeyStats, i18n.KeyBack,
		i18n.StatusReady, i18n.StatusLoading, i18n.StatusResumeFound,
		i18n.StatusStatsLoaded, i18n.StatusSessionStarted, i18n.StatusSaved,
		i18n.StatusCheckRate, i18n.StatusCorrect, i18n.StatusSelectRating,
		i18n.StatusEscThenRate, i18n.StatusEscToReturn, i18n.StatusLoadingStats,
		i18n.StatusActiveFound, i18n.StatusConfirmDiscard, i18n.StatusConfirmReviewFallback, i18n.StatusStartingReview, i18n.StatusStartingNew,
		i18n.StatusStartingLearn, i18n.StatusResuming, i18n.StatusSaving,
		i18n.StatusReturningHome, i18n.StatusBackHome, i18n.StatusHelp, i18n.StatusWriteContinue, i18n.StatusReviewPracticeContinue,
		i18n.StatusWriteBasicEmpty, i18n.StatusAudioDisabled, i18n.StatusAudioUnavailable, i18n.StatusAudioFailed,
		i18n.StatusAutoplayOn, i18n.StatusAutoplayOff, i18n.StatusKeymapEditing, i18n.StatusKeymapRecording,
		i18n.StatusKeymapRecorded, i18n.StatusKeymapConflict, i18n.StatusKeymapCleared,
		i18n.StatusKeymapReset, i18n.StatusKeymapSaved, i18n.StatusKeymapSavedFocus,
		i18n.CLIRootShort, i18n.CLIDoctorHeader, i18n.CLIDoctorOK,
		i18n.CLIDoctorErrors, i18n.CLIDoctorWarnings, i18n.CLIDoctorBoth,
		i18n.CLIResetHeader, i18n.CLIResetCleared, i18n.CLIResetReseeded,
	}

	for _, k := range keys {
		got := i18n.T(k)
		if got == k {
			t.Errorf("key %q has no Japanese translation (returned raw key)", k)
		}
	}

	// Now switch to English and verify the same keys exist.
	if err := i18n.Load("en"); err != nil {
		t.Fatalf("Load(en): %v", err)
	}
	for _, k := range keys {
		got := i18n.T(k)
		if got == k {
			t.Errorf("key %q has no English translation (returned raw key)", k)
		}
	}
}
