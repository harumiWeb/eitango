//go:build windows

package audio

import (
	"strings"
	"testing"
)

func TestNewSpeakerOnWindowsUsesPowerShellWhenAvailable(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	const path = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	resetVoiceCatalogCache()
	windowsLookPath = func(file string) (string, error) {
		if file != "powershell.exe" {
			t.Fatalf("lookPath file = %q, want powershell.exe", file)
		}
		return path, nil
	}
	windowsListVoices = func(command string) ([]byte, error) {
		if command != "powershell.exe" {
			t.Fatalf("listVoices command = %q, want %q", command, "powershell.exe")
		}
		return []byte(`{"Name":"Microsoft David Desktop","Locale":"en-US"}`), nil
	}
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsListVoices = previousVoices
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
	if command.command != "powershell.exe" {
		t.Fatalf("command = %q, want %q", command.command, "powershell.exe")
	}
}

func TestNewSpeakerOnWindowsReturnsNoopWhenEnglishVoiceUnavailable(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	const path = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return path, nil
	}
	windowsListVoices = func(string) ([]byte, error) {
		return []byte(`{"Name":"Haruka","Locale":"ja-JP"}`), nil
	}
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsListVoices = previousVoices
		resetVoiceCatalogCache()
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if speaker.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
}

func TestNewSpeakerOnWindowsUsesConfiguredVoiceEvenWithoutEnglishFallback(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	const path = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return path, nil
	}
	windowsListVoices = func(string) ([]byte, error) {
		return []byte(`{"Name":"Haruka","Locale":"ja-JP"}`), nil
	}
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsListVoices = previousVoices
		resetVoiceCatalogCache()
	})

	speaker := NewSpeaker(Config{Enabled: true, Voice: "Haruka"})
	if !speaker.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}
	command, ok := speaker.(commandSpeaker)
	if !ok {
		t.Fatalf("speaker type = %T, want commandSpeaker", speaker)
	}
	args := command.buildArgs("begin")
	if !strings.Contains(args[3], "$synth.SelectVoice('Haruka')") {
		t.Fatalf("script = %q, want configured Haruka voice", args[3])
	}
}

func TestWindowsSpeechArgsEscapesSingleQuotes(t *testing.T) {
	t.Parallel()

	args := windowsSpeechArgs("can't", "")
	if len(args) != 4 {
		t.Fatalf("len(args) = %d, want 4", len(args))
	}
	if args[0] != "-NoProfile" || args[1] != "-NonInteractive" || args[2] != "-Command" {
		t.Fatalf("args prefix = %v, want PowerShell flags", args[:3])
	}
	if !strings.Contains(args[3], "$_.Culture.Name -eq 'en-US'") {
		t.Fatalf("script = %q, want en-US voice selection", args[3])
	}
	if !strings.Contains(args[3], "throw 'no english voice installed'") {
		t.Fatalf("script = %q, want missing voice failure", args[3])
	}
	if !strings.Contains(args[3], "$synth.SelectVoice($voice.Name)") {
		t.Fatalf("script = %q, want explicit voice selection", args[3])
	}
	if !strings.Contains(args[3], "$synth.Speak('can''t')") {
		t.Fatalf("script = %q, want escaped single quote", args[3])
	}
}

func TestWindowsSpeechArgsUsesExplicitVoiceWhenProvided(t *testing.T) {
	t.Parallel()

	args := windowsSpeechArgs("begin", "Haruka")
	if !strings.Contains(args[3], "$synth.SelectVoice('Haruka')") {
		t.Fatalf("script = %q, want configured voice selection", args[3])
	}
	if strings.Contains(args[3], "$_.Culture.Name -eq 'en-US'") {
		t.Fatalf("script = %q, must not use english auto-selection when voice is explicit", args[3])
	}
}

func TestWindowsVoicesAcceptSingleObjectJSON(t *testing.T) {
	previous := windowsListVoices
	resetVoiceCatalogCache()
	windowsListVoices = func(command string) ([]byte, error) {
		if command != "powershell.exe" {
			t.Fatalf("listVoices command = %q, want %q", command, "powershell.exe")
		}
		return []byte(`{"Name":"Microsoft David Desktop","Locale":"en-US"}`), nil
	}
	t.Cleanup(func() {
		windowsListVoices = previous
		resetVoiceCatalogCache()
	})

	voices, err := windowsVoices("powershell.exe")
	if err != nil {
		t.Fatalf("windowsVoices() error = %v", err)
	}
	if len(voices) != 1 {
		t.Fatalf("len(voices) = %d, want 1", len(voices))
	}
	if voices[0].ID != "Microsoft David Desktop" {
		t.Fatalf("voices[0].ID = %q, want %q", voices[0].ID, "Microsoft David Desktop")
	}
}
