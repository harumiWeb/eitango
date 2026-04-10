package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/srs"
)

func TestMigrateSeedWordsAndHomeSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	entries := testEntries()

	if err := st.SeedWords(ctx, entries, "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	snapshot, err := st.LoadHomeSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadHomeSnapshot() error = %v", err)
	}
	if snapshot.DueCount != 0 {
		t.Fatalf("DueCount = %d, want 0", snapshot.DueCount)
	}
	if snapshot.NewCount != len(entries) {
		t.Fatalf("NewCount = %d, want %d", snapshot.NewCount, len(entries))
	}
	if snapshot.StreakDays != 0 {
		t.Fatalf("StreakDays = %d, want 0", snapshot.StreakDays)
	}
	if snapshot.ActiveSession != nil {
		t.Fatalf("ActiveSession = %+v, want nil", snapshot.ActiveSession)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	if len(words) != len(entries) {
		t.Fatalf("len(ListNewWords()) = %d, want %d", len(words), len(entries))
	}
	if words[0].Lemma != "abandon" || words[1].Lemma != "apply" || words[2].Lemma != "benefit" {
		t.Fatalf("unexpected word ordering: %+v", []string{words[0].Lemma, words[1].Lemma, words[2].Lemma})
	}

	verbs, err := st.ListWordsByPOS(ctx, "verb", 10, []int64{words[0].ID})
	if err != nil {
		t.Fatalf("ListWordsByPOS() error = %v", err)
	}
	if len(verbs) != 1 || verbs[0].Lemma != "apply" {
		t.Fatalf("unexpected verb list: %+v", verbs)
	}

	if err := st.SeedWords(ctx, entries, "test-v1"); err != nil {
		t.Fatalf("SeedWords() second call error = %v", err)
	}
	if got := mustCountRows(t, st, "words"); got != len(entries) {
		t.Fatalf("word count after same-version reseed = %d, want %d", got, len(entries))
	}
}

func TestListDistractorCandidatesPrioritizeGroupAndLevel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	entries := []dict.Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "core-1", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "collect", Pos: "verb", MeaningJA: "収集する", Level: "core-1", FrequencyRank: 140, DistractorGroup: "basic-verb-action"},
		{Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "core-1", FrequencyRank: 160, DistractorGroup: "basic-verb-action"},
		{Lemma: "approve", Pos: "verb", MeaningJA: "承認する", Level: "core-3", FrequencyRank: 5000, DistractorGroup: "business-verb"},
		{Lemma: "assign", Pos: "verb", MeaningJA: "割り当てる", Level: "core-3", FrequencyRank: 5010, DistractorGroup: "business-verb"},
		{Lemma: "budget", Pos: "verb", MeaningJA: "予算計上する", Level: "core-3", FrequencyRank: 5020, DistractorGroup: "business-verb"},
		{Lemma: "delegate", Pos: "verb", MeaningJA: "委任する", Level: "core-3", FrequencyRank: 5030, DistractorGroup: "business-verb"},
		{Lemma: "expand", Pos: "verb", MeaningJA: "拡大する", Level: "core-3", FrequencyRank: 6500, DistractorGroup: "change-verb"},
	}

	if err := st.SeedWords(ctx, entries, "test-distractors-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 20, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	var correct Word
	for _, word := range words {
		if word.Lemma == "approve" {
			correct = word
			break
		}
	}
	if correct.ID == 0 {
		t.Fatal("approve word not found")
	}

	pool, err := st.ListDistractorCandidates(ctx, correct, 4, []int64{correct.ID})
	if err != nil {
		t.Fatalf("ListDistractorCandidates() error = %v", err)
	}
	if len(pool) != 4 {
		t.Fatalf("len(pool) = %d, want 4", len(pool))
	}
	if pool[0].Lemma != "assign" || pool[1].Lemma != "budget" || pool[2].Lemma != "delegate" {
		t.Fatalf("unexpected prioritized distractors: %+v", []string{pool[0].Lemma, pool[1].Lemma, pool[2].Lemma, pool[3].Lemma})
	}
}

