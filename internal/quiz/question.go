package quiz

import "github.com/yourname/eitango/internal/store"

type Choice struct {
	WordID  int64
	Meaning string
}

type Question struct {
	Word         store.Word
	Choices      []Choice
	CorrectIndex int
	Ordinal      int
	Total        int
	Kind         string
}

type Feedback struct {
	Question      Question
	SelectedIndex int
	Correct       bool
	ResponseMS    int64
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

func (q Question) CorrectChoice() Choice {
	if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Choices) {
		return Choice{}
	}
	return q.Choices[q.CorrectIndex]
}

func (q Question) DistractorIDs() []int64 {
	ids := make([]int64, 0, len(q.Choices)-1)
	for i, choice := range q.Choices {
		if i == q.CorrectIndex {
			continue
		}
		ids = append(ids, choice.WordID)
	}
	return ids
}
