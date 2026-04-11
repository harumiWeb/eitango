package app

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
	_ "modernc.org/sqlite"
)

func TestSessionCmdReviewResumesActiveSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	record, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
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

	activeRecord, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
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

func TestSessionCmdReviewWithoutDueReturnsFallbackPromptWhenReviewedWordsExist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	wordID := mustWordIDByIndex(t, st, 0)
	recordReviewInMode(t, st, wordID, store.AnswerModeChoice, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode: store.ModeReview,
		Plan: session.PlanOptions{QuestionCount: 3, ReviewRatio: 0.2},
	}, nil)()

	prompt := mustReviewFallbackPromptMsg(t, msg)
	if prompt.Request.Mode != store.ModeReview {
		t.Fatalf("prompt mode = %q, want %q", prompt.Request.Mode, store.ModeReview)
	}
	if !prompt.Request.AllowReviewFallback {
		t.Fatal("AllowReviewFallback = false, want true")
	}

	active, items, err := st.LoadActiveRuntime(ctx)
	if err != nil {
		t.Fatalf("LoadActiveRuntime() error = %v", err)
	}
	if active != nil || len(items) != 0 {
		t.Fatalf("active runtime = %+v / %+v, want none before confirmation", active, items)
	}
}

func TestSessionCmdReviewFallbackStartsReviewedOnlySession(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)
	first := mustWordIDByIndex(t, st, 0)
	second := mustWordIDByIndex(t, st, 1)
	recordReviewInMode(t, st, first, store.AnswerModeChoice, time.Now().UTC())
	recordReviewInMode(t, st, second, store.AnswerModeWrite, time.Now().UTC().Add(1*time.Minute))

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeReview,
		AnswerMode:          store.AnswerModeWrite,
		AllowReviewFallback: true,
		Plan:                session.PlanOptions{QuestionCount: 5, ReviewRatio: 0.2},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.Mode != store.ModeReviewInfinite {
		t.Fatalf("session mode = %q, want %q", loaded.Runtime.Session.Mode, store.ModeReviewInfinite)
	}
	if loaded.Runtime.Session.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("session answer mode = %q, want %q", loaded.Runtime.Session.AnswerMode, store.AnswerModeWrite)
	}
	if got := loaded.Runtime.Total(); got != 2 {
		t.Fatalf("Total() = %d, want 2 reviewed words", got)
	}

	gotIDs := map[int64]struct{}{}
	for _, item := range loaded.Runtime.Items {
		if item.Kind != store.ItemKindReview {
			t.Fatalf("item kind = %q, want %q", item.Kind, store.ItemKindReview)
		}
		gotIDs[item.WordID] = struct{}{}
	}
	for _, want := range []int64{first, second} {
		if _, ok := gotIDs[want]; !ok {
			t.Fatalf("runtime word ids = %+v, want %d", gotIDs, want)
		}
	}
}

func TestSessionCmdReplaceActiveReviewFallbackPromptKeepsExistingActiveSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	activeRecord, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: mustWordIDByIndex(t, st, 0), Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	recordReviewInMode(t, st, mustWordIDByIndex(t, st, 1), store.AnswerModeChoice, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:          store.ModeReview,
		ReplaceActive: true,
		Plan:          session.PlanOptions{QuestionCount: 3, ReviewRatio: 0.2},
	}, nil)()
	_ = mustReviewFallbackPromptMsg(t, msg)

	record, err := st.LoadSession(ctx, activeRecord.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if record.Status != store.SessionStatusActive {
		t.Fatalf("record status = %q, want %q", record.Status, store.SessionStatusActive)
	}
}

func TestSessionCmdDoesNotResumeAbandonedInfiniteReviewSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	wordID := mustWordIDByIndex(t, st, 0)
	activeRecord, _, err := st.CreateSession(ctx, store.ModeReviewInfinite, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: wordID, Kind: store.ItemKindReview},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode: store.ModeLearn,
		Plan: session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.ID == activeRecord.ID {
		t.Fatalf("session id = %q, want stale infinite review to be discarded", loaded.Runtime.Session.ID)
	}

	record, err := st.LoadSession(ctx, activeRecord.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if record.Status != store.SessionStatusAbandoned {
		t.Fatalf("record status = %q, want %q", record.Status, store.SessionStatusAbandoned)
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

func TestSessionCmdWriteSessionPersistsAnswerMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyHard,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Runtime.Session.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("session answer mode = %q, want %q", loaded.Runtime.Session.AnswerMode, store.AnswerModeWrite)
	}
	if loaded.Question.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("question answer mode = %q, want %q", loaded.Question.AnswerMode, store.AnswerModeWrite)
	}

	record, err := st.LoadSession(ctx, loaded.Runtime.Session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if record.AnswerMode != store.AnswerModeWrite {
		t.Fatalf("stored answer mode = %q, want %q", record.AnswerMode, store.AnswerModeWrite)
	}
}

