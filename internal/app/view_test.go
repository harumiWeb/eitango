package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/harumiWeb/eitango/internal/audio"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/keymap"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/tui"
	"github.com/mattn/go-runewidth"
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
	if err := i18n.Load(i18n.DefaultLang); err != nil {
		t.Fatalf("Load(default lang) error = %v", err)
	}
	t.Cleanup(func() {
		if err := i18n.Load(i18n.DefaultLang); err != nil {
			t.Fatalf("restore default lang error = %v", err)
		}
	})

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
				model.settingsThemeMode = config.ThemeModeDefault
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

func TestRenderKeymapEditorFitsWindowHeightAndScrollsRows(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         config.DefaultSettings().SessionSize,
			ReviewRatio:         config.DefaultSettings().ReviewRatio,
			WriteModeDifficulty: config.DefaultSettings().WriteModeDifficulty,
			AudioEnabled:        config.DefaultSettings().AudioEnabled,
			AudioAutoplay:       config.DefaultSettings().AudioAutoplay,
			Language:            i18n.LangJA,
			ThemeMode:           config.ThemeModeNoColor,
		},
	})
	model.loading = false
	model.height = 20
	model.width = 80
	model = model.openKeymapEditor()
	if model.keymapEditor == nil {
		t.Fatal("keymapEditor = nil")
	}

	rows := model.keymapEditor.rows()
	model.keymapEditor.cursor = len(rows) - 2

	view := model.View().Content
	if got := lipgloss.Height(view); got > model.height {
		t.Fatalf("View height = %d, want <= %d\n%s", got, model.height, view)
	}
	if got := model.View().MouseMode; got != tea.MouseModeCellMotion {
		t.Fatalf("MouseMode = %v, want %v", got, tea.MouseModeCellMotion)
	}
	if !strings.Contains(view, keymap.ContextLabel(keymap.ContextHelp)) {
		t.Fatalf("View missing selected context near bottom:\n%s", view)
	}
	if !strings.Contains(view, keymap.ActionLabel(keymap.ActionBack)) {
		t.Fatalf("View missing selected action near bottom:\n%s", view)
	}
	if strings.Contains(view, keymap.ActionLabel(keymap.ActionToggleAnswerMode)) {
		t.Fatalf("View unexpectedly contains first-row action; list did not scroll:\n%s", view)
	}
	if !strings.Contains(view, "█") || !strings.Contains(view, "│") {
		t.Fatalf("View missing scrollbar track/thumb:\n%s", view)
	}

	lines := strings.Split(view, "\n")
	defaultCols := make([]int, 0, 3)
	for _, line := range lines {
		plain := ansi.Strip(line)
		if !strings.Contains(plain, "default") {
			continue
		}
		for _, ctx := range keymap.Contexts() {
			if strings.Contains(plain, keymap.ContextLabel(ctx)) {
				index := strings.Index(plain, "default")
				defaultCols = append(defaultCols, runewidth.StringWidth(plain[:index]))
				break
			}
		}
	}
	if len(defaultCols) < 2 {
		t.Fatalf("View missing enough list rows to verify alignment:\n%s", view)
	}
	if defaultCols[0] != defaultCols[1] {
		t.Fatalf("default column is not aligned: %v\n%s", defaultCols[:2], view)
	}
}

