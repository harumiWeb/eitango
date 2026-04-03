package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/audio"
	"github.com/harumiWeb/eitango/internal/config"
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
				model.settingsWriteDifficulty = config.WriteModeDifficultyBasic
				model.settingsAudioEnabled = true
				model.settingsAudioAutoplay = false
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
	model.selectedAnswerMode = store.AnswerModeWrite

	got := model.renderHome()
	if !strings.Contains(got, "5.5 min") {
		t.Fatalf("renderHome() missing wait metric:\n%s", got)
	}
	if !strings.Contains(got, i18n.T(i18n.AnswerModeWrite)) {
		t.Fatalf("renderHome() missing selected answer mode:\n%s", got)
	}
}

func TestRenderHomeLocalizesActiveSessionMode(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.home.ActiveSession = &store.SessionRecord{
		Mode:              store.ModeLearn,
		AnswerMode:        store.AnswerModeChoice,
		AnsweredQuestions: 2,
		TotalQuestions:    5,
	}

	got := model.renderHome()
	wantDetail := i18n.Tf(i18n.HomeActiveDetail, 2, 5, i18n.T(i18n.StartModeLearn), i18n.T(i18n.AnswerModeChoice))
	if !strings.Contains(got, wantDetail) {
		t.Fatalf("renderHome() missing localized active session detail %q:\n%s", wantDetail, got)
	}
	if strings.Contains(got, "learn / "+i18n.T(i18n.AnswerModeChoice)) {
		t.Fatalf("renderHome() unexpectedly contains raw session mode:\n%s", got)
	}
}

func TestRenderWriteFeedbackShowsMeaningHintsAndSkippedState(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			AnswerMode: store.AnswerModeWrite,
			Word: store.Word{
				Lemma:     "begin",
				MeaningJA: "始める",
			},
		},
		SelectedText: "began",
		Correct:      false,
		ResponseMS:   1700,
		HintCount:    2,
	}

	got := model.renderFeedback()
	for _, want := range []string{"begin", "始める", "began", "2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderFeedback() missing %q:\n%s", want, got)
		}
	}
}

func TestRenderWriteQuizAndHelpShowCtrlShortcuts(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.writeInput = "begin"
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

	quizView := model.renderQuiz()
	for _, want := range []string{"Tab=ヒント", "Ctrl+S", "Esc=終了"} {
		if !strings.Contains(quizView, want) {
			t.Fatalf("renderQuiz() missing %q:\n%s", want, quizView)
		}
	}
	for _, unwanted := range []string{"Ctrl+P", "Shift+Tab", "ON"} {
		if strings.Contains(quizView, unwanted) {
			t.Fatalf("renderQuiz() unexpectedly contains %q:\n%s", unwanted, quizView)
		}
	}
	if !strings.Contains(quizView, "b e g i n") {
		t.Fatalf("renderQuiz() should render spaced input:\n%s", quizView)
	}
	if strings.Contains(quizView, "A-Z") {
		t.Fatalf("renderQuiz() unexpectedly shows A-Z typing hint:\n%s", quizView)
	}

	model = model.openHelp()
	helpView := model.renderHelp()
	for _, want := range []string{"tab", "ctrl+s", "esc"} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{"ctrl+p", "shift+tab"} {
		if strings.Contains(helpView, unwanted) {
			t.Fatalf("renderHelp() unexpectedly contains %q:\n%s", unwanted, helpView)
		}
	}
	if strings.Contains(helpView, "A-Z") {
		t.Fatalf("renderHelp() unexpectedly shows A-Z typing hint:\n%s", helpView)
	}
}

func TestRenderWriteFeedbackHelpShowsEnterOnly(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.screen = ScreenFeedback
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			AnswerMode: store.AnswerModeWrite,
			Word: store.Word{
				Lemma:     "begin",
				MeaningJA: "始める",
			},
		},
		Correct: true,
	}

	helpView := model.openHelp().renderHelp()
	for _, want := range []string{
		helpLine(model.keymap.Confirm),
		helpLine(model.keymap.Speak),
		helpLine(model.keymap.ToggleAutoplay),
		fmt.Sprintf("%-10s %s", "q", i18n.T(i18n.HelpQuitDisabledWrite)),
	} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{
		helpLine(model.keymap.Again),
		helpLine(model.keymap.Hard),
		helpLine(model.keymap.Good),
		helpLine(model.keymap.Easy),
	} {
		if strings.Contains(helpView, unwanted) {
			t.Fatalf("renderHelp() unexpectedly contains %q:\n%s", unwanted, helpView)
		}
	}
}

func TestRenderWriteFeedbackShowsAudioControls(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.autoplayEnabled = true
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			AnswerMode: store.AnswerModeWrite,
			Word: store.Word{
				Lemma:     "begin",
				MeaningJA: "始める",
			},
		},
		Correct: true,
	}

	got := model.renderFeedback()
	for _, want := range []string{"Ctrl+P", "Shift+Tab", "ON"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderFeedback() missing %q:\n%s", want, got)
		}
	}
}