func TestSessionCmdWriteBasicPrefersChoiceSeenWriteUnseenWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	recordReviewInMode(t, st, words[1].ID, store.AnswerModeChoice, time.Now().UTC())
	recordReviewInMode(t, st, words[0].ID, store.AnswerModeChoice, time.Now().UTC())
	recordReviewInMode(t, st, words[0].ID, store.AnswerModeWrite, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Question.Word.ID != words[1].ID {
		t.Fatalf("question word = %d, want choice-seen word %d", loaded.Question.Word.ID, words[1].ID)
	}
	if len(loaded.Runtime.Items) != 1 || loaded.Runtime.Items[0].WordID != words[1].ID {
		t.Fatalf("runtime items = %+v, want choice-seen word only", loaded.Runtime.Items)
	}
}

func TestSessionCmdWriteBasicFallsBackToChoiceSeenWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	recordReviewInMode(t, st, words[0].ID, store.AnswerModeChoice, time.Now().UTC())
	recordReviewInMode(t, st, words[0].ID, store.AnswerModeWrite, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Question.Word.ID != words[0].ID {
		t.Fatalf("question word = %d, want fallback choice-seen word %d", loaded.Question.Word.ID, words[0].ID)
	}
}

func TestSessionCmdWriteBasicDoesNotReuseDueWordsBeyondDueSelectionLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	now := time.Now().UTC()
	markWordDue(t, st, words[0].ID, now.AddDate(0, 0, -4))
	markWordDue(t, st, words[1].ID, now.AddDate(0, 0, -5))
	recordReviewInMode(t, st, words[2].ID, store.AnswerModeChoice, now)

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Question.Word.ID != words[2].ID {
		t.Fatalf("question word = %d, want non-due basic candidate %d", loaded.Question.Word.ID, words[2].ID)
	}
}

func TestSessionCmdWriteHardKeepsUsingNewWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	recordReviewInMode(t, st, words[0].ID, store.AnswerModeChoice, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyHard,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Question.Word.ID == words[0].ID {
		t.Fatalf("question word = %d, want an unseen new word under hard mode", loaded.Question.Word.ID)
	}
}

func TestSessionCmdWriteDefaultsToBasicWhenDifficultyUnset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}

	recordReviewInMode(t, st, words[2].ID, store.AnswerModeChoice, time.Now().UTC())

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:       store.ModeLearn,
		AnswerMode: store.AnswerModeWrite,
		Plan:       session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()
	loaded := mustSessionLoadedMsg(t, msg)

	if loaded.Question.Word.ID != words[2].ID {
		t.Fatalf("question word = %d, want basic default choice-seen word %d", loaded.Question.Word.ID, words[2].ID)
	}
}

func TestSessionCmdWriteBasicReturnsModeAwareErrorWhenChoicePoolIsEmpty(t *testing.T) {
	t.Parallel()

	st := newTestStore(t)

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()

	err := mustErrMsg(t, msg)
	if err.Error() != i18n.T(i18n.StatusWriteBasicEmpty) {
		t.Fatalf("err = %v, want %q", err, i18n.T(i18n.StatusWriteBasicEmpty))
	}
}

func TestSessionCmdReplaceActiveWriteBasicEmptyKeepsExistingActiveSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	activeRecord, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: mustWordIDByIndex(t, st, 0), Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	msg := sessionCmd(st, quiz.NewService(st), sessionRequest{
		Mode:                store.ModeLearn,
		AnswerMode:          store.AnswerModeWrite,
		WriteModeDifficulty: config.WriteModeDifficultyBasic,
		ReplaceActive:       true,
		Plan:                session.PlanOptions{QuestionCount: 1, ReviewRatio: 0},
	}, nil)()

	gotErr := mustErrMsg(t, msg)
	if gotErr.Error() != i18n.T(i18n.StatusWriteBasicEmpty) {
		t.Fatalf("err = %v, want %q", gotErr, i18n.T(i18n.StatusWriteBasicEmpty))
	}

	record, err := st.LoadSession(ctx, activeRecord.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if record.Status != store.SessionStatusActive {
		t.Fatalf("record status = %q, want %q", record.Status, store.SessionStatusActive)
	}
}