func TestRenderKeymapEditorFitsWindowHeightWhenVerySmall(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings: config.Settings{
			SessionSize:         config.DefaultSettings().SessionSize,
			ReviewRatio:         config.DefaultSettings().ReviewRatio,
			WriteModeDifficulty: config.DefaultSettings().WriteModeDifficulty,
			AudioEnabled:        config.DefaultSettings().AudioEnabled,
			AudioAutoplay:       config.DefaultSettings().AudioAutoplay,
			Language:            i18n.LangJA,
			ThemeMode:           config.ThemeModeNoColor,
		},
	})
	model.loading = false
	model.width = 80

	for _, height := range []int{6, 8, 10, 12} {
		model.height = height
		m := model.openKeymapEditor()
		if m.keymapEditor == nil {
			t.Fatalf("height=%d: keymapEditor = nil", height)
		}
		// Enable recording and a conflict to exercise all optional sections.
		m.keymapEditor.recording = true
		m.keymapEditor.conflict = &keymapConflictState{
			Token: "x",
			Conflicts: []keymap.Conflict{{
				Context: keymap.ContextHome,
				Action:  keymap.ActionConfirm,
			}},
		}
		view := m.View().Content
		if got := lipgloss.Height(view); got > height {
			t.Fatalf("height=%d: View height = %d, want <= %d\n%s", height, got, height, view)
		}
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
	if err := i18n.Load(i18n.DefaultLang); err != nil {
		t.Fatalf("Load(default lang) error = %v", err)
	}
	t.Cleanup(func() {
		if err := i18n.Load(i18n.DefaultLang); err != nil {
			t.Fatalf("restore default lang error = %v", err)
		}
	})

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
	for _, want := range []string{
		helpLine(model.binding(keymap.ContextQuizWrite, keymap.ActionHint)),
		helpLine(model.binding(keymap.ContextQuizWrite, keymap.ActionSkip)),
		helpLine(model.binding(keymap.ContextQuizWrite, keymap.ActionWriteQuit)),
	} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{
		helpLine(model.binding(keymap.ContextQuizChoice, keymap.ActionSpeak)),
		helpLine(model.binding(keymap.ContextQuizChoice, keymap.ActionToggleAutoplay)),
	} {
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
		helpLine(model.binding(keymap.ContextFeedbackWrite, keymap.ActionConfirm)),
		helpLine(model.binding(keymap.ContextFeedbackWrite, keymap.ActionSpeak)),
		helpLine(model.binding(keymap.ContextFeedbackWrite, keymap.ActionToggleAutoplay)),
		disabledHelpLine(model.binding(keymap.ContextFeedbackWrite, keymap.ActionQuit), i18n.T(i18n.HelpQuitDisabledWrite)),
	} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{
		helpLine(model.binding(keymap.ContextFeedbackRate, keymap.ActionAgain)),
		helpLine(model.binding(keymap.ContextFeedbackRate, keymap.ActionHard)),
		helpLine(model.binding(keymap.ContextFeedbackRate, keymap.ActionGood)),
		helpLine(model.binding(keymap.ContextFeedbackRate, keymap.ActionEasy)),
	} {
		if strings.Contains(helpView, unwanted) {
			t.Fatalf("renderHelp() unexpectedly contains %q:\n%s", unwanted, helpView)
		}
	}
}

func TestRenderChoiceQuizOmitsChoiceBindingsFromInlineGuide(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.screen = ScreenQuiz
	model.autoplayEnabled = true
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word: store.Word{
			Lemma: "begin",
			Pos:   "verb",
		},
		Choices: []quiz.Choice{
			{Meaning: "始める"},
			{Meaning: "開始する"},
			{Meaning: "続ける"},
			{Meaning: "終える"},
		},
		Ordinal: 1,
		Total:   4,
		Kind:    store.ItemKindNew,
	}

	quizView := model.renderQuiz()
	for _, want := range []string{"j=下に移動", "k=上に移動", "Enter=決定", "Ctrl+P=現在の単語を再生", "Shift+Tab=自動再生を切り替える", "?=ヘルプ", "q/Ctrl+C=終了"} {
		if !strings.Contains(quizView, want) {
			t.Fatalf("renderQuiz() missing %q:\n%s", want, quizView)
		}
	}
	for _, unwanted := range []string{"1=選択肢 1", "2=選択肢 2", "3=選択肢 3", "4=選択肢 4"} {
		if strings.Contains(quizView, unwanted) {
			t.Fatalf("renderQuiz() unexpectedly contains %q:\n%s", unwanted, quizView)
		}
	}
}

