package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
)

func TestUpdateHomeConfirmWithoutActiveStartsSessionImmediately(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)
	model := NewModel(st, Options{
		Plan: session.PlanOptions{QuestionCount: 3, ReviewRatio: 0.4},
	})
	model.loading = false

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated, ok := next.(RootModel)
	if !ok {
		t.Fatalf("Update(enter) returned %T, want RootModel", next)
	}
	if !updated.loading {
		t.Fatal("loading = false, want true")
	}
	if updated.status != i18n.T(i18n.StatusStartingLearn) {
		t.Fatalf("status = %q, want starting learn status", updated.status)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want session start command")
	}

	loaded := mustSessionLoadedMsg(t, cmd())
	if loaded.Runtime.Total() != 3 {
		t.Fatalf("Total() = %d, want 3", loaded.Runtime.Total())
	}
}

func TestUpdateHomeSettingsOpensOverlay(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false

	next, _ := model.Update(tea.KeyPressMsg{Text: "c"})
	updated, ok := next.(RootModel)
	if !ok {
		t.Fatalf("Update(c) returned %T, want RootModel", next)
	}
	if !updated.settingsOpen {
		t.Fatal("settingsOpen = false, want true")
	}
	if updated.status != i18n.T(i18n.StatusConfiguringSettings) {
		t.Fatalf("status = %q, want configuring settings status", updated.status)
	}
}

func TestUpdateHomeTabTogglesAnswerMode(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	updated := next.(RootModel)
	if updated.selectedAnswerMode != store.AnswerModeWrite {
		t.Fatalf("selectedAnswerMode = %q, want %q", updated.selectedAnswerMode, store.AnswerModeWrite)
	}

	next, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	updated = next.(RootModel)
	if updated.selectedAnswerMode != store.AnswerModeChoice {
		t.Fatalf("selectedAnswerMode = %q, want %q", updated.selectedAnswerMode, store.AnswerModeChoice)
	}
}

