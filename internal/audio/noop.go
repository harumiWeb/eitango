package audio

import "context"

type NoopSpeaker struct{}

func (NoopSpeaker) Speak(context.Context, string) error {
	return nil
}

func (NoopSpeaker) Enabled() bool {
	return false
}