func TestRenderChoiceHelpKeepsChoiceBindingsWithCustomKeymap(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.screen = ScreenQuiz
	if err := model.keymap.SetKeys(keymap.ContextQuizChoice, keymap.ActionSelect1, []string{"z"}); err != nil {
		t.Fatalf("SetKeys(quiz.choice.select1) error = %v", err)
	}
	if err := model.keymap.SetKeys(keymap.ContextQuizChoice, keymap.ActionHelp, []string{"f1"}); err != nil {
		t.Fatalf("SetKeys(quiz.choice.help) error = %v", err)
	}
	model.currentQ = &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word: store.Word{
			Lemma: "begin",
			Pos:   "verb",
		},
		Choices: []quiz.Choice{
			{Meaning: "始める"},
			{Meaning: "開始する"},
			{Meaning: "続ける"},
			{Meaning: "終える"},
		},
		Ordinal: 1,
		Total:   4,
		Kind:    store.ItemKindNew,
	}

	quizView := model.renderQuiz()
	if strings.Contains(quizView, "z=選択肢 1") {
		t.Fatalf("renderQuiz() unexpectedly shows custom select binding inline:\n%s", quizView)
	}
	if !strings.Contains(quizView, "f1=ヘルプ") {
		t.Fatalf("renderQuiz() missing custom help binding inline:\n%s", quizView)
	}

	helpView := model.openHelp().renderHelp()
	if !strings.Contains(helpView, helpLine(model.binding(keymap.ContextQuizChoice, keymap.ActionSelect1))) {
		t.Fatalf("renderHelp() missing custom select binding:\n%s", helpView)
	}
	if !strings.Contains(helpView, helpLine(model.binding(keymap.ContextQuizChoice, keymap.ActionHelp))) {
		t.Fatalf("renderHelp() missing custom help binding:\n%s", helpView)
	}
}