func TestRenderHomeWithSettingsOverlayUsesScreenSwitch(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.settingsOpen = true
	model.settingsInput = "10"
	model.settingsWriteDifficulty = config.WriteModeDifficultyHard
	model.settingsAudioEnabled = true
	model.settingsAudioAutoplay = true
	model.settingsLanguage = i18n.LangJA

	got := model.renderHomeWithSettingsOverlay()
	if !strings.Contains(got, i18n.T(i18n.SettingsTitle)) {
		t.Fatalf("renderHomeWithSettingsOverlay() missing settings title:\n%s", got)
	}
	for _, want := range []string{
		i18n.T(i18n.SettingsWriteDifficulty),
		i18n.T(i18n.SettingsWriteDifficultyHard),
		i18n.T(i18n.SettingsAudioEnabled),
		i18n.T(i18n.SettingsAudioAutoplay),
		i18n.T(i18n.AudioStateOn),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHomeWithSettingsOverlay() missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("renderHomeWithSettingsOverlay() should not include home background when settings are open:\n%s", got)
	}
}

func TestRenderHomeWithSettingsOverlayShowsAutoplayOffWhenUnavailable(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         10,
			ReviewRatio:         0.4,
			WriteModeDifficulty: config.WriteModeDifficultyHard,
			AudioEnabled:        true,
			AudioAutoplay:       true,
			Language:            i18n.LangJA,
		},
		SpeakerFactory: newStubSpeakerFactory(false),
	})
	model.loading = false
	model.settingsOpen = true
	model.settingsInput = "10"
	model.settingsWriteDifficulty = config.WriteModeDifficultyHard
	model.settingsAudioEnabled = true
	model.settingsAudioAutoplay = true
	model.settingsLanguage = i18n.LangJA

	got := model.renderHomeWithSettingsOverlay()
	if !strings.Contains(got, i18n.T(i18n.SettingsAudioAutoplay)) || !strings.Contains(got, i18n.T(i18n.AudioStateOff)) {
		t.Fatalf("renderHomeWithSettingsOverlay() should show autoplay OFF:\n%s", got)
	}
}

func TestRenderHomeWithSettingsOverlayDoesNotProbeAudioOnRender(t *testing.T) {
	t.Parallel()

	settings := newAudioEnabledSettings()
	probes := 0
	model := NewModel(nil, Options{
		Settings: settings,
		SpeakerFactory: func(cfg audio.Config) audio.Speaker {
			probes++
			return &stubSpeaker{enabled: cfg.Enabled}
		},
	})
	model.loading = false
	model = model.openSettingsOverlay()
	initialProbes := probes

	_ = model.renderHomeWithSettingsOverlay()
	_ = model.renderHomeWithSettingsOverlay()

	if probes != initialProbes {
		t.Fatalf("speaker probes during render = %d, want %d", probes, initialProbes)
	}
}

func TestRenderHomeWithConfirmOverlayUsesScreenSwitch(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.home.ActiveSession = &store.SessionRecord{
		Mode:              store.ModeLearn,
		AnswerMode:        store.AnswerModeChoice,
		AnsweredQuestions: 2,
		TotalQuestions:    5,
	}
	model.homeConfirm = &homeConfirmState{
		Request: sessionRequest{
			Mode:       store.ModeReview,
			AnswerMode: store.AnswerModeWrite,
		},
	}

	got := model.renderHomeWithConfirmOverlay()
	currentDetail := i18n.Tf(
		i18n.HomeActiveDetail,
		model.home.ActiveSession.AnsweredQuestions,
		model.home.ActiveSession.TotalQuestions,
		sessionModeLabel(model.home.ActiveSession.Mode),
		answerModeLabel(model.home.ActiveSession.AnswerMode),
	)
	for _, want := range []string{
		i18n.T(i18n.HomeConfirmTitle),
		i18n.T(i18n.HomeConfirmBody),
		i18n.T(i18n.HomeConfirmCurrent),
		i18n.T(i18n.HomeConfirmTarget),
		currentDetail,
		i18n.T(i18n.StartModeLearn),
		i18n.T(i18n.StartModeReview),
		i18n.T(i18n.AnswerModeWrite),
		i18n.T(i18n.HomeConfirmKeys),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHomeWithConfirmOverlay() missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("renderHomeWithConfirmOverlay() should not include home background when confirmation is open:\n%s", got)
	}
}

func TestRenderHelpFromHomeConfirmShowsConfirmAndBack(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.home.ActiveSession = &store.SessionRecord{
		Mode:              store.ModeLearn,
		AnswerMode:        store.AnswerModeChoice,
		AnsweredQuestions: 1,
		TotalQuestions:    3,
	}
	model.homeConfirm = &homeConfirmState{
		Request: sessionRequest{
			Mode:       store.ModeLearn,
			AnswerMode: store.AnswerModeWrite,
		},
	}

	helpView := model.openHelp().renderHelp()
	for _, want := range []string{
		helpLine(model.keymap.Confirm),
		helpLine(model.keymap.Back),
	} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{
		helpLine(model.keymap.NewSession),
		helpLine(model.keymap.Review),
		helpLine(model.keymap.ToggleAnswerMode),
	} {
		if strings.Contains(helpView, unwanted) {
			t.Fatalf("renderHelp() unexpectedly contains %q:\n%s", unwanted, helpView)
		}
	}
}
