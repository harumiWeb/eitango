//go:build darwin

package audio

import (
	"os/exec"
	"regexp"
	"strings"
)

var darwinLookPath = exec.LookPath
var darwinListVoices = defaultDarwinListVoices

var darwinVoiceLocalePattern = regexp.MustCompile(`\s([[:alnum:]_-]+)\s+#`)

func newPlatformSpeaker() Speaker {
	command, err := darwinLookPath("say")
	if err != nil {
		return NoopSpeaker{}
	}

	voice := darwinPreferredVoice()
	return commandSpeaker{
		command: command,
		buildArgs: func(text string) []string {
			if voice != "" {
				return []string{"-v", voice, text}
			}
			return []string{text}
		},
		runCommand: defaultRunCommand,
	}
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
