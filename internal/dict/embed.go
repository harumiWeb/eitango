package dict

import (
	"fmt"

	projectassets "github.com/yourname/eitango/assets"
)

const CoreWordsVersion = "2026-03-24-mvp"

func LoadCoreWords() ([]Entry, error) {
	file, err := projectassets.Embedded.Open("words_core.jsonl")
	if err != nil {
		return nil, fmt.Errorf("open embedded words: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	entries, err := ParseJSONL(file)
	if err != nil {
		return nil, fmt.Errorf("parse embedded words: %w", err)
	}

	return entries, nil
}