func TestSessionStartErrMsgReloadsHomeEvenWhenStatsReloadFails(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "broken-stats.db")
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = rawDB.Close()
	})

	schema := []string{
		`CREATE TABLE words (
			id INTEGER PRIMARY KEY,
			lemma TEXT NOT NULL,
			pos TEXT,
			meaning_ja TEXT NOT NULL,
			level TEXT,
			frequency_rank INTEGER,
			distractor_group TEXT,
			example_en TEXT,
			example_ja TEXT,
			source TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE progress (
			word_id INTEGER PRIMARY KEY,
			state TEXT NOT NULL,
			due_at TEXT,
			interval_days INTEGER NOT NULL,
			ease_factor REAL NOT NULL,
			last_seen_at TEXT,
			streak_correct INTEGER NOT NULL,
			total_correct INTEGER NOT NULL,
			total_wrong INTEGER NOT NULL,
			lapses INTEGER NOT NULL
		)`,
		`CREATE TABLE reviews (
			id INTEGER PRIMARY KEY,
			word_id INTEGER NOT NULL,
			session_id TEXT,
			answered_at TEXT NOT NULL,
			is_correct INTEGER NOT NULL
		)`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			started_at TEXT NOT NULL,
			finished_at TEXT,
			mode TEXT NOT NULL,
			answer_mode TEXT NOT NULL,
			total_questions INTEGER NOT NULL,
			answered_questions INTEGER NOT NULL,
			status TEXT NOT NULL
		)`,
	}
	for _, stmt := range schema {
		if _, err := rawDB.Exec(stmt); err != nil {
			t.Fatalf("Exec(%q) error = %v", stmt, err)
		}
	}

	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	msg := sessionStartErrMsg(st, errors.New("start failed"), true)
	reloaded, ok := msg.(homeReloadedErrMsg)
	if !ok {
		t.Fatalf("sessionStartErrMsg() returned %T, want homeReloadedErrMsg", msg)
	}
	if reloaded.Home.ActiveSession != nil {
		t.Fatalf("Home.ActiveSession = %+v, want nil", reloaded.Home.ActiveSession)
	}
	if reloaded.Stats != nil {
		t.Fatalf("Stats = %+v, want nil when stats reload fails", reloaded.Stats)
	}
	if reloaded.err == nil || reloaded.err.Error() != "start failed" {
		t.Fatalf("err = %v, want start failed", reloaded.err)
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
	record, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
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

func recordReviewInMode(t *testing.T, st *store.Store, wordID int64, answerMode string, answeredAt time.Time) {
	t.Helper()

	ctx := context.Background()
	record, _, err := st.CreateSession(ctx, store.ModeLearn, answerMode, []store.SessionItemPlan{
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

func mustErrMsg(t *testing.T, msg any) error {
	t.Helper()

	switch typed := msg.(type) {
	case errMsg:
		if typed.err == nil {
			t.Fatal("errMsg.err = nil, want error")
		}
		return typed.err
	default:
		t.Fatalf("unexpected msg type %T", msg)
	}
	return nil
}

func mustReviewFallbackPromptMsg(t *testing.T, msg any) reviewFallbackPromptMsg {
	t.Helper()

	switch typed := msg.(type) {
	case reviewFallbackPromptMsg:
		return typed
	case errMsg:
		t.Fatalf("sessionCmd() error = %v", typed.err)
	default:
		t.Fatalf("unexpected msg type %T", msg)
	}
	return reviewFallbackPromptMsg{}
}

func mustWordIDByIndex(t *testing.T, st *store.Store, index int) int64 {
	t.Helper()

	words, err := st.ListNewWords(context.Background(), index+1, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	if len(words) <= index {
		t.Fatalf("ListNewWords() returned %d words, want index %d", len(words), index)
	}
	return words[index].ID
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
