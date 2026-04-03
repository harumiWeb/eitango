package audio

import (
	"context"
	"testing"
)

func TestNoopSpeakerDisabled(t *testing.T) {
	t.Parallel()

	speaker := NoopSpeaker{}
	if speaker.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
	if err := speaker.Speak(context.Background(), "begin"); err != nil {
		t.Fatalf("Speak() error = %v", err)
	}
}

func TestNewSpeakerDisabledReturnsNoop(t *testing.T) {
	t.Parallel()

	speaker := NewSpeaker(Config{Enabled: false})
	if speaker.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
}

func TestCommandSpeakerSkipsBlankInput(t *testing.T) {
	t.Parallel()

	called := false
	speaker := commandSpeaker{
		command: "test-bin",
		buildArgs: func(string) []string {
			t.Fatal("buildArgs should not be called for blank input")
			return nil
		},
		runCommand: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}

	if err := speaker.Speak(context.Background(), "   "); err != nil {
		t.Fatalf("Speak() error = %v", err)
	}
	if called {
		t.Fatal("runCommand was called for blank input")
	}
}

func TestCommandSpeakerReturnsErrorWhenUninitialized(t *testing.T) {
	t.Parallel()

	speaker := commandSpeaker{command: "say"}
	if err := speaker.Speak(context.Background(), "begin"); err == nil {
		t.Fatal("Speak() error = nil, want initialization error")
	}
}
