package dict

import "testing"

func TestValidateImportEntriesAcceptsValidEntries(t *testing.T) {
	t.Parallel()

	err := ValidateImportEntries([]Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", FrequencyRank: 100},
		{Lemma: "budget", Pos: "noun", MeaningJA: "予算", FrequencyRank: 200},
	})
	if err != nil {
		t.Fatalf("ValidateImportEntries() error = %v", err)
	}
}

func TestValidateImportEntriesRejectsDuplicateLemmaPos(t *testing.T) {
	t.Parallel()

	err := ValidateImportEntries([]Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる"},
		{Lemma: " accept ", Pos: " verb ", MeaningJA: "承諾する"},
	})
	if err == nil {
		t.Fatal("ValidateImportEntries() error = nil, want duplicate lemma/pos error")
	}
	if got := err.Error(); got != "import entry 2 (accept/verb) duplicates lemma/pos from entry 1" {
		t.Fatalf("ValidateImportEntries() error = %q, want duplicate guidance", got)
	}
}

func TestValidateImportEntriesRejectsDuplicateFrequencyRank(t *testing.T) {
	t.Parallel()

	err := ValidateImportEntries([]Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", FrequencyRank: 100},
		{Lemma: "budget", Pos: "noun", MeaningJA: "予算", FrequencyRank: 100},
	})
	if err == nil {
		t.Fatal("ValidateImportEntries() error = nil, want duplicate rank error")
	}
	if got := err.Error(); got != "import entry 2 (budget) duplicates frequency_rank 100 from entry 1" {
		t.Fatalf("ValidateImportEntries() error = %q, want duplicate-rank guidance", got)
	}
}

func TestSummarizeEntries(t *testing.T) {
	t.Parallel()

	summary := SummarizeEntries([]Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "budget", Pos: "noun", MeaningJA: "予算", Level: "core-3", FrequencyRank: 200, DistractorGroup: "business-noun"},
		{Lemma: "calm", Pos: "adjective", MeaningJA: "落ち着いた", Level: "core-1", DistractorGroup: "emotion-adjective"},
	})

	if summary.EntryCount != 3 || summary.PosCount != 3 || summary.LevelCount != 2 || summary.DistractorGroupCount != 3 || summary.FrequencyRankedEntries != 2 {
		t.Fatalf("SummarizeEntries() = %+v, want counts 3/3/2/3/2", summary)
	}
}