func TestListWriteBasicCandidatesPrioritizeWriteUnseenAndExcludeDue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	base := stableUTCNoon()
	recordReviewInMode(t, st, words[2].ID, AnswerModeChoice, base.Add(1*time.Minute))
	recordReviewInMode(t, st, words[1].ID, AnswerModeChoice, base.Add(2*time.Minute))
	recordReviewInMode(t, st, words[1].ID, AnswerModeWrite, base.Add(3*time.Minute))
	recordReviewInMode(t, st, words[0].ID, AnswerModeChoice, base.AddDate(0, 0, -4))

	seen, err := st.ListWriteBasicCandidates(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListWriteBasicCandidates() error = %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("len(ListWriteBasicCandidates()) = %d, want 2", len(seen))
	}
	if seen[0].ID != words[2].ID || seen[1].ID != words[1].ID {
		t.Fatalf("unexpected choice-seen order: %+v", []int64{seen[0].ID, seen[1].ID})
	}

	seen, err = st.ListWriteBasicCandidates(ctx, 10, []int64{words[2].ID})
	if err != nil {
		t.Fatalf("ListWriteBasicCandidates(exclude) error = %v", err)
	}
	if len(seen) != 1 || seen[0].ID != words[1].ID {
		t.Fatalf("unexpected excluded result: %+v", seen)
	}
}

func TestListReviewedWordsRandomReturnsDistinctReviewedWordsOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	base := stableUTCNoon()
	recordReviewInMode(t, st, words[0].ID, AnswerModeChoice, base.Add(1*time.Minute))
	recordReviewInMode(t, st, words[0].ID, AnswerModeWrite, base.Add(2*time.Minute))
	recordReviewInMode(t, st, words[2].ID, AnswerModeChoice, base.Add(3*time.Minute))

	reviewed, err := st.ListReviewedWordsRandom(ctx, 10)
	if err != nil {
		t.Fatalf("ListReviewedWordsRandom() error = %v", err)
	}
	if len(reviewed) != 2 {
		t.Fatalf("len(ListReviewedWordsRandom()) = %d, want 2", len(reviewed))
	}

	got := map[int64]struct{}{}
	for _, word := range reviewed {
		got[word.ID] = struct{}{}
	}
	for _, want := range []int64{words[0].ID, words[2].ID} {
		if _, ok := got[want]; !ok {
			t.Fatalf("reviewed ids = %+v, want %d to be present", got, want)
		}
	}
	if _, ok := got[words[1].ID]; ok {
		t.Fatalf("reviewed ids = %+v, want unseen word %d to be excluded", got, words[1].ID)
	}
}

func TestCreateSessionPersistsAnswerMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeWrite, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if record.AnswerMode != AnswerModeWrite {
		t.Fatalf("record.AnswerMode = %q, want %q", record.AnswerMode, AnswerModeWrite)
	}

	active, _, err := st.LoadActiveRuntime(ctx)
	if err != nil {
		t.Fatalf("LoadActiveRuntime() error = %v", err)
	}
	if active == nil || active.AnswerMode != AnswerModeWrite {
		t.Fatalf("active session = %+v, want write answer mode", active)
	}
}

func TestSaveAnswerPersistsReviewAnswerMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeWrite, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         words[0].ID,
		Kind:           ItemKindNew,
		AnswerMode:     AnswerModeWrite,
		SelectedChoice: 0,
		CorrectChoice:  0,
		IsCorrect:      true,
		Rating:         srs.Easy,
		AnsweredAt:     time.Now().UTC(),
		ResponseMS:     900,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}

	var answerMode string
	if err := st.db.QueryRowContext(ctx, `SELECT answer_mode FROM reviews WHERE session_id = ? LIMIT 1`, record.ID).Scan(&answerMode); err != nil {
		t.Fatalf("load review answer_mode: %v", err)
	}
	if answerMode != AnswerModeWrite {
		t.Fatalf("review answer_mode = %q, want %q", answerMode, AnswerModeWrite)
	}
}

