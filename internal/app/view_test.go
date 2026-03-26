package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/quiz"
	"github.com/yourname/eitango/internal/stats"
	"github.com/yourname/eitango/internal/store"
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
	if !strings.Contains(got, "Example EN   : She will apply for the role.") {
		t.Fatalf("renderFeedback() missing english example:\n%s", got)
	}
	if !strings.Contains(got, "Example JA   : 彼女はその役割に応募する。") {
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
	if strings.Contains(got, "Example EN") || strings.Contains(got, "Example JA") {
		t.Fatalf("renderFeedback() unexpectedly rendered empty examples:\n%s", got)
	}
}

func TestHelpScreenRoundTripFromAllScreens(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		screen    Screen
		helpTitle string
		prepare   func(*RootModel)
	}{
		{
			name:      "home",
			screen:    ScreenHome,
			helpTitle: "Home controls",
		},
		{
			name:      "quiz",
			screen:    ScreenQuiz,
			helpTitle: "Quiz controls",
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
			helpTitle: "Feedback controls",
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
			helpTitle: "Results controls",
		},
		{
			name:      "stats",
			screen:    ScreenStats,
			helpTitle: "Stats controls",
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
			if !strings.Contains(helpView, "Esc=back") {
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
	if !strings.Contains(got, "Wait today   : 5.5 min") {
		t.Fatalf("renderHome() missing wait metric:\n%s", got)
	}
}
