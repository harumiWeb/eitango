//go:build darwin

package audio

import "testing"

func TestNewSpeakerOnDarwinUsesSayWhenAvailable(t *testing.T) {
	previous := darwinLookPath
	previousVoices := darwinListVoices
	const path = "/usr/bin/say"
	resetVoiceCatalogCache()
	darwinLookPath = func(file string) (string, error) {
		if file != "say" {
			t.Fatalf("lookPath file = %q, want say", file)
		}
		return path, nil
	}
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Samantha  en_US  # Hello, my name is Samantha.\nKyoko  ja_JP  # こんにちは、私の名前はKyokoです。"), nil
	}
	t.Cleanup(func() {
		darwinLookPath = previous
		darwinListVoices = previousVoices
		resetVoiceCatalogCache()
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if !speaker.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}

	command, ok := speaker.(commandSpeaker)
	if !ok {
		t.Fatalf("speaker type = %T, want commandSpeaker", speaker)
	}
	if command.command != "/usr/bin/say" {
		t.Fatalf("command = %q, want %q", command.command, "/usr/bin/say")
	}
	args := command.buildArgs("begin")
	if len(args) != 3 {
		t.Fatalf("len(args) = %d, want 3", len(args))
	}
	if args[0] != "-v" || args[1] != "Samantha" || args[2] != "begin" {
		t.Fatalf("args = %v, want [-v Samantha begin]", args)
	}
}

func TestNewSpeakerOnDarwinUsesConfiguredVoice(t *testing.T) {
	previous := darwinLookPath
	previousVoices := darwinListVoices
	const path = "/usr/bin/say"
	resetVoiceCatalogCache()
	darwinLookPath = func(string) (string, error) {
		return path, nil
	}
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Kyoko  ja_JP  # こんにちは。\nSamantha  en_US  # Hello."), nil
	}
	t.Cleanup(func() {
		darwinLookPath = previous
		darwinListVoices = previousVoices
		resetVoiceCatalogCache()
	})

	speaker := NewSpeaker(Config{Enabled: true, Voice: "Kyoko"})
	command, ok := speaker.(commandSpeaker)
	if !ok {
		t.Fatalf("speaker type = %T, want commandSpeaker", speaker)
	}
	args := command.buildArgs("begin")
	if len(args) != 3 || args[1] != "Kyoko" {
		t.Fatalf("args = %v, want configured Kyoko voice", args)
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
	const path = "/usr/bin/say"
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Daniel  en_GB  # Hello, my name is Daniel.\nKyoko  ja_JP  # こんにちは。"), nil
	}
	t.Cleanup(func() {
		darwinListVoices = previousVoices
	})

	voice := darwinPreferredVoice(path)
	if voice != "Daniel" {
		t.Fatalf("voice = %q, want %q", voice, "Daniel")
	}
}

func TestDarwinPreferredVoicePrefersNaturalVoiceOverEarlierEnglishFallback(t *testing.T) {
	previousVoices := darwinListVoices
	const path = "/usr/bin/say"
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Whisper  en_US  # Hello.\nFred  en_US  # Hello.\nDaniel  en_GB  # Hello.\nKyoko  ja_JP  # こんにちは。"), nil
	}
	t.Cleanup(func() {
		darwinListVoices = previousVoices
	})

	voice := darwinPreferredVoice(path)
	if voice != "Daniel" {
		t.Fatalf("voice = %q, want %q", voice, "Daniel")
	}
}

func TestDarwinPreferredVoiceSkipsNoveltyVoicesDuringFallback(t *testing.T) {
	previousVoices := darwinListVoices
	const path = "/usr/bin/say"
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Whisper  en_US  # Hello.\nZarvox  en_US  # Hello.\nFiona  en-scotland  # Hello.\nKyoko  ja_JP  # こんにちは。"), nil
	}
	t.Cleanup(func() {
		darwinListVoices = previousVoices
	})

	voice := darwinPreferredVoice(path)
	if voice != "Fiona" {
		t.Fatalf("voice = %q, want %q", voice, "Fiona")
	}
}

func TestDarwinPreferredVoiceReturnsEmptyWhenOnlyNoveltyEnglishVoicesExist(t *testing.T) {
	previousVoices := darwinListVoices
	const path = "/usr/bin/say"
	darwinListVoices = func(command string) ([]byte, error) {
		if command != path {
			t.Fatalf("listVoices command = %q, want %q", command, path)
		}
		return []byte("Whisper  en_US  # Hello.\nZarvox  en_US  # Hello.\nKyoko  ja_JP  # こんにちは。"), nil
	}
	t.Cleanup(func() {
		darwinListVoices = previousVoices
	})

	voice := darwinPreferredVoice(path)
	if voice != "" {
		t.Fatalf("voice = %q, want empty string", voice)
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
