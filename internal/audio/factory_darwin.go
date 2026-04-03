//go:build darwin

package audio

import "os/exec"

var darwinLookPath = exec.LookPath

func newPlatformSpeaker() Speaker {
	command, err := darwinLookPath("say")
	if err != nil {
		return NoopSpeaker{}
	}
	return commandSpeaker{
		command: command,
		buildArgs: func(text string) []string {
			return []string{text}
		},
		runCommand: defaultRunCommand,
	}
}
