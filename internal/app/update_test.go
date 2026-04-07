package app

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/harumiWeb/eitango/internal/audio"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
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

func TestUpdateHomeNewSessionWithoutActiveStartsImmediately(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)
	settings := config.DefaultSettings()
	settings.WriteModeDifficulty = config.WriteModeDifficultyHard
	model := NewModel(st, Options{Settings: settings})
	model.loading = false
	model.selectedAnswerMode = store.AnswerModeWrite

	next, cmd := model.Update(tea.KeyPressMsg{Text: "n", Code: 'n'})
	updated := next.(RootModel)
	if updated.homeConfirm != nil {
		t.Fatalf("homeConfirm = %+v, want nil", updated.homeConfirm)
	}
	if !updated.loading {
		t.Fatal("loading = false, want true")
	}
	if updated.status != i18n.T(i18n.StatusStartingNew) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusStartingNew))
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want new session command")
	}

	loaded := mustSessionLoadedMsg(t, cmd())
	if loaded.Runtime.Session.Mode != store.ModeLearn {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeLearn)
	}
	if loaded.Runtime.Session.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("session answer mode = %q, want %q", loaded.Runtime.Session.AnswerMode, store.AnswerModeWrite)
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
	model.settingsCursor = settingsRowLanguage
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
	if final.binding(keymap.ContextHome, keymap.ActionSettings).Help().Desc != i18n.T(i18n.KeySettings) {
		t.Fatalf("settings key help = %q, want %q", final.binding(keymap.ContextHome, keymap.ActionSettings).Help().Desc, i18n.T(i18n.KeySettings))
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
			AudioEnabled:        true,
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

func TestUpdateKeymapEditorSavesOverrideAndAppliesImmediately(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	model := NewModel(nil, Options{
		Settings:   config.DefaultSettings(),
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowKeymap

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	editor := next.(RootModel)
	if editor.screen != ScreenKeymap {
		t.Fatalf("screen = %v, want %v", editor.screen, ScreenKeymap)
	}
	if editor.keymapEditor == nil {
		t.Fatal("keymapEditor = nil, want editor state")
	}

	next, _ = editor.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	editor = next.(RootModel)
	next, _ = editor.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	editor = next.(RootModel)
	if editor.keymapEditor == nil || !editor.keymapEditor.recording {
		t.Fatal("recording = false, want true after a")
	}

	next, _ = editor.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	editor = next.(RootModel)
	if editor.keymapEditor == nil || editor.keymapEditor.recording {
		t.Fatal("recording = true, want false after capture")
	}
	if got := editor.keymapEditor.draft.Keys(keymap.ContextHome, keymap.ActionToggleAnswerMode); !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("draft home toggle keys = %v, want [x]", got)
	}

	next, cmd := editor.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	saving := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want save keymap command")
	}
	if !saving.loading {
		t.Fatal("loading = false, want true while saving")
	}

	saved, _ := saving.Update(cmd())
	final := saved.(RootModel)
	if final.screen != ScreenHome {
		t.Fatalf("screen = %v, want %v", final.screen, ScreenHome)
	}
	if !final.settingsOpen {
		t.Fatal("settingsOpen = false, want true after returning from keymap editor")
	}
	if got := final.keymap.Keys(keymap.ContextHome, keymap.ActionToggleAnswerMode); !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("effective home toggle keys = %v, want [x]", got)
	}

	savedSettings, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(saved config) error = %v", err)
	}
	if got := savedSettings.Keymap.Home["toggle_answer_mode"]; !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("saved keymap override = %v, want [x]", got)
	}
}

