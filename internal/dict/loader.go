package dict

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
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

func normalizeEntry(entry Entry) Entry {
	entry.Lemma = strings.TrimSpace(entry.Lemma)
	entry.Pos = strings.TrimSpace(entry.Pos)
	entry.MeaningJA = strings.TrimSpace(entry.MeaningJA)
	entry.Level = strings.TrimSpace(entry.Level)
	entry.DistractorGroup = strings.TrimSpace(entry.DistractorGroup)
	entry.ExampleEN = strings.TrimSpace(entry.ExampleEN)
	entry.ExampleJA = strings.TrimSpace(entry.ExampleJA)
	return entry
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

		entry = normalizeEntry(entry)

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
		"frequency_rank",
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

		frequencyRank, err := csvOptionalPositiveInt(record, indexByColumn, "frequency_rank", rowNo)
		if err != nil {
			return nil, err
		}

		entry := normalizeEntry(Entry{
			Lemma:           csvField(record, indexByColumn, "lemma"),
			Pos:             csvField(record, indexByColumn, "pos"),
			MeaningJA:       csvField(record, indexByColumn, "meaning_ja"),
			Level:           csvField(record, indexByColumn, "level"),
			FrequencyRank:   frequencyRank,
			DistractorGroup: csvField(record, indexByColumn, "distractor_group"),
			ExampleEN:       csvField(record, indexByColumn, "example_en"),
			ExampleJA:       csvField(record, indexByColumn, "example_ja"),
		})
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

func csvOptionalPositiveInt(record []string, indexByColumn map[string]int, column string, rowNo int) (int, error) {
	raw := csvField(record, indexByColumn, column)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse csv row %d: %s must be an integer", rowNo, column)
	}
	if value <= 0 {
		return 0, fmt.Errorf("parse csv row %d: %s must be positive", rowNo, column)
	}
	return value, nil
}

func ValidateCoreEntries(entries []Entry) error {
	if len(entries) == 0 {
		return fmt.Errorf("no core entries found")
	}

	seenLemmaPos := make(map[string]int, len(entries))
	seenRanks := make(map[int]int, len(entries))
	groupCounts := make(map[string]int)

	for i, rawEntry := range entries {
		entryNo := i + 1
		entry := normalizeEntry(rawEntry)

		if entry.Lemma == "" {
			return fmt.Errorf("core entry %d: lemma is required", entryNo)
		}
		if entry.MeaningJA == "" {
			return fmt.Errorf("core entry %d (%s): meaning_ja is required", entryNo, entry.Lemma)
		}
		if entry.Pos == "" {
			return fmt.Errorf("core entry %d (%s): pos is required", entryNo, entry.Lemma)
		}
		if entry.Level == "" {
			return fmt.Errorf("core entry %d (%s): level is required", entryNo, entry.Lemma)
		}
		if entry.FrequencyRank <= 0 {
			return fmt.Errorf("core entry %d (%s): frequency_rank must be positive", entryNo, entry.Lemma)
		}
		if entry.DistractorGroup == "" {
			return fmt.Errorf("core entry %d (%s): distractor_group is required", entryNo, entry.Lemma)
		}

		key := strings.ToLower(entry.Pos + "\x00" + entry.Lemma)
		if previousEntry, exists := seenLemmaPos[key]; exists {
			return fmt.Errorf("core entry %d (%s/%s) duplicates lemma/pos from entry %d", entryNo, entry.Lemma, entry.Pos, previousEntry)
		}
		seenLemmaPos[key] = entryNo

		if previousEntry, exists := seenRanks[entry.FrequencyRank]; exists {
			return fmt.Errorf("core entry %d (%s) duplicates frequency_rank %d from entry %d", entryNo, entry.Lemma, entry.FrequencyRank, previousEntry)
		}
		seenRanks[entry.FrequencyRank] = entryNo
		groupCounts[entry.DistractorGroup]++
	}

	var smallGroups []string
	for group, count := range groupCounts {
		if count < 4 {
			smallGroups = append(smallGroups, group)
		}
	}
	if len(smallGroups) > 0 {
		slices.Sort(smallGroups)
		group := smallGroups[0]
		return fmt.Errorf("core distractor_group %q has %d entries, want at least 4", group, groupCounts[group])
	}

	return nil
}
