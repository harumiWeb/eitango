//go:build windows

package audio

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	windowsTestPowerShellPath    = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	windowsTestEnglishVoiceJSON  = `{"Name":"Microsoft David Desktop","Locale":"en-US"}`
	windowsTestJapaneseVoiceJSON = `{"Name":"Haruka","Locale":"ja-JP"}`
)

func TestNewSpeakerOnWindowsUsesPowerShellWhenAvailable(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	resetVoiceCatalogCache()
	windowsLookPath = func(file string) (string, error) {
		if file != "powershell.exe" {
			t.Fatalf("lookPath file = %q, want powershell.exe", file)
		}
		return windowsTestPowerShellPath, nil
	}
	windowsListVoices = func(command string) ([]byte, error) {
		if command != "powershell.exe" {
			t.Fatalf("listVoices command = %q, want %q", command, "powershell.exe")
		}
		return []byte(windowsTestEnglishVoiceJSON), nil
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
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return windowsTestPowerShellPath, nil
	}
	windowsListVoices = func(string) ([]byte, error) {
		return []byte(windowsTestJapaneseVoiceJSON), nil
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
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return windowsTestPowerShellPath, nil
	}
	windowsListVoices = func(string) ([]byte, error) {
		return []byte(windowsTestJapaneseVoiceJSON), nil
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
	if want := windowsSpeechArgs("begin", "Haruka"); !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestNewSpeakerOnWindowsFallsBackToAutoSelectionWhenCatalogUnavailable(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return windowsTestPowerShellPath, nil
	}
	windowsListVoices = func(string) ([]byte, error) {
		return nil, errors.New("temporary voice listing failure")
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
	args := command.buildArgs("begin")
	if want := windowsSpeechArgs("begin", ""); !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestInstalledVoicesOnWindowsRetriesAfterTransientFailure(t *testing.T) {
	previous := windowsLookPath
	previousVoices := windowsListVoices
	resetVoiceCatalogCache()
	windowsLookPath = func(string) (string, error) {
		return windowsTestPowerShellPath, nil
	}
	calls := 0
	windowsListVoices = func(string) ([]byte, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("temporary voice listing failure")
		}
		return []byte(windowsTestEnglishVoiceJSON), nil
	}
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsListVoices = previousVoices
		resetVoiceCatalogCache()
	})

	voices, available := InstalledVoices()
	if available {
		t.Fatal("available = true, want false after first failure")
	}
	if len(voices) != 0 {
		t.Fatalf("len(voices) = %d, want 0 after first failure", len(voices))
	}

	voices, available = InstalledVoices()
	if !available {
		t.Fatal("available = false, want true after retry")
	}
	if len(voices) != 1 || voices[0].ID != "Microsoft David Desktop" {
		t.Fatalf("voices = %+v, want recovered cached voice list", voices)
	}
	if calls != 2 {
		t.Fatalf("windowsListVoices calls = %d, want 2", calls)
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
		return []byte(windowsTestEnglishVoiceJSON), nil
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
