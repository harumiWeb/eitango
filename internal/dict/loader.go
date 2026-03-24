package dict

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Entry struct {
	Lemma           string `json:"lemma"`
	Pos             string `json:"pos"`
	MeaningJA       string `json:"meaning_ja"`
	Level           string `json:"level"`
	FrequencyRank   int    `json:"frequency_rank"`
	DistractorGroup string `json:"distractor_group"`
	ExampleEN       string `json:"example_en"`
	ExampleJA       string `json:"example_ja"`
}

func ParseJSONL(r io.Reader) ([]Entry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	entries := make([]Entry, 0, 128)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parse jsonl line %d: %w", lineNo, err)
		}

		entry.Lemma = strings.TrimSpace(entry.Lemma)
		entry.Pos = strings.TrimSpace(entry.Pos)
		entry.MeaningJA = strings.TrimSpace(entry.MeaningJA)
		entry.Level = strings.TrimSpace(entry.Level)
		entry.DistractorGroup = strings.TrimSpace(entry.DistractorGroup)
		entry.ExampleEN = strings.TrimSpace(entry.ExampleEN)
		entry.ExampleJA = strings.TrimSpace(entry.ExampleJA)

		if entry.Lemma == "" {
			return nil, fmt.Errorf("parse jsonl line %d: lemma is required", lineNo)
		}
		if entry.MeaningJA == "" {
			return nil, fmt.Errorf("parse jsonl line %d: meaning_ja is required", lineNo)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan jsonl: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in jsonl")
	}

	return entries, nil
}
