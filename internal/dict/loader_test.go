package dict

import (
	"strings"
	"testing"
)

func TestParseCSVParsesRequiredAndOptionalColumns(t *testing.T) {
	t.Parallel()

	entries, err := ParseCSV(strings.NewReader(strings.Join([]string{
		"lemma,meaning_ja,pos,level,frequency_rank,distractor_group,example_en,example_ja",
		" apply , 応募する , verb , toeic600 , 1200 , basic-verb-action , She will apply. , 彼女は応募する。 ",
	}, "\n")))
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Lemma != "apply" || entry.MeaningJA != "応募する" {
		t.Fatalf("unexpected parsed entry: %+v", entry)
	}
	if entry.Pos != "verb" || entry.Level != "toeic600" || entry.DistractorGroup != "basic-verb-action" {
		t.Fatalf("unexpected optional fields: %+v", entry)
	}
	if entry.FrequencyRank != 1200 {
		t.Fatalf("entry.FrequencyRank = %d, want 1200", entry.FrequencyRank)
	}
	if entry.ExampleEN != "She will apply." || entry.ExampleJA != "彼女は応募する。" {
		t.Fatalf("unexpected examples: %+v", entry)
	}
}

func TestParseCSVRejectsInvalidFrequencyRank(t *testing.T) {
	t.Parallel()

	_, err := ParseCSV(strings.NewReader(strings.Join([]string{
		"lemma,meaning_ja,frequency_rank",
		"apply,応募する,not-a-number",
	}, "\n")))
	if err == nil {
		t.Fatal("ParseCSV() error = nil, want frequency_rank validation error")
	}
	if got := err.Error(); got != "parse csv row 2: frequency_rank must be an integer" {
		t.Fatalf("ParseCSV() error = %q, want frequency_rank guidance", got)
	}
}

func TestParseCSVRejectsMissingRequiredHeader(t *testing.T) {
	t.Parallel()

	_, err := ParseCSV(strings.NewReader("lemma,pos\napply,verb\n"))
	if err == nil {
		t.Fatal("ParseCSV() error = nil, want header validation error")
	}
	if got := err.Error(); got != "parse csv header: meaning_ja column is required" {
		t.Fatalf("ParseCSV() error = %q, want meaning_ja guidance", got)
	}
}

func TestParseCSVHandlesUTF8BOMHeader(t *testing.T) {
	t.Parallel()

	entries, err := ParseCSV(strings.NewReader("\ufefflemma,meaning_ja\napply,応募する\n"))
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Lemma != "apply" {
		t.Fatalf("unexpected BOM-parsed entries: %+v", entries)
	}
}

func TestParseCSVRejectsDuplicateHeader(t *testing.T) {
	t.Parallel()

	_, err := ParseCSV(strings.NewReader("lemma,meaning_ja,lemma\napply,応募する,ignored\n"))
	if err == nil {
		t.Fatal("ParseCSV() error = nil, want duplicate-header error")
	}
	if got := err.Error(); got != `parse csv header: duplicate column "lemma"` {
		t.Fatalf("ParseCSV() error = %q, want duplicate header guidance", got)
	}
}

func TestValidateCoreEntriesAcceptsValidEntries(t *testing.T) {
	t.Parallel()

	if err := ValidateCoreEntries(validCoreEntries()); err != nil {
		t.Fatalf("ValidateCoreEntries() error = %v", err)
	}
}

func TestValidateCoreEntriesRejectsMissingCoreMetadata(t *testing.T) {
	t.Parallel()

	entries := validCoreEntries()
	entries[0].Level = ""

	err := ValidateCoreEntries(entries)
	if err == nil {
		t.Fatal("ValidateCoreEntries() error = nil, want missing metadata error")
	}
	if got := err.Error(); got != "core entry 1 (accept): level is required" {
		t.Fatalf("ValidateCoreEntries() error = %q, want level guidance", got)
	}
}

func TestValidateCoreEntriesRejectsDuplicateFrequencyRank(t *testing.T) {
	t.Parallel()

	entries := validCoreEntries()
	entries[1].FrequencyRank = entries[0].FrequencyRank

	err := ValidateCoreEntries(entries)
	if err == nil {
		t.Fatal("ValidateCoreEntries() error = nil, want duplicate rank error")
	}
	if got := err.Error(); got != "core entry 2 (avoid) duplicates frequency_rank 100 from entry 1" {
		t.Fatalf("ValidateCoreEntries() error = %q, want duplicate-rank guidance", got)
	}
}

func TestValidateCoreEntriesRejectsSmallDistractorGroup(t *testing.T) {
	t.Parallel()

	err := ValidateCoreEntries([]Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", Level: "toeic600", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "toeic600", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "collect", Pos: "verb", MeaningJA: "収集する", Level: "toeic600", FrequencyRank: 140, DistractorGroup: "basic-verb-action"},
	})
	if err == nil {
		t.Fatal("ValidateCoreEntries() error = nil, want distractor-group validation error")
	}
	if got := err.Error(); got != `core distractor_group "basic-verb-action" has 3 entries, want at least 4` {
		t.Fatalf("ValidateCoreEntries() error = %q, want distractor-group guidance", got)
	}
}

func validCoreEntries() []Entry {
	return []Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", Level: "toeic600", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "toeic600", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "collect", Pos: "verb", MeaningJA: "収集する", Level: "toeic600", FrequencyRank: 140, DistractorGroup: "basic-verb-action"},
		{Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "toeic600", FrequencyRank: 160, DistractorGroup: "basic-verb-action"},
	}
}