func TestSaveAnswerReviewInfiniteDoesNotUpdateProgressOrRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	target := words[0]
	base := stableUTCNoon()
	recordReviewInMode(t, st, target.ID, AnswerModeChoice, base)
	before := mustLoadProgress(t, st, target.ID)

	record, _, err := st.CreateSession(ctx, ModeReviewInfinite, AnswerModeChoice, []SessionItemPlan{
		{WordID: target.ID, Kind: ItemKindReview},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		SessionMode:    ModeReviewInfinite,
		ItemOrdinal:    1,
		WordID:         target.ID,
		Kind:           ItemKindReview,
		AnswerMode:     AnswerModeChoice,
		SelectedChoice: 1,
		CorrectChoice:  0,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     base.Add(30 * time.Minute),
		ResponseMS:     900,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}

	after := mustLoadProgress(t, st, target.ID)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("progress changed in review infinite mode:\nbefore=%+v\nafter=%+v", before, after)
	}

	var rating any
	if err := st.db.QueryRowContext(ctx, `SELECT rating FROM reviews WHERE session_id = ? LIMIT 1`, record.ID).Scan(&rating); err != nil {
		t.Fatalf("load review rating: %v", err)
	}
	if rating != nil {
		t.Fatalf("rating = %v, want nil for review infinite answer", rating)
	}
}

