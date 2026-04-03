//go:build darwin

package audio

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

var darwinLookPath = exec.LookPath
var darwinListVoices = defaultDarwinListVoices

var darwinVoiceLocalePattern = regexp.MustCompile(`\s([[:alnum:]_-]+)\s+#`)

func newPlatformSpeaker() Speaker {
	if _, err := darwinLookPath("say"); err != nil {
		return NoopSpeaker{}
	}

	voice := darwinPreferredVoice()
	return commandSpeaker{
		command: "say",
		buildArgs: func(text string) []string {
			if voice != "" {
				return []string{"-v", voice, text}
			}
			return []string{text}
		},
		runCommand: runDarwinSay,
	}
}

func runDarwinSay(ctx context.Context, _ string, args ...string) error {
	return exec.CommandContext(ctx, "say", args...).Run()
}

func defaultDarwinListVoices() ([]byte, error) {
	return exec.Command("say", "-v", "?").Output()
}

func darwinPreferredVoice() string {
	output, err := darwinListVoices()
	if err != nil {
		return ""
	}
	fallback := ""
	for _, line := range strings.Split(string(output), "\n") {
		voice, locale, ok := parseDarwinVoiceLine(line)
		if !ok {
			continue
		}
		if locale == "en_US" {
			return voice
		}
		if fallback == "" && strings.HasPrefix(strings.ToLower(locale), "en") {
			fallback = voice
		}
	}
	return fallback
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
