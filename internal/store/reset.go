package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/harumiWeb/eitango/internal/dict"
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
		clearedWords, err := deleteWordsBySourceTx(ctx, tx, WordSourceCore)
		if err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}
		result.ClearedWords = clearedWords

		counts, err := upsertWordsTx(ctx, tx, WordSourceCore, coreWords)
		if err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}
		if err := s.setMetaTx(ctx, tx, "dict_version", coreVersion); err != nil {
			_ = tx.Rollback()
			return ResetResult{}, err
		}

		result.SeededWords = counts.inserted + counts.updated
		result.DictVersion = coreVersion
	}

	if err := tx.Commit(); err != nil {
		return ResetResult{}, fmt.Errorf("commit reset: %w", err)
	}

	return result, nil
}

func resetLearningTablesTx(ctx context.Context, tx *sql.Tx, result *ResetResult) error {
	type resetStep struct {
		table string
		apply func(int)
	}
	steps := []resetStep{
		{
			table: "session_items",
			apply: func(count int) {
				if result != nil {
					result.ClearedSessionItems = count
				}
			},
		},
		{
			table: "reviews",
			apply: func(count int) {
				if result != nil {
					result.ClearedReviews = count
				}
			},
		},
		{
			table: "progress",
			apply: func(count int) {
				if result != nil {
					result.ClearedProgress = count
				}
			},
		},
		{
			table: "sessions",
			apply: func(count int) {
				if result != nil {
					result.ClearedSessions = count
				}
			},
		},
	}

	for _, step := range steps {
		count, err := countRowsTx(ctx, tx, step.table)
		if err != nil {
			return err
		}
		step.apply(count)

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