func TestUpdateHomeSettingsSavePersistsAndAppliesLanguage(t *testing.T) {
	if err := i18n.Load(i18n.LangJA); err != nil {
		t.Fatalf("Load(ja) error = %v", err)
	}
	defer func() {
		if err := i18n.Load(i18n.LangJA); err != nil {
			t.Fatalf("restore Load(ja) error = %v", err)
		}
	}()

	path := filepath.Join(t.TempDir(), "config.toml")
	initial := config.Settings{
		SessionSize:         12,
		ReviewRatio:         0.4,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Language:            i18n.LangJA,
	}

	model := NewModel(nil, Options{
		Settings:   initial,
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsInput = "8"
	model.settingsEditing = true
	model.settingsWriteDifficulty = config.WriteModeDifficultyHard
	model.settingsCursor = 2
	model.settingsLanguage = i18n.LangEN

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated, ok := next.(RootModel)
	if !ok {
		t.Fatalf("Update(enter) returned %T, want RootModel", next)
	}
	if !updated.loading {
		t.Fatal("loading = false, want true")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want save settings command")
	}

	saved, _ := updated.Update(cmd())
	final, ok := saved.(RootModel)
	if !ok {
		t.Fatalf("Update(settingsSavedMsg) returned %T, want RootModel", saved)
	}
	if final.loading {
		t.Fatal("loading = true, want false")
	}
	if final.settingsOpen {
		t.Fatal("settingsOpen = true, want false")
	}
	if final.settings.SessionSize != 8 {
		t.Fatalf("SessionSize = %d, want 8", final.settings.SessionSize)
	}
	if final.settings.Language != i18n.LangEN {
		t.Fatalf("Language = %q, want %q", final.settings.Language, i18n.LangEN)
	}
	if final.settings.WriteModeDifficulty != config.WriteModeDifficultyHard {
		t.Fatalf("WriteModeDifficulty = %q, want %q", final.settings.WriteModeDifficulty, config.WriteModeDifficultyHard)
	}
	if final.planOptions.QuestionCount != 8 {
		t.Fatalf("QuestionCount = %d, want 8", final.planOptions.QuestionCount)
	}
	if final.keymap.Settings.Help().Desc != i18n.T(i18n.KeySettings) {
		t.Fatalf("settings key help = %q, want %q", final.keymap.Settings.Help().Desc, i18n.T(i18n.KeySettings))
	}

	savedSettings, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(saved config) error = %v", err)
	}
	if savedSettings.SessionSize != 8 {
		t.Fatalf("saved SessionSize = %d, want 8", savedSettings.SessionSize)
	}
	if savedSettings.Language != i18n.LangEN {
		t.Fatalf("saved Language = %q, want %q", savedSettings.Language, i18n.LangEN)
	}
	if savedSettings.WriteModeDifficulty != config.WriteModeDifficultyHard {
		t.Fatalf("saved WriteModeDifficulty = %q, want %q", savedSettings.WriteModeDifficulty, config.WriteModeDifficultyHard)
	}
}

func TestUpdateHomeSettingsSaveDisablesFocusModeDefaultOnQuestionChange(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	initial := config.Settings{
		SessionSize:         12,
		ReviewRatio:         0.4,
		FocusModeDefault:    true,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Language:            i18n.LangJA,
	}

	model := NewModel(nil, Options{
		Settings:   initial,
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsInput = "9"
	model.settingsEditing = true

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want save settings command")
	}

	saved, _ := updated.Update(cmd())
	final := saved.(RootModel)
	if final.settings.FocusModeDefault {
		t.Fatal("FocusModeDefault = true, want false")
	}
	if final.planOptions.QuestionCount != 9 {
		t.Fatalf("QuestionCount = %d, want 9", final.planOptions.QuestionCount)
	}
	if final.status != i18n.T(i18n.StatusSettingsSavedFocus) {
		t.Fatalf("status = %q, want focus-disabled save status", final.status)
	}
}

func TestUpdateHomeSettingsDifficultySwitchesWithArrowKeys(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			Language:            i18n.LangJA,
		},
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = 1

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated := next.(RootModel)
	if updated.settingsWriteDifficulty != config.WriteModeDifficultyHard {
		t.Fatalf("settingsWriteDifficulty after right = %q, want %q", updated.settingsWriteDifficulty, config.WriteModeDifficultyHard)
	}

	next, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated = next.(RootModel)
	if updated.settingsWriteDifficulty != config.WriteModeDifficultyBasic {
		t.Fatalf("settingsWriteDifficulty after left = %q, want %q", updated.settingsWriteDifficulty, config.WriteModeDifficultyBasic)
	}
}

func TestUpdateCheckedMsgStoresLatestVersionForHomeNotice(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{CurrentVersion: "v1.1.0"})

	next, _ := model.Update(updateCheckedMsg{Result: updatecheck.Result{
		Latest:          updatecheck.ReleaseInfo{TagName: "v1.2.0"},
		UpdateAvailable: true,
		ShouldNotify:    true,
	}})
	updated := next.(RootModel)

	if updated.updateLatestTag != "v1.2.0" {
		t.Fatalf("updateLatestTag = %q, want v1.2.0", updated.updateLatestTag)
	}
}

func TestRenderHomeShowsUpdateNotice(t *testing.T) {
	if err := i18n.Load(i18n.LangEN); err != nil {
		t.Fatalf("Load(en) error = %v", err)
	}
	defer func() {
		if err := i18n.Load(i18n.LangJA); err != nil {
			t.Fatalf("restore Load(ja) error = %v", err)
		}
	}()

	model := NewModel(nil, Options{CurrentVersion: "v1.1.0"})
	model.loading = false
	model.updateLatestTag = "v1.2.0"

	rendered := model.renderHome()
	if !strings.Contains(rendered, i18n.T(i18n.HomeUpdate)) {
		t.Fatalf("renderHome() = %q, want update title", rendered)
	}
	if !strings.Contains(rendered, "v1.2.0") {
		t.Fatalf("renderHome() = %q, want latest version", rendered)
	}
}

