package app

import (
	"context"
	"errors"
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
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
)

const (
	testResponseDuration   = 2 * time.Second
	beginRevealHintPresses = 4
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

func TestUpdateHomeConfirmWithDifferentAnswerModeOpensDiscardOverlay(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)
	active := mustCreateActiveSession(t, st, store.ModeLearn, store.AnswerModeChoice)
	model := NewModel(st, Options{})
	model.loading = false
	model.home.ActiveSession = &active
	model.selectedAnswerMode = store.AnswerModeWrite

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.loading {
		t.Fatal("loading = true, want false")
	}
	if updated.homeConfirm == nil {
		t.Fatal("homeConfirm = nil, want discard confirmation")
	}
	if updated.status != i18n.T(i18n.StatusConfirmDiscard) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusConfirmDiscard))
	}
	if updated.homeConfirm.Request.Mode != store.ModeLearn {
		t.Fatalf("pending mode = %q, want %q", updated.homeConfirm.Request.Mode, store.ModeLearn)
	}
	if updated.homeConfirm.Request.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("pending answer mode = %q, want %q", updated.homeConfirm.Request.AnswerMode, store.AnswerModeWrite)
	}
	if !updated.homeConfirm.Request.ReplaceActive {
		t.Fatal("pending request ReplaceActive = false, want true")
	}
}

func TestUpdateHomeConfirmDiscardStartsRequestedSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	active := mustCreateActiveSession(t, st, store.ModeLearn, store.AnswerModeChoice)
	settings := config.DefaultSettings()
	settings.WriteModeDifficulty = config.WriteModeDifficultyHard
	model := NewModel(st, Options{Settings: settings})
	model.loading = false
	model.home.ActiveSession = &active
	model.selectedAnswerMode = store.AnswerModeWrite

	opened, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	pending := opened.(RootModel)

	next, cmd := pending.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if !updated.loading {
		t.Fatal("loading = false, want true")
	}
	if updated.homeConfirm != nil {
		t.Fatalf("homeConfirm = %+v, want nil after confirmation", updated.homeConfirm)
	}
	if updated.status != i18n.T(i18n.StatusStartingLearn) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusStartingLearn))
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want session start command")
	}

	loaded := mustSessionLoadedMsg(t, cmd())
	if loaded.Runtime.Session.ID == active.ID {
		t.Fatalf("session id = %q, want a fresh session", loaded.Runtime.Session.ID)
	}
	if loaded.Runtime.Session.Mode != store.ModeLearn {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeLearn)
	}
	if loaded.Runtime.Session.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("session answer mode = %q, want %q", loaded.Runtime.Session.AnswerMode, store.AnswerModeWrite)
	}

	abandoned, err := st.LoadSession(ctx, active.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if abandoned.Status != store.SessionStatusAbandoned {
		t.Fatalf("abandoned status = %q, want %q", abandoned.Status, store.SessionStatusAbandoned)
	}
}

func TestUpdateHomeConfirmCancelKeepsActiveSessionAndSelection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	active := mustCreateActiveSession(t, st, store.ModeLearn, store.AnswerModeChoice)
	model := NewModel(st, Options{})
	model.loading = false
	model.home.ActiveSession = &active
	model.selectedAnswerMode = store.AnswerModeWrite

	opened, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	pending := opened.(RootModel)

	next, cmd := pending.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.homeConfirm != nil {
		t.Fatalf("homeConfirm = %+v, want nil after cancel", updated.homeConfirm)
	}
	if updated.selectedAnswerMode != store.AnswerModeWrite {
		t.Fatalf("selectedAnswerMode = %q, want %q", updated.selectedAnswerMode, store.AnswerModeWrite)
	}
	if updated.status != i18n.T(i18n.StatusResumeFound) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusResumeFound))
	}

	record, err := st.LoadSession(ctx, active.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if record.Status != store.SessionStatusActive {
		t.Fatalf("record status = %q, want %q", record.Status, store.SessionStatusActive)
	}
}

