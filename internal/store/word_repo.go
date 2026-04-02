package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/stats"
)

func (s *Store) LoadHomeSnapshot(ctx context.Context) (HomeSnapshot, error) {
	dueCount, err := s.countDueWords(ctx)
	if err != nil {
		return HomeSnapshot{}, err
	}

	newCount, err := s.countNewWords(ctx)
	if err != nil {
		return HomeSnapshot{}, err
	}

	streakDays, err := s.countStreakDays(ctx)
	if err != nil {
		return HomeSnapshot{}, err
	}

	activeSession, err := s.loadActiveSession(ctx)
	if err != nil {
		return HomeSnapshot{}, err
	}

	return HomeSnapshot{
		DueCount:      dueCount,
		NewCount:      newCount,
		StreakDays:    streakDays,
		ActiveSession: activeSession,
	}, nil
}

func (s *Store) LoadStatsSnapshot(ctx context.Context) (stats.Snapshot, error) {
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	today, err := s.reviewWindow(ctx, &todayStart)
	if err != nil {
		return stats.Snapshot{}, err
	}
	sevenDaysStart := now.AddDate(0, 0, -7)
	sevenDays, err := s.reviewWindow(ctx, &sevenDaysStart)
	if err != nil {
		return stats.Snapshot{}, err
	}
	thirtyDaysStart := now.AddDate(0, 0, -30)
	thirtyDays, err := s.reviewWindow(ctx, &thirtyDaysStart)
	if err != nil {
		return stats.Snapshot{}, err
	}
	total, err := s.reviewWindow(ctx, nil)
	if err != nil {
		return stats.Snapshot{}, err
	}

	dueCount, err := s.countDueWords(ctx)
	if err != nil {
		return stats.Snapshot{}, err
	}
	newCount, err := s.countNewWords(ctx)
	if err != nil {
		return stats.Snapshot{}, err
	}
	streakDays, err := s.countStreakDays(ctx)
	if err != nil {
		return stats.Snapshot{}, err
	}

	return stats.Snapshot{
		Today: stats.Window{
			Label:       "Today",
			Reviews:     today.reviews,
			Correct:     today.correct,
			WaitMinutes: waitMinutesFromResponseMS(today.responseMS),
		},
		SevenDays: stats.Window{
			Label:       "7 days",
			Reviews:     sevenDays.reviews,
			Correct:     sevenDays.correct,
			WaitMinutes: waitMinutesFromResponseMS(sevenDays.responseMS),
		},
		ThirtyDays: stats.Window{
			Label:       "30 days",
			Reviews:     thirtyDays.reviews,
			Correct:     thirtyDays.correct,
			WaitMinutes: waitMinutesFromResponseMS(thirtyDays.responseMS),
		},
		Total: stats.Window{
			Label:       "Total",
			Reviews:     total.reviews,
			Correct:     total.correct,
			WaitMinutes: waitMinutesFromResponseMS(total.responseMS),
		},
		DueCount:   dueCount,
		NewCount:   newCount,
		StreakDays: streakDays,
	}, nil
}

func (s *Store) ListDueWords(ctx context.Context, limit int) ([]Word, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT w.id, w.lemma, w.pos, w.meaning_ja, w.level, w.frequency_rank,
       w.distractor_group, w.example_en, w.example_ja, w.source, w.created_at
FROM words w
JOIN progress p ON p.word_id = w.id
WHERE p.due_at IS NOT NULL AND p.due_at <= ?
ORDER BY p.due_at ASC, COALESCE(w.frequency_rank, 999999) ASC, w.id ASC
LIMIT ?
`, formatTime(time.Now().UTC()), limit)
	if err != nil {
		return nil, fmt.Errorf("query due words: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanWords(rows)
}

func (s *Store) ListNewWords(ctx context.Context, limit int, excludeIDs []int64) ([]Word, error) {
	query := `
SELECT w.id, w.lemma, w.pos, w.meaning_ja, w.level, w.frequency_rank,
       w.distractor_group, w.example_en, w.example_ja, w.source, w.created_at
FROM words w
LEFT JOIN progress p ON p.word_id = w.id
WHERE (p.word_id IS NULL OR p.state = 'new')
`
	args := make([]any, 0, len(excludeIDs)+1)
	if len(excludeIDs) > 0 {
		query += " AND w.id NOT IN (" + placeholders(len(excludeIDs)) + ")"
		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}
	query += ` ORDER BY COALESCE(w.frequency_rank, 999999) ASC, w.id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query new words: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanWords(rows)
}

