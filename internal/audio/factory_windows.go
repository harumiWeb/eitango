//go:build windows

package audio

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var windowsLookPath = exec.LookPath
var windowsVoiceProbe = probeWindowsEnglishVoice

func newPlatformSpeaker() Speaker {
	command, err := windowsLookPath("powershell.exe")
	if err != nil {
		return NoopSpeaker{}
	}
	command, ok := normalizeWindowsPowerShellCommand(command)
	if !ok {
		return NoopSpeaker{}
	}
	if !windowsVoiceProbe(command) {
		return NoopSpeaker{}
	}
	return commandSpeaker{
		command:    command,
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

func probeWindowsEnglishVoice(command string) bool {
	if _, ok := normalizeWindowsPowerShellCommand(command); !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "powershell.exe", windowsSpeechProbeArgs()...).Run() == nil
}

func runWindowsPowerShell(ctx context.Context, name string, args ...string) error {
	if _, ok := normalizeWindowsPowerShellCommand(name); !ok {
		return errors.New("unsupported powershell command")
	}
	return exec.CommandContext(ctx, "powershell.exe", args...).Run()
}

func escapePowerShellSingleQuoted(text string) string {
	return strings.ReplaceAll(text, "'", "''")
}

func normalizeWindowsPowerShellCommand(name string) (string, bool) {
	cleaned := strings.ToLower(filepath.Clean(strings.TrimSpace(name)))
	switch cleaned {
	case "powershell.exe":
		return "powershell.exe", true
	case strings.ToLower(`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`):
		return "powershell.exe", true
	default:
		return "", false
	}
}
