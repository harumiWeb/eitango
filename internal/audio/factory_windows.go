//go:build windows

package audio

import (
	"os/exec"
	"strings"
)

var windowsLookPath = exec.LookPath

func newPlatformSpeaker() Speaker {
	command, err := windowsLookPath("powershell.exe")
	if err != nil {
		return NoopSpeaker{}
	}
	return commandSpeaker{
		command:    command,
		buildArgs:  windowsSpeechArgs,
		runCommand: defaultRunCommand,
	}
}

func windowsSpeechArgs(text string) []string {
	quoted := "'" + escapePowerShellSingleQuoted(text) + "'"
	script := "$ErrorActionPreference='Stop'; " +
		"Add-Type -AssemblyName System.Speech; " +
		"$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; " +
		"$voice = $synth.GetInstalledVoices() | " +
		"ForEach-Object { $_.VoiceInfo } | " +
		"Where-Object { $_.Culture.Name -eq 'en-US' } | " +
		"Select-Object -First 1; " +
		"if ($null -eq $voice) { " +
		"$voice = $synth.GetInstalledVoices() | " +
		"ForEach-Object { $_.VoiceInfo } | " +
		"Where-Object { $_.Culture.Name -like 'en-*' } | " +
		"Select-Object -First 1 " +
		"}; " +
		"if ($null -ne $voice) { $synth.SelectVoice($voice.Name) }; " +
		"$synth.Speak(" + quoted + ")"
	return []string{"-NoProfile", "-NonInteractive", "-Command", script}
}

func escapePowerShellSingleQuoted(text string) string {
	return strings.ReplaceAll(text, "'", "''")
}
