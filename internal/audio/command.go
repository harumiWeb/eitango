package audio

import (
	"context"
	"os/exec"
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
	if trimmed == "" {
		return nil
	}
	return s.runCommand(ctx, s.command, s.buildArgs(trimmed)...)
}

func (s commandSpeaker) Enabled() bool {
	return strings.TrimSpace(s.command) != ""
}

func defaultRunCommand(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}
