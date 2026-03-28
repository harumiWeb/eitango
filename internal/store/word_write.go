package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/harumiWeb/eitango/internal/dict"
)

const (
	WordSourceCore         = "core"
	wordSourceImportPrefix = "import:"
)

type upsertWordCounts struct {
	inserted int
	updated  int
}

func NormalizeImportSource(name string) (string, error) {
	sourceName := strings.TrimSpace(name)
	if sourceName == "" {
		return "", fmt.Errorf("import source name is required")
	}
	if strings.EqualFold(sourceName, WordSourceCore) {
		return "", fmt.Errorf("import source name %q is reserved", sourceName)
	}
	if strings.HasPrefix(strings.ToLower(sourceName), wordSourceImportPrefix) {
		sourceName = strings.TrimSpace(sourceName[len(wordSourceImportPrefix):])
	}
	if sourceName == "" {
		return "", fmt.Errorf("import source name is required")
	}
	return wordSourceImportPrefix + sourceName, nil
}

func DefaultImportSource(filePath string) (string, error) {
	base := filepath.Base(strings.TrimSpace(filePath))
	if base == "" || base == "." {
		return "", fmt.Errorf("derive import source: file path is required")
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "", fmt.Errorf("derive import source from %q: base name is empty", filePath)
	}
	return NormalizeImportSource(name)
}

func (s *Store) ImportWords(ctx context.Context, source string, entries []dict.Entry) (ImportResult, error) {
	if len(entries) == 0 {
		return ImportResult{}, fmt.Errorf("import words: no entries provided")
	}
	normalizedSource, err := NormalizeImportSource(source)
	if err != nil {
		return ImportResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import words: %w", err)
	}

	counts, err := upsertWordsTx(ctx, tx, normalizedSource, entries)
	if err != nil {
		_ = tx.Rollback()
		return ImportResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit import words: %w", err)
	}

	return ImportResult{
		Source:        normalizedSource,
		InsertedWords: counts.inserted,
		UpdatedWords:  counts.updated,
	}, nil
}

func (s *Store) countWordsBySource(ctx context.Context, source string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM words WHERE source = ?`, source).Scan(&count); err != nil {
		return 0, fmt.Errorf("count words for source %q: %w", source, err)
	}
	return count, nil
}

func countWordsBySourceTx(ctx context.Context, tx *sql.Tx, source string) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM words WHERE source = ?`, source).Scan(&count); err != nil {
		return 0, fmt.Errorf("count words for source %q: %w", source, err)
	}
	return count, nil
}

func deleteWordsBySourceTx(ctx context.Context, tx *sql.Tx, source string) (int, error) {
	count, err := countWordsBySourceTx(ctx, tx, source)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM words WHERE source = ?`, source); err != nil {
		return 0, fmt.Errorf("delete words for source %q: %w", source, err)
	}
	return count, nil
}

func upsertWordsTx(ctx context.Context, tx *sql.Tx, source string, entries []dict.Entry) (upsertWordCounts, error) {
	existingIDs, err := listExistingWordIDsBySourceTx(ctx, tx, source)
	if err != nil {
		return upsertWordCounts{}, err
	}

	insertStmt, err := tx.PrepareContext(ctx, `
INSERT INTO words (
lemma,
pos,
meaning_ja,
level,
frequency_rank,
distractor_group,
example_en,
example_ja,
source
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return upsertWordCounts{}, fmt.Errorf("prepare word insert for source %q: %w", source, err)
	}
	defer func() {
		_ = insertStmt.Close()
	}()

	updateStmt, err := tx.PrepareContext(ctx, `
UPDATE words
SET meaning_ja = ?,
    level = ?,
    frequency_rank = ?,
    distractor_group = ?,
    example_en = ?,
    example_ja = ?
WHERE id = ?
`)
	if err != nil {
		return upsertWordCounts{}, fmt.Errorf("prepare word update for source %q: %w", source, err)
	}
	defer func() {
		_ = updateStmt.Close()
	}()

	counts := upsertWordCounts{}
	for _, entry := range entries {
		existingID, exists := existingIDs[wordKey(entry)]
		if exists {
			if err := updateWordTx(ctx, updateStmt, existingID, entry); err != nil {
				return upsertWordCounts{}, err
			}
			counts.updated++
			continue
		}
		if err := insertWordTx(ctx, insertStmt, source, entry); err != nil {
			return upsertWordCounts{}, err
		}
		counts.inserted++
	}

	return counts, nil
}

func listExistingWordIDsBySourceTx(ctx context.Context, tx *sql.Tx, source string) (map[string]int64, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, lemma, IFNULL(pos, '')
FROM words
WHERE source = ?
`, source)
	if err != nil {
		return nil, fmt.Errorf("list existing words for source %q: %w", source, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	existingIDs := make(map[string]int64)
	for rows.Next() {
		var (
			id    int64
			lemma string
			pos   string
		)
		if err := rows.Scan(&id, &lemma, &pos); err != nil {
			return nil, fmt.Errorf("scan existing word for source %q: %w", source, err)
		}
		key := strings.ToLower(strings.TrimSpace(lemma) + "\x00" + strings.TrimSpace(pos))
		if _, exists := existingIDs[key]; !exists {
			existingIDs[key] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing words for source %q: %w", source, err)
	}
	return existingIDs, nil
}

func wordKey(entry dict.Entry) string {
	return strings.ToLower(strings.TrimSpace(entry.Lemma) + "\x00" + strings.TrimSpace(entry.Pos))
}

func insertWordTx(ctx context.Context, stmt *sql.Stmt, source string, entry dict.Entry) error {
	rank := nullableFrequencyRank(entry.FrequencyRank)
	if _, err := stmt.ExecContext(
		ctx,
		nullableString(entry.Lemma),
		nullableString(entry.Pos),
		nullableString(entry.MeaningJA),
		nullableString(entry.Level),
		rank,
		nullableString(entry.DistractorGroup),
		nullableString(entry.ExampleEN),
		nullableString(entry.ExampleJA),
		source,
	); err != nil {
		return fmt.Errorf("insert word %s for source %q: %w", entry.Lemma, source, err)
	}
	return nil
}

func updateWordTx(ctx context.Context, stmt *sql.Stmt, id int64, entry dict.Entry) error {
	if _, err := stmt.ExecContext(
		ctx,
		nullableString(entry.MeaningJA),
		nullableString(entry.Level),
		nullableFrequencyRank(entry.FrequencyRank),
		nullableString(entry.DistractorGroup),
		nullableString(entry.ExampleEN),
		nullableString(entry.ExampleJA),
		id,
	); err != nil {
		return fmt.Errorf("update word %s (%d): %w", entry.Lemma, id, err)
	}
	return nil
}

func nullableFrequencyRank(rank int) any {
	if rank <= 0 {
		return nil
	}
	return rank
}
