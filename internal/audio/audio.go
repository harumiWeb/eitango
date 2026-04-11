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

var voiceCatalogMu sync.Mutex
var voiceCatalogCached bool
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
		return value
	}
	return selectVoiceID(value, voices)
}

func cachedVoiceCatalog() voiceCatalogState {
	voiceCatalogMu.Lock()
	defer voiceCatalogMu.Unlock()

	if !voiceCatalogCached {
		voices, err := listPlatformVoices()
		if err != nil {
			return voiceCatalogState{}
		}
		voiceCatalog = voiceCatalogState{
			voices:    cloneVoices(voices),
			available: true,
		}
		voiceCatalogCached = true
	}
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
