package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
)

func TestExportWrongWordsCommandWritesCSV(t *testing.T) {
	dataDir := t.TempDir()
	fixture := seedExportCommandFixture(t, dataDir)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"export", "wrong-words", "--format", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rows, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("wrong-words csv rows = %d, want 2", len(rows))
	}
	expectedHeader := []string{
		"lemma",
		"pos",
		"meaning_ja",
		"level",
		"frequency_rank",
		"distractor_group",
		"source",
		"state",
		"wrong_reviews",
		"correct_reviews",
		"total_reviews",
		"last_wrong_at",
		"last_correct_at",
		"due_at",
		"example_en",
		"example_ja",
	}
	if got := rows[0]; len(got) != len(expectedHeader) {
		t.Fatalf("wrong-words header len = %d, want %d", len(got), len(expectedHeader))
	} else {
		for i := range expectedHeader {
			if got[i] != expectedHeader[i] {
				t.Fatalf("wrong-words header[%d] = %q, want %q", i, got[i], expectedHeader[i])
			}
		}
	}

	row := rows[1]
	if row[0] != "accept" || row[2] != "受け入れる" {
		t.Fatalf("wrong-words first row = %+v, want accept", row)
	}
	if row[6] != store.WordSourceCore {
		t.Fatalf("wrong-words source = %q, want %q", row[6], store.WordSourceCore)
	}
	if row[7] != "review" || row[8] != "1" || row[9] != "1" || row[10] != "2" {
		t.Fatalf("unexpected wrong-words counters row = %+v", row)
	}
	if row[11] != fixture.firstWrongAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("last_wrong_at = %q, want %q", row[11], fixture.firstWrongAt.UTC().Format(time.RFC3339Nano))
	}
	if row[12] != fixture.retryCorrectAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("last_correct_at = %q, want %q", row[12], fixture.retryCorrectAt.UTC().Format(time.RFC3339Nano))
	}
	if row[13] == "" {
		t.Fatal("due_at should not be empty for reviewed wrong word")
	}
}

func TestExportProgressCommandWritesJSON(t *testing.T) {
	dataDir := t.TempDir()
	fixture := seedExportCommandFixture(t, dataDir)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"export", "progress", "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var document progressExportDocument
	if err := json.Unmarshal(out.Bytes(), &document); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if document.DictVersion != fixture.dictVersion {
		t.Fatalf("dict_version = %q, want %q", document.DictVersion, fixture.dictVersion)
	}
	if document.Summary.TotalWords != 3 || document.Summary.NewWords != 1 || document.Summary.ReviewWords != 2 {
		t.Fatalf("unexpected progress summary counts: %+v", document.Summary)
	}
	if document.Summary.LearningWords != 0 || document.Summary.ReviewedWords != 2 || document.Summary.WrongWords != 1 {
		t.Fatalf("unexpected progress summary stats: %+v", document.Summary)
	}
	if document.ExportedAt == "" {
		t.Fatal("exported_at is empty")
	}
	if len(document.Words) != 3 {
		t.Fatalf("len(words) = %d, want 3", len(document.Words))
	}

	accept := mustFindProgressExportWord(t, document.Words, "accept")
	if accept.Word.Source != store.WordSourceCore {
		t.Fatalf("accept source = %q, want %q", accept.Word.Source, store.WordSourceCore)
	}
	if accept.Progress.State != "review" || accept.ReviewStats.WrongReviews != 1 || accept.ReviewStats.TotalReviews != 2 {
		t.Fatalf("unexpected accept export record: %+v", accept)
	}
	if accept.ReviewStats.LastWrongAt == nil || *accept.ReviewStats.LastWrongAt != fixture.firstWrongAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("accept last_wrong_at = %+v, want %q", accept.ReviewStats.LastWrongAt, fixture.firstWrongAt.UTC().Format(time.RFC3339Nano))
	}
	if accept.Progress.DueAt == nil {
		t.Fatal("accept progress due_at is nil")
	}

	budget := mustFindProgressExportWord(t, document.Words, "budget")
	if budget.Progress.State != "new" {
		t.Fatalf("budget progress state = %q, want new", budget.Progress.State)
	}
	if budget.ReviewStats.TotalReviews != 0 || budget.ReviewStats.LastAnsweredAt != nil {
		t.Fatalf("unexpected budget review stats: %+v", budget.ReviewStats)
	}
}

