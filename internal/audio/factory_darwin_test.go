//go:build darwin

package audio

import "testing"

func TestNewSpeakerOnDarwinUsesSayWhenAvailable(t *testing.T) {
	t.Parallel()

	previous := darwinLookPath
	darwinLookPath = func(file string) (string, error) {
		if file != "say" {
			t.Fatalf("lookPath file = %q, want say", file)
		}
		return "/usr/bin/say", nil
	}
	t.Cleanup(func() {
		darwinLookPath = previous
	})

	speaker := NewSpeaker(Config{Enabled: true})
	if !speaker.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}
}