func TestSaveAnswerCreatesRetryCompletesSessionAndUpdatesStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	if len(words) == 0 {
		t.Fatal("ListNewWords() returned no words")
	}
	target := words[0]

	record, items, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
		{WordID: target.ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if record.Status != SessionStatusActive {
		t.Fatalf("session status = %q, want %q", record.Status, SessionStatusActive)
	}
	if len(items) != 1 || items[0].Status != ItemStatusPending {
		t.Fatalf("unexpected created session items: %+v", items)
	}

	activeRecord, activeItems, err := st.LoadActiveRuntime(ctx)
	if err != nil {
		t.Fatalf("LoadActiveRuntime() error = %v", err)
	}
	if activeRecord == nil || activeRecord.ID != record.ID {
		t.Fatalf("active record = %+v, want session %s", activeRecord, record.ID)
	}
	if len(activeItems) != 1 || activeItems[0].WordID != target.ID {
		t.Fatalf("unexpected active items: %+v", activeItems)
	}

	firstAnsweredAt := stableUTCNoon()
	record, items, err = st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         target.ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  2,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     firstAnsweredAt,
		ResponseMS:     1200,
	})
	if err != nil {
		t.Fatalf("SaveAnswer() first call error = %v", err)
	}
	if record.Status != SessionStatusActive {
		t.Fatalf("session status after wrong answer = %q, want %q", record.Status, SessionStatusActive)
	}
	if record.AnsweredQuestions != 1 {
		t.Fatalf("AnsweredQuestions after wrong answer = %d, want 1", record.AnsweredQuestions)
	}
	if record.TotalQuestions != 2 {
		t.Fatalf("TotalQuestions after wrong answer = %d, want 2", record.TotalQuestions)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) after wrong answer = %d, want 2", len(items))
	}
	if items[0].Status != ItemStatusAnswered {
		t.Fatalf("first item status = %q, want %q", items[0].Status, ItemStatusAnswered)
	}
	if items[1].Kind != ItemKindRetry || items[1].Status != ItemStatusPending {
		t.Fatalf("retry item = %+v, want pending retry", items[1])
	}
	if items[1].SourceOrdinal == nil || *items[1].SourceOrdinal != 1 {
		t.Fatalf("retry source ordinal = %+v, want 1", items[1].SourceOrdinal)
	}

	progress := mustLoadProgress(t, st, target.ID)
	if progress.State != "learning" {
		t.Fatalf("progress state after Again = %q, want learning", progress.State)
	}
	if progress.TotalWrong != 1 || progress.Lapses != 1 {
		t.Fatalf("unexpected wrong-answer progress counters: %+v", progress)
	}
	if progress.DueAt == nil {
		t.Fatal("DueAt after Again is nil")
	}
	if diff := progress.DueAt.Sub(firstAnsweredAt); diff < 9*time.Minute || diff > 11*time.Minute {
		t.Fatalf("DueAt after Again diff = %s, want about 10m", diff)
	}

	secondAnsweredAt := firstAnsweredAt.Add(20 * time.Minute)
	record, items, err = st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    2,
		WordID:         target.ID,
		Kind:           ItemKindRetry,
		SelectedChoice: 2,
		CorrectChoice:  2,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     secondAnsweredAt,
		ResponseMS:     900,
	})
	if err != nil {
		t.Fatalf("SaveAnswer() second call error = %v", err)
	}
	if record.Status != SessionStatusCompleted {
		t.Fatalf("session status after retry answer = %q, want %q", record.Status, SessionStatusCompleted)
	}
	if record.AnsweredQuestions != 2 || record.TotalQuestions != 2 {
		t.Fatalf("unexpected completed session counters: %+v", record)
	}
	if record.FinishedAt == nil {
		t.Fatal("FinishedAt after completion is nil")
	}
	if len(items) != 2 || items[1].Status != ItemStatusAnswered {
		t.Fatalf("unexpected session items after completion: %+v", items)
	}

	progress = mustLoadProgress(t, st, target.ID)
	if progress.State != "review" {
		t.Fatalf("progress state after Good = %q, want review", progress.State)
	}
	if progress.IntervalDays != 3 {
		t.Fatalf("IntervalDays after Good = %v, want 3", progress.IntervalDays)
	}
	if progress.TotalCorrect != 1 || progress.TotalWrong != 1 || progress.StreakCorrect != 1 {
		t.Fatalf("unexpected completed progress counters: %+v", progress)
	}
	if progress.DueAt == nil {
		t.Fatal("DueAt after Good is nil")
	}
	if diff := progress.DueAt.Sub(secondAnsweredAt); diff < 71*time.Hour || diff > 73*time.Hour {
		t.Fatalf("DueAt after Good diff = %s, want about 72h", diff)
	}

	summary, err := st.LoadSessionSummary(ctx, record.ID)
	if err != nil {
		t.Fatalf("LoadSessionSummary() error = %v", err)
	}
	if summary.TotalQuestions != 2 || summary.CorrectAnswers != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.Accuracy != 50 {
		t.Fatalf("summary accuracy = %v, want 50", summary.Accuracy)
	}
	if summary.NewCount != 1 || summary.ReviewCount != 0 || summary.RetryCount != 1 {
		t.Fatalf("unexpected summary mix counts: %+v", summary)
	}
	if len(summary.HardWords) != 1 || summary.HardWords[0].ID != target.ID {
		t.Fatalf("unexpected hard words: %+v", summary.HardWords)
	}

	statsSnapshot, err := st.LoadStatsSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadStatsSnapshot() error = %v", err)
	}
	if statsSnapshot.Today.Reviews != 2 || statsSnapshot.Today.Correct != 1 {
		t.Fatalf("unexpected today stats: %+v", statsSnapshot.Today)
	}
	if diff := statsSnapshot.Today.WaitMinutes - 0.035; diff < -0.000001 || diff > 0.000001 {
		t.Fatalf("Today.WaitMinutes = %v, want about 0.035", statsSnapshot.Today.WaitMinutes)
	}
	if statsSnapshot.Total.Reviews != 2 || statsSnapshot.Total.Correct != 1 {
		t.Fatalf("unexpected total stats: %+v", statsSnapshot.Total)
	}
	if diff := statsSnapshot.Total.WaitMinutes - 0.035; diff < -0.000001 || diff > 0.000001 {
		t.Fatalf("Total.WaitMinutes = %v, want about 0.035", statsSnapshot.Total.WaitMinutes)
	}
	if statsSnapshot.NewCount != 2 {
		t.Fatalf("NewCount after completing one word = %d, want 2", statsSnapshot.NewCount)
	}
	if statsSnapshot.DueCount != 0 {
		t.Fatalf("DueCount after Good review = %d, want 0", statsSnapshot.DueCount)
	}
	if statsSnapshot.StreakDays != 1 {
		t.Fatalf("StreakDays after same-day reviews = %d, want 1", statsSnapshot.StreakDays)
	}

	activeRecord, activeItems, err = st.LoadActiveRuntime(ctx)
	if err != nil {
		t.Fatalf("LoadActiveRuntime() after completion error = %v", err)
	}
	if activeRecord != nil || activeItems != nil {
		t.Fatalf("active runtime after completion = %+v / %+v, want nil", activeRecord, activeItems)
	}
}