func TestUpdateHomeReviewWithActiveSessionConfirmsThenStartsReview(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	active := mustCreateActiveSession(t, st, store.ModeLearn, store.AnswerModeChoice)
	words, err := st.ListNewWords(ctx, 3, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	markWordDue(t, st, words[1].ID, time.Now().UTC().AddDate(0, 0, -4))

	model := NewModel(st, Options{})
	model.loading = false
	model.home.ActiveSession = &active

	opened, cmd := model.Update(tea.KeyPressMsg{Text: "r", Code: 'r'})
	pending := opened.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if pending.homeConfirm == nil {
		t.Fatal("homeConfirm = nil, want discard confirmation for review")
	}
	if pending.homeConfirm.StartStatus != i18n.StatusStartingReview {
		t.Fatalf("StartStatus = %q, want %q", pending.homeConfirm.StartStatus, i18n.StatusStartingReview)
	}

	next, cmd := pending.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if updated.status != i18n.T(i18n.StatusStartingReview) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusStartingReview))
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want review session command")
	}

	loaded := mustSessionLoadedMsg(t, cmd())
	if loaded.Runtime.Session.Mode != store.ModeReview {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeReview)
	}
}

func TestUpdateHomeNewSessionWithActiveSessionOpensDiscardOverlay(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)
	active := mustCreateActiveSession(t, st, store.ModeLearn, store.AnswerModeChoice)
	model := NewModel(st, Options{})
	model.loading = false
	model.home.ActiveSession = &active
	model.selectedAnswerMode = store.AnswerModeWrite

	next, cmd := model.Update(tea.KeyPressMsg{Text: "n", Code: 'n'})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.homeConfirm == nil {
		t.Fatal("homeConfirm = nil, want discard confirmation for new session")
	}
	if updated.homeConfirm.StartStatus != i18n.StatusStartingNew {
		t.Fatalf("StartStatus = %q, want %q", updated.homeConfirm.StartStatus, i18n.StatusStartingNew)
	}
	if updated.homeConfirm.Request.Mode != store.ModeLearn {
		t.Fatalf("pending mode = %q, want %q", updated.homeConfirm.Request.Mode, store.ModeLearn)
	}
	if updated.homeConfirm.Request.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("pending answer mode = %q, want %q", updated.homeConfirm.Request.AnswerMode, store.AnswerModeWrite)
	}
	if !updated.homeConfirm.Request.ReplaceActive {
		t.Fatal("pending request ReplaceActive = false, want true")
	}
}

func TestUpdateHomeReloadedErrMsgReplacesStaleActiveSnapshot(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = true
	model.home.ActiveSession = &store.SessionRecord{
		ID:                "stale-session",
		Mode:              store.ModeLearn,
		AnswerMode:        store.AnswerModeChoice,
		AnsweredQuestions: 2,
		TotalQuestions:    5,
		Status:            store.SessionStatusActive,
	}
	model.homeConfirm = &homeConfirmState{
		Request: sessionRequest{
			Mode:          store.ModeLearn,
			AnswerMode:    store.AnswerModeWrite,
			ReplaceActive: true,
		},
		StartStatus: i18n.StatusStartingLearn,
	}
	model.selectedAnswerMode = store.AnswerModeWrite
	wantErr := errors.New("no words available for this session")

	next, cmd := model.Update(homeReloadedErrMsg{
		Home: store.HomeSnapshot{},
		Stats: &stats.Snapshot{
			Today: stats.Window{Label: "Today", WaitMinutes: 1.5},
		},
		err: wantErr,
	})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.loading {
		t.Fatal("loading = true, want false")
	}
	if updated.home.ActiveSession != nil {
		t.Fatalf("home.ActiveSession = %+v, want nil", updated.home.ActiveSession)
	}
	if updated.homeConfirm != nil {
		t.Fatalf("homeConfirm = %+v, want nil", updated.homeConfirm)
	}
	if updated.err == nil || updated.err.Error() != wantErr.Error() {
		t.Fatalf("err = %v, want %v", updated.err, wantErr)
	}
	if updated.status != wantErr.Error() {
		t.Fatalf("status = %q, want %q", updated.status, wantErr.Error())
	}
	if updated.selectedAnswerMode != store.AnswerModeWrite {
		t.Fatalf("selectedAnswerMode = %q, want %q", updated.selectedAnswerMode, store.AnswerModeWrite)
	}
	if updated.stats.Today.WaitMinutes != 1.5 {
		t.Fatalf("stats.Today.WaitMinutes = %v, want 1.5", updated.stats.Today.WaitMinutes)
	}
}

