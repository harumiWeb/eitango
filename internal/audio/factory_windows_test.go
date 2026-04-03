//go:build windows

package audio

import (
	"strings"
	"testing"
)

func TestNewSpeakerOnWindowsUsesPowerShellWhenAvailable(t *testing.T) {
	previous := windowsLookPath
	previousProbe := windowsVoiceProbe
	windowsLookPath = func(file string) (string, error) {
		if file != "powershell.exe" {
			t.Fatalf("lookPath file = %q, want powershell.exe", file)
		}
		return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", nil
	}
	windowsVoiceProbe = func() bool { return true }
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsVoiceProbe = previousProbe
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if !speaker.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}
}

func TestNewSpeakerOnWindowsReturnsNoopWhenEnglishVoiceUnavailable(t *testing.T) {
	previous := windowsLookPath
	previousProbe := windowsVoiceProbe
	windowsLookPath = func(string) (string, error) {
		return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", nil
	}
	windowsVoiceProbe = func() bool { return false }
	t.Cleanup(func() {
		windowsLookPath = previous
		windowsVoiceProbe = previousProbe
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if speaker.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
}

func TestWindowsSpeechArgsEscapesSingleQuotes(t *testing.T) {
	t.Parallel()

	args := windowsSpeechArgs("can't")
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
