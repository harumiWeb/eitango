// Package i18n provides message localisation for the eitango TUI.
//
// Locale files (TOML) are embedded from assets/locale/ at compile time.
// Call Load once at startup; after that T and Tf are safe for concurrent use
// from Bubble Tea's View (called every frame) because the underlying map is
// read-only after initialisation.
package i18n

import (
	"fmt"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/harumiWeb/eitango/assets"
)

const (
	LangJA = "ja"
	LangEN = "en"

	DefaultLang = LangJA
)

// messages holds the flattened key→text map for the active language.
var messages map[string]string

// fallback holds the flattened key→text map for the fallback language (en).
var fallback map[string]string

var mu sync.RWMutex

func init() {
	// Pre-load with the default language so T() works even if Load() is
	// never called (e.g. in tests that don't care about language).
	_ = Load(DefaultLang)
}

// Load parses the embedded locale file for lang and sets it as the active
// language. Unknown languages fall back to the default.
func Load(lang string) error {
	lang = normaliseLang(lang)

	primary, err := loadLocale(lang)
	if err != nil {
		return fmt.Errorf("load locale %s: %w", lang, err)
	}

	fb, err := loadLocale(fallbackLang(lang))
	if err != nil {
		// If the fallback itself fails, use an empty map so T() still
		// returns something useful (the key).
		fb = map[string]string{}
	}

	mu.Lock()
	messages = primary
	fallback = fb
	mu.Unlock()
	return nil
}

// T returns the localised string for key.
// Falls back to the other language, then to the raw key.
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if v, ok := messages[key]; ok {
		return v
	}
	if v, ok := fallback[key]; ok {
		return v
	}
	return key
}

// Tf is a convenience wrapper around fmt.Sprintf(T(key), args...).
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// ValidLang reports whether lang is a supported language code.
func ValidLang(lang string) bool {
	switch normaliseLang(lang) {
	case LangJA, LangEN:
		return true
	}
	return false
}

// --- internal helpers ---

func loadLocale(lang string) (map[string]string, error) {
	path := "locale/" + lang + ".toml"
	data, err := assets.Embedded.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	flat := make(map[string]string)
	flatten("", raw, flat)
	return flat, nil
}

// flatten recursively walks nested TOML tables and produces dotted keys.
//
//	[home]
//	subtitle = "…"
//
// becomes  "home.subtitle" → "…"
func flatten(prefix string, m map[string]any, out map[string]string) {
	for k, v := range m {
		full := k
		if prefix != "" {
			full = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]any:
			flatten(full, val, out)
		case string:
			out[full] = val
		default:
			out[full] = fmt.Sprint(val)
		}
	}
}

func normaliseLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	// Accept "ja_JP", "ja-JP" etc.
	if strings.HasPrefix(lang, "ja") {
		return LangJA
	}
	if strings.HasPrefix(lang, "en") {
		return LangEN
	}
	if lang == "" {
		return DefaultLang
	}
	return DefaultLang
}

func fallbackLang(lang string) string {
	if lang == LangEN {
		return LangJA
	}
	return LangEN
}