func TestRenderWriteFeedbackHelpShowsUnboundQuitPlaceholder(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.screen = ScreenFeedback
	if err := model.keymap.SetKeys(keymap.ContextFeedbackWrite, keymap.ActionQuit, nil); err != nil {
		t.Fatalf("SetKeys(feedback.write.quit=nil) error = %v", err)
	}
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
	want := fmt.Sprintf("%-10s %s", i18n.T(i18n.KeymapUnbound), i18n.T(i18n.HelpQuitDisabledWrite))
	if !strings.Contains(helpView, want) {
		t.Fatalf("renderHelp() missing unbound quit placeholder %q:\n%s", want, helpView)
	}
	unwanted := fmt.Sprintf("%-10s %s", i18n.T(i18n.KeyQuit), i18n.T(i18n.HelpQuitDisabledWrite))
	if strings.Contains(helpView, unwanted) {
		t.Fatalf("renderHelp() unexpectedly uses action label for unbound quit %q:\n%s", unwanted, helpView)
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

func TestRenderChoiceFeedbackShowsAudioControlsInKeyGuide(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{
		Settings:       newAudioEnabledSettings(),
		SpeakerFactory: newStubSpeakerFactory(true),
	})
	model.loading = false
	model.autoplayEnabled = true
	model.feedback = &quiz.Feedback{
		Question: quiz.Question{
			AnswerMode:   store.AnswerModeChoice,
			Word:         store.Word{Lemma: "begin"},
			Choices:      []quiz.Choice{{Meaning: "始める"}},
			CorrectIndex: 0,
		},
		Correct:       true,
		SelectedIndex: 0,
		ResponseMS:    1200,
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
	model.settingsThemeMode = config.ThemeModeNeon

	got := model.renderHomeWithSettingsOverlay()
	if !strings.Contains(got, i18n.T(i18n.SettingsTitle)) {
		t.Fatalf("renderHomeWithSettingsOverlay() missing settings title:\n%s", got)
	}
	for _, want := range []string{
		i18n.T(i18n.SettingsWriteDifficulty),
		i18n.T(i18n.SettingsWriteDifficultyHard),
		i18n.T(i18n.SettingsAudioEnabled),
		i18n.T(i18n.SettingsAudioAutoplay),
		i18n.T(i18n.SettingsTheme),
		i18n.T(i18n.SettingsThemeNeon),
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
	model.settingsThemeMode = config.ThemeModeCustom

	got := model.renderHomeWithSettingsOverlay()
	if !strings.Contains(got, i18n.T(i18n.SettingsAudioAutoplay)) || !strings.Contains(got, i18n.T(i18n.AudioStateOff)) {
		t.Fatalf("renderHomeWithSettingsOverlay() should show autoplay OFF:\n%s", got)
	}
	if !strings.Contains(got, i18n.T(i18n.SettingsThemeCustomNote)) {
		t.Fatalf("renderHomeWithSettingsOverlay() missing custom theme note:\n%s", got)
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
		model.renderInlineGuides(keymap.ContextHomeConfirm, keymap.ActionConfirm, keymap.ActionBack),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHomeWithConfirmOverlay() missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("renderHomeWithConfirmOverlay() should not include home background when confirmation is open:\n%s", got)
	}
}

func TestRenderHomeMarksSelectedAnswerModeWithBrackets(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.selectedAnswerMode = store.AnswerModeWrite

	got := model.renderHome()
	if !strings.Contains(got, "["+i18n.T(i18n.AnswerModeWrite)+"]") {
		t.Fatalf("renderHome() missing selected answer mode brackets:\n%s", got)
	}
}

func TestRenderStatusLineUsesErrorPrefix(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.status = "boom"
	model.err = errors.New("boom")

	got := model.renderStatusLine()
	if !strings.Contains(got, "error: boom") {
		t.Fatalf("renderStatusLine() = %q, want error prefix", got)
	}
}

func TestViewShowsNarrowWidthFallbackByScreen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		width     int
		wantLabel string
		unwanted  string
		mouseMode tea.MouseMode
		prepare   func(*RootModel)
	}{
		{
			name:      "home",
			width:     compactWidthStandard - 1,
			wantLabel: i18n.T(i18n.HelpScreenHome),
			unwanted:  i18n.T(i18n.HomeSubtitle),
			mouseMode: tea.MouseModeNone,
			prepare: func(model *RootModel) {
				model.screen = ScreenHome
				model.home = store.HomeSnapshot{DueCount: 3, NewCount: 2, StreakDays: 1}
			},
		},
		{
			name:      "settings overlay",
			width:     compactWidthWide - 1,
			wantLabel: i18n.T(i18n.SettingsTitle),
			unwanted:  i18n.T(i18n.SettingsQuestions),
			mouseMode: tea.MouseModeNone,
			prepare: func(model *RootModel) {
				prepareCompactSettingsModel(model)
			},
		},
		{
			name:      "choice quiz",
			width:     compactWidthWide - 1,
			wantLabel: i18n.T(i18n.HelpScreenQuiz),
			unwanted:  "始める",
			mouseMode: tea.MouseModeNone,
			prepare: func(model *RootModel) {
				model.screen = ScreenQuiz
				model.currentQ = sampleChoiceQuestion()
			},
		},
		{
			name:      "keymap editor",
			width:     normalWidthKeymap - 1,
			wantLabel: i18n.T(i18n.KeymapTitle),
			unwanted:  i18n.T(i18n.KeymapContext),
			mouseMode: tea.MouseModeCellMotion,
			prepare: func(model *RootModel) {
				model.screen = ScreenKeymap
				*model = model.openKeymapEditor()
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel(nil, Options{})
			model.loading = false
			model.width = tc.width
			model.status = "status line should wrap safely in narrow mode and remain within the terminal width"
			if tc.prepare != nil {
				tc.prepare(&model)
			}

			view := model.View()
			if !strings.Contains(view.Content, tc.wantLabel) {
				t.Fatalf("View() missing label %q:\n%s", tc.wantLabel, view.Content)
			}
			if strings.Contains(view.Content, tc.unwanted) {
				t.Fatalf("View() unexpectedly contains %q:\n%s", tc.unwanted, view.Content)
			}
			if !strings.Contains(view.Content, "cols") {
				t.Fatalf("View() missing narrow width body:\n%s", view.Content)
			}
			if view.MouseMode != tc.mouseMode {
				t.Fatalf("MouseMode = %v, want %v", view.MouseMode, tc.mouseMode)
			}
			assertViewFitsWidth(t, view.Content, tc.width)
		})
	}
}

func TestViewShowsCompactLayoutByScreen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		width        int
		want         string
		unwanted     string
		normalMarker string
		prepare      func(*RootModel)
	}{
		{
			name:         "home",
			width:        compactWidthStandard,
			want:         i18n.T(i18n.HomeSubtitle),
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: "1,2,3,4=選択",
			prepare: func(model *RootModel) {
				model.screen = ScreenHome
				model.home = store.HomeSnapshot{DueCount: 3, NewCount: 2, StreakDays: 1}
				model.stats.Today.WaitMinutes = 2.5
			},
		},
		{
			name:         "settings overlay",
			width:        compactWidthWide,
			want:         i18n.T(i18n.SettingsQuestions),
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: tui.AlignLabel(i18n.T(i18n.SettingsQuestions), 18),
			prepare: func(model *RootModel) {
				prepareCompactSettingsModel(model)
			},
		},
		{
			name:         "choice quiz",
			width:        compactWidthWide,
			want:         "始める",
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: "  •  ",
			prepare: func(model *RootModel) {
				model.screen = ScreenQuiz
				model.currentQ = sampleChoiceQuestion()
			},
		},
		{
			name:         "write quiz",
			width:        compactWidthStandard,
			want:         i18n.T(i18n.QuizMeaning),
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: tui.AlignLabel(i18n.T(i18n.QuizMeaning), 14),
			prepare: func(model *RootModel) {
				model.screen = ScreenQuiz
				model.currentQ = sampleWriteQuestion()
				model.writeInput = "ap"
				model.writeHintCount = 1
			},
		},
		{
			name:         "choice feedback",
			width:        compactWidthWide,
			want:         i18n.T(i18n.FbCorrectAnswer),
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: tui.AlignLabel(i18n.T(i18n.FbCorrectAnswer), 14),
			prepare: func(model *RootModel) {
				model.screen = ScreenFeedback
				model.feedback = sampleChoiceFeedback()
			},
		},
		{
			name:         "help",
			width:        compactWidthWide,
			want:         i18n.T(i18n.HelpTitle),
			unwanted:     i18n.T(i18n.NarrowWidthTitle),
			normalMarker: "",
			prepare: func(model *RootModel) {
				model.screen = ScreenHelp
				model.helpReturn = ScreenQuiz
				model.currentQ = sampleChoiceQuestion()
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel(nil, Options{})
			model.loading = false
			model.width = tc.width
			model.status = "compact layout should stay within the available width"
			if tc.prepare != nil {
				tc.prepare(&model)
			}

			view := model.View().Content
			if !strings.Contains(view, tc.want) {
				t.Fatalf("View() missing compact marker %q:\n%s", tc.want, view)
			}
			if strings.Contains(view, tc.unwanted) {
				t.Fatalf("View() unexpectedly contains %q:\n%s", tc.unwanted, view)
			}
			if tc.normalMarker != "" && strings.Contains(view, tc.normalMarker) {
				t.Fatalf("View() unexpectedly contains normal-layout marker %q:\n%s", tc.normalMarker, view)
			}
			assertViewFitsWidth(t, view, tc.width)
		})
	}
}

