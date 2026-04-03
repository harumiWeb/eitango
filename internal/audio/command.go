package audio

import (
	"context"
	"errors"
	"strings"
)

type runCommandFunc func(ctx context.Context, name string, args ...string) error

type commandSpeaker struct {
	command    string
	buildArgs  func(text string) []string
	runCommand runCommandFunc
}

func (s commandSpeaker) Speak(ctx context.Context, text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !s.Enabled() {
		return nil
	}
	if s.buildArgs == nil || s.runCommand == nil {
		return errors.New("audio command speaker is not initialized")
	}
	return s.runCommand(ctx, s.command, s.buildArgs(trimmed)...)
}

func (s commandSpeaker) Enabled() bool {
	return strings.TrimSpace(s.command) != ""
}
