package session

import (
	"testing"

	"github.com/yourname/eitango/internal/store"
)

func TestMakePlanLearn(t *testing.T) {
	plan := MakePlan(DefaultPlanOptions(), 7, 5, store.ModeLearn)
	if plan.ReviewCount != 7 {
		t.Fatalf("ReviewCount = %d, want 7", plan.ReviewCount)
	}
	if plan.NewCount != 3 {
		t.Fatalf("NewCount = %d, want 3", plan.NewCount)
	}
	if plan.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", plan.RetryCount)
	}
}

func TestMakePlanUsesCustomOptions(t *testing.T) {
	plan := MakePlan(PlanOptions{QuestionCount: 5, ReviewRatio: 0.4}, 4, 6, store.ModeLearn)
	if plan.ReviewCount != 2 {
		t.Fatalf("ReviewCount = %d, want 2", plan.ReviewCount)
	}
	if plan.NewCount != 3 {
		t.Fatalf("NewCount = %d, want 3", plan.NewCount)
	}
}

func TestPlanOptionsNormalize(t *testing.T) {
	options := (PlanOptions{ReviewRatio: -1}).Normalize()
	if options.QuestionCount != DefaultQuestionCount {
		t.Fatalf("QuestionCount = %d, want %d", options.QuestionCount, DefaultQuestionCount)
	}
	if options.ReviewRatio != DefaultReviewRatio {
		t.Fatalf("ReviewRatio = %v, want %v", options.ReviewRatio, DefaultReviewRatio)
	}
}

func TestBuildSessionItemsInterleaves(t *testing.T) {
	reviewWords := []store.Word{{ID: 1}, {ID: 2}, {ID: 3}}
	newWords := []store.Word{{ID: 4}, {ID: 5}}
	items := BuildSessionItems(reviewWords, newWords)
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}
	if items[0].WordID != 1 || items[1].WordID != 2 || items[2].WordID != 4 {
		t.Fatalf("unexpected first batch ordering: %+v", items[:3])
	}
}
