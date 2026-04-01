package quiz

import (
	"context"
	"math/rand"
	"testing"

	"github.com/harumiWeb/eitango/internal/store"
)

func TestBuildChoices(t *testing.T) {
	correct := store.Word{ID: 1, Lemma: "abandon", Pos: "verb", MeaningJA: "捨てる", Level: "core-1", FrequencyRank: 3400, DistractorGroup: "basic-verb-action"}
	pool := []store.Word{
		{ID: 2, Lemma: "acquire", Pos: "verb", MeaningJA: "得る", Level: "core-1", FrequencyRank: 3600, DistractorGroup: "basic-verb-action"},
		{ID: 3, Lemma: "arrange", Pos: "verb", MeaningJA: "手配する", Level: "core-1", FrequencyRank: 3900, DistractorGroup: "basic-verb-action"},
		{ID: 4, Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "core-1", FrequencyRank: 3200, DistractorGroup: "basic-verb-action"},
		{ID: 5, Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "core-1", FrequencyRank: 3700, DistractorGroup: "basic-verb-action"},
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

func TestBuildWriteQuestion(t *testing.T) {
	t.Parallel()

	svc := NewService(stubWordStore{
		word: store.Word{
			ID:        1,
			Lemma:     "begin",
			MeaningJA: "始める",
			Pos:       "verb",
		},
	})

	question, err := svc.BuildQuestion(context.Background(), store.SessionItem{
		WordID:    1,
		Ordinal:   2,
		Kind:      store.ItemKindReview,
		SessionID: "session-1",
	}, 7, store.AnswerModeWrite, nil)
	if err != nil {
		t.Fatalf("BuildQuestion() error = %v", err)
	}
	if question.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("AnswerMode = %q, want %q", question.AnswerMode, store.AnswerModeWrite)
	}
	if len(question.Choices) != 0 {
		t.Fatalf("len(Choices) = %d, want 0", len(question.Choices))
	}
	if question.Word.Lemma != "begin" || question.Word.MeaningJA != "始める" {
		t.Fatalf("unexpected write question word: %+v", question.Word)
	}
}

func TestNormalizeWriteAnswer(t *testing.T) {
	t.Parallel()

	if got := NormalizeWriteAnswer(" Begin "); got != "begin" {
		t.Fatalf("NormalizeWriteAnswer() = %q, want %q", got, "begin")
	}
}

type stubWordStore struct {
	word store.Word
}

func (s stubWordStore) GetWord(context.Context, int64) (store.Word, error) {
	return s.word, nil
}

func (s stubWordStore) ListWordsByPOS(context.Context, string, int, []int64) ([]store.Word, error) {
	return nil, nil
}

func (s stubWordStore) ListDistractorCandidates(context.Context, store.Word, int, []int64) ([]store.Word, error) {
	return nil, nil
}