func TestUpdateKeymapEditorSavePersistsSettingsOverlayDraft(t *testing.T) {
	t.Parallel()

	if err := i18n.Load(i18n.DefaultLang); err != nil {
		t.Fatalf("Load(default lang) error = %v", err)
	}
	t.Cleanup(func() {
		if err := i18n.Load(i18n.DefaultLang); err != nil {
			t.Fatalf("restore default lang error = %v", err)
		}
	})

	path := filepath.Join(t.TempDir(), "config.toml")
	settings := config.DefaultSettings()
	settings.FocusModeDefault = true
	settings.SessionSize = 5

	model := NewModel(nil, Options{
		Settings:   settings,
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsInput = "12"
	model.settingsLanguage = i18n.LangEN
	model.settingsCursor = settingsRowKeymap

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	editor := next.(RootModel)
	if editor.keymapEditor == nil {
		t.Fatal("keymapEditor = nil, want editor state")
	}

	next, _ = editor.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	editor = next.(RootModel)
	next, _ = editor.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	editor = next.(RootModel)
	next, _ = editor.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	editor = next.(RootModel)

	next, cmd := editor.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	saving := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want save keymap command")
	}

	saved, _ := saving.Update(cmd())
	final := saved.(RootModel)
	if final.settings.SessionSize != 12 {
		t.Fatalf("final session size = %d, want 12", final.settings.SessionSize)
	}
	if final.settings.Language != i18n.LangEN {
		t.Fatalf("final language = %q, want %q", final.settings.Language, i18n.LangEN)
	}
	if final.settings.FocusModeDefault {
		t.Fatal("FocusModeDefault = true, want false after saving updated session size")
	}
	if final.status != i18n.T(i18n.StatusKeymapSavedFocus) {
		t.Fatalf("status = %q, want %q", final.status, i18n.T(i18n.StatusKeymapSavedFocus))
	}

	savedSettings, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(saved config) error = %v", err)
	}
	if savedSettings.SessionSize != 12 {
		t.Fatalf("saved session size = %d, want 12", savedSettings.SessionSize)
	}
	if savedSettings.Language != i18n.LangEN {
		t.Fatalf("saved language = %q, want %q", savedSettings.Language, i18n.LangEN)
	}
	if savedSettings.FocusModeDefault {
		t.Fatal("saved FocusModeDefault = true, want false")
	}
}

func TestUpdateKeymapEditorRecordsEsc(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{Settings: config.DefaultSettings()})
	model.loading = false
	model = model.openKeymapEditor()
	if model.keymapEditor == nil {
		t.Fatal("keymapEditor = nil")
	}

	next, _ := model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	editor := next.(RootModel)
	if editor.keymapEditor == nil || !editor.keymapEditor.recording {
		t.Fatal("recording = false, want true")
	}

	next, _ = editor.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	editor = next.(RootModel)
	if editor.keymapEditor == nil || editor.keymapEditor.recording {
		t.Fatal("recording = true, want false after recording esc")
	}
	if got := editor.keymapEditor.draft.Keys(keymap.ContextHome, keymap.ActionToggleAnswerMode); !reflect.DeepEqual(got, []string{"tab", "esc"}) {
		t.Fatalf("draft home toggle keys = %v, want [tab esc]", got)
	}
}

func TestUpdateKeymapEditorCtrlGCancelsRecording(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{Settings: config.DefaultSettings()})
	model.loading = false
	model = model.openKeymapEditor()
	if model.keymapEditor == nil {
		t.Fatal("keymapEditor = nil")
	}

	next, _ := model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	editor := next.(RootModel)
	if editor.keymapEditor == nil || !editor.keymapEditor.recording {
		t.Fatal("recording = false, want true")
	}

	next, _ = editor.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	editor = next.(RootModel)
	if editor.keymapEditor == nil {
		t.Fatal("keymapEditor = nil after cancel")
	}
	if editor.keymapEditor.recording {
		t.Fatal("recording = true, want false after Ctrl+G")
	}
	if got := editor.keymapEditor.draft.Keys(keymap.ContextHome, keymap.ActionToggleAnswerMode); !reflect.DeepEqual(got, keymap.DefaultKeys(keymap.ContextHome, keymap.ActionToggleAnswerMode)) {
		t.Fatalf("draft home toggle keys = %v, want defaults", got)
	}
}

func TestUpdateKeymapEditorMouseWheelScrollsCursor(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.DefaultSettings(),
	})
	model.loading = false
	model = model.openKeymapEditor()
	if model.keymapEditor == nil {
		t.Fatal("keymapEditor = nil")
	}

	next, _ := model.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	scrolled := next.(RootModel)
	if scrolled.keymapEditor == nil || scrolled.keymapEditor.cursor != 1 {
		t.Fatalf("cursor after wheel down = %v, want 1", scrolled.keymapEditor)
	}

	next, _ = scrolled.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	scrolled = next.(RootModel)
	if scrolled.keymapEditor == nil || scrolled.keymapEditor.cursor != 0 {
		t.Fatalf("cursor after wheel up = %v, want 0", scrolled.keymapEditor)
	}
}