func (s *Store) ListWriteBasicCandidates(ctx context.Context, limit int, excludeIDs []int64) ([]Word, error) {
	query := `
SELECT id, lemma, pos, meaning_ja, level, frequency_rank,
       distractor_group, example_en, example_ja, source, created_at
FROM (
	SELECT w.id, w.lemma, w.pos, w.meaning_ja, w.level, w.frequency_rank,
	       w.distractor_group, w.example_en, w.example_ja, w.source, w.created_at,
	       CASE WHEN EXISTS (
	           SELECT 1
	           FROM reviews r
	           WHERE r.word_id = w.id AND r.answer_mode = ?
	       ) THEN 1 ELSE 0 END AS write_seen
	FROM words w
	WHERE EXISTS (
		SELECT 1
		FROM reviews r
		WHERE r.word_id = w.id AND r.answer_mode = ?
	)
	AND NOT EXISTS (
		SELECT 1
		FROM progress p
		WHERE p.word_id = w.id AND p.due_at IS NOT NULL AND p.due_at <= ?
	)
)
WHERE 1 = 1
`
	args := make([]any, 0, len(excludeIDs)+4)
	args = append(args, AnswerModeWrite, AnswerModeChoice, formatTime(time.Now().UTC()))
	if len(excludeIDs) > 0 {
		query += " AND id NOT IN (" + placeholders(len(excludeIDs)) + ")"
		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}
	query += ` ORDER BY write_seen ASC, COALESCE(frequency_rank, 999999) ASC, id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query write basic candidates: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanWords(rows)
}

func (s *Store) GetWord(ctx context.Context, wordID int64) (Word, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, lemma, pos, meaning_ja, level, frequency_rank,
       distractor_group, example_en, example_ja, source, created_at
FROM words
WHERE id = ?
`, wordID)

	word, err := scanWord(row)
	if err != nil {
		return Word{}, fmt.Errorf("get word %d: %w", wordID, err)
	}
	return word, nil
}

