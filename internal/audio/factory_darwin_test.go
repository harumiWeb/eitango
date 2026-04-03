//go:build darwin

package audio

import "testing"

func TestNewSpeakerOnDarwinUsesSayWhenAvailable(t *testing.T) {
	previous := darwinLookPath
	previousVoices := darwinListVoices
	darwinLookPath = func(file string) (string, error) {
		if file != "say" {
			t.Fatalf("lookPath file = %q, want say", file)
		}
		return "/usr/bin/say", nil
	}
	darwinListVoices = func() ([]byte, error) {
		return []byte("Samantha  en_US  # Hello, my name is Samantha.\nKyoko  ja_JP  # こんにちは、私の名前はKyokoです。"), nil
	}
	t.Cleanup(func() {
		darwinLookPath = previous
		darwinListVoices = previousVoices
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if !speaker.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}

	command, ok := speaker.(commandSpeaker)
	if !ok {
		t.Fatalf("speaker type = %T, want commandSpeaker", speaker)
	}
	args := command.buildArgs("begin")
	if len(args) != 3 {
		t.Fatalf("len(args) = %d, want 3", len(args))
	}
	if args[0] != "-v" || args[1] != "Samantha" || args[2] != "begin" {
		t.Fatalf("args = %v, want [-v Samantha begin]", args)
	}
}

func TestParseDarwinVoiceLineExtractsVoiceAndLocale(t *testing.T) {
	t.Parallel()

	voice, locale, ok := parseDarwinVoiceLine("Grandpa (English (US))    en_US    # Hello! My name is Grandpa.")
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if voice != "Grandpa (English (US))" {
		t.Fatalf("voice = %q, want %q", voice, "Grandpa (English (US))")
	}
	if locale != "en_US" {
		t.Fatalf("locale = %q, want %q", locale, "en_US")
	}
}

func TestDarwinPreferredVoiceFallsBackToOtherEnglishLocale(t *testing.T) {
	previousVoices := darwinListVoices
	darwinListVoices = func() ([]byte, error) {
		return []byte("Daniel  en_GB  # Hello, my name is Daniel.\nKyoko  ja_JP  # こんにちは。"), nil
	}
	t.Cleanup(func() {
		darwinListVoices = previousVoices
	})

	voice := darwinPreferredVoice()
	if voice != "Daniel" {
		t.Fatalf("voice = %q, want %q", voice, "Daniel")
	}
}

func TestParseDarwinVoiceLineAcceptsHyphenatedEnglishLocale(t *testing.T) {
	t.Parallel()

	voice, locale, ok := parseDarwinVoiceLine("Fiona    en-scotland    # Hello! My name is Fiona.")
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if voice != "Fiona" {
		t.Fatalf("voice = %q, want %q", voice, "Fiona")
	}
	if locale != "en-scotland" {
		t.Fatalf("locale = %q, want %q", locale, "en-scotland")
	}
}
