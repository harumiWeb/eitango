package dict

import (
	"fmt"
	"strings"
)

type ValidationSummary struct {
	EntryCount             int
	PosCount               int
	LevelCount             int
	DistractorGroupCount   int
	FrequencyRankedEntries int
}

func ValidateImportEntries(entries []Entry) error {
	if len(entries) == 0 {
		return fmt.Errorf("no import entries found")
	}

	seenLemmaPos := make(map[string]int, len(entries))
	seenRanks := make(map[int]int, len(entries))

	for i, rawEntry := range entries {
		entryNo := i + 1
		entry := normalizeEntry(rawEntry)

		if entry.Lemma == "" {
			return fmt.Errorf("import entry %d: lemma is required", entryNo)
		}
		if entry.MeaningJA == "" {
			return fmt.Errorf("import entry %d (%s): meaning_ja is required", entryNo, entry.Lemma)
		}

		key := strings.ToLower(entry.Pos + "\x00" + entry.Lemma)
		if previousEntry, exists := seenLemmaPos[key]; exists {
			return fmt.Errorf("import entry %d (%s/%s) duplicates lemma/pos from entry %d", entryNo, entry.Lemma, entry.Pos, previousEntry)
		}
		seenLemmaPos[key] = entryNo

		if entry.FrequencyRank > 0 {
			if previousEntry, exists := seenRanks[entry.FrequencyRank]; exists {
				return fmt.Errorf("import entry %d (%s) duplicates frequency_rank %d from entry %d", entryNo, entry.Lemma, entry.FrequencyRank, previousEntry)
			}
			seenRanks[entry.FrequencyRank] = entryNo
		}
	}

	return nil
}

func SummarizeEntries(entries []Entry) ValidationSummary {
	posSet := make(map[string]struct{})
	levelSet := make(map[string]struct{})
	groupSet := make(map[string]struct{})
	frequencyRankedEntries := 0

	for _, rawEntry := range entries {
		entry := normalizeEntry(rawEntry)
		if entry.Pos != "" {
			posSet[entry.Pos] = struct{}{}
		}
		if entry.Level != "" {
			levelSet[entry.Level] = struct{}{}
		}
		if entry.DistractorGroup != "" {
			groupSet[entry.DistractorGroup] = struct{}{}
		}
		if entry.FrequencyRank > 0 {
			frequencyRankedEntries++
		}
	}

	return ValidationSummary{
		EntryCount:             len(entries),
		PosCount:               len(posSet),
		LevelCount:             len(levelSet),
		DistractorGroupCount:   len(groupSet),
		FrequencyRankedEntries: frequencyRankedEntries,
	}
}
