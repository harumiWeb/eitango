package keymap

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveAppliesGlobalAndContextOverrides(t *testing.T) {
	t.Parallel()

	state, err := Resolve(Config{
		Global: map[string][]string{
			string(ActionConfirm): {"space"},
		},
		Results: map[string][]string{
			string(ActionConfirm): {"enter"},
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got := state.Keys(ContextHome, ActionConfirm); !reflect.DeepEqual(got, []string{"space"}) {
		t.Fatalf("home confirm = %v, want [space]", got)
	}
	if got := state.Keys(ContextResults, ActionConfirm); !reflect.DeepEqual(got, []string{"enter"}) {
		t.Fatalf("results confirm = %v, want [enter]", got)
	}
}

func TestResolveRejectsLetterBindingsInWriteContext(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Config{
		Quiz: QuizConfig{
			Write: map[string][]string{
				string(ActionSkip): {"s"},
			},
		},
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "write input") {
		t.Fatalf("Resolve() error = %v, want write input validation", err)
	}
}

func TestResolveAllowsNonASCIILetterBindingsInWriteContext(t *testing.T) {
	t.Parallel()

	state, err := Resolve(Config{
		Quiz: QuizConfig{
			Write: map[string][]string{
				string(ActionSkip): {"あ"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got := state.Keys(ContextQuizWrite, ActionSkip); !reflect.DeepEqual(got, []string{"あ"}) {
		t.Fatalf("quiz.write.skip = %v, want [あ]", got)
	}
}

func TestResolveRejectsHelpWithoutEscapeBinding(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Config{
		Help: map[string][]string{
			string(ActionBack): {},
			string(ActionQuit): {},
		},
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least one of back or quit") {
		t.Fatalf("Resolve() error = %v, want help escape validation", err)
	}
}

func TestStateSetKeysRejectsClearingLastHelpEscapeBinding(t *testing.T) {
	t.Parallel()

	state := DefaultState()
	if err := state.SetKeys(ContextHelp, ActionBack, nil); err != nil {
		t.Fatalf("SetKeys(help.back=nil) error = %v", err)
	}
	if err := state.SetKeys(ContextHelp, ActionQuit, nil); err == nil {
		t.Fatal("SetKeys(help.quit=nil) error = nil, want error")
	}
}

func TestStateToConfigRoundTripsOverride(t *testing.T) {
	t.Parallel()

	state := DefaultState()
	if err := state.SetKeys(ContextHome, ActionToggleAnswerMode, []string{"x"}); err != nil {
		t.Fatalf("SetKeys() error = %v", err)
	}

	cfg := state.ToConfig()
	if got := cfg.Home[string(ActionToggleAnswerMode)]; !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("config home.toggle_answer_mode = %v, want [x]", got)
	}

	resolved, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got := resolved.Keys(ContextHome, ActionToggleAnswerMode); !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("resolved keys = %v, want [x]", got)
	}
}
