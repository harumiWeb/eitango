package app

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/session"
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
		SessionSize: 12,
		ReviewRatio: 0.4,
		Language:    i18n.LangJA,
	}

	model := NewModel(nil, Options{
		Settings:   initial,
		ConfigPath: path,
	})
	model.loading = false
	model = model.openSettingsOverlay()
	model.settingsInput = "8"
	model.settingsEditing = true
	model.settingsCursor = 1
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
}

func TestUpdateHomeSettingsSaveDisablesFocusModeDefaultOnQuestionChange(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	initial := config.Settings{
		SessionSize:      12,
		ReviewRatio:      0.4,
		FocusModeDefault: true,
		Language:         i18n.LangJA,
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