func TestViewAtOrAboveNormalWidthThresholdRendersNormalScreen(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenQuiz
	model.width = normalWidthWide
	model.currentQ = sampleChoiceQuestion()

	view := model.View().Content
	if strings.Contains(view, i18n.T(i18n.NarrowWidthTitle)) {
		t.Fatalf("View() unexpectedly shows narrow width fallback at threshold:\n%s", view)
	}
	if !strings.Contains(view, "  •  ") {
		t.Fatalf("View() missing normal quiz meta separator at threshold:\n%s", view)
	}
}

func TestViewNonCompactScreensStillUseNarrowFallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		width   int
		prepare func(*RootModel)
	}{
		{
			name:  "results",
			width: normalWidthStandard - 1,
			prepare: func(model *RootModel) {
				model.screen = ScreenResults
				model.summary = &store.SessionSummary{Accuracy: 80, CorrectAnswers: 4, TotalQuestions: 5}
			},
		},
		{
			name:  "stats",
			width: normalWidthStandard - 1,
			prepare: func(model *RootModel) {
				model.screen = ScreenStats
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel(nil, Options{})
			model.loading = false
			model.width = tc.width
			tc.prepare(&model)

			view := model.View().Content
			if !strings.Contains(view, i18n.T(i18n.NarrowWidthTitle)) {
				t.Fatalf("View() missing narrow fallback for %s:\n%s", tc.name, view)
			}
			assertViewFitsWidth(t, view, tc.width)
		})
	}
}

func TestViewWidthZeroDoesNotTriggerNarrowWidthFallback(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenHome
	model.width = 0

	view := model.View().Content
	if strings.Contains(view, i18n.T(i18n.NarrowWidthTitle)) {
		t.Fatalf("View() unexpectedly shows narrow width fallback when width is unknown:\n%s", view)
	}
	if !strings.Contains(view, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("View() missing normal home content when width is unknown:\n%s", view)
	}
}

