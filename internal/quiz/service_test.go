package quiz

import (
	"math/rand"
	"testing"

	"github.com/yourname/eitango/internal/store"
)

func TestBuildChoices(t *testing.T) {
	correct := store.Word{ID: 1, Lemma: "abandon", Pos: "verb", MeaningJA: "捨てる", Level: "toeic600", FrequencyRank: 3400, DistractorGroup: "basic-verb-action"}
	pool := []store.Word{
		{ID: 2, Lemma: "acquire", Pos: "verb", MeaningJA: "得る", Level: "toeic600", FrequencyRank: 3600, DistractorGroup: "basic-verb-action"},
		{ID: 3, Lemma: "arrange", Pos: "verb", MeaningJA: "手配する", Level: "toeic600", FrequencyRank: 3900, DistractorGroup: "basic-verb-action"},
		{ID: 4, Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "toeic600", FrequencyRank: 3200, DistractorGroup: "basic-verb-action"},
		{ID: 5, Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "toeic600", FrequencyRank: 3700, DistractorGroup: "basic-verb-action"},
	}

	choices, err := BuildChoices(correct, pool, 4, nil, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("BuildChoices() error = %v", err)
	}
	if len(choices) != 4 {
		t.Fatalf("len(choices) = %d, want 4", len(choices))
	}

	foundCorrect := false
	for _, choice := range choices {
		if choice.WordID == correct.ID {
			foundCorrect = true
		}
	}
	if !foundCorrect {
		t.Fatalf("correct answer missing from choices: %+v", choices)
	}
}