func TestUpdateHomeSettingsAudioRowsSwitchWithArrowKeys(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       false,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: func(audio.Config) audio.Speaker { return &stubSpeaker{enabled: true} },
	})
	model.loading = false
	model = model.openSettingsOverlay()

	model.settingsCursor = settingsRowAudioEnabled
	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated := next.(RootModel)
	if updated.settingsAudioEnabled {
		t.Fatal("settingsAudioEnabled after left = true, want false")
	}
	if updated.settingsAudioAutoplay {
		t.Fatal("settingsAudioAutoplay after disabling audio = true, want false")
	}

	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowAudioAutoplay
	next, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated = next.(RootModel)
	if !updated.settingsAudioAutoplay {
		t.Fatal("settingsAudioAutoplay after right = false, want true")
	}
}

func TestUpdateHomeSettingsThemeRowSwitchesWithArrowKeys(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			Language:            i18n.LangJA,
			ThemeMode:           config.ThemeModeDefault,
		},
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowTheme

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated := next.(RootModel)
	if updated.settingsThemeMode != config.ThemeModeNoColor {
		t.Fatalf("settingsThemeMode after right = %q, want %q", updated.settingsThemeMode, config.ThemeModeNoColor)
	}

	next, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated = next.(RootModel)
	if updated.settingsThemeMode != config.ThemeModeDefault {
		t.Fatalf("settingsThemeMode after left = %q, want %q", updated.settingsThemeMode, config.ThemeModeDefault)
	}
}

func TestUpdateHomeSettingsSaveAppliesThemeStyles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			Language:            i18n.LangJA,
			ThemeMode:           config.ThemeModeDefault,
		},
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowTheme
	model.settingsThemeMode = config.ThemeModeNeon

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want save settings command")
	}

	saved, _ := updated.Update(cmd())
	final := saved.(RootModel)
	if final.settings.ThemeMode != config.ThemeModeNeon {
		t.Fatalf("settings.ThemeMode = %q, want %q", final.settings.ThemeMode, config.ThemeModeNeon)
	}
	if final.styles.Title.GetForeground() != lipgloss.Color("#A6FF00") {
		t.Fatalf("Title foreground = %#v, want %#v", final.styles.Title.GetForeground(), lipgloss.Color("#A6FF00"))
	}
}

func TestUpdateHomeSettingsAutoplayUnavailableStaysOff(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       false,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: func(audio.Config) audio.Speaker { return &stubSpeaker{enabled: false} },
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowAudioAutoplay

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.settingsAudioAutoplay {
		t.Fatal("settingsAudioAutoplay = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusAudioUnavailable) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioUnavailable))
	}
}

func TestUpdateHomeSettingsAutoplayDisabledPromptsToEnableAudio(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioDisabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsCursor = settingsRowAudioAutoplay

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.settingsAudioAutoplay {
		t.Fatal("settingsAudioAutoplay = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusAudioDisabled) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioDisabled))
	}
}

func TestUpdateHomeSettingsSaveNormalizesUnavailableAutoplay(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       true,
			Language:            i18n.LangJA,
		},
		ConfigPath:     path,
		SpeakerFactory: func(audio.Config) audio.Speaker { return &stubSpeaker{enabled: false} },
	})
	model.loading = false
	model = model.openSettingsOverlay()

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want save settings command")
	}
	if !updated.loading {
		t.Fatal("loading = false, want true")
	}

	saved, _ := updated.Update(cmd())
	final := saved.(RootModel)
	if final.settings.AudioAutoplay {
		t.Fatal("settings.AudioAutoplay = true, want false")
	}
	if final.settingsAudioAutoplay {
		t.Fatal("settingsAudioAutoplay = true, want false")
	}

	savedSettings, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(saved config) error = %v", err)
	}
	if savedSettings.AudioAutoplay {
		t.Fatal("saved AudioAutoplay = true, want false")
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

func TestUpdateQuizSpeakUsesSpeaker(t *testing.T) {
	t.Parallel()

	speaker := &stubSpeaker{enabled: true}
	settings := newAudioEnabledSettings()
	model := NewModel(nil, Options{
		Settings:       settings,
		SpeakerFactory: newPinnedSpeakerFactory(speaker),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updated := next.(RootModel)
	if cmd == nil {
		t.Fatal("cmd = nil, want speak command")
	}
	if updated.status == i18n.T(i18n.StatusAudioUnavailable) {
		t.Fatalf("status = %q, want speaker to be available", updated.status)
	}

	if msg := cmd(); msg != nil {
		t.Fatalf("cmd() = %T, want nil on success", msg)
	}
	if speaker.calls != 1 {
		t.Fatalf("speaker calls = %d, want 1", speaker.calls)
	}
	if speaker.lastText != "begin" {
		t.Fatalf("speaker lastText = %q, want %q", speaker.lastText, "begin")
	}
}

func TestUpdateQuizSpeakWithoutSpeakerSetsUnavailableStatus(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.status != i18n.T(i18n.StatusAudioUnavailable) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioUnavailable))
	}
}

