//go:build windows

package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var windowsLookPath = exec.LookPath
var windowsListVoices = defaultWindowsListVoices

type windowsVoiceInfo struct {
	Name   string `json:"Name"`
	Locale string `json:"Locale"`
}

func newPlatformSpeaker(config Config) Speaker {
	command, err := windowsPowerShellCommand()
	if err != nil {
		return NoopSpeaker{}
	}

	voices, available := InstalledVoices()
	voice := strings.TrimSpace(config.Voice)
	if available {
		voice = selectVoiceID(voice, voices)
		if voice == "" {
			voice = windowsPreferredVoice(voices)
		}
		if voice == "" {
			return NoopSpeaker{}
		}
	}
	return commandSpeaker{
		command: command,
		buildArgs: func(text string) []string {
			return windowsSpeechArgs(text, voice)
		},
		runCommand: runWindowsPowerShell,
	}
}

func listPlatformVoices() ([]Voice, error) {
	command, err := windowsPowerShellCommand()
	if err != nil {
		return nil, err
	}
	return windowsVoices(command)
}

func windowsSpeechArgs(text string, voice string) []string {
	quoted := "'" + escapePowerShellSingleQuoted(text) + "'"
	return windowsPowerShellArgs(windowsSpeechScript(voice, "$synth.Speak("+quoted+")"))
}

func windowsListVoicesArgs() []string {
	return windowsPowerShellArgs(windowsListVoicesScript())
}

func windowsPowerShellArgs(script string) []string {
	return []string{"-NoProfile", "-NonInteractive", "-Command", script}
}

func windowsSpeechScript(voice string, body string) string {
	return "$ErrorActionPreference='Stop'; " +
		"Add-Type -AssemblyName System.Speech; " +
		"$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; " +
		windowsVoiceSelectionScript(voice) +
		body
}

func windowsVoiceSelectionScript(voice string) string {
	if strings.TrimSpace(voice) != "" {
		return "$synth.SelectVoice('" + escapePowerShellSingleQuoted(voice) + "'); "
	}
	return windowsEnglishVoiceSelectionScript()
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

func windowsListVoicesScript() string {
	return "$ErrorActionPreference='Stop'; " +
		"Add-Type -AssemblyName System.Speech; " +
		"$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; " +
		"@($synth.GetInstalledVoices() | ForEach-Object { " +
		"$voice = $_.VoiceInfo; " +
		"[PSCustomObject]@{ Name = $voice.Name; Locale = $voice.Culture.Name }" +
		"}) | ConvertTo-Json -Compress"
}

func defaultWindowsListVoices(command string) ([]byte, error) {
	if _, ok := normalizeWindowsPowerShellCommand(command); !ok {
		return nil, errors.New("unsupported powershell command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "powershell.exe", windowsListVoicesArgs()...).Output()
}

func runWindowsPowerShell(ctx context.Context, name string, args ...string) error {
	if _, ok := normalizeWindowsPowerShellCommand(name); !ok {
		return errors.New("unsupported powershell command")
	}
	return exec.CommandContext(ctx, "powershell.exe", args...).Run()
}

func windowsVoices(command string) ([]Voice, error) {
	output, err := windowsListVoices(command)
	if err != nil {
		return nil, err
	}

	raw, err := decodeWindowsVoiceList(output)
	if err != nil {
		return nil, err
	}

	voices := make([]Voice, 0, len(raw))
	for _, voice := range raw {
		name := strings.TrimSpace(voice.Name)
		if name == "" {
			continue
		}
		voices = append(voices, Voice{
			ID:     name,
			Label:  name,
			Locale: strings.TrimSpace(voice.Locale),
		})
	}
	return voices, nil
}

func decodeWindowsVoiceList(output []byte) ([]windowsVoiceInfo, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, errors.New("empty windows voice list")
	}

	if trimmed[0] == '{' {
		var single windowsVoiceInfo
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return nil, err
		}
		return []windowsVoiceInfo{single}, nil
	}

	var raw []windowsVoiceInfo
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func windowsPreferredVoice(voices []Voice) string {
	for _, voice := range voices {
		if strings.EqualFold(strings.TrimSpace(voice.Locale), "en-US") {
			return voice.ID
		}
	}
	for _, voice := range voices {
		if isWindowsEnglishLocale(voice.Locale) {
			return voice.ID
		}
	}
	return ""
}

func isWindowsEnglishLocale(locale string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en")
}

func windowsPowerShellCommand() (string, error) {
	command, err := windowsLookPath("powershell.exe")
	if err != nil {
		return "", err
	}
	command, ok := normalizeWindowsPowerShellCommand(command)
	if !ok {
		return "", errors.New("unsupported powershell command")
	}
	return command, nil
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
