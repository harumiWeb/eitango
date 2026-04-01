package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	projectassets "github.com/harumiWeb/eitango/assets"
	"github.com/harumiWeb/eitango/internal/dict"
)

func TestRunDiagnosticsHealthyStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	report := st.RunDiagnostics(ctx)
	if report.HasIssues() {
		t.Fatalf("RunDiagnostics() reported issues on a healthy store: %+v", report.Checks)
	}

	quizability, ok := report.Check("quizability")
	if !ok {
		t.Fatal("quizability check not found")
	}
	if quizability.Status != DiagnosticStatusOK {
		t.Fatalf("quizability status = %q, want %q", quizability.Status, DiagnosticStatusOK)
	}
}

func TestRunDiagnosticsDetectsDictionaryVersionMismatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), "old-version"); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	report := st.RunDiagnostics(ctx)

	dictionary, ok := report.Check("dictionary")
	if !ok {
		t.Fatal("dictionary check not found")
	}
	if dictionary.Status != DiagnosticStatusWarning {
		t.Fatalf("dictionary status = %q, want %q", dictionary.Status, DiagnosticStatusWarning)
	}
	if !strings.Contains(dictionary.Summary, "embedded core words") {
		t.Fatalf("dictionary summary = %q, want embedded version mismatch", dictionary.Summary)
	}
}

func TestRunDiagnosticsDetectsOrphanRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF;`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.db.ExecContext(context.Background(), `PRAGMA foreign_keys = ON;`)
	})

	if _, err := st.db.ExecContext(ctx, `INSERT INTO progress (word_id, state) VALUES (9999, 'review')`); err != nil {
		t.Fatalf("insert orphan progress: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `
INSERT INTO reviews (word_id, session_id, answered_at, selected_choice, correct_choice, is_correct)
VALUES (9999, 'missing-session', CURRENT_TIMESTAMP, 1, 1, 1)
`); err != nil {
		t.Fatalf("insert orphan review: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `
INSERT INTO session_items (session_id, ordinal, word_id, kind, status)
VALUES ('missing-session', 1, 9999, ?, ?)
`, ItemKindReview, ItemStatusPending); err != nil {
		t.Fatalf("insert orphan session item: %v", err)
	}

	report := st.RunDiagnostics(ctx)

	progressCheck, ok := report.Check("orphan progress")
	if !ok {
		t.Fatal("orphan progress check not found")
	}
	if progressCheck.Status != DiagnosticStatusError {
		t.Fatalf("orphan progress status = %q, want %q", progressCheck.Status, DiagnosticStatusError)
	}

	reviewsCheck, ok := report.Check("orphan reviews")
	if !ok {
		t.Fatal("orphan reviews check not found")
	}
	if reviewsCheck.Status != DiagnosticStatusError {
		t.Fatalf("orphan reviews status = %q, want %q", reviewsCheck.Status, DiagnosticStatusError)
	}

	itemsCheck, ok := report.Check("orphan session items")
	if !ok {
		t.Fatal("orphan session items check not found")
	}
	if itemsCheck.Status != DiagnosticStatusError {
		t.Fatalf("orphan session items status = %q, want %q", itemsCheck.Status, DiagnosticStatusError)
	}
}

func TestRunDiagnosticsDetectsActiveSessionInconsistencies(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListWordsByPOS(ctx, "verb", 4, nil)
	if err != nil {
		t.Fatalf("ListWordsByPOS() error = %v", err)
	}
	if len(words) < 4 {
		t.Fatalf("len(words) = %d, want at least 4", len(words))
	}

	record, _, err := st.CreateSession(ctx, ModeLearn, AnswerModeChoice, []SessionItemPlan{
		{WordID: words[0].ID, Kind: ItemKindNew},
		{WordID: words[1].ID, Kind: ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if _, err := st.db.ExecContext(ctx, `
