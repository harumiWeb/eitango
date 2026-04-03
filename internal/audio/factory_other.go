//go:build !darwin && !windows

package audio

func newPlatformSpeaker() Speaker {
	return NoopSpeaker{}
}
