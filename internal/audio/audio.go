package audio

import "context"

type Config struct {
	Enabled bool
}

type Speaker interface {
	Speak(ctx context.Context, text string) error
	Enabled() bool
}

func NewSpeaker(config Config) Speaker {
	if !config.Enabled {
		return NoopSpeaker{}
	}
	return newPlatformSpeaker()
}
