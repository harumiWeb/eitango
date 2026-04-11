package audio

import (
	"context"
	"strings"
	"sync"
)

type Config struct {
	Enabled bool
	Voice   string
}

type Voice struct {
	ID     string
	Label  string
	Locale string
}

type Speaker interface {
	Speak(ctx context.Context, text string) error
	Enabled() bool
}

type voiceCatalogState struct {
	voices    []Voice
	available bool
}

var voiceCatalogOnce sync.Once
var voiceCatalog voiceCatalogState

func NewSpeaker(config Config) Speaker {
	if !config.Enabled {
		return NoopSpeaker{}
	}
	config.Voice = strings.TrimSpace(config.Voice)
	return newPlatformSpeaker(config)
}

func InstalledVoices() ([]Voice, bool) {
	catalog := cachedVoiceCatalog()
	return cloneVoices(catalog.voices), catalog.available
}

func NormalizeVoiceID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	voices, available := InstalledVoices()
	if !available {
		return ""
	}
	return selectVoiceID(value, voices)
}

func cachedVoiceCatalog() voiceCatalogState {
	voiceCatalogOnce.Do(func() {
		voices, err := listPlatformVoices()
		if err != nil {
			voiceCatalog = voiceCatalogState{}
			return
		}
		voiceCatalog = voiceCatalogState{
			voices:    cloneVoices(voices),
			available: true,
		}
	})
	return voiceCatalogState{
		voices:    cloneVoices(voiceCatalog.voices),
		available: voiceCatalog.available,
	}
}

func selectVoiceID(requested string, voices []Voice) string {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return ""
	}
	for _, voice := range voices {
		if voice.ID == requested {
			return voice.ID
		}
	}
	return ""
}

func cloneVoices(voices []Voice) []Voice {
	if len(voices) == 0 {
		return nil
	}
	cloned := make([]Voice, len(voices))
	copy(cloned, voices)
	return cloned
}

func resetVoiceCatalogCache() {
	voiceCatalogOnce = sync.Once{}
	voiceCatalog = voiceCatalogState{}
}