UPDATE sessions
SET answered_questions = 3, total_questions = 1, finished_at = CURRENT_TIMESTAMP
WHERE id = ?
`, record.ID); err != nil {
		t.Fatalf("corrupt active session: %v", err)
	}

	report := st.RunDiagnostics(ctx)

	active, ok := report.Check("active sessions")
	if !ok {
		t.Fatal("active sessions check not found")
	}
	if active.Status != DiagnosticStatusError {
		t.Fatalf("active sessions status = %q, want %q", active.Status, DiagnosticStatusError)
	}
	if strings.Contains(active.Summary, "could not be inspected") || strings.Contains(strings.Join(active.Details, "\n"), "context deadline exceeded") {
		t.Fatalf("active session check timed out instead of reporting inconsistencies: %+v", active)
	}
	if !strings.Contains(strings.Join(active.Details, "\n"), record.ID) {
		t.Fatalf("active session details = %+v, want session id %s", active.Details, record.ID)
	}
}

func TestRunDiagnosticsReadOnlyLegacySessionsFallbackToChoiceAnswerMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := legacyDoctorDBPath(t)

	st, err := OpenReadOnly(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly() error = %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	report := st.RunDiagnostics(ctx)

	migrations, ok := report.Check("migrations")
	if !ok {
		t.Fatal("migrations check not found")
	}
	if migrations.Status != DiagnosticStatusError {
		t.Fatalf("migrations status = %q, want %q", migrations.Status, DiagnosticStatusError)
	}
	if !strings.Contains(strings.Join(migrations.Details, "\n"), "005_answer_modes.sql") {
		t.Fatalf("migrations details = %+v, want missing 005 migration", migrations.Details)
	}

	active, ok := report.Check("active sessions")
	if !ok {
		t.Fatal("active sessions check not found")
	}
	if active.Status != DiagnosticStatusOK {
		t.Fatalf("active sessions status = %q, want %q", active.Status, DiagnosticStatusOK)
	}
	if strings.Contains(active.Summary, "could not be read") || strings.Contains(strings.Join(active.Details, "\n"), "answer_mode") {
		t.Fatalf("active sessions = %+v, want legacy fallback without read failure", active)
	}
}

func TestRunDiagnosticsDetectsUnquizzableWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, testEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	report := st.RunDiagnostics(ctx)

	quizability, ok := report.Check("quizability")
	if !ok {
		t.Fatal("quizability check not found")
	}
	if quizability.Status != DiagnosticStatusError {
		t.Fatalf("quizability status = %q, want %q", quizability.Status, DiagnosticStatusError)
	}
	if !strings.Contains(quizability.Summary, "cannot form") {
		t.Fatalf("quizability summary = %q, want failure detail", quizability.Summary)
	}
}

func TestRunDiagnosticsWarnsOnCrossSourceDuplicates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if _, err := st.ImportWords(ctx, "travel-pack", []dict.Entry{
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "申請する",
			Level:           "core-2",
			DistractorGroup: "import-verb",
		},
	}); err != nil {
		t.Fatalf("ImportWords() error = %v", err)
	}

	report := st.RunDiagnostics(ctx)

	wordSources, ok := report.Check("word sources")
	if !ok {
		t.Fatal("word sources check not found")
	}
	if wordSources.Status != DiagnosticStatusWarning {
		t.Fatalf("word sources status = %q, want %q", wordSources.Status, DiagnosticStatusWarning)
	}
	if !strings.Contains(strings.Join(wordSources.Details, "\n"), "apply [verb]") {
		t.Fatalf("word sources details = %+v, want apply sample", wordSources.Details)
	}
}

func TestRunDiagnosticsWarnsOnMissingWordMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, []dict.Entry{
		{Lemma: "adopt", Pos: "verb", MeaningJA: "採用する", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "apply", Pos: "verb", MeaningJA: "応募する", Level: "core-1", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "cancel", Pos: "verb", MeaningJA: "取り消す", Level: "", FrequencyRank: 0, DistractorGroup: ""},
		{Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "core-1", FrequencyRank: 160, DistractorGroup: "basic-verb-action"},
	}, dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	report := st.RunDiagnostics(ctx)

	metadata, ok := report.Check("word metadata")
	if !ok {
		t.Fatal("word metadata check not found")
	}
	if metadata.Status != DiagnosticStatusWarning {
		t.Fatalf("word metadata status = %q, want %q", metadata.Status, DiagnosticStatusWarning)
	}
	if !strings.Contains(strings.Join(metadata.Details, "\n"), "cancel") {
		t.Fatalf("word metadata details = %+v, want missing-metadata sample", metadata.Details)
	}
}

func TestRunDiagnosticsErrorsOnSameSourceDuplicates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newTestStore(t)
	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `
