package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	projectassets "github.com/harumiWeb/eitango/assets"
	"github.com/harumiWeb/eitango/internal/dict"
)

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
version TEXT PRIMARY KEY,
applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrations, err := embeddedMigrationNames()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := s.hasMigration(ctx, migration)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sqlBytes, err := projectassets.Embedded.ReadFile("migrations/" + migration)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migration, err)
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration, err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, migration); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration, err)
		}
	}

	return nil
}

func (s *Store) SeedWords(ctx context.Context, entries []dict.Entry, version string) error {
	if len(entries) == 0 {
		return fmt.Errorf("seed words: no entries provided")
	}
	if version == "" {
		return fmt.Errorf("seed words: version is required")
	}

	coreWordCount, err := s.countWordsBySource(ctx, WordSourceCore)
	if err != nil {
		return err
	}

	currentVersion, err := s.metaValue(ctx, "dict_version")
	if err != nil {
		return err
	}
	if coreWordCount > 0 && currentVersion == version {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed words: %w", err)
	}

	if coreWordCount > 0 && currentVersion != version {
		if err := resetLearningTablesTx(ctx, tx, nil); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("reset before seed: %w", err)
		}
		if _, err := deleteWordsBySourceTx(ctx, tx, WordSourceCore); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("replace core words before seed: %w", err)
		}
	}

	if _, err := upsertWordsTx(ctx, tx, WordSourceCore, entries); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := s.setMetaTx(ctx, tx, "dict_version", version); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed words: %w", err)
	}

	return nil
}

func (s *Store) hasMigration(ctx context.Context, version string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM schema_migrations WHERE version = ?`, version).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, fmt.Errorf("check migration %s: %w", version, err)
}

func (s *Store) metaValue(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, key).Scan(&value)
	if err == nil {
		return value, nil
	}
	if err == sql.ErrNoRows {
		return "", nil
	}
	return "", fmt.Errorf("load app_meta %s: %w", key, err)
}

func (s *Store) setMetaTx(ctx context.Context, tx *sql.Tx, key, value string) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO app_meta (key, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET
value = excluded.value,
updated_at = CURRENT_TIMESTAMP
`, key, value); err != nil {
		return fmt.Errorf("upsert app_meta %s: %w", key, err)
	}
	return nil
}

func (s *Store) wordCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM words`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count words: %w", err)
	}
	return count, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func embeddedMigrationNames() ([]string, error) {
	entries, err := projectassets.Embedded.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	migrations := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		migrations = append(migrations, entry.Name())
	}
	return migrations, nil
}
