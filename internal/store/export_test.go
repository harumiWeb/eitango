package store

import (
	"context"
	"testing"
	"time"

	"github.com/yourname/eitango/internal/srs"
)

func TestListExportWordSnapshotsIncludesProgressAndReviewStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	fixture := seedExportFixture(t, st)

	snapshots, err := st.ListExportWordSnapshots(ctx)
	if err != nil {
		t.Fatalf("ListExportWordSnapshots() error = %v", err)
	}
	if len(snapshots) != len(testEntries()) {
		t.Fatalf("len(ListExportWordSnapshots()) = %d, want %d", len(snapshots), len(testEntries()))
	}

	abandon := mustFindExportSnapshot(t, snapshots, "abandon")
	if abandon.Progress.State != "review" {
		t.Fatalf("abandon progress state = %q, want review", abandon.Progress.State)
	}
	if abandon.ReviewStats.TotalReviews != 2 || abandon.ReviewStats.CorrectReviews != 1 || abandon.ReviewStats.WrongReviews != 1 {
		t.Fatalf("unexpected abandon review stats: %+v", abandon.ReviewStats)
	}
	if abandon.Progress.TotalCorrect != 1 || abandon.Progress.TotalWrong != 1 || abandon.Progress.Lapses != 1 {
		t.Fatalf("unexpected abandon progress counters: %+v", abandon.Progress)
	}
	if abandon.ReviewStats.LastWrongAt == nil || !abandon.ReviewStats.LastWrongAt.Equal(fixture.firstWrongAt) {
		t.Fatalf("abandon last wrong = %+v, want %s", abandon.ReviewStats.LastWrongAt, fixture.firstWrongAt)
	}
	if abandon.ReviewStats.LastCorrectAt == nil || !abandon.ReviewStats.LastCorrectAt.Equal(fixture.retryCorrectAt) {
		t.Fatalf("abandon last correct = %+v, want %s", abandon.ReviewStats.LastCorrectAt, fixture.retryCorrectAt)
	}
	if abandon.ReviewStats.LastAnsweredAt == nil || !abandon.ReviewStats.LastAnsweredAt.Equal(fixture.retryCorrectAt) {
		t.Fatalf("abandon last answered = %+v, want %s", abandon.ReviewStats.LastAnsweredAt, fixture.retryCorrectAt)
	}
	if abandon.Progress.DueAt == nil {
		t.Fatal("abandon progress due_at is nil")
	}

	apply := mustFindExportSnapshot(t, snapshots, "apply")
	if apply.Progress.State != "review" {
		t.Fatalf("apply progress state = %q, want review", apply.Progress.State)
	}
	if apply.ReviewStats.TotalReviews != 1 || apply.ReviewStats.CorrectReviews != 1 || apply.ReviewStats.WrongReviews != 0 {
		t.Fatalf("unexpected apply review stats: %+v", apply.ReviewStats)
	}
	if apply.ReviewStats.LastCorrectAt == nil || !apply.ReviewStats.LastCorrectAt.Equal(fixture.secondCorrectAt) {
		t.Fatalf("apply last correct = %+v, want %s", apply.ReviewStats.LastCorrectAt, fixture.secondCorrectAt)
	}

	benefit := mustFindExportSnapshot(t, snapshots, "benefit")
	if benefit.Progress.State != "new" {
		t.Fatalf("benefit progress state = %q, want new", benefit.Progress.State)
	}
	if benefit.ReviewStats.TotalReviews != 0 || benefit.ReviewStats.WrongReviews != 0 || benefit.ReviewStats.LastAnsweredAt != nil {
		t.Fatalf("unexpected benefit review stats: %+v", benefit.ReviewStats)
	}
	if benefit.Progress.LastSeenAt != nil || benefit.Progress.DueAt != nil {
		t.Fatalf("unexpected benefit progress timestamps: %+v", benefit.Progress)
	}
}

