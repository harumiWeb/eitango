package dict

import (
	"strings"
	"testing"
)

func TestParseCSVParsesRequiredAndOptionalColumns(t *testing.T) {
	t.Parallel()

	entries, err := ParseCSV(strings.NewReader(strings.Join([]string{
		"lemma,meaning_ja,pos,level,distractor_group,example_en,example_ja",
		" apply , 応募する , verb , toeic600 , basic-verb-action , She will apply. , 彼女は応募する。 ",
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
	if entry.ExampleEN != "She will apply." || entry.ExampleJA != "彼女は応募する。" {
		t.Fatalf("unexpected examples: %+v", entry)
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