INSERT INTO words (lemma, pos, meaning_ja, source)
VALUES ('apply', 'verb', '重複データ', ?)
`, WordSourceCore); err != nil {
		t.Fatalf("insert same-source duplicate: %v", err)
	}

	report := st.RunDiagnostics(ctx)

	wordSources, ok := report.Check("word sources")
	if !ok {
		t.Fatal("word sources check not found")
	}
	if wordSources.Status != DiagnosticStatusError {
		t.Fatalf("word sources status = %q, want %q", wordSources.Status, DiagnosticStatusError)
	}
	if !strings.Contains(strings.Join(wordSources.Details, "\n"), WordSourceCore+" -> apply [verb]") {
		t.Fatalf("word sources details = %+v, want same-source duplicate sample", wordSources.Details)
	}
}

func doctorTestEntries() []dict.Entry {
	return []dict.Entry{
		{
			Lemma:           "adopt",
			Pos:             "verb",
			MeaningJA:       "採用する",
			Level:           "core-1",
			FrequencyRank:   100,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "apply",
			Pos:             "verb",
			MeaningJA:       "応募する",
			Level:           "core-1",
			FrequencyRank:   120,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "cancel",
			Pos:             "verb",
			MeaningJA:       "取り消す",
			Level:           "core-1",
			FrequencyRank:   140,
			DistractorGroup: "basic-verb-action",
		},
		{
			Lemma:           "deliver",
			Pos:             "verb",
			MeaningJA:       "届ける",
			Level:           "core-1",
			FrequencyRank:   160,
			DistractorGroup: "basic-verb-action",
		},
	}
}

func legacyDoctorDBPath(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-doctor.db")
	st, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	if _, err := st.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
version TEXT PRIMARY KEY,
applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	for _, migration := range []string{
		"001_init.sql",
		"002_indexes.sql",
		"003_words_pos_rank.sql",
		"004_words_source.sql",
	} {
		sqlBytes, err := projectassets.Embedded.ReadFile("migrations/" + migration)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", migration, err)
		}
		if _, err := st.db.ExecContext(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", migration, err)
		}
		if _, err := st.db.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, migration); err != nil {
			t.Fatalf("record %s: %v", migration, err)
		}
	}

	if err := st.SeedWords(ctx, doctorTestEntries(), dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListWordsByPOS(ctx, "verb", 1, nil)
	if err != nil {
		t.Fatalf("ListWordsByPOS() error = %v", err)
	}
	if len(words) == 0 {
		t.Fatal("ListWordsByPOS() returned no words")
	}

	if _, err := st.db.ExecContext(ctx, `
INSERT INTO sessions (id, started_at, mode, total_questions, answered_questions, status)
VALUES (?, CURRENT_TIMESTAMP, ?, 1, 0, ?)
`, "legacy-active", ModeLearn, SessionStatusActive); err != nil {
		t.Fatalf("insert legacy session: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `
INSERT INTO session_items (session_id, ordinal, word_id, kind, status)
VALUES (?, 1, ?, ?, ?)
`, "legacy-active", words[0].ID, ItemKindNew, ItemStatusPending); err != nil {
		t.Fatalf("insert legacy session item: %v", err)
	}

	return dbPath
}
