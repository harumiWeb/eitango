package quiz

import (
	"math/rand"
	"testing"

	"github.com/yourname/eitango/internal/store"
)

func BenchmarkBuildChoices(b *testing.B) {
	correct := store.Word{ID: 1, Lemma: "abandon", Pos: "verb", MeaningJA: "捨てる", Level: "toeic600", FrequencyRank: 3400, DistractorGroup: "basic-verb-action"}
	pool := []store.Word{
		{ID: 2, Lemma: "acquire", Pos: "verb", MeaningJA: "得る", Level: "toeic600", FrequencyRank: 3600, DistractorGroup: "basic-verb-action"},
		{ID: 3, Lemma: "arrange", Pos: "verb", MeaningJA: "手配する", Level: "toeic600", FrequencyRank: 3900, DistractorGroup: "basic-verb-action"},
		{ID: 4, Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "toeic600", FrequencyRank: 3200, DistractorGroup: "basic-verb-action"},
		{ID: 5, Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "toeic600", FrequencyRank: 3700, DistractorGroup: "basic-verb-action"},
		{ID: 6, Lemma: "delegate", Pos: "verb", MeaningJA: "委任する", Level: "toeic800", FrequencyRank: 5200, DistractorGroup: "business-verb"},
		{ID: 7, Lemma: "expand", Pos: "verb", MeaningJA: "拡大する", Level: "toeic800", FrequencyRank: 6500, DistractorGroup: "change-verb"},
	}

	rng := rand.New(rand.NewSource(1))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := BuildChoices(correct, pool, 4, nil, rng); err != nil {
			b.Fatalf("BuildChoices() error = %v", err)
		}
	}
}
