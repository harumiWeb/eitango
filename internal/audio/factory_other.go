//go:build !darwin && !windows

package audio

import "errors"

func newPlatformSpeaker(Config) Speaker {
	return NoopSpeaker{}
}

func listPlatformVoices() ([]Voice, error) {
	return nil, errors.New("audio unavailable")
}