func TestLoadStatsSnapshotCountsConsecutiveReviewDays(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	if len(words) < 2 {
		t.Fatalf("len(ListNewWords()) = %d, want at least 2", len(words))
	}

	yesterday := stableUTCNoon().AddDate(0, 0, -1)
	today := yesterday.AddDate(0, 0, 1)

	for _, review := range []struct {
		wordID     int64
		answeredAt time.Time
	}{
		{wordID: words[0].ID, answeredAt: yesterday},
		{wordID: words[1].ID, answeredAt: today},
	} {
		record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
			{WordID: review.wordID, Kind: ItemKindNew},
		})
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
			SessionID:      record.ID,
			ItemOrdinal:    1,
			WordID:         review.wordID,
			Kind:           ItemKindNew,
			SelectedChoice: 1,
			CorrectChoice:  1,
			IsCorrect:      true,
			Rating:         srs.Good,
			AnsweredAt:     review.answeredAt,
			ResponseMS:     800,
		}); err != nil {
			t.Fatalf("SaveAnswer() error = %v", err)
		}
	}

	homeSnapshot, err := st.LoadHomeSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadHomeSnapshot() error = %v", err)
	}
	if homeSnapshot.StreakDays != 2 {
		t.Fatalf("HomeSnapshot streak = %d, want 2", homeSnapshot.StreakDays)
	}

	statsSnapshot, err := st.LoadStatsSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadStatsSnapshot() error = %v", err)
	}
	if statsSnapshot.StreakDays != 2 {
		t.Fatalf("StatsSnapshot streak = %d, want 2", statsSnapshot.StreakDays)
	}
}