func (s *Store) ListWordsByPOS(ctx context.Context, pos string, limit int, excludeIDs []int64) ([]Word, error) {
	query := `
SELECT id, lemma, pos, meaning_ja, level, frequency_rank,
       distractor_group, example_en, example_ja, source, created_at
FROM words
WHERE pos = ?
`
	args := make([]any, 0, len(excludeIDs)+2)
	args = append(args, pos)
	if len(excludeIDs) > 0 {
		query += " AND id NOT IN (" + placeholders(len(excludeIDs)) + ")"
		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}
	query += ` ORDER BY COALESCE(frequency_rank, 999999) ASC, id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query words by pos %q: %w", pos, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanWords(rows)
}

func (s *Store) ListDistractorCandidates(ctx context.Context, correct Word, limit int, excludeIDs []int64) ([]Word, error) {
	query := `
SELECT id, lemma, pos, meaning_ja, level, frequency_rank,
       distractor_group, example_en, example_ja, source, created_at
FROM words
WHERE pos = ?
  AND meaning_ja != ?
`
	args := make([]any, 0, len(excludeIDs)+8)
	args = append(args, correct.Pos, correct.MeaningJA)
	if len(excludeIDs) > 0 {
		query += " AND id NOT IN (" + placeholders(len(excludeIDs)) + ")"
		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}
	query += `
ORDER BY
  CASE
    WHEN ? <> '' AND COALESCE(distractor_group, '') = ? THEN 0
    ELSE 1
  END,
  CASE
    WHEN ? <> '' AND COALESCE(level, '') = ? THEN 0
    ELSE 1
  END,
  ABS(COALESCE(frequency_rank, 999999) - ?) ASC,
  COALESCE(frequency_rank, 999999) ASC,
  id ASC
LIMIT ?
`
	args = append(
		args,
		correct.DistractorGroup,
		correct.DistractorGroup,
		correct.Level,
		correct.Level,
		correct.FrequencyRank,
		limit,
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query distractor candidates for %q: %w", correct.Lemma, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanWords(rows)
}

func (s *Store) AbandonActiveSession(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE sessions
SET status = ?, finished_at = ?
WHERE status = ?
`, SessionStatusAbandoned, formatTime(time.Now().UTC()), SessionStatusActive)
	if err != nil {
		return fmt.Errorf("abandon active session: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, mode, answerMode string, items []SessionItemPlan) (SessionRecord, []SessionItem, error) {
	if len(items) == 0 {
		return SessionRecord{}, nil, fmt.Errorf("create session: no session items")
	}

	now := time.Now().UTC()
	record := SessionRecord{
		ID:                uuid.NewString(),
		StartedAt:         now,
		Mode:              mode,
		AnswerMode:        NormalizeAnswerMode(answerMode),
		TotalQuestions:    len(items),
		AnsweredQuestions: 0,
		Status:            SessionStatusActive,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionRecord{}, nil, fmt.Errorf("begin create session: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO sessions (id, started_at, mode, answer_mode, total_questions, answered_questions, status)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, record.ID, formatTime(record.StartedAt), record.Mode, record.AnswerMode, record.TotalQuestions, record.AnsweredQuestions, record.Status); err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("insert session: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO session_items (session_id, ordinal, word_id, kind, status, source_ordinal)
VALUES (?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("prepare session items: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	createdItems := make([]SessionItem, 0, len(items))
	for i, item := range items {
		ordinal := i + 1
		var sourceOrdinal any
		if item.SourceOrdinal > 0 {
			sourceOrdinal = item.SourceOrdinal
		}
		if _, err := stmt.ExecContext(ctx, record.ID, ordinal, item.WordID, item.Kind, ItemStatusPending, sourceOrdinal); err != nil {
			_ = tx.Rollback()
			return SessionRecord{}, nil, fmt.Errorf("insert session item %d: %w", ordinal, err)
		}

		createdItems = append(createdItems, SessionItem{
			SessionID: record.ID,
			Ordinal:   ordinal,
			WordID:    item.WordID,
			Kind:      item.Kind,
			Status:    ItemStatusPending,
			CreatedAt: now,
		})
	}

	if err := tx.Commit(); err != nil {
		return SessionRecord{}, nil, fmt.Errorf("commit create session: %w", err)
	}

	return record, createdItems, nil
}

func (s *Store) LoadActiveRuntime(ctx context.Context) (*SessionRecord, []SessionItem, error) {
	record, err := s.loadActiveSession(ctx)
	if err != nil {
		return nil, nil, err
	}
	if record == nil {
		return nil, nil, nil
	}

	items, err := s.LoadSessionItems(ctx, record.ID)
	if err != nil {
		return nil, nil, err
	}

	return record, items, nil
}

func (s *Store) LoadSession(ctx context.Context, sessionID string) (SessionRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, started_at, finished_at, mode, answer_mode, total_questions, answered_questions, status
FROM sessions
WHERE id = ?
`, sessionID)

	record, err := scanSessionRecord(row)
	if err != nil {
		return SessionRecord{}, fmt.Errorf("load session %s: %w", sessionID, err)
	}
	return record, nil
}

func (s *Store) LoadSessionItems(ctx context.Context, sessionID string) ([]SessionItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT session_id, ordinal, word_id, kind, status, source_ordinal, answered_review_id, created_at
FROM session_items
WHERE session_id = ?
ORDER BY ordinal ASC
`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session items %s: %w", sessionID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]SessionItem, 0, 16)
	for rows.Next() {
		var item SessionItem
		var sourceOrdinal sql.NullInt64
		var answeredReviewID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&item.SessionID, &item.Ordinal, &item.WordID, &item.Kind, &item.Status, &sourceOrdinal, &answeredReviewID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan session item: %w", err)
		}
		if sourceOrdinal.Valid {
			value := int(sourceOrdinal.Int64)
			item.SourceOrdinal = &value
		}
		if answeredReviewID.Valid {
			value := answeredReviewID.Int64
			item.AnsweredReviewID = &value
		}
		parsedCreatedAt, err := parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse session item created_at: %w", err)
		}
		item.CreatedAt = parsedCreatedAt
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session items: %w", err)
	}

	return items, nil
}

func (s *Store) SaveAnswer(ctx context.Context, event ReviewEvent) (SessionRecord, []SessionItem, error) {
	now := event.AnsweredAt.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionRecord{}, nil, fmt.Errorf("begin save answer: %w", err)
	}

	progress, err := loadProgressTx(ctx, tx, event.WordID)
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, err
	}
	updatedProgress := srs.Update(progress, event.Rating, now)
	if err := saveProgressTx(ctx, tx, event.WordID, updatedProgress); err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, err
	}

	reviewResult, err := tx.ExecContext(ctx, `
INSERT INTO reviews (
word_id, session_id, answered_at, answer_mode, selected_choice,
correct_choice, is_correct, response_ms, rating
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, event.WordID, event.SessionID, formatTime(now), NormalizeAnswerMode(event.AnswerMode), event.SelectedChoice, event.CorrectChoice, boolToInt(event.IsCorrect), event.ResponseMS, string(event.Rating))
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("insert review: %w", err)
	}

	reviewID, err := reviewResult.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("read review id: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE session_items
SET status = ?, answered_review_id = ?
WHERE session_id = ? AND ordinal = ?
`, ItemStatusAnswered, reviewID, event.SessionID, event.ItemOrdinal); err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("mark session item answered: %w", err)
	}

	totalQuestions, err := sessionItemCountTx(ctx, tx, event.SessionID)
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, err
	}

	if !event.IsCorrect && event.Kind != ItemKindRetry {
		hasRetry, err := sessionHasRetryTx(ctx, tx, event.SessionID, event.WordID)
		if err != nil {
			_ = tx.Rollback()
			return SessionRecord{}, nil, err
		}
		if !hasRetry {
			maxOrdinal, err := maxSessionOrdinalTx(ctx, tx, event.SessionID)
			if err != nil {
				_ = tx.Rollback()
				return SessionRecord{}, nil, err
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO session_items (session_id, ordinal, word_id, kind, status, source_ordinal)
VALUES (?, ?, ?, ?, ?, ?)
`, event.SessionID, maxOrdinal+1, event.WordID, ItemKindRetry, ItemStatusPending, event.ItemOrdinal); err != nil {
				_ = tx.Rollback()
				return SessionRecord{}, nil, fmt.Errorf("insert retry session item: %w", err)
			}
			totalQuestions++
		}
	}

	answeredQuestions, err := answeredSessionItemCountTx(ctx, tx, event.SessionID)
	if err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, err
	}

	status := SessionStatusActive
	var finishedAt any
	if answeredQuestions >= totalQuestions {
		status = SessionStatusCompleted
		finishedAt = formatTime(now)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE sessions
SET answered_questions = ?, total_questions = ?, status = ?, finished_at = ?
WHERE id = ?
`, answeredQuestions, totalQuestions, status, finishedAt, event.SessionID); err != nil {
		_ = tx.Rollback()
		return SessionRecord{}, nil, fmt.Errorf("update session progress: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SessionRecord{}, nil, fmt.Errorf("commit save answer: %w", err)
	}

	record, err := s.LoadSession(ctx, event.SessionID)
	if err != nil {
		return SessionRecord{}, nil, err
	}
	items, err := s.LoadSessionItems(ctx, event.SessionID)
	if err != nil {
		return SessionRecord{}, nil, err
	}

	return record, items, nil
}

func (s *Store) LoadSessionSummary(ctx context.Context, sessionID string) (SessionSummary, error) {
	record, err := s.LoadSession(ctx, sessionID)
	if err != nil {
		return SessionSummary{}, err
	}

	var correctAnswers int
	if err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(is_correct), 0)
FROM reviews
WHERE session_id = ?
`, sessionID).Scan(&correctAnswers); err != nil {
		return SessionSummary{}, fmt.Errorf("count correct answers: %w", err)
	}

	summary := SessionSummary{
		SessionID:      sessionID,
		TotalQuestions: record.TotalQuestions,
		CorrectAnswers: correctAnswers,
	}
	if summary.TotalQuestions > 0 {
		summary.Accuracy = float64(summary.CorrectAnswers) / float64(summary.TotalQuestions) * 100
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT kind, COUNT(*)
FROM session_items
WHERE session_id = ?
GROUP BY kind
`, sessionID)
	if err != nil {
		return SessionSummary{}, fmt.Errorf("count session item kinds: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			return SessionSummary{}, fmt.Errorf("scan session item kind count: %w", err)
		}
		switch kind {
		case ItemKindNew:
			summary.NewCount = count
		case ItemKindReview:
			summary.ReviewCount = count
		case ItemKindRetry:
			summary.RetryCount = count
		}
	}
	if err := rows.Err(); err != nil {
		return SessionSummary{}, fmt.Errorf("iterate session item kinds: %w", err)
	}

	hardRows, err := s.db.QueryContext(ctx, `
SELECT w.id, w.lemma, w.pos, w.meaning_ja, w.level, w.frequency_rank,
       w.distractor_group, w.example_en, w.example_ja, w.source, w.created_at
FROM reviews r
JOIN words w ON w.id = r.word_id
WHERE r.session_id = ? AND r.is_correct = 0
GROUP BY w.id
ORDER BY COUNT(*) DESC, MAX(r.answered_at) DESC
LIMIT 5
`, sessionID)
	if err != nil {
		return SessionSummary{}, fmt.Errorf("load hard words: %w", err)
	}
	defer func() {
		_ = hardRows.Close()
	}()

	hardWords, err := scanWords(hardRows)
	if err != nil {
		return SessionSummary{}, err
	}
	summary.HardWords = hardWords

	return summary, nil
}