func TestUpdateQuizSpeakWithAudioDisabledSetsDisabledStatus(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioDisabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.status != i18n.T(i18n.StatusAudioDisabled) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioDisabled))
	}
}

func TestUpdateQuizShiftTabTogglesAutoplayForCurrentSession(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = false
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	updated := next.(RootModel)
	if !updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = false, want true")
	}
	if updated.status != i18n.T(i18n.StatusAutoplayOn) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAutoplayOn))
	}
}

func TestUpdateQuizShiftTabWithoutSpeakerKeepsAutoplayOff(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = false
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusAudioUnavailable) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioUnavailable))
	}
}

func TestUpdateQuizShiftTabWithAudioDisabledKeepsAutoplayOff(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioDisabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = false
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin"},
	}

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusAudioDisabled) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioDisabled))
	}
}

func TestSessionLoadedMsgInitializesAutoplayButDoesNotSpeakInWriteQuiz(t *testing.T) {
	t.Parallel()

	speaker := &stubSpeaker{enabled: true}
	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       true,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: newPinnedSpeakerFactory(speaker),
	})

	record := store.SessionRecord{
		ID:             "session-1",
		Mode:           store.ModeLearn,
		AnswerMode:     store.AnswerModeChoice,
		TotalQuestions: 1,
		Status:         store.SessionStatusActive,
	}
	runtime := session.NewRuntime(record, []store.SessionItem{
		{SessionID: record.ID, Ordinal: 1, WordID: 1, Kind: store.ItemKindNew},
	})

	next, cmd := model.Update(sessionLoadedMsg{
		Runtime: runtime,
		Question: quiz.Question{
			AnswerMode: store.AnswerModeWrite,
			Word:       store.Word{Lemma: "begin"},
			Ordinal:    1,
			Total:      1,
			Kind:       store.ItemKindNew,
		},
	})
	updated := next.(RootModel)
	if !updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = false, want true")
	}
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil when write quiz autoplay is disabled", cmd)
	}
	if speaker.calls != 0 {
		t.Fatalf("speaker calls = %d, want 0", speaker.calls)
	}
}

func TestSessionLoadedMsgDisablesAutoplayWhenSpeakerUnavailable(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       true,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: newStubSpeakerFactory(false),
	})

	record := store.SessionRecord{
		ID:             "session-1",
		Mode:           store.ModeLearn,
		AnswerMode:     store.AnswerModeChoice,
		TotalQuestions: 1,
		Status:         store.SessionStatusActive,
	}
	runtime := session.NewRuntime(record, []store.SessionItem{
		{SessionID: record.ID, Ordinal: 1, WordID: 1, Kind: store.ItemKindNew},
	})

	next, cmd := model.Update(sessionLoadedMsg{
		Runtime: runtime,
		Question: quiz.Question{
			AnswerMode:   store.AnswerModeChoice,
			Word:         store.Word{Lemma: "begin"},
			Choices:      []quiz.Choice{{WordID: 2, Meaning: "始める"}},
			CorrectIndex: 0,
			Ordinal:      1,
			Total:        1,
			Kind:         store.ItemKindNew,
		},
	})
	updated := next.(RootModel)
	if updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = true, want false")
	}
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
}