func TestExportProgressCommandWritesOutputFile(t *testing.T) {
	dataDir := t.TempDir()
	seedExportCommandFixture(t, dataDir)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	outputPath := filepath.Join(dataDir, "progress-export.json")
	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"export", "progress", "--format", "json", "--output", outputPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout should be empty when writing file, got %q", out.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var document progressExportDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("json.Unmarshal() output file error = %v", err)
	}
	if document.Summary.TotalWords != 3 {
		t.Fatalf("file total_words = %d, want 3", document.Summary.TotalWords)
	}
}

func TestExportProgressCommandRejectsUnsupportedFormatBeforeOpeningDB(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"export", "progress", "--format", "csv"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want format validation error")
	}
	if got := err.Error(); got != "eitango export progress only supports --format json" {
		t.Fatalf("Execute() error = %q, want format guidance", got)
	}
	if _, statErr := os.Stat(dbPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("user.db should not be created, stat error = %v", statErr)
	}
}

type exportCommandFixture struct {
	dictVersion     string
	firstWrongAt    time.Time
	retryCorrectAt  time.Time
	secondCorrectAt time.Time
}

func seedExportCommandFixture(t *testing.T, dataDir string) exportCommandFixture {
	t.Helper()

	dbPath := filepath.Join(dataDir, "user.db")
	ctx := context.Background()
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	entries := []dict.Entry{
		{
			Lemma:           "accept",
			Pos:             "verb",
			MeaningJA:       "受け入れる",
			Level:           "core-1",
			FrequencyRank:   100,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "They accept the updated plan.",
			ExampleJA:       "彼らは更新された計画を受け入れる。",
		},
		{
			Lemma:           "avoid",
			Pos:             "verb",
			MeaningJA:       "避ける",
			Level:           "core-1",
			FrequencyRank:   120,
			DistractorGroup: "basic-verb-action",
			ExampleEN:       "Try to avoid the traffic jam.",
			ExampleJA:       "交通渋滞を避けるようにしてください。",
		},
		{
			Lemma:           "budget",
			Pos:             "noun",
			MeaningJA:       "予算",
			Level:           "core-1",
			FrequencyRank:   140,
			DistractorGroup: "basic-noun-business",
			ExampleEN:       "We reviewed the annual budget.",
			ExampleJA:       "私たちは年間予算を見直した。",
		},
	}
	const dictVersion = "test-export-v1"
	if err := st.SeedWords(ctx, entries, dictVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	wordsByLemma := make(map[string]store.Word, len(words))
	for _, word := range words {
		wordsByLemma[word.Lemma] = word
	}

	firstWrongAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	retryCorrectAt := firstWrongAt.Add(20 * time.Minute)
	record, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: wordsByLemma["accept"].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() accept session error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, store.ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordsByLemma["accept"].ID,
		Kind:           store.ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  2,
		IsCorrect:      false,
		Rating:         srs.Again,
		AnsweredAt:     firstWrongAt,
		ResponseMS:     1100,
	}); err != nil {
		t.Fatalf("SaveAnswer() accept wrong error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, store.ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    2,
		WordID:         wordsByLemma["accept"].ID,
		Kind:           store.ItemKindRetry,
		SelectedChoice: 2,
		CorrectChoice:  2,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     retryCorrectAt,
		ResponseMS:     850,
	}); err != nil {
		t.Fatalf("SaveAnswer() accept retry error = %v", err)
	}

	secondCorrectAt := retryCorrectAt.Add(40 * time.Minute)
	record, _, err = st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: wordsByLemma["avoid"].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() avoid session error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, store.ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         wordsByLemma["avoid"].ID,
		Kind:           store.ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     secondCorrectAt,
		ResponseMS:     780,
	}); err != nil {
		t.Fatalf("SaveAnswer() avoid correct error = %v", err)
	}

	return exportCommandFixture{
		dictVersion:     dictVersion,
		firstWrongAt:    firstWrongAt,
		retryCorrectAt:  retryCorrectAt,
		secondCorrectAt: secondCorrectAt,
	}
}

func mustFindProgressExportWord(t *testing.T, words []progressExportWord, lemma string) progressExportWord {
	t.Helper()

	for _, word := range words {
		if word.Word.Lemma == lemma {
			return word
		}
	}
	t.Fatalf("progress export word %q not found", lemma)
	return progressExportWord{}
}