type reviewCounts struct {
	reviews    int
	correct    int
	responseMS int64
}

func (s *Store) reviewWindow(ctx context.Context, since *time.Time) (reviewCounts, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(is_correct), 0), COALESCE(SUM(response_ms), 0) FROM reviews`
	args := make([]any, 0, 1)
	if since != nil {
		query += ` WHERE answered_at >= ?`
		args = append(args, formatTime(*since))
	}

	var result reviewCounts
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&result.reviews, &result.correct, &result.responseMS); err != nil {
		return reviewCounts{}, fmt.Errorf("query review window: %w", err)
	}
	return result, nil
}

func waitMinutesFromResponseMS(responseMS int64) float64 {
	return float64(responseMS) / 60000.0
}

func (s *Store) countDueWords(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM progress
WHERE due_at IS NOT NULL AND due_at <= ?
`, formatTime(time.Now().UTC())).Scan(&count); err != nil {
		return 0, fmt.Errorf("count due words: %w", err)
	}
	return count, nil
}

func (s *Store) countNewWords(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM words w
LEFT JOIN progress p ON p.word_id = w.id
WHERE p.word_id IS NULL OR p.state = 'new'
`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count new words: %w", err)
	}
	return count, nil
}

func (s *Store) countStreakDays(ctx context.Context) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT SUBSTR(answered_at, 1, 10) AS review_day
FROM reviews
ORDER BY review_day DESC
`)
	if err != nil {
		return 0, fmt.Errorf("query streak days: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	today := time.Now().UTC().Truncate(24 * time.Hour)
	expected := today
	streak := 0
	index := 0

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, fmt.Errorf("scan streak date: %w", err)
		}
		day, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return 0, fmt.Errorf("parse streak date %q: %w", raw, err)
		}
		day = day.UTC()

		if index == 0 && !day.Equal(today) {
			return 0, nil
		}
		if !day.Equal(expected) {
			break
		}

		streak++
		expected = expected.AddDate(0, 0, -1)
		index++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate streak dates: %w", err)
	}

	return streak, nil
}

