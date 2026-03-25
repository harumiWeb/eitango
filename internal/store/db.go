package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dbPath string) (*Store, error) {
	return open(ctx, dbPath, false)
}

func OpenReadOnly(ctx context.Context, dbPath string) (*Store, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("stat sqlite %s: %w", dbPath, err)
	}
	return open(ctx, sqliteURI(dbPath, "ro"), true)
}

func open(ctx context.Context, dsn string, readOnly bool) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.configure(ctx, readOnly); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) configure(ctx context.Context, readOnly bool) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	if !readOnly {
		pragmas = append(pragmas,
			"PRAGMA journal_mode = WAL;",
			"PRAGMA synchronous = NORMAL;",
		)
	}
	for _, pragma := range pragmas {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("apply %s: %w", pragma, err)
		}
	}

	return nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
	} {
		var (
			parsed time.Time
			err    error
		)
		if layout == "2006-01-02 15:04:05" {
			parsed, err = time.ParseInLocation(layout, raw, time.UTC)
		} else {
			parsed, err = time.Parse(layout, raw)
		}
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q: unsupported format", raw)
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func sqliteURI(dbPath, mode string) string {
	parts := strings.Split(filepath.ToSlash(dbPath), "/")
	for i, part := range parts {
		if i == 0 && strings.HasSuffix(part, ":") {
			continue
		}
		parts[i] = url.PathEscape(part)
	}
	return "file:" + strings.Join(parts, "/") + "?mode=" + url.QueryEscape(mode)
}
