package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/harumiWeb/eitango/internal/srs"
)

func (s *Store) ListExportWordSnapshots(ctx context.Context) ([]ExportWordSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT w.id, w.lemma, w.pos, w.meaning_ja, w.level, w.frequency_rank,
       w.distractor_group, w.example_en, w.example_ja, w.source, w.created_at,
       p.state, p.due_at, p.interval_days, p.ease_factor, p.last_seen_at,
       p.streak_correct, p.total_correct, p.total_wrong, p.lapses,
       COALESCE(review_stats.total_reviews, 0),
       COALESCE(review_stats.correct_reviews, 0),
       COALESCE(review_stats.wrong_reviews, 0),
       review_stats.last_answered_at,
       review_stats.last_wrong_at,
       review_stats.last_correct_at
FROM words w
LEFT JOIN progress p ON p.word_id = w.id
LEFT JOIN (
  SELECT word_id,
         COUNT(*) AS total_reviews,
         COALESCE(SUM(is_correct), 0) AS correct_reviews,
         COUNT(*) - COALESCE(SUM(is_correct), 0) AS wrong_reviews,
         MAX(answered_at) AS last_answered_at,
         MAX(CASE WHEN is_correct = 0 THEN answered_at END) AS last_wrong_at,
         MAX(CASE WHEN is_correct = 1 THEN answered_at END) AS last_correct_at
  FROM reviews
  GROUP BY word_id
) review_stats ON review_stats.word_id = w.id
ORDER BY COALESCE(w.frequency_rank, 999999) ASC, w.id ASC
`)
	if err != nil {
		return nil, fmt.Errorf("query export word snapshots: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanExportWordSnapshots(rows)
}

func (s *Store) ListWrongWordSnapshots(ctx context.Context) ([]ExportWordSnapshot, error) {
	snapshots, err := s.ListExportWordSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	wrongWords := make([]ExportWordSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.ReviewStats.WrongReviews > 0 {
			wrongWords = append(wrongWords, snapshot)
		}
	}

	sort.Slice(wrongWords, func(i, j int) bool {
		left := wrongWords[i]
		right := wrongWords[j]
		if left.ReviewStats.WrongReviews != right.ReviewStats.WrongReviews {
			return left.ReviewStats.WrongReviews > right.ReviewStats.WrongReviews
		}
		if !sameNullableTime(left.ReviewStats.LastWrongAt, right.ReviewStats.LastWrongAt) {
			return nullableTimeAfter(left.ReviewStats.LastWrongAt, right.ReviewStats.LastWrongAt)
		}
		leftRank := sortableFrequencyRank(left.Word.FrequencyRank)
		rightRank := sortableFrequencyRank(right.Word.FrequencyRank)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return left.Word.ID < right.Word.ID
	})

	return wrongWords, nil
}

func (s *Store) DictionaryVersion(ctx context.Context) (string, error) {
	version, err := s.metaValue(ctx, "dict_version")
	if err != nil {
		return "", fmt.Errorf("load dictionary version: %w", err)
	}
	return version, nil
}

func scanExportWordSnapshots(rows *sql.Rows) ([]ExportWordSnapshot, error) {
	snapshots := make([]ExportWordSnapshot, 0, 16)
	for rows.Next() {
		snapshot, err := scanExportWordSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export word snapshots: %w", err)
	}
	return snapshots, nil
}

func scanExportWordSnapshot(scanner rowScanner) (ExportWordSnapshot, error) {
	var snapshot ExportWordSnapshot
	var (
		pos             sql.NullString
		level           sql.NullString
		frequencyRank   sql.NullInt64
		distractorGroup sql.NullString
		exampleEN       sql.NullString
		exampleJA       sql.NullString
		source          string
		createdAt       string
		state           sql.NullString
		dueAt           sql.NullString
		intervalDays    sql.NullFloat64
		easeFactor      sql.NullFloat64
		lastSeenAt      sql.NullString
		streakCorrect   sql.NullInt64
		totalCorrect    sql.NullInt64
		totalWrong      sql.NullInt64
		lapses          sql.NullInt64
		lastAnsweredAt  sql.NullString
		lastWrongAt     sql.NullString
		lastCorrectAt   sql.NullString
	)

	if err := scanner.Scan(
		&snapshot.Word.ID,
		&snapshot.Word.Lemma,
		&pos,
		&snapshot.Word.MeaningJA,
		&level,
		&frequencyRank,
		&distractorGroup,
		&exampleEN,
		&exampleJA,
		&source,
		&createdAt,
		&state,
		&dueAt,
		&intervalDays,
		&easeFactor,
		&lastSeenAt,
		&streakCorrect,
		&totalCorrect,
		&totalWrong,
		&lapses,
		&snapshot.ReviewStats.TotalReviews,
		&snapshot.ReviewStats.CorrectReviews,
		&snapshot.ReviewStats.WrongReviews,
		&lastAnsweredAt,
		&lastWrongAt,
		&lastCorrectAt,
	); err != nil {
		return ExportWordSnapshot{}, err
	}

	snapshot.Word.Pos = pos.String
	snapshot.Word.Level = level.String
	snapshot.Word.FrequencyRank = int(frequencyRank.Int64)
	snapshot.Word.DistractorGroup = distractorGroup.String
	snapshot.Word.ExampleEN = exampleEN.String
	snapshot.Word.ExampleJA = exampleJA.String
	snapshot.Word.Source = source

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return ExportWordSnapshot{}, fmt.Errorf("parse export word created_at: %w", err)
	}
	snapshot.Word.CreatedAt = parsedCreatedAt

	snapshot.Progress = srs.DefaultProgress()
	if state.Valid && state.String != "" {
		snapshot.Progress.State = state.String
	}
	if dueAt.Valid {
		parsedDueAt, err := parseTime(dueAt.String)
		if err != nil {
			return ExportWordSnapshot{}, fmt.Errorf("parse export progress due_at: %w", err)
		}
		snapshot.Progress.DueAt = &parsedDueAt
	}
	if intervalDays.Valid {
		snapshot.Progress.IntervalDays = intervalDays.Float64
	}
	if easeFactor.Valid && easeFactor.Float64 != 0 {
		snapshot.Progress.EaseFactor = easeFactor.Float64
	}
	if lastSeenAt.Valid {
		parsedLastSeenAt, err := parseTime(lastSeenAt.String)
		if err != nil {
			return ExportWordSnapshot{}, fmt.Errorf("parse export progress last_seen_at: %w", err)
		}
		snapshot.Progress.LastSeenAt = &parsedLastSeenAt
	}
	if streakCorrect.Valid {
		snapshot.Progress.StreakCorrect = int(streakCorrect.Int64)
	}
	if totalCorrect.Valid {
		snapshot.Progress.TotalCorrect = int(totalCorrect.Int64)
	}
	if totalWrong.Valid {
		snapshot.Progress.TotalWrong = int(totalWrong.Int64)
	}
	if lapses.Valid {
		snapshot.Progress.Lapses = int(lapses.Int64)
	}

	snapshot.ReviewStats.LastAnsweredAt, err = parseNullableExportTime(lastAnsweredAt, "last_answered_at")
	if err != nil {
		return ExportWordSnapshot{}, err
	}
	snapshot.ReviewStats.LastWrongAt, err = parseNullableExportTime(lastWrongAt, "last_wrong_at")
	if err != nil {
		return ExportWordSnapshot{}, err
	}
	snapshot.ReviewStats.LastCorrectAt, err = parseNullableExportTime(lastCorrectAt, "last_correct_at")
	if err != nil {
		return ExportWordSnapshot{}, err
	}

	return snapshot, nil
}

func parseNullableExportTime(raw sql.NullString, field string) (*time.Time, error) {
	if !raw.Valid {
		return nil, nil
	}
	parsed, err := parseTime(raw.String)
	if err != nil {
		return nil, fmt.Errorf("parse export %s: %w", field, err)
	}
	return &parsed, nil
}

func sameNullableTime(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

func nullableTimeAfter(left, right *time.Time) bool {
	switch {
	case left == nil:
		return false
	case right == nil:
		return true
	default:
		return left.After(*right)
	}
}

func sortableFrequencyRank(rank int) int {
	if rank <= 0 {
		return 999999
	}
	return rank
}
