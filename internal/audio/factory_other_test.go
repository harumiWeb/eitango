//go:build !darwin && !windows

package audio

import "testing"

func TestNewSpeakerUnsupportedReturnsNoop(t *testing.T) {
	t.Parallel()

	speaker := NewSpeaker(Config{Enabled: true})
	if speaker.Enabled() {
		t.Fatal("Enabled() = true, want false on unsupported platform")
	}
}
