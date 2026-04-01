package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/srs"
)

func TestImportWordsUpsertsWithinSourceAndAllowsCrossSourceDuplicates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	firstImport := []dict.Entry{
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "申し込む",
			Level:           "core-2",
			DistractorGroup: "import-verb",
		},
		{
			Lemma:           "coordinate",
			Pos:             "verb",
			MeaningJA:       "調整する",
			Level:           "core-2",
			DistractorGroup: "import-verb",
			ExampleEN:       "They coordinate each release.",
			ExampleJA:       "彼らは各リリースを調整する。",
		},
	}
	result, err := st.ImportWords(ctx, "travel-pack", firstImport)
	if err != nil {
		t.Fatalf("ImportWords() first import error = %v", err)
	}
	if result.Source != "import:travel-pack" {
		t.Fatalf("first import source = %q, want import:travel-pack", result.Source)
	}
	if result.InsertedWords != 2 || result.UpdatedWords != 0 {
		t.Fatalf("unexpected first import counts: %+v", result)
	}

	secondImport := []dict.Entry{
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "申請する",
			Level:           "core-3",
			DistractorGroup: "import-verb-2",
			ExampleEN:       "Apply for the updated permit.",
			ExampleJA:       "更新された許可を申請する。",
		},
	}
	result, err = st.ImportWords(ctx, "import:travel-pack", secondImport)
	if err != nil {
		t.Fatalf("ImportWords() second import error = %v", err)
	}
	if result.InsertedWords != 0 || result.UpdatedWords != 1 {
		t.Fatalf("unexpected second import counts: %+v", result)
	}

	otherSourceResult, err := st.ImportWords(ctx, "business-pack", []dict.Entry{
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "適用する",
			Level:           "core-3",
			DistractorGroup: "business-verb",
		},
	})
	if err != nil {
		t.Fatalf("ImportWords() cross-source duplicate error = %v", err)
	}
	if otherSourceResult.InsertedWords != 1 || otherSourceResult.UpdatedWords != 0 {
		t.Fatalf("unexpected cross-source import counts: %+v", otherSourceResult)
	}

	rows, err := st.db.QueryContext(ctx, `
SELECT source, meaning_ja, COALESCE(level, ''), COALESCE(example_en, '')
FROM words
WHERE lemma = 'apply'
ORDER BY source ASC
`)
	if err != nil {
		t.Fatalf("query imported apply rows: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	type applyRow struct {
		source    string
		meaningJA string
		level     string
		exampleEN string
	}
	var applyRows []applyRow
	for rows.Next() {
		var row applyRow
		if err := rows.Scan(&row.source, &row.meaningJA, &row.level, &row.exampleEN); err != nil {
			t.Fatalf("scan apply row: %v", err)
		}
		applyRows = append(applyRows, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate apply rows: %v", err)
	}
	if len(applyRows) != 3 {
		t.Fatalf("len(applyRows) = %d, want 3", len(applyRows))
	}

	if applyRows[0].source != WordSourceCore {
		t.Fatalf("core apply row source = %q, want %q", applyRows[0].source, WordSourceCore)
	}
	if applyRows[1].source != "import:business-pack" || applyRows[1].meaningJA != "適用する" {
		t.Fatalf("unexpected business-pack apply row: %+v", applyRows[1])
	}
	if applyRows[2].source != "import:travel-pack" || applyRows[2].meaningJA != "申請する" || applyRows[2].level != "core-3" {
		t.Fatalf("unexpected travel-pack apply row: %+v", applyRows[2])
	}
	if applyRows[2].exampleEN != "Apply for the updated permit." {
		t.Fatalf("travel-pack apply example_en = %q, want updated text", applyRows[2].exampleEN)
	}
}

func TestSeedWordsVersionChangePreservesImportedWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if _, err := st.ImportWords(ctx, "travel-pack", []dict.Entry{
		{
			Lemma:           "coordinate",
			Pos:             "verb",
			MeaningJA:       "調整する",
			Level:           "core-2",
			DistractorGroup: "import-verb",
		},
	}); err != nil {
		t.Fatalf("ImportWords() error = %v", err)
	}

	nextEntries := append(testEntries(), dict.Entry{
		Lemma:           "coach",
		Pos:             "verb",
		MeaningJA:       "指導する",
		Level:           "core-1",
		FrequencyRank:   400,
		DistractorGroup: "basic-verb-action",
	})
	if err := st.SeedWords(ctx, nextEntries, "test-v2"); err != nil {
		t.Fatalf("SeedWords() version bump error = %v", err)
	}

	if got := mustCountRows(t, st, "words"); got != len(nextEntries)+1 {
		t.Fatalf("words after version bump = %d, want %d", got, len(nextEntries)+1)
	}
	importCount, err := st.countWordsBySource(ctx, "import:travel-pack")
	if err != nil {
		t.Fatalf("countWordsBySource(import:travel-pack) error = %v", err)
	}
	if importCount != 1 {
		t.Fatalf("imported words after version bump = %d, want 1", importCount)
	}
	if got := mustCountRows(t, st, "sessions"); got != 0 {
		t.Fatalf("sessions after version bump = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 0 {
		t.Fatalf("reviews after version bump = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 0 {
		t.Fatalf("progress after version bump = %d, want 0", got)
	}
}

func TestResetReseedPreservesImportedWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)

	if err := st.SeedWords(ctx, testEntries(), "test-v1"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if _, err := st.ImportWords(ctx, "travel-pack", []dict.Entry{
		{
			Lemma:           "coordinate",
			Pos:             "verb",
			MeaningJA:       "調整する",
			Level:           "core-2",
			DistractorGroup: "import-verb",
		},
	}); err != nil {
		t.Fatalf("ImportWords() error = %v", err)
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

	replacementEntries := []dict.Entry{
		{
			Lemma:           "coach",
			Pos:             "verb",
			MeaningJA:       "指導する",
			Level:           "core-1",
			FrequencyRank:   400,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "demand",
			Pos:             "verb",
			MeaningJA:       "要求する",
			Level:           "core-1",
			FrequencyRank:   500,
			DistractorGroup: "basic-verb-action",
		},
	}
	result, err := st.Reset(ctx, ResetOptions{Reseed: true}, replacementEntries, "test-v1")
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if result.ClearedWords != len(testEntries()) {
		t.Fatalf("ClearedWords = %d, want %d", result.ClearedWords, len(testEntries()))
	}
	if got := mustCountRows(t, st, "words"); got != len(replacementEntries)+1 {
		t.Fatalf("words after reseed = %d, want %d", got, len(replacementEntries)+1)
	}
	if importCount, err := st.countWordsBySource(ctx, "import:travel-pack"); err != nil {
		t.Fatalf("countWordsBySource(import:travel-pack) error = %v", err)
	} else if importCount != 1 {
		t.Fatalf("imported words after reseed = %d, want 1", importCount)
	}
	if got := mustCountRows(t, st, "sessions"); got != 0 {
		t.Fatalf("sessions after reseed = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "reviews"); got != 0 {
		t.Fatalf("reviews after reseed = %d, want 0", got)
	}
	if got := mustCountRows(t, st, "progress"); got != 0 {
		t.Fatalf("progress after reseed = %d, want 0", got)
	}
}

func TestDefaultImportSourceDerivesFromFileName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "windows separators",
			filePath: `C:\tmp\My Words.csv`,
			want:     "import:My Words",
		},
		{
			name:     "unix separators",
			filePath: "/tmp/My Words.csv",
			want:     "import:My Words",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source, err := DefaultImportSource(tc.filePath)
			if err != nil {
				t.Fatalf("DefaultImportSource() error = %v", err)
			}
			if source != tc.want {
				t.Fatalf("DefaultImportSource() = %q, want %s", source, tc.want)
			}
		})
	}
}

func TestNormalizeImportSourceRejectsReservedCore(t *testing.T) {
	t.Parallel()

	_, err := NormalizeImportSource("core")
	if err == nil {
		t.Fatal("NormalizeImportSource() error = nil, want reserved-name error")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("NormalizeImportSource() error = %q, want reserved guidance", err.Error())
	}
}
