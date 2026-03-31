package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
)

func TestSessionCmdReviewResumesActiveSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	record, _, err := st.CreateSession(ctx, store.ModeLearn, []store.SessionItemPlan{
		{WordID: words[0].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode: store.ModeReview,
		Plan: session.DefaultPlanOptions(),
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.ID != record.ID {
		t.Fatalf("resumed session id = %q, want %q", loaded.Runtime.Session.ID, record.ID)
	}
	if loaded.Runtime.Session.Mode != store.ModeLearn {
		t.Fatalf("resumed session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeLearn)
	}
}

func TestSessionCmdReplaceActiveStartsFreshReviewSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	for _, word := range words[1:6] {
		markWordDue(t, st, word.ID, time.Now().UTC().AddDate(0, 0, -4))
	}

	activeRecord, _, err := st.CreateSession(ctx, store.ModeLearn, []store.SessionItemPlan{
		{WordID: words[0].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:          store.ModeReview,
		ReplaceActive: true,
		Plan:          session.PlanOptions{QuestionCount: 5, ReviewRatio: 0.7},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.ID == activeRecord.ID {
		t.Fatalf("new review session reused active session %q", activeRecord.ID)
	}
	if loaded.Runtime.Session.Mode != store.ModeReview {
		t.Fatalf("new session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeReview)
	}

	abandoned, err := st.LoadSession(ctx, activeRecord.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if abandoned.Status != store.SessionStatusAbandoned {
		t.Fatalf("abandoned session status = %q, want %q", abandoned.Status, store.SessionStatusAbandoned)
	}
}

func TestSessionCmdReviewStartsDueOnlySession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	for _, word := range words[:5] {
		markWordDue(t, st, word.ID, time.Now().UTC().AddDate(0, 0, -4))
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode: store.ModeReview,
		Plan: session.PlanOptions{QuestionCount: 5, ReviewRatio: 0.2},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.Mode != store.ModeReview {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeReview)
	}
	if loaded.Runtime.Total() != 5 {
		t.Fatalf("Total() = %d, want 5", loaded.Runtime.Total())
	}
	if len(loaded.Runtime.Items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(loaded.Runtime.Items))
	}
	for _, item := range loaded.Runtime.Items {
		if item.Kind != store.ItemKindReview {
			t.Fatalf("item kind = %q, want %q", item.Kind, store.ItemKindReview)
		}
	}
}

func TestSessionCmdLearnUsesPlanOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	for _, word := range words[:2] {
		markWordDue(t, st, word.ID, time.Now().UTC().AddDate(0, 0, -4))
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode: store.ModeLearn,
		Plan: session.PlanOptions{QuestionCount: 5, ReviewRatio: 0.4},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.Mode != store.ModeLearn {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeLearn)
	}
	if loaded.Runtime.Total() != 5 {
		t.Fatalf("Total() = %d, want 5", loaded.Runtime.Total())
	}

	var reviewCount, newCount int
	for _, item := range loaded.Runtime.Items {
		switch item.Kind {
		case store.ItemKindReview:
			reviewCount++
		case store.ItemKindNew:
			newCount++
		}
	}
	if reviewCount != 2 {
		t.Fatalf("review item count = %d, want 2", reviewCount)
	}
	if newCount != 3 {
		t.Fatalf("new item count = %d, want 3", newCount)
	}
}

func TestUpdateCheckCmdUsesCheckNowAndReturnsResultEvenWhenServiceErrors(t *testing.T) {
	t.Parallel()

	service := &stubUpdateService{
		checkNowResult: updatecheck.Result{
			Latest:          updatecheck.ReleaseInfo{TagName: "v1.2.0"},
			UpdateAvailable: true,
			ShouldNotify:    true,
		},
		checkNowErr: errors.New("timeout"),
	}

	msg := updateCheckCmd(service, "v1.1.0")()
	checked, ok := msg.(updateCheckedMsg)
	if !ok {
		t.Fatalf("updateCheckCmd() returned %T, want updateCheckedMsg", msg)
	}
	if service.checkCalls != 0 {
		t.Fatalf("checkCalls = %d, want 0", service.checkCalls)
	}
	if service.checkNowCalls != 1 {
		t.Fatalf("checkNowCalls = %d, want 1", service.checkNowCalls)
	}
	if !checked.Result.ShouldNotify {
		t.Fatal("ShouldNotify = false, want true")
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "eitango-app-test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	return st
}

type stubUpdateService struct {
	checkResult    updatecheck.Result
	checkErr       error
	checkCalls     int
	checkNowResult updatecheck.Result
	checkNowErr    error
	checkNowCalls  int
}

func (s *stubUpdateService) Check(context.Context, string) (updatecheck.Result, error) {
	s.checkCalls++
	return s.checkResult, s.checkErr
}

func (s *stubUpdateService) CheckNow(context.Context, string) (updatecheck.Result, error) {
	s.checkNowCalls++
	return s.checkNowResult, s.checkNowErr
}

func markWordDue(t *testing.T, st *store.Store, wordID int64, answeredAt time.Time) {
	t.Helper()

	ctx := context.Background()
	record, _, err := st.CreateSession(ctx, store.ModeLearn, []store.SessionItemPlan{
		{WordID: wordID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, store.ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordID,
		Kind:           store.ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     answeredAt,
		ResponseMS:     800,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}
}

func mustSessionLoadedMsg(t *testing.T, msg any) sessionLoadedMsg {
	t.Helper()

	switch typed := msg.(type) {
	case sessionLoadedMsg:
		return typed
	case errMsg:
		t.Fatalf("sessionCmd() error = %v", typed.err)
	default:
		t.Fatalf("unexpected msg type %T", msg)
	}
	return sessionLoadedMsg{}
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
		},
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "応募する",
			Level:           "core-1",
			FrequencyRank:   200,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "arrange",
			Pos:             "verb",
			MeaningJA:       "手配する",
			Level:           "core-1",
			FrequencyRank:   300,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "avoid",
			Pos:             "verb",
			MeaningJA:       "避ける",
			Level:           "core-1",
			FrequencyRank:   400,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "coach",
			Pos:             "verb",
			MeaningJA:       "指導する",
			Level:           "core-1",
			FrequencyRank:   500,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "deliver",
			Pos:             "verb",
			MeaningJA:       "届ける",
			Level:           "core-1",
			FrequencyRank:   600,
			DistractorGroup: "basic-verb-action",
		},
	}
}
