package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yourname/eitango/internal/dict"
)

type ResetOptions struct {
	Progress bool
	Reseed   bool
}

func (o ResetOptions) Validate() error {
	if !o.Progress && !o.Reseed {
		return fmt.Errorf("reset requires at least one scope flag: --progress or --reseed")
	}
	return nil
}

type ResetResult struct {
	Options             ResetOptions
	ClearedSessionItems int
	ClearedReviews      int
	ClearedProgress     int
	ClearedSessions     int
	ClearedWords        int
	SeededWords         int
	DictVersion         string
}

func (s *Store) Reset(ctx context.Context, options ResetOptions, coreWords []dict.Entry, coreVersion string) (ResetResult, error) {
	if err := options.Validate(); err != nil {
		return ResetResult{}, err
	}
	if options.Reseed && len(coreWords) == 0 {
		return ResetResult{}, fmt.Errorf("reset reseed: no core words provided")
	}
	if options.Reseed && coreVersion == "" {
		return ResetResult{}, fmt.Errorf("reset reseed: core version is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ResetResult{}, fmt.Errorf("begin reset: %w", err)
	}

	result := ResetResult{Options: options}
	if err := resetLearningTablesTx(ctx, tx, &result); err != nil {
		_ = tx.Rollback()
		return ResetResult{}, err
	}

	if options.Reseed {
		clearedWords, err := countRowsTx(ctx, tx, "words")
		if err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}
		result.ClearedWords = clearedWords

		if _, err := tx.ExecContext(ctx, `DELETE FROM words`); err != nil {
			_ = tx.Rollback()
			return ResetResult{}, fmt.Errorf("reset words: %w", err)
		}
		if err := insertSeedWordsTx(ctx, tx, coreWords); err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}
		if err := s.setMetaTx(ctx, tx, "dict_version", coreVersion); err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}

		result.SeededWords = len(coreWords)
		result.DictVersion = coreVersion
	}

	if err := tx.Commit(); err != nil {
		return ResetResult{}, fmt.Errorf("commit reset: %w", err)
	}

	return result, nil
}

func resetLearningTablesTx(ctx context.Context, tx *sql.Tx, result *ResetResult) error {
	steps := []struct {
		table string
		dest  *int
	}{
		{table: "session_items", dest: &result.ClearedSessionItems},
		{table: "reviews", dest: &result.ClearedReviews},
		{table: "progress", dest: &result.ClearedProgress},
		{table: "sessions", dest: &result.ClearedSessions},
	}

	for _, step := range steps {
		count, err := countRowsTx(ctx, tx, step.table)
		if err != nil {
			return err
		}
		*step.dest = count

		if _, err := tx.ExecContext(ctx, "DELETE FROM "+step.table); err != nil {
			return fmt.Errorf("reset %s: %w", step.table, err)
		}
	}

	return nil
}

func countRowsTx(ctx context.Context, tx *sql.Tx, table string) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}

func insertSeedWordsTx(ctx context.Context, tx *sql.Tx, entries []dict.Entry) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO words (
lemma,
pos,
meaning_ja,
level,
frequency_rank,
distractor_group,
example_en,
example_ja
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare seed words: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, entry := range entries {
		var rank any
		if entry.FrequencyRank > 0 {
			rank = entry.FrequencyRank
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
		); err != nil {
			return fmt.Errorf("insert seed word %s: %w", entry.Lemma, err)
		}
	}

	return nil
}
