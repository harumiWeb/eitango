package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
)

func TestRenderFeedbackShowsExamplesWhenAvailable(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			Word: store.Word{
				Lemma:     "apply",
				ExampleEN: "She will apply for the role.",
				ExampleJA: "彼女はその役割に応募する。",
			},
			Choices:      []quiz.Choice{{Meaning: "応募する"}},
			CorrectIndex: 0,
		},
		Correct:    true,
		ResponseMS: 2300,
	}

	got := model.renderFeedback()
	if !strings.Contains(got, "She will apply for the role.") {
		t.Fatalf("renderFeedback() missing english example:\n%s", got)
	}
	if !strings.Contains(got, "彼女はその役割に応募する。") {
		t.Fatalf("renderFeedback() missing japanese example:\n%s", got)
	}
}

func TestRenderFeedbackOmitsExampleLabelsWhenAbsent(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			Word:         store.Word{Lemma: "apply"},
			Choices:      []quiz.Choice{{Meaning: "応募する"}},
			CorrectIndex: 0,
		},
		Correct:    true,
		ResponseMS: 2300,
	}

	got := model.renderFeedback()
	if strings.Contains(got, i18n.T(i18n.FbExampleEN)) || strings.Contains(got, i18n.T(i18n.FbExampleJA)) {
		t.Fatalf("renderFeedback() unexpectedly rendered empty examples:\n%s", got)
	}
}

func TestHelpScreenRoundTripFromAllScreens(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		screen             Screen
		helpTitle          string
		expectSettingsOpen bool
		prepare            func(*RootModel)
	}{
		{
			name:      "home",
			screen:    ScreenHome,
			helpTitle: i18n.T(i18n.HelpScreenHome),
		},
		{
			name:               "home-settings",
			screen:             ScreenHome,
			helpTitle:          i18n.T(i18n.HelpScreenSettings),
			expectSettingsOpen: true,
			prepare: func(model *RootModel) {
				model.settingsOpen = true
				model.settingsInput = "10"
				model.settingsLanguage = i18n.LangJA
			},
		},
		{
			name:      "quiz",
			screen:    ScreenQuiz,
			helpTitle: i18n.T(i18n.HelpScreenQuiz),
			prepare: func(model *RootModel) {
				model.currentQ = &quiz.Question{
					Choices: []quiz.Choice{
						{Meaning: "応募する"},
						{Meaning: "手配する"},
						{Meaning: "避ける"},
						{Meaning: "受け入れる"},
					},
				}
			},
		},
		{
			name:      "feedback",
			screen:    ScreenFeedback,
			helpTitle: i18n.T(i18n.HelpScreenFeedback),
			prepare: func(model *RootModel) {
				model.feedback = &quiz.Feedback{
					Question: quiz.Question{
						Word:         store.Word{Lemma: "apply"},
						Choices:      []quiz.Choice{{Meaning: "応募する"}},
						CorrectIndex: 0,
					},
					Correct: true,
				}
			},
		},
		{
			name:      "results",
			screen:    ScreenResults,
			helpTitle: i18n.T(i18n.HelpScreenResults),
		},
		{
			name:      "stats",
			screen:    ScreenStats,
			helpTitle: i18n.T(i18n.HelpScreenStats),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := NewModel(nil, Options{})
			model.loading = false
			model.screen = tc.screen
			model.status = "original status"
			if tc.prepare != nil {
				tc.prepare(&model)
			}

			next, _ := model.Update(tea.KeyPressMsg{Text: "?"})
			helpModel, ok := next.(RootModel)
			if !ok {
				t.Fatalf("Update(?) returned %T, want RootModel", next)
			}
			if helpModel.screen != ScreenHelp {
				t.Fatalf("screen after ? = %v, want %v", helpModel.screen, ScreenHelp)
			}
			if helpModel.helpReturn != tc.screen {
				t.Fatalf("help return screen = %v, want %v", helpModel.helpReturn, tc.screen)
			}
			if helpModel.helpStatus != "original status" {
				t.Fatalf("help status backup = %q, want original status", helpModel.helpStatus)
			}

			helpView := helpModel.renderHelp()
			if !strings.Contains(helpView, tc.helpTitle) {
				t.Fatalf("renderHelp() missing title %q:\n%s", tc.helpTitle, helpView)
			}
			if !strings.Contains(helpView, i18n.T(i18n.HelpBack)) {
				t.Fatalf("renderHelp() missing back hint:\n%s", helpView)
			}

			closed, _ := helpModel.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
			restored, ok := closed.(RootModel)
			if !ok {
				t.Fatalf("Update(Esc) returned %T, want RootModel", closed)
			}
			if restored.screen != tc.screen {
				t.Fatalf("screen after Esc = %v, want %v", restored.screen, tc.screen)
			}
			if restored.status != "original status" {
				t.Fatalf("status after Esc = %q, want original status", restored.status)
			}
			if restored.settingsOpen != tc.expectSettingsOpen {
				t.Fatalf("settingsOpen after Esc = %v, want %v", restored.settingsOpen, tc.expectSettingsOpen)
			}
		})
	}
}

func TestRenderHomeShowsWaitToday(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.home = store.HomeSnapshot{
		DueCount:   4,
		NewCount:   7,
		StreakDays: 3,
	}
	model.stats = stats.Snapshot{
		Today: stats.Window{Label: "Today", WaitMinutes: 5.5},
	}

	got := model.renderHome()
	if !strings.Contains(got, "5.5 min") {
		t.Fatalf("renderHome() missing wait metric:\n%s", got)
	}
}

func TestRenderHomeWithSettingsOverlayUsesScreenSwitch(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.settingsOpen = true
	model.settingsInput = "10"
	model.settingsLanguage = i18n.LangJA

	got := model.renderHomeWithSettingsOverlay()
	if !strings.Contains(got, i18n.T(i18n.SettingsTitle)) {
		t.Fatalf("renderHomeWithSettingsOverlay() missing settings title:\n%s", got)
	}
	if strings.Contains(got, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("renderHomeWithSettingsOverlay() should not include home background when settings are open:\n%s", got)
	}
}
