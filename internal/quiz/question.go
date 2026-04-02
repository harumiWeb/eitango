package quiz

import (
	"strings"

	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
)

type Choice struct {
	WordID  int64
	Meaning string
}

type Question struct {
	Word         store.Word
	AnswerMode   string
	Choices      []Choice
	CorrectIndex int
	Ordinal      int
	Total        int
	Kind         string
}

type Feedback struct {
	Question      Question
	SelectedIndex int
	SelectedText  string
	Correct       bool
	ResponseMS    int64
	HintCount     int
	Skipped       bool
	Rating        srs.Rating
}

func BuildFeedback(question Question, selectedIndex int, responseMS int64) Feedback {
	correct := selectedIndex == question.CorrectIndex
	return Feedback{
		Question:      question,
		SelectedIndex: selectedIndex,
		Correct:       correct,
		ResponseMS:    responseMS,
	}
}

func BuildWriteFeedback(question Question, typed string, hintCount int, skipped bool, forceIncorrect bool, responseMS int64) Feedback {
	normalizedTyped := NormalizeWriteAnswer(typed)
	correct := !forceIncorrect && !skipped && normalizedTyped == NormalizeWriteAnswer(question.Word.Lemma)
	rating := srs.Again
	switch {
	case correct && hintCount == 0:
		rating = srs.Easy
	case correct:
		rating = srs.Good
	}
	return Feedback{
		Question:     question,
		SelectedText: strings.TrimSpace(typed),
		Correct:      correct,
		ResponseMS:   responseMS,
		HintCount:    hintCount,
		Skipped:      skipped,
		Rating:       rating,
	}
}

func (q Question) CorrectChoice() Choice {
	if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Choices) {
		return Choice{}
	}
	return q.Choices[q.CorrectIndex]
}

func (q Question) DistractorIDs() []int64 {
	if q.AnswerMode != store.AnswerModeChoice {
		return nil
	}
	ids := make([]int64, 0, len(q.Choices)-1)
	for i, choice := range q.Choices {
		if i == q.CorrectIndex {
			continue
		}
		ids = append(ids, choice.WordID)
	}
	return ids
}

func NormalizeWriteAnswer(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