func TestListWrongWordSnapshotsFiltersAndSortsByWrongReviews(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	fixture := seedExportFixture(t, st)
	benefitID := fixture.wordsByLemma["benefit"].ID

	firstBenefitWrongAt := fixture.secondCorrectAt.Add(20 * time.Minute)
	record, _, err := st.CreateSession(ctx, ModeReview, []SessionItemPlan{
		{WordID: benefitID, Kind: ItemKindReview},
	})
	if err != nil {
		t.Fatalf("CreateSession() first benefit review error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         benefitID,
		Kind:           ItemKindReview,
		SelectedChoice: 1,
		CorrectChoice:  2,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     firstBenefitWrongAt,
		ResponseMS:     1100,
	}); err != nil {
		t.Fatalf("SaveAnswer() first benefit wrong error = %v", err)
	}
	if err := st.AbandonActiveSession(ctx); err != nil {
		t.Fatalf("AbandonActiveSession() after first benefit wrong error = %v", err)
	}

	secondBenefitWrongAt := firstBenefitWrongAt.Add(30 * time.Minute)
	record, _, err = st.CreateSession(ctx, ModeReview, []SessionItemPlan{
		{WordID: benefitID, Kind: ItemKindReview},
	})
	if err != nil {
		t.Fatalf("CreateSession() second benefit review error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         benefitID,
		Kind:           ItemKindReview,
		SelectedChoice: 1,
		CorrectChoice:  2,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     secondBenefitWrongAt,
		ResponseMS:     1050,
	}); err != nil {
		t.Fatalf("SaveAnswer() second benefit wrong error = %v", err)
	}

	snapshots, err := st.ListWrongWordSnapshots(ctx)
	if err != nil {
		t.Fatalf("ListWrongWordSnapshots() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("len(ListWrongWordSnapshots()) = %d, want 2", len(snapshots))
	}
	if snapshots[0].Word.Lemma != "benefit" || snapshots[1].Word.Lemma != "abandon" {
		t.Fatalf("unexpected wrong-word order: %+v", []string{snapshots[0].Word.Lemma, snapshots[1].Word.Lemma})
	}
	if snapshots[0].ReviewStats.WrongReviews != 2 {
		t.Fatalf("benefit wrong reviews = %d, want 2", snapshots[0].ReviewStats.WrongReviews)
	}
	if snapshots[0].ReviewStats.LastWrongAt == nil || !snapshots[0].ReviewStats.LastWrongAt.Equal(secondBenefitWrongAt) {
		t.Fatalf("benefit last wrong = %+v, want %s", snapshots[0].ReviewStats.LastWrongAt, secondBenefitWrongAt)
	}
}

type exportFixture struct {
	wordsByLemma    map[string]Word
	firstWrongAt    time.Time
	retryCorrectAt  time.Time
	secondCorrectAt time.Time
}

func seedExportFixture(t *testing.T, st *Store) exportFixture {
	t.Helper()

	ctx := context.Background()
	if err := st.SeedWords(ctx, testEntries(), "test-export-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	wordsByLemma := make(map[string]Word, len(words))
	for _, word := range words {
		wordsByLemma[word.Lemma] = word
	}

	firstWrongAt := time.Date(2026, time.March, 25, 9, 0, 0, 0, time.UTC)
	retryCorrectAt := firstWrongAt.Add(20 * time.Minute)

	record, _, err := st.CreateSession(ctx, ModeLearn, []SessionItemPlan{
		{WordID: wordsByLemma["abandon"].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() abandon session error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordsByLemma["abandon"].ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  2,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     firstWrongAt,
		ResponseMS:     1200,
	}); err != nil {
		t.Fatalf("SaveAnswer() abandon wrong error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    2,
		WordID:         wordsByLemma["abandon"].ID,
		Kind:           ItemKindRetry,
		SelectedChoice: 2,
		CorrectChoice:  2,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     retryCorrectAt,
		ResponseMS:     900,
	}); err != nil {
		t.Fatalf("SaveAnswer() abandon retry error = %v", err)
	}

	secondCorrectAt := retryCorrectAt.Add(40 * time.Minute)
	record, _, err = st.CreateSession(ctx, ModeLearn, []SessionItemPlan{
		{WordID: wordsByLemma["apply"].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() apply session error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordsByLemma["apply"].ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     secondCorrectAt,
		ResponseMS:     800,
	}); err != nil {
		t.Fatalf("SaveAnswer() apply correct error = %v", err)
	}

	return exportFixture{
		wordsByLemma:    wordsByLemma,
		firstWrongAt:    firstWrongAt,
		retryCorrectAt:  retryCorrectAt,
		secondCorrectAt: secondCorrectAt,
	}
}

func mustFindExportSnapshot(t *testing.T, snapshots []ExportWordSnapshot, lemma string) ExportWordSnapshot {
	t.Helper()

	for _, snapshot := range snapshots {
		if snapshot.Word.Lemma == lemma {
			return snapshot
		}
	}
	t.Fatalf("snapshot for lemma %q not found", lemma)
	return ExportWordSnapshot{}
}