func TestUpdateHomeReloadedErrMsgWithoutStatsKeepsExistingStats(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = true
	model.home.ActiveSession = &store.SessionRecord{
		ID:                "stale-session",
		Mode:              store.ModeLearn,
		AnswerMode:        store.AnswerModeChoice,
		AnsweredQuestions: 2,
		TotalQuestions:    5,
		Status:            store.SessionStatusActive,
	}
	model.stats = stats.Snapshot{
		Today: stats.Window{Label: "Today", WaitMinutes: 4.25},
	}
	wantErr := errors.New("no words available for this session")

	next, _ := model.Update(homeReloadedErrMsg{
		Home:  store.HomeSnapshot{},
		Stats: nil,
		err:   wantErr,
	})
	updated := next.(RootModel)
	if updated.home.ActiveSession != nil {
		t.Fatalf("home.ActiveSession = %+v, want nil", updated.home.ActiveSession)
	}
	if updated.stats.Today.WaitMinutes != 4.25 {
		t.Fatalf("stats.Today.WaitMinutes = %v, want 4.25", updated.stats.Today.WaitMinutes)
	}
	if updated.status != wantErr.Error() {
		t.Fatalf("status = %q, want %q", updated.status, wantErr.Error())
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
	model.questionStarted = time.Now().UTC().Add(-testResponseDuration)

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

func TestUpdateWriteQuizAllHintsAutoFail(t *testing.T) {
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
	model.questionStarted = time.Now().UTC().Add(-testResponseDuration)

	updated := model
	for i := 0; i < beginRevealHintPresses; i++ {
		next, _ := updated.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		updated = next.(RootModel)
	}

	if updated.screen != ScreenFeedback {
		t.Fatalf("screen after all hints = %v, want %v", updated.screen, ScreenFeedback)
	}
	if updated.feedback == nil {
		t.Fatal("feedback after all hints = nil, want write feedback")
	}
	if updated.feedback.Correct {
		t.Fatalf("feedback.Correct = true, want false: %+v", updated.feedback)
	}
	if updated.feedback.Skipped {
		t.Fatalf("feedback.Skipped = true, want false: %+v", updated.feedback)
	}
	if updated.feedback.HintCount != beginRevealHintPresses {
		t.Fatalf("feedback.HintCount = %d, want %d", updated.feedback.HintCount, beginRevealHintPresses)
	}
	if updated.feedback.Rating != srs.Again {
		t.Fatalf("feedback.Rating = %q, want %q", updated.feedback.Rating, srs.Again)
	}
}

func TestUpdateWriteQuizAllHintsAutoFailWithCorrectPrefilledInput(t *testing.T) {
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
	model.writeInput = "begin"
	model.questionStarted = time.Now().UTC().Add(-testResponseDuration)

	updated := model
	for i := 0; i < beginRevealHintPresses; i++ {
		next, _ := updated.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		updated = next.(RootModel)
	}

	if updated.screen != ScreenFeedback {
		t.Fatalf("screen after all hints = %v, want %v", updated.screen, ScreenFeedback)
	}
	if updated.feedback == nil {
		t.Fatal("feedback after all hints = nil, want write feedback")
	}
	if updated.feedback.Correct {
		t.Fatalf("feedback.Correct = true, want false: %+v", updated.feedback)
	}
	if updated.feedback.Rating != srs.Again {
		t.Fatalf("feedback.Rating = %q, want %q", updated.feedback.Rating, srs.Again)
	}
	if updated.feedback.SelectedText != "begin" {
		t.Fatalf("feedback.SelectedText = %q, want %q", updated.feedback.SelectedText, "begin")
	}
	if updated.feedback.Skipped {
		t.Fatalf("feedback.Skipped = true, want false: %+v", updated.feedback)
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

func mustCreateActiveSession(t *testing.T, st *store.Store, mode, answerMode string) store.SessionRecord {
	t.Helper()

	ctx := context.Background()
	words, err := st.ListNewWords(ctx, 1, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	record, _, err := st.CreateSession(ctx, mode, answerMode, []store.SessionItemPlan{
		{WordID: words[0].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return record
}
