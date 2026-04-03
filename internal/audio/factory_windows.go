//go:build windows

package audio

import (
	"context"
	"os/exec"
	"strings"
)

var windowsLookPath = exec.LookPath
var windowsVoiceProbe = probeWindowsEnglishVoice

func newPlatformSpeaker() Speaker {
	if _, err := windowsLookPath("powershell.exe"); err != nil {
		return NoopSpeaker{}
	}
	if !windowsVoiceProbe() {
		return NoopSpeaker{}
	}
	return commandSpeaker{
		command:    "powershell.exe",
		buildArgs:  windowsSpeechArgs,
		runCommand: runWindowsPowerShell,
	}
}

func windowsSpeechArgs(text string) []string {
	quoted := "'" + escapePowerShellSingleQuoted(text) + "'"
	return windowsPowerShellArgs(windowsSpeechScript("$synth.Speak(" + quoted + ")"))
}

func windowsSpeechProbeArgs() []string {
	return windowsPowerShellArgs(windowsSpeechScript("exit 0"))
}

func windowsPowerShellArgs(script string) []string {
	return []string{"-NoProfile", "-NonInteractive", "-Command", script}
}

func windowsSpeechScript(body string) string {
	return "$ErrorActionPreference='Stop'; " +
		"Add-Type -AssemblyName System.Speech; " +
		"$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; " +
		windowsEnglishVoiceSelectionScript() +
		body
}

func windowsEnglishVoiceSelectionScript() string {
	return "$voice = $synth.GetInstalledVoices() | " +
		"ForEach-Object { $_.VoiceInfo } | " +
		"Where-Object { $_.Culture.Name -eq 'en-US' } | " +
		"Select-Object -First 1; " +
		"if ($null -eq $voice) { " +
		"$voice = $synth.GetInstalledVoices() | " +
		"ForEach-Object { $_.VoiceInfo } | " +
		"Where-Object { $_.Culture.Name -like 'en-*' } | " +
		"Select-Object -First 1 " +
		"}; " +
		"if ($null -eq $voice) { throw 'no english voice installed' }; " +
		"$synth.SelectVoice($voice.Name); "
}

func probeWindowsEnglishVoice() bool {
	return exec.Command("powershell.exe", windowsSpeechProbeArgs()...).Run() == nil
}

func runWindowsPowerShell(ctx context.Context, _ string, args ...string) error {
	return exec.CommandContext(ctx, "powershell.exe", args...).Run()
}

func escapePowerShellSingleQuoted(text string) string {
	return strings.ReplaceAll(text, "'", "''")
}