func (s *Store) loadActiveSession(ctx context.Context) (*SessionRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, started_at, finished_at, mode, answer_mode, total_questions, answered_questions, status
FROM sessions
WHERE status = ?
ORDER BY started_at DESC
LIMIT 1
`, SessionStatusActive)

	record, err := scanSessionRecord(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active session: %w", err)
	}
	return &record, nil
}

func loadProgressTx(ctx context.Context, tx *sql.Tx, wordID int64) (Progress, error) {
	row := tx.QueryRowContext(ctx, `
SELECT state, due_at, interval_days, ease_factor, last_seen_at,
       streak_correct, total_correct, total_wrong, lapses
FROM progress
WHERE word_id = ?
`, wordID)

	var progress Progress
	var dueAt sql.NullString
	var lastSeenAt sql.NullString
	err := row.Scan(
		&progress.State,
		&dueAt,
		&progress.IntervalDays,
		&progress.EaseFactor,
		&lastSeenAt,
		&progress.StreakCorrect,
		&progress.TotalCorrect,
		&progress.TotalWrong,
		&progress.Lapses,
	)
	if err == sql.ErrNoRows {
		return srs.DefaultProgress(), nil
	}
	if err != nil {
		return Progress{}, fmt.Errorf("load progress for word %d: %w", wordID, err)
	}
	if dueAt.Valid {
		parsed, err := parseTime(dueAt.String)
		if err != nil {
			return Progress{}, fmt.Errorf("parse progress due_at: %w", err)
		}
		progress.DueAt = &parsed
	}
	if lastSeenAt.Valid {
		parsed, err := parseTime(lastSeenAt.String)
		if err != nil {
			return Progress{}, fmt.Errorf("parse progress last_seen_at: %w", err)
		}
		progress.LastSeenAt = &parsed
	}

	return progress, nil
}

func saveProgressTx(ctx context.Context, tx *sql.Tx, wordID int64, progress Progress) error {
	var dueAt any
	if progress.DueAt != nil {
		dueAt = formatTime(*progress.DueAt)
	}
	var lastSeenAt any
	if progress.LastSeenAt != nil {
		lastSeenAt = formatTime(*progress.LastSeenAt)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO progress (
word_id, state, due_at, interval_days, ease_factor,
last_seen_at, streak_correct, total_correct, total_wrong, lapses
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(word_id) DO UPDATE SET
state = excluded.state,
due_at = excluded.due_at,
interval_days = excluded.interval_days,
ease_factor = excluded.ease_factor,
last_seen_at = excluded.last_seen_at,
streak_correct = excluded.streak_correct,
total_correct = excluded.total_correct,
total_wrong = excluded.total_wrong,
lapses = excluded.lapses
`, wordID, progress.State, dueAt, progress.IntervalDays, progress.EaseFactor, lastSeenAt, progress.StreakCorrect, progress.TotalCorrect, progress.TotalWrong, progress.Lapses); err != nil {
		return fmt.Errorf("save progress for word %d: %w", wordID, err)
	}
	return nil
}

