package quiz

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/harumiWeb/eitango/internal/store"
)

type WordStore interface {
	GetWord(ctx context.Context, wordID int64) (store.Word, error)
	ListWordsByPOS(ctx context.Context, pos string, limit int, excludeIDs []int64) ([]store.Word, error)
	ListDistractorCandidates(ctx context.Context, correct store.Word, limit int, excludeIDs []int64) ([]store.Word, error)
}

type Service struct {
	store WordStore
	rng   *rand.Rand
}

func NewService(store WordStore) *Service {
	return &Service{
		store: store,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) BuildQuestion(ctx context.Context, item store.SessionItem, total int, recentDistractors []int64) (Question, error) {
	correct, err := s.store.GetWord(ctx, item.WordID)
	if err != nil {
		return Question{}, err
	}

	exclude := uniqueIDs(append([]int64{correct.ID}, recentDistractors...))
	pool, err := s.store.ListDistractorCandidates(ctx, correct, 64, exclude)
	if err != nil {
		return Question{}, err
	}
	if len(pool) < 3 {
		fallbackPool, err := s.store.ListDistractorCandidates(ctx, correct, 64, []int64{correct.ID})
		if err != nil {
			return Question{}, err
		}
		pool = mergePools(pool, fallbackPool)
	}

	choices, err := BuildChoices(correct, pool, 4, recentDistractors, s.rng)
	if err != nil {
		return Question{}, err
	}

	correctIndex := 0
	for i, choice := range choices {
		if choice.WordID == correct.ID {
			correctIndex = i
			break
		}
	}

	return Question{
		Word:         correct,
		Choices:      choices,
		CorrectIndex: correctIndex,
		Ordinal:      item.Ordinal,
		Total:        total,
		Kind:         item.Kind,
	}, nil
}

func BuildChoices(correct store.Word, pool []store.Word, n int, recentDistractors []int64, rng *rand.Rand) ([]Choice, error) {
	if n < 2 {
		return nil, fmt.Errorf("choice count must be at least 2")
	}

	recentPenalty := make(map[int64]int, len(recentDistractors))
	for _, id := range recentDistractors {
		recentPenalty[id]++
	}

	type scoredWord struct {
		word  store.Word
		score int
		diff  int
	}

	scored := make([]scoredWord, 0, len(pool))
	for _, candidate := range pool {
		if candidate.ID == correct.ID {
			continue
		}
		if candidate.MeaningJA == correct.MeaningJA {
			continue
		}
		diff := absInt(candidate.FrequencyRank - correct.FrequencyRank)
		score := 0
		if candidate.DistractorGroup != "" && candidate.DistractorGroup == correct.DistractorGroup {
			score += 5
		}
		if candidate.Level != "" && candidate.Level == correct.Level {
			score += 3
		}
		switch {
		case diff <= 1500:
			score += 2
		case diff <= 4000:
			score += 1
		}
		if penalty, ok := recentPenalty[candidate.ID]; ok {
			score -= 4 * penalty
		}

		scored = append(scored, scoredWord{word: candidate, score: score, diff: diff})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].diff != scored[j].diff {
			return scored[i].diff < scored[j].diff
		}
		return scored[i].word.Lemma < scored[j].word.Lemma
	})

	chosen := make([]store.Word, 0, n-1)
	usedMeanings := map[string]struct{}{correct.MeaningJA: {}}
	for _, candidate := range scored {
		if _, exists := usedMeanings[candidate.word.MeaningJA]; exists {
			continue
		}
		chosen = append(chosen, candidate.word)
		usedMeanings[candidate.word.MeaningJA] = struct{}{}
		if len(chosen) == n-1 {
			break
		}
	}

	if len(chosen) < n-1 {
		return nil, fmt.Errorf("not enough distractors for %s", correct.Lemma)
	}

	words := make([]store.Word, 0, n)
	words = append(words, correct)
	words = append(words, chosen...)
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	rng.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})

	choices := make([]Choice, 0, len(words))
	for _, word := range words {
		choices = append(choices, Choice{WordID: word.ID, Meaning: word.MeaningJA})
	}

	return choices, nil
}

func mergePools(primary, fallback []store.Word) []store.Word {
	seen := make(map[int64]struct{}, len(primary))
	merged := make([]store.Word, 0, len(primary)+len(fallback))
	for _, word := range primary {
		seen[word.ID] = struct{}{}
		merged = append(merged, word)
	}
	for _, word := range fallback {
		if _, exists := seen[word.ID]; exists {
			continue
		}
		seen[word.ID] = struct{}{}
		merged = append(merged, word)
	}
	return merged
}

func uniqueIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func absInt(v int) int {
	return int(math.Abs(float64(v)))
}