func TestSeedWordsVersionChangeResetsUserData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         words[0].ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     time.Now().UTC(),
		ResponseMS:     800,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}

	if got := mustCountRows(t, st, "sessions"); got != 1 {
		t.Fatalf("sessions before reseed = %d, want 1", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 1 {
		t.Fatalf("reviews before reseed = %d, want 1", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 1 {
		t.Fatalf("progress before reseed = %d, want 1", got)
	}

	nextEntries := append(testEntries(), dict.Entry{
		Lemma:           "coach",
		Pos:             "verb",
		MeaningJA:       "指導する",
		Level:           "core-1",
		FrequencyRank:   400,
		DistractorGroup: "basic-verb-action",
		ExampleEN:       "They coach the team every weekend.",
		ExampleJA:       "彼らは毎週末チームを指導する。",
	})
	if err := st.SeedWords(ctx, nextEntries, "test-v2"); err != nil {
		t.Fatalf("SeedWords() version bump error = %v", err)
	}

	if got := mustCountRows(t, st, "words"); got != len(nextEntries) {
		t.Fatalf("words after version bump = %d, want %d", got, len(nextEntries))
	}
	if got := mustCountRows(t, st, "sessions"); got != 0 {
		t.Fatalf("sessions after version bump = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "session_items"); got != 0 {
		t.Fatalf("session_items after version bump = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 0 {
		t.Fatalf("reviews after version bump = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 0 {
		t.Fatalf("progress after version bump = %d, want 0", got)
	}

	version, err := st.metaValue(ctx, "dict_version")
	if err != nil {
		t.Fatalf("metaValue(dict_version) error = %v", err)
	}
	if version != "test-v2" {
		t.Fatalf("dict_version = %q, want test-v2", version)
	}

	snapshot, err := st.LoadHomeSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadHomeSnapshot() error = %v", err)
	}
	if snapshot.NewCount != len(nextEntries) {
		t.Fatalf("NewCount after version bump = %d, want %d", snapshot.NewCount, len(nextEntries))
	}
	if snapshot.ActiveSession != nil {
		t.Fatalf("ActiveSession after version bump = %+v, want nil", snapshot.ActiveSession)
	}
}

func TestResetProgressClearsLearningHistoryOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         words[0].ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     time.Now().UTC(),
		ResponseMS:     700,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}

	result, err := st.Reset(ctx, ResetOptions{Progress: true}, nil, "")
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if result.ClearedSessionItems != 1 || result.ClearedReviews != 1 || result.ClearedProgress != 1 || result.ClearedSessions != 1 {
		t.Fatalf("unexpected reset counts: %+v", result)
	}
	if result.ClearedWords != 0 || result.SeededWords != 0 {
		t.Fatalf("unexpected word reset counts: %+v", result)
	}

	if got := mustCountRows(t, st, "words"); got != len(testEntries()) {
		t.Fatalf("words after progress reset = %d, want %d", got, len(testEntries()))
	}
	if got := mustCountRows(t, st, "sessions"); got != 0 {
		t.Fatalf("sessions after progress reset = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "session_items"); got != 0 {
		t.Fatalf("session_items after progress reset = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 0 {
		t.Fatalf("reviews after progress reset = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 0 {
		t.Fatalf("progress after progress reset = %d, want 0", got)
	}

	version, err := st.metaValue(ctx, "dict_version")
	if err != nil {
		t.Fatalf("metaValue(dict_version) error = %v", err)
	}
	if version != "test-v1" {
		t.Fatalf("dict_version after progress reset = %q, want test-v1", version)
	}
}

func TestResetReseedForcesReplaceSameVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         words[0].ID,
		Kind:           ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     time.Now().UTC(),
		ResponseMS:     650,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}

	replacementEntries := []dict.Entry{
		{
			Lemma:           "coach",
			Pos:             "verb",
			MeaningJA:       "指導する",
			Level:           "core-1",
			FrequencyRank:   400,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "They coach the team every weekend.",
			ExampleJA:       "彼らは毎週末チームを指導する。",
		},
		{
			Lemma:           "demand",
			Pos:             "verb",
			MeaningJA:       "要求する",
			Level:           "core-1",
			FrequencyRank:   500,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "Customers demand faster delivery.",
			ExampleJA:       "顧客はより速い配送を求める。",
		},
	}

	result, err := st.Reset(ctx, ResetOptions{Reseed: true}, replacementEntries, "test-v1")
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if result.ClearedWords != len(testEntries()) {
		t.Fatalf("ClearedWords = %d, want %d", result.ClearedWords, len(testEntries()))
	}
	if result.SeededWords != len(replacementEntries) {
		t.Fatalf("SeededWords = %d, want %d", result.SeededWords, len(replacementEntries))
	}

	if got := mustCountRows(t, st, "words"); got != len(replacementEntries) {
		t.Fatalf("words after reseed = %d, want %d", got, len(replacementEntries))
	}
	if got := mustCountRows(t, st, "sessions"); got != 0 {
		t.Fatalf("sessions after reseed = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "session_items"); got != 0 {
		t.Fatalf("session_items after reseed = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 0 {
		t.Fatalf("reviews after reseed = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 0 {
		t.Fatalf("progress after reseed = %d, want 0", got)
	}

	newWords, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() after reseed error = %v", err)
	}
	if len(newWords) != len(replacementEntries) || newWords[0].Lemma != "coach" || newWords[1].Lemma != "demand" {
		t.Fatalf("unexpected words after reseed: %+v", newWords)
	}

	version, err := st.metaValue(ctx, "dict_version")
	if err != nil {
		t.Fatalf("metaValue(dict_version) error = %v", err)
	}
	if version != "test-v1" {
		t.Fatalf("dict_version after reseed = %q, want test-v1", version)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "eitango-test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return st
}

func testEntries() []dict.Entry {
	return []dict.Entry{
		{
			Lemma:           "abandon",
			Pos:             "verb",
			MeaningJA:       "捨てる",
			Level:           "core-1",
			FrequencyRank:   100,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "They had to abandon the plan.",
			ExampleJA:       "彼らはその計画を捨てなければならなかった。",
		},
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "応募する",
			Level:           "core-1",
			FrequencyRank:   200,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "She will apply for the job.",
			ExampleJA:       "彼女はその仕事に応募するつもりだ。",
		},
		{
			Lemma:           "benefit",
			Pos:             "noun",
			MeaningJA:       "利益",
			Level:           "core-1",
			FrequencyRank:   300,
			DistractorGroup: "basic-noun-business",
			ExampleEN:       "The company offers a strong benefit package.",
			ExampleJA:       "その会社は充実した福利厚生を提供している。",
		},
	}
}

func mustCountRows(t *testing.T, st *Store, table string) int {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM " + table
	if err := st.db.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

func mustLoadProgress(t *testing.T, st *Store, wordID int64) Progress {
	t.Helper()

	tx, err := st.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	progress, err := loadProgressTx(context.Background(), tx, wordID)
	if err != nil {
		t.Fatalf("loadProgressTx() error = %v", err)
	}
	return progress
}

func stableUTCNoon() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
}

func recordReviewInMode(t *testing.T, st *Store, wordID int64, answerMode string, answeredAt time.Time) {
	t.Helper()

	ctx := context.Background()
	record, _, err := st.CreateSession(ctx, ModeLearn, answerMode, []SessionItemPlan{
		{WordID: wordID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordID,
		Kind:           ItemKindNew,
		AnswerMode:     answerMode,
		SelectedChoice: 0,
		CorrectChoice:  0,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     answeredAt,
		ResponseMS:     800,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}
}