func TestViewNarrowWidthFallbackFitsVerySmallTerminal(t *testing.T) {
	t.Parallel()

	model := NewModel(nil, Options{})
	model.loading = false
	model.screen = ScreenHome
	model.width = 5
	model.status = "status line should still fit"

	view := model.View().Content
	if strings.Contains(view, i18n.T(i18n.HomeSubtitle)) {
		t.Fatalf("View() unexpectedly contains normal home content in very small terminal:\n%s", view)
	}
	assertViewFitsWidth(t, view, model.width)
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
		helpLine(model.binding(keymap.ContextHomeConfirm, keymap.ActionConfirm)),
		helpLine(model.binding(keymap.ContextHomeConfirm, keymap.ActionBack)),
	} {
		if !strings.Contains(helpView, want) {
			t.Fatalf("renderHelp() missing %q:\n%s", want, helpView)
		}
	}
	for _, unwanted := range []string{
		helpLine(model.binding(keymap.ContextHome, keymap.ActionNewSession)),
		helpLine(model.binding(keymap.ContextHome, keymap.ActionReview)),
		helpLine(model.binding(keymap.ContextHome, keymap.ActionToggleAnswerMode)),
	} {
		if strings.Contains(helpView, unwanted) {
			t.Fatalf("renderHelp() unexpectedly contains %q:\n%s", unwanted, helpView)
		}
	}
}

func TestCompactHelpersFitWidth(t *testing.T) {
	t.Parallel()

	if got := packTextPartsWidth([]string{"tab=answer mode", "enter=start", "s=settings"}, 18, "  "); strings.Contains(got, "tab=answer mode  enter=start") {
		t.Fatalf("packTextPartsWidth() failed to wrap:\n%s", got)
	}
	assertViewFitsWidth(t, packTextPartsWidth([]string{"tab=answer mode", "enter=start", "s=settings"}, 18, "  "), 18)
	assertViewFitsWidth(t, renderStackedField("Meaning", "a very long explanation that must wrap", 18), 18)
	assertViewFitsWidth(t, renderPrefixedWrap("ctrl+h: ", "show the help screen and keep every line within width", 20), 20)
}

func prepareCompactSettingsModel(model *RootModel) {
	model.screen = ScreenHome
	model.settingsOpen = true
	model.settingsInput = "10"
	model.settingsWriteDifficulty = config.WriteModeDifficultyBasic
	model.settingsAudioEnabled = true
	model.settingsLanguage = i18n.LangJA
	model.settingsThemeMode = config.ThemeModeDefault
}

func sampleChoiceQuestion() *quiz.Question {
	return &quiz.Question{
		AnswerMode: store.AnswerModeChoice,
		Word:       store.Word{Lemma: "begin", Pos: "verb"},
		Choices: []quiz.Choice{
			{Meaning: "始める"},
			{Meaning: "開始する"},
			{Meaning: "続ける"},
			{Meaning: "終える"},
		},
		Ordinal: 1,
		Total:   4,
		Kind:    store.ItemKindNew,
	}
}

func sampleWriteQuestion() *quiz.Question {
	return &quiz.Question{
		AnswerMode: store.AnswerModeWrite,
		Word: store.Word{
			Lemma:     "apply",
			MeaningJA: "応募する",
			Pos:       "verb",
		},
		Ordinal: 2,
		Total:   4,
		Kind:    store.ItemKindReview,
	}
}

func sampleChoiceFeedback() *quiz.Feedback {
	return &quiz.Feedback{
		Question: quiz.Question{
			Word:         store.Word{Lemma: "begin", ExampleEN: "We begin at dawn.", ExampleJA: "夜明けに始める。"},
			Choices:      []quiz.Choice{{Meaning: "始める"}, {Meaning: "終える"}},
			CorrectIndex: 0,
		},
		Correct:       false,
		SelectedIndex: 1,
		ResponseMS:    1250,
	}
}

func assertViewFitsWidth(t *testing.T, view string, width int) {
	t.Helper()

	for _, line := range strings.Split(view, "\n") {
		plain := ansi.Strip(line)
		if got := runewidth.StringWidth(plain); got > width {
			t.Fatalf("line width = %d, want <= %d\nline=%q\nview=\n%s", got, width, plain, view)
		}
	}
}