func TestUpdateWriteQuizHintSkipAndAutoRating(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word: store.Word{
			Lemma:     "begin",
			MeaningJA: "始める",
			Pos:       "verb",
		},
		Ordinal: 1,
		Total:   1,
		Kind:    store.ItemKindNew,
	}
	model.questionStarted = time.Now().UTC().Add(-2 * time.Second)

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	withHint := next.(RootModel)
	if withHint.writeHintCount != 1 {
		t.Fatalf("writeHintCount = %d, want 1", withHint.writeHintCount)
	}
	if got := renderSlots(withHint.currentQ.Word.Lemma, withHint.writeHintIndices); got != "b _ _ _ n" {
		t.Fatalf("renderSlots() after first hint = %q, want %q", got, "b _ _ _ n")
	}

	next, _ = withHint.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	skipped := next.(RootModel)
	if skipped.screen != ScreenFeedback {
		t.Fatalf("screen after skip = %v, want %v", skipped.screen, ScreenFeedback)
	}
	if skipped.feedback == nil || !skipped.feedback.Skipped {
		t.Fatalf("feedback after skip = %+v, want skipped feedback", skipped.feedback)
	}
	if skipped.feedback.Rating != srs.Again {
		t.Fatalf("skip rating = %q, want %q", skipped.feedback.Rating, srs.Again)
	}
}

func TestUpdateWriteQuizLettersRemainTypable(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word: store.Word{
			Lemma:     "ship",
			MeaningJA: "船",
			Pos:       "noun",
		},
		Ordinal: 1,
		Total:   1,
		Kind:    store.ItemKindNew,
	}

	next, _ := model.Update(tea.KeyPressMsg{Text: "h", Code: 'h'})
	updated := next.(RootModel)
	if updated.writeHintCount != 0 {
		t.Fatalf("writeHintCount after typing h = %d, want 0", updated.writeHintCount)
	}
	if updated.writeInput != "h" {
		t.Fatalf("writeInput after typing h = %q, want %q", updated.writeInput, "h")
	}

	next, _ = updated.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	updated = next.(RootModel)
	if updated.screen != ScreenQuiz {
		t.Fatalf("screen after typing s = %v, want %v", updated.screen, ScreenQuiz)
	}
	if updated.writeInput != "hs" {
		t.Fatalf("writeInput after typing s = %q, want %q", updated.writeInput, "hs")
	}

	next, _ = updated.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
	updated = next.(RootModel)
	if updated.screen != ScreenQuiz {
		t.Fatalf("screen after typing q = %v, want %v", updated.screen, ScreenQuiz)
	}
	if updated.writeInput != "hsq" {
		t.Fatalf("writeInput after typing q = %q, want %q", updated.writeInput, "hsq")
	}
}

func TestUpdateWriteQuizEscQuits(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word: store.Word{
			Lemma:     "quit",
			MeaningJA: "やめる",
			Pos:       "verb",
		},
	}

	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("cmd = nil, want tea.Quit command")
	}
}

func TestUpdateWriteQuizEnterBuildsCorrectFeedback(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word: store.Word{
			Lemma:     "begin",
			MeaningJA: "始める",
			Pos:       "verb",
		},
		Ordinal: 1,
		Total:   1,
		Kind:    store.ItemKindNew,
	}
	model.writeInput = "Begin"
	model.questionStarted = time.Now().UTC().Add(-1500 * time.Millisecond)

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if updated.feedback == nil {
		t.Fatal("feedback = nil, want write feedback")
	}
	if !updated.feedback.Correct {
		t.Fatalf("feedback.Correct = false, want true: %+v", updated.feedback)
	}
	if updated.feedback.Rating != srs.Easy {
		t.Fatalf("feedback.Rating = %q, want %q", updated.feedback.Rating, srs.Easy)
	}
}

func TestUpdateHelpQuitFromWriteFeedbackShowsWriteContinueStatus(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenHelp
	model.helpReturn = ScreenFeedback
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			AnswerMode: store.AnswerModeWrite,
			Word:       store.Word{Lemma: "begin"},
		},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.status != i18n.T(i18n.StatusWriteContinue) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusWriteContinue))
	}
	if updated.screen != ScreenHelp {
		t.Fatalf("screen = %v, want %v", updated.screen, ScreenHelp)
	}
}
