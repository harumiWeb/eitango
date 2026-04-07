//go:build darwin

package audio

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var darwinLookPath = exec.LookPath
var darwinListVoices = defaultDarwinListVoices

var darwinVoiceLocalePattern = regexp.MustCompile(`\s([[:alnum:]_-]+)\s+#`)

var darwinPreferredVoiceNames = []string{
	"Samantha",
	"Alex",
	"Daniel",
	"Karen",
	"Moira",
}

func newPlatformSpeaker() Speaker {
	command, err := darwinLookPath("say")
	if err != nil {
		return NoopSpeaker{}
	}
	command, ok := normalizeDarwinSayCommand(command)
	if !ok {
		return NoopSpeaker{}
	}

	voice := darwinPreferredVoice(command)
	return commandSpeaker{
		command: command,
		buildArgs: func(text string) []string {
			if voice != "" {
				return []string{"-v", voice, text}
			}
			return []string{text}
		},
		runCommand: runDarwinSay,
	}
}

func runDarwinSay(ctx context.Context, name string, args ...string) error {
	if _, ok := normalizeDarwinSayCommand(name); !ok {
		return errors.New("unsupported say command")
	}
	return exec.CommandContext(ctx, "/usr/bin/say", args...).Run()
}

func defaultDarwinListVoices(name string) ([]byte, error) {
	if _, ok := normalizeDarwinSayCommand(name); !ok {
		return nil, errors.New("unsupported say command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "/usr/bin/say", "-v", "?").Output()
}

func darwinPreferredVoice(command string) string {
	output, err := darwinListVoices(command)
	if err != nil {
		return ""
	}

	type voiceInfo struct {
		name   string
		locale string
	}

	var voices []voiceInfo
	for _, line := range strings.Split(string(output), "\n") {
		voice, locale, ok := parseDarwinVoiceLine(line)
		if !ok {
			continue
		}
		voices = append(voices, voiceInfo{name: voice, locale: locale})
	}

	for _, preferred := range darwinPreferredVoiceNames {
		for _, voice := range voices {
			if voice.name == preferred && isDarwinEnglishLocale(voice.locale) {
				return voice.name
			}
		}
	}

	for _, voice := range voices {
		if isDarwinEnglishLocale(voice.locale) && !isDarwinNoveltyVoice(voice.name) {
			return voice.name
		}
	}

	return ""
}

func parseDarwinVoiceLine(line string) (voice string, locale string, ok bool) {
	match := darwinVoiceLocalePattern.FindStringSubmatchIndex(line)
	if match == nil {
		return "", "", false
	}
	voice = strings.TrimSpace(line[:match[2]])
	locale = line[match[2]:match[3]]
	if voice == "" || locale == "" {
		return "", "", false
	}
	return voice, locale, true
}

func isDarwinEnglishLocale(locale string) bool {
	return strings.HasPrefix(strings.ToLower(locale), "en")
}

func isDarwinNoveltyVoice(name string) bool {
	switch strings.ToLower(name) {
	case "bad news", "bells", "boing", "bubbles", "cellos",
		"deranged", "good news", "hysterical", "pipe organ",
		"princess", "trinoids", "whisper", "zarvox":
		return true
	default:
		return false
	}
}

func normalizeDarwinSayCommand(name string) (string, bool) {
	cleaned := filepath.Clean(strings.TrimSpace(name))
	switch cleaned {
	case "say", "/usr/bin/say":
		return "/usr/bin/say", true
	default:
		return "", false
	}
}
