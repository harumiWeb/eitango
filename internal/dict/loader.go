package dict

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
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

func ParseCSV(r io.Reader) ([]Entry, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("no entries found in csv")
		}
		return nil, fmt.Errorf("read csv header: %w", err)
	}

	indexByColumn := make(map[string]int, len(header))
	for i, raw := range header {
		name := strings.TrimSpace(raw)
		if i == 0 {
			name = strings.TrimPrefix(name, "\ufeff")
		}
		if name == "" {
			continue
		}
		if _, exists := indexByColumn[name]; exists {
			return nil, fmt.Errorf("parse csv header: duplicate column %q", name)
		}
		indexByColumn[name] = i
	}

	for _, column := range []string{"lemma", "meaning_ja"} {
		if _, ok := indexByColumn[column]; !ok {
			return nil, fmt.Errorf("parse csv header: %s column is required", column)
		}
	}

	allowedColumns := []string{
		"lemma",
		"meaning_ja",
		"pos",
		"level",
		"distractor_group",
		"example_en",
		"example_ja",
	}
	for column := range indexByColumn {
		if !slices.Contains(allowedColumns, column) {
			return nil, fmt.Errorf("parse csv header: unsupported column %q", column)
		}
	}

	entries := make([]Entry, 0, 128)
	rowNo := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row %d: %w", rowNo+1, err)
		}
		rowNo++

		entry := Entry{
			Lemma:           csvField(record, indexByColumn, "lemma"),
			Pos:             csvField(record, indexByColumn, "pos"),
			MeaningJA:       csvField(record, indexByColumn, "meaning_ja"),
			Level:           csvField(record, indexByColumn, "level"),
			DistractorGroup: csvField(record, indexByColumn, "distractor_group"),
			ExampleEN:       csvField(record, indexByColumn, "example_en"),
			ExampleJA:       csvField(record, indexByColumn, "example_ja"),
		}
		if entry.Lemma == "" {
			return nil, fmt.Errorf("parse csv row %d: lemma is required", rowNo)
		}
		if entry.MeaningJA == "" {
			return nil, fmt.Errorf("parse csv row %d: meaning_ja is required", rowNo)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in csv")
	}

	return entries, nil
}

func csvField(record []string, indexByColumn map[string]int, column string) string {
	index, ok := indexByColumn[column]
	if !ok || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}
