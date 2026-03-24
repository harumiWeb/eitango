package session

import (
	"math"

	"github.com/yourname/eitango/internal/store"
)

const DefaultQuestionCount = 10

type Plan struct {
	NewCount    int
	ReviewCount int
	RetryCount  int
}

type Runtime struct {
	Session store.SessionRecord
	Items   []store.SessionItem
}

func MakePlan(total, dueAvailable, newAvailable int, mode string) Plan {
	if total <= 0 {
		total = DefaultQuestionCount
	}

	if mode == store.ModeReview {
		reviewCount := minInt(total, dueAvailable)
		return Plan{ReviewCount: reviewCount}
	}

	retryBudget := 0
	if total >= 5 {
		retryBudget = 1
	}

	reviewTarget := int(math.Round(float64(total) * 0.7))
	if reviewTarget > total {
		reviewTarget = total
	}
	if reviewTarget == 0 && dueAvailable > 0 {
		reviewTarget = 1
	}
	newTarget := total - reviewTarget

	reviewCount := minInt(reviewTarget, dueAvailable)
	newCount := minInt(newTarget, newAvailable)

	remaining := total - reviewCount - newCount
	if remaining > 0 {
		extraReview := minInt(remaining, dueAvailable-reviewCount)
		reviewCount += extraReview
		remaining -= extraReview
	}
	if remaining > 0 {
		extraNew := minInt(remaining, newAvailable-newCount)
		newCount += extraNew
	}

	return Plan{
		NewCount:    newCount,
		ReviewCount: reviewCount,
		RetryCount:  retryBudget,
	}
}

func BuildSessionItems(reviewWords, newWords []store.Word) []store.SessionItemPlan {
	items := make([]store.SessionItemPlan, 0, len(reviewWords)+len(newWords))

	reviewIndex := 0
	newIndex := 0
	for reviewIndex < len(reviewWords) || newIndex < len(newWords) {
		for i := 0; i < 2 && reviewIndex < len(reviewWords); i++ {
			items = append(items, store.SessionItemPlan{WordID: reviewWords[reviewIndex].ID, Kind: store.ItemKindReview})
			reviewIndex++
		}
		if newIndex < len(newWords) {
			items = append(items, store.SessionItemPlan{WordID: newWords[newIndex].ID, Kind: store.ItemKindNew})
			newIndex++
		}
	}

	return items
}

func NewRuntime(record store.SessionRecord, items []store.SessionItem) *Runtime {
	return &Runtime{Session: record, Items: items}
}

func (r *Runtime) CurrentItem() (store.SessionItem, bool) {
	for _, item := range r.Items {
		if item.Status == store.ItemStatusPending {
			return item, true
		}
	}
	return store.SessionItem{}, false
}

func (r *Runtime) AnsweredCount() int {
	count := 0
	for _, item := range r.Items {
		if item.Status == store.ItemStatusAnswered {
			count++
		}
	}
	return count
}

func (r *Runtime) PendingCount() int {
	count := 0
	for _, item := range r.Items {
		if item.Status == store.ItemStatusPending {
			count++
		}
	}
	return count
}

func (r *Runtime) Total() int {
	if r.Session.TotalQuestions > 0 {
		return r.Session.TotalQuestions
	}
	return len(r.Items)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