func sessionItemCountTx(ctx context.Context, tx *sql.Tx, sessionID string) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM session_items WHERE session_id = ?`, sessionID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count session items: %w", err)
	}
	return count, nil
}

func answeredSessionItemCountTx(ctx context.Context, tx *sql.Tx, sessionID string) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM session_items
WHERE session_id = ? AND status = ?
`, sessionID, ItemStatusAnswered).Scan(&count); err != nil {
		return 0, fmt.Errorf("count answered session items: %w", err)
	}
	return count, nil
}

func sessionHasRetryTx(ctx context.Context, tx *sql.Tx, sessionID string, wordID int64) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM session_items
WHERE session_id = ? AND word_id = ? AND kind = ?
`, sessionID, wordID, ItemKindRetry).Scan(&count); err != nil {
		return false, fmt.Errorf("check retry session item: %w", err)
	}
	return count > 0, nil
}

func maxSessionOrdinalTx(ctx context.Context, tx *sql.Tx, sessionID string) (int, error) {
	var ordinal int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal), 0) FROM session_items WHERE session_id = ?`, sessionID).Scan(&ordinal); err != nil {
		return 0, fmt.Errorf("max session ordinal: %w", err)
	}
	return ordinal, nil
}

func scanWords(rows *sql.Rows) ([]Word, error) {
	words := make([]Word, 0, 16)
	for rows.Next() {
		word, err := scanWord(rows)
		if err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate words: %w", err)
	}
	return words, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWord(scanner rowScanner) (Word, error) {
	var word Word
	var pos sql.NullString
	var level sql.NullString
	var frequencyRank sql.NullInt64
	var distractorGroup sql.NullString
	var exampleEN sql.NullString
	var exampleJA sql.NullString
	var source string
	var createdAt string

	if err := scanner.Scan(
		&word.ID,
		&word.Lemma,
		&pos,
		&word.MeaningJA,
		&level,
		&frequencyRank,
		&distractorGroup,
		&exampleEN,
		&exampleJA,
		&source,
		&createdAt,
	); err != nil {
		return Word{}, err
	}

	word.Pos = pos.String
	word.Level = level.String
	word.FrequencyRank = int(frequencyRank.Int64)
	word.DistractorGroup = distractorGroup.String
	word.ExampleEN = exampleEN.String
	word.ExampleJA = exampleJA.String
	word.Source = source
	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return Word{}, fmt.Errorf("parse word created_at: %w", err)
	}
	word.CreatedAt = parsedCreatedAt
	return word, nil
}

func scanSessionRecord(scanner rowScanner) (SessionRecord, error) {
	var record SessionRecord
	var startedAt string
	var finishedAt sql.NullString
	if err := scanner.Scan(&record.ID, &startedAt, &finishedAt, &record.Mode, &record.AnswerMode, &record.TotalQuestions, &record.AnsweredQuestions, &record.Status); err != nil {
		return SessionRecord{}, err
	}
	record.AnswerMode = NormalizeAnswerMode(record.AnswerMode)

	parsedStartedAt, err := parseTime(startedAt)
	if err != nil {
		return SessionRecord{}, fmt.Errorf("parse session started_at: %w", err)
	}
	record.StartedAt = parsedStartedAt
	if finishedAt.Valid {
		parsedFinishedAt, err := parseTime(finishedAt.String)
		if err != nil {
			return SessionRecord{}, fmt.Errorf("parse session finished_at: %w", err)
		}
		record.FinishedAt = &parsedFinishedAt
	}

	return record, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
