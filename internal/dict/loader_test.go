package dict

import (
	"strings"
	"testing"
)

func TestParseJSONL(t *testing.T) {
	input := strings.NewReader(`
{"lemma":"abandon","pos":"verb","meaning_ja":"捨てる","level":"toeic600","frequency_rank":3400,"distractor_group":"basic-verb-action"}
{"lemma":"budget","pos":"noun","meaning_ja":"予算","level":"toeic600","frequency_rank":3500,"distractor_group":"business-noun"}
`)

	entries, err := ParseJSONL(input)
	if err != nil {
		t.Fatalf("ParseJSONL() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Lemma != "abandon" {
		t.Fatalf("entries[0].Lemma = %q, want abandon", entries[0].Lemma)
	}
}