func TestUpdateWriteQuizCtrlPDoesNotSpeak(t *testing.T) {
	t.Parallel()

	speaker := &stubSpeaker{enabled: true}
	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newPinnedSpeakerFactory(speaker),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word:       store.Word{Lemma: "begin"},
	}
	model.status = i18n.T(i18n.StatusReady)

	next, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.status != i18n.T(i18n.StatusReady) {
		t.Fatalf("status = %q, want unchanged ready status", updated.status)
	}
	if speaker.calls != 0 {
		t.Fatalf("speaker calls = %d, want 0", speaker.calls)
	}
}

func TestUpdateWriteQuizShiftTabDoesNotToggleAutoplay(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = false
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word:       store.Word{Lemma: "begin"},
	}
	model.status = i18n.T(i18n.StatusReady)

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusReady) {
		t.Fatalf("status = %q, want unchanged ready status", updated.status)
	}
}

func TestUpdateWriteQuizSkipAutoplaySpeaksOnFeedback(t *testing.T) {
	t.Parallel()

	speaker := &stubSpeaker{enabled: true}
	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyBasic,
			AudioEnabled:        true,
			AudioAutoplay:       false,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: newPinnedSpeakerFactory(speaker),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = true
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

	next, cmd := model.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	updated := next.(RootModel)
	if updated.screen != ScreenFeedback {
		t.Fatalf("screen = %v, want %v", updated.screen, ScreenFeedback)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want autoplay command on write feedback")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("cmd() = %T, want nil on success", msg)
	}
	if speaker.calls != 1 {
		t.Fatalf("speaker calls = %d, want 1", speaker.calls)
	}
	if speaker.lastText != "begin" {
		t.Fatalf("speaker lastText = %q, want %q", speaker.lastText, "begin")
	}
}

func TestUpdateAudioErrMsgFromAutoplayDisablesAutoplay(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.autoplayEnabled = true

	next, cmd := model.Update(audioErrMsg{fromAutoplay: true, err: wantErr})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = true, want false")
	}
	if updated.status != i18n.T(i18n.StatusAudioFailed) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioFailed))
	}
	if !errors.Is(updated.err, wantErr) {
		t.Fatalf("err = %v, want %v", updated.err, wantErr)
	}
}

func TestUpdateAudioErrMsgFromManualSpeakKeepsAutoplayEnabled(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.autoplayEnabled = true

	next, cmd := model.Update(audioErrMsg{fromAutoplay: false})
	updated := next.(RootModel)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if !updated.autoplayEnabled {
		t.Fatal("autoplayEnabled = false, want true")
	}
	if updated.status != i18n.T(i18n.StatusAudioFailed) {
		t.Fatalf("status = %q, want %q", updated.status, i18n.T(i18n.StatusAudioFailed))
	}
}

func TestSpeakCmdReturnsAutoplayAudioErrMsg(t *testing.T) {
	t.Parallel()

	cmd := speakCmd(&stubSpeaker{enabled: true, err: errors.New("boom")}, "begin", true)
	if cmd == nil {
		t.Fatal("cmd = nil, want speak command")
	}
	msg := cmd()
	errMsg, ok := msg.(audioErrMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want audioErrMsg", msg)
	}
	if !errMsg.fromAutoplay {
		t.Fatal("fromAutoplay = false, want true")
	}
	if errMsg.err == nil || errMsg.err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", errMsg.err)
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

type stubSpeaker struct {
	enabled  bool
	err      error
	calls    int
	lastText string
}

func (s *stubSpeaker) Speak(_ context.Context, text string) error {
	s.calls++
	s.lastText = text
	return s.err
}

func (s *stubSpeaker) Enabled() bool {
	return s.enabled
}

func newAudioEnabledSettings() config.Settings {
	settings := config.DefaultSettings()
	settings.AudioEnabled = true
	return settings
}

func newAudioDisabledSettings() config.Settings {
	settings := config.DefaultSettings()
	settings.AudioEnabled = false
	settings.AudioAutoplay = false
	return settings
}

func newStubSpeakerFactory(enabled bool) func(audio.Config) audio.Speaker {
	return func(cfg audio.Config) audio.Speaker {
		return &stubSpeaker{enabled: enabled && cfg.Enabled}
	}
}

func newPinnedSpeakerFactory(speaker *stubSpeaker) func(audio.Config) audio.Speaker {
	return func(cfg audio.Config) audio.Speaker {
		if !cfg.Enabled {
			return &stubSpeaker{enabled: false}
		}
		return speaker
	}
}
