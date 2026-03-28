package dict

import (
	"strings"
	"testing"
)

func TestLoadCoreWordsPhase1Pack(t *testing.T) {
	entries, err := LoadCoreWords()
	if err != nil {
		t.Fatalf("LoadCoreWords() error = %v", err)
	}

	if len(entries) < 1000 {
		t.Fatalf("len(entries) = %d, want at least 1000", len(entries))
	}

	posCounts := make(map[string]int)
	groupCounts := make(map[string]int)
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Lemma == "" || entry.Pos == "" || entry.MeaningJA == "" || entry.Level == "" || entry.DistractorGroup == "" {
			t.Fatalf("embedded entry has blank required field: %+v", entry)
		}
		if entry.FrequencyRank <= 0 {
			t.Fatalf("embedded entry has invalid frequency rank: %+v", entry)
		}

		key := strings.ToLower(entry.Pos + "\x00" + entry.Lemma)
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate lemma/pos in embedded core words: %s %s", entry.Pos, entry.Lemma)
		}
		seen[key] = struct{}{}

		posCounts[entry.Pos]++
		groupCounts[entry.DistractorGroup]++
	}

	if len(posCounts) < 4 {
		t.Fatalf("len(posCounts) = %d, want at least 4", len(posCounts))
	}
	for group, count := range groupCounts {
		if count < 4 {
			t.Fatalf("distractor group %q has %d entries, want at least 4", group, count)
		}
	}
}
