package store

import (
	"context"
	"database/sql"
	"fmt"
	"path"
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

type existingWordRow struct {
	id       int64
	isActive bool
}

type syncCoreWordCounts struct {
	inserted int
	updated  int
	retired  int
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
	base := path.Base(strings.ReplaceAll(strings.TrimSpace(filePath), `\`, `/`))
	if base == "" || base == "." {
		return "", fmt.Errorf("derive import source: file path is required")
	}
	name := strings.TrimSuffix(base, path.Ext(base))
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

func (s *Store) countWordsBySourceActive(ctx context.Context, source string, isActive bool) (int, error) {
	activeValue := 0
	if isActive {
		activeValue = 1
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM words WHERE source = ? AND is_active = ?`, source, activeValue).Scan(&count); err != nil {
		return 0, fmt.Errorf("count words for source %q active=%t: %w", source, isActive, err)
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
	if err := validateEntryKeys(entries, source); err != nil {
		return upsertWordCounts{}, err
	}

	existingRows, err := listExistingWordRowsBySourceTx(ctx, tx, source)
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
source,
is_active
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return upsertWordCounts{}, fmt.Errorf("prepare word insert for source %q: %w", source, err)
	}
	defer func() {
		_ = insertStmt.Close()
	}()

	updateStmt, err := tx.PrepareContext(ctx, `
UPDATE words
SET lemma = ?,
    pos = ?,
    meaning_ja = ?,
    level = ?,
    frequency_rank = ?,
    distractor_group = ?,
    example_en = ?,
    example_ja = ?,
    source = ?,
    is_active = ?
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
		existing, exists := existingRows[wordKey(entry)]
		if exists {
			if err := updateWordTx(ctx, updateStmt, existing.id, source, true, entry); err != nil {
				return upsertWordCounts{}, err
			}
			counts.updated++
			continue
		}
		if err := insertWordTx(ctx, insertStmt, source, true, entry); err != nil {
			return upsertWordCounts{}, err
		}
		counts.inserted++
	}

	return counts, nil
}

func syncCoreWordsTx(ctx context.Context, tx *sql.Tx, entries []dict.Entry) (syncCoreWordCounts, error) {
	if err := validateEntryKeys(entries, WordSourceCore); err != nil {
		return syncCoreWordCounts{}, err
	}

	existingRows, err := listExistingWordRowsBySourceTx(ctx, tx, WordSourceCore)
	if err != nil {
		return syncCoreWordCounts{}, err
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
source,
is_active
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return syncCoreWordCounts{}, fmt.Errorf("prepare core word insert: %w", err)
	}
	defer func() {
		_ = insertStmt.Close()
	}()

	updateStmt, err := tx.PrepareContext(ctx, `
UPDATE words
SET lemma = ?,
    pos = ?,
    meaning_ja = ?,
    level = ?,
    frequency_rank = ?,
    distractor_group = ?,
    example_en = ?,
    example_ja = ?,
    source = ?,
    is_active = ?
WHERE id = ?
`)
	if err != nil {
		return syncCoreWordCounts{}, fmt.Errorf("prepare core word update: %w", err)
	}
	defer func() {
		_ = updateStmt.Close()
	}()

	counts := syncCoreWordCounts{}
	for _, entry := range entries {
		key := wordKey(entry)
		existing, exists := existingRows[key]
		if exists {
			if err := updateWordTx(ctx, updateStmt, existing.id, WordSourceCore, true, entry); err != nil {
				return syncCoreWordCounts{}, err
			}
			delete(existingRows, key)
			counts.updated++
			continue
		}
		if err := insertWordTx(ctx, insertStmt, WordSourceCore, true, entry); err != nil {
			return syncCoreWordCounts{}, err
		}
		counts.inserted++
	}

	for _, existing := range existingRows {
		if !existing.isActive {
			continue
		}
		// False positive: the SQL is static and the id stays parameterized.
		// nosemgrep
		if _, err := tx.ExecContext(ctx, `UPDATE words SET is_active = 0 WHERE id = ?`, existing.id); err != nil {
			return syncCoreWordCounts{}, fmt.Errorf("retire core word %d: %w", existing.id, err)
		}
		counts.retired++
	}

	return counts, nil
}

func listExistingWordRowsBySourceTx(ctx context.Context, tx *sql.Tx, source string) (map[string]existingWordRow, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, lemma, IFNULL(pos, ''), is_active
FROM words
WHERE source = ?
`, source)
	if err != nil {
		return nil, fmt.Errorf("list existing words for source %q: %w", source, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	existingRows := make(map[string]existingWordRow)
	for rows.Next() {
		var (
			id       int64
			lemma    string
			pos      string
			isActive int
		)
		if err := rows.Scan(&id, &lemma, &pos, &isActive); err != nil {
			return nil, fmt.Errorf("scan existing word for source %q: %w", source, err)
		}
		key := strings.ToLower(strings.TrimSpace(lemma) + "\x00" + strings.TrimSpace(pos))
		if _, exists := existingRows[key]; exists {
			return nil, fmt.Errorf("duplicate word key %s already exists in source %q", formatWordKeyLabel(lemma, pos), source)
		}
		existingRows[key] = existingWordRow{id: id, isActive: isActive != 0}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing words for source %q: %w", source, err)
	}
	return existingRows, nil
}

func wordKey(entry dict.Entry) string {
	return strings.ToLower(strings.TrimSpace(entry.Lemma) + "\x00" + strings.TrimSpace(entry.Pos))
}

func validateEntryKeys(entries []dict.Entry, source string) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key := wordKey(entry)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate word key %s in source %q", formatWordKeyLabel(entry.Lemma, entry.Pos), source)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func formatWordKeyLabel(lemma, pos string) string {
	trimmedLemma := strings.TrimSpace(lemma)
	trimmedPos := strings.TrimSpace(pos)
	if trimmedPos == "" {
		trimmedPos = "no-pos"
	}
	return fmt.Sprintf("%q [%s]", trimmedLemma, trimmedPos)
}

func insertWordTx(ctx context.Context, stmt *sql.Stmt, source string, isActive bool, entry dict.Entry) error {
	rank := nullableFrequencyRank(entry.FrequencyRank)
	activeValue := 0
	if isActive {
		activeValue = 1
	}
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
		activeValue,
	); err != nil {
		return fmt.Errorf("insert word %s for source %q: %w", entry.Lemma, source, err)
	}
	return nil
}

func updateWordTx(ctx context.Context, stmt *sql.Stmt, id int64, source string, isActive bool, entry dict.Entry) error {
	activeValue := 0
	if isActive {
		activeValue = 1
	}
	if _, err := stmt.ExecContext(
		ctx,
		nullableString(entry.Lemma),
		nullableString(entry.Pos),
		nullableString(entry.MeaningJA),
		nullableString(entry.Level),
		nullableFrequencyRank(entry.FrequencyRank),
		nullableString(entry.DistractorGroup),
		nullableString(entry.ExampleEN),
		nullableString(entry.ExampleJA),
		source,
		activeValue,
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
