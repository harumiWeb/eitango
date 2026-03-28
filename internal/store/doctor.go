package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/harumiWeb/eitango/internal/dict"
)

const (
	doctorSampleLimit    = 5
	doctorQuizChoiceSize = 4
	doctorQuizPoolLimit  = 64
)

type DiagnosticStatus string

const (
	DiagnosticStatusOK      DiagnosticStatus = "ok"
	DiagnosticStatusWarning DiagnosticStatus = "warning"
	DiagnosticStatusError   DiagnosticStatus = "error"
)

type DiagnosticCheck struct {
	Name    string
	Status  DiagnosticStatus
	Summary string
	Details []string
}

type DiagnosticReport struct {
	Checks []DiagnosticCheck
}

func (r DiagnosticReport) Check(name string) (DiagnosticCheck, bool) {
	for _, check := range r.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func (r DiagnosticReport) WarningCount() int {
	count := 0
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusWarning {
			count++
		}
	}
	return count
}

func (r DiagnosticReport) ErrorCount() int {
	count := 0
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusError {
			count++
		}
	}
	return count
}

func (r DiagnosticReport) HasIssues() bool {
	return r.WarningCount() > 0 || r.ErrorCount() > 0
}

func (s *Store) RunDiagnostics(ctx context.Context) DiagnosticReport {
	return DiagnosticReport{
		Checks: []DiagnosticCheck{
			s.checkDatabase(ctx),
			s.checkMigrations(ctx),
			s.checkDictionary(ctx),
			s.checkWordSources(ctx),
			s.checkWordMetadata(ctx),
			s.checkOrphanProgress(ctx),
			s.checkOrphanReviews(ctx),
			s.checkOrphanSessionItems(ctx),
			s.checkActiveSessions(ctx),
			s.checkQuizability(ctx),
		},
	}
}

func (s *Store) checkDatabase(ctx context.Context) DiagnosticCheck {
	var value int
	if err := s.db.QueryRowContext(ctx, `SELECT 1`).Scan(&value); err != nil {
		return diagnosticCheckError("database", "database is not readable", err.Error())
	}
	return diagnosticCheckOK("database", "database is readable")
}

func (s *Store) checkMigrations(ctx context.Context) DiagnosticCheck {
	expected, err := embeddedMigrationNames()
	if err != nil {
		return diagnosticCheckError("migrations", "embedded migration list is unavailable", err.Error())
	}

	rows, err := s.db.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return diagnosticCheckError("migrations", "schema_migrations is not readable", err.Error())
	}
	defer func() {
		_ = rows.Close()
	}()

	applied := make(map[string]struct{}, len(expected))
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return diagnosticCheckError("migrations", "schema_migrations could not be scanned", err.Error())
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return diagnosticCheckError("migrations", "schema_migrations could not be iterated", err.Error())
	}

	expectedSet := make(map[string]struct{}, len(expected))
	missing := make([]string, 0)
	for _, version := range expected {
		expectedSet[version] = struct{}{}
		if _, ok := applied[version]; !ok {
			missing = append(missing, version)
		}
	}

	extra := make([]string, 0)
	for version := range applied {
		if _, ok := expectedSet[version]; !ok {
			extra = append(extra, version)
		}
	}
	sort.Strings(extra)

	if len(missing) > 0 {
		details := []string{formatStringSamples("missing", len(missing), missing)}
		if len(extra) > 0 {
			details = append(details, formatStringSamples("unexpected", len(extra), extra))
		}
		return diagnosticCheckError("migrations", fmt.Sprintf("%d embedded migration(s) are missing", len(missing)), details...)
	}
	if len(extra) > 0 {
		return diagnosticCheckWarning("migrations", fmt.Sprintf("%d unexpected migration(s) are recorded", len(extra)), formatStringSamples("unexpected", len(extra), extra))
	}

	return diagnosticCheckOK("migrations", fmt.Sprintf("%d/%d embedded migrations are recorded", len(expected), len(expected)))
}

func (s *Store) checkDictionary(ctx context.Context) DiagnosticCheck {
	version, err := s.metaValue(ctx, "dict_version")
	if err != nil {
		return diagnosticCheckError("dictionary", "dict_version could not be read", err.Error())
	}

	wordCount, err := s.wordCount(ctx)
	if err != nil {
		return diagnosticCheckError("dictionary", "word count could not be read", err.Error())
	}
	coreWordCount, err := s.countWordsBySource(ctx, WordSourceCore)
	if err != nil {
		return diagnosticCheckError("dictionary", "core word count could not be read", err.Error())
	}
	importWordCount := wordCount - coreWordCount

	switch {
	case version == "" && coreWordCount == 0:
		return diagnosticCheckError("dictionary", "core words are not seeded", fmt.Sprintf("expected dict_version %q", dict.CoreWordsVersion))
	case version == "":
		return diagnosticCheckError("dictionary", "dict_version is missing", fmt.Sprintf("core words: %d", coreWordCount), fmt.Sprintf("imported words: %d", importWordCount))
	case coreWordCount == 0:
		return diagnosticCheckError("dictionary", "dict_version exists but core words are missing", fmt.Sprintf("dict_version: %s", version), fmt.Sprintf("imported words: %d", importWordCount))
	case version != dict.CoreWordsVersion:
		details := []string{fmt.Sprintf("core words: %d", coreWordCount)}
		if importWordCount > 0 {
			details = append(details, fmt.Sprintf("imported words: %d", importWordCount))
		}
		return diagnosticCheckWarning(
			"dictionary",
			fmt.Sprintf("dict_version is %q but embedded core words are %q", version, dict.CoreWordsVersion),
			details...,
		)
	default:
		summary := fmt.Sprintf("%d core words seeded at %s", coreWordCount, version)
		if importWordCount > 0 {
			summary += fmt.Sprintf(" (+%d imported)", importWordCount)
		}
		return diagnosticCheckOK("dictionary", summary)
	}
}

func (s *Store) checkWordSources(ctx context.Context) DiagnosticCheck {
	missingSourceCount, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM words
WHERE TRIM(COALESCE(source, '')) = ''
`)
	if err != nil {
		return diagnosticCheckError("word sources", "word sources could not be checked", err.Error())
	}

	duplicateCount, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM (
  SELECT lemma, IFNULL(pos, '') AS pos_key
  FROM words
  GROUP BY lemma, pos_key
  HAVING COUNT(DISTINCT source) > 1
)
`)
	if err != nil {
		return diagnosticCheckError("word sources", "cross-source duplicates could not be counted", err.Error())
	}
	sameSourceDuplicateCount, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM (
  SELECT source, lemma, IFNULL(pos, '') AS pos_key
  FROM words
  GROUP BY source, lemma, pos_key
  HAVING COUNT(*) > 1
)
`)
	if err != nil {
		return diagnosticCheckError("word sources", "same-source duplicates could not be counted", err.Error())
	}

	details := make([]string, 0, 3)
	if missingSourceCount > 0 {
		rows, err := s.db.QueryContext(ctx, `
SELECT id, lemma
FROM words
WHERE TRIM(COALESCE(source, '')) = ''
ORDER BY id ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("word sources", "missing word sources were found but samples could not be loaded", err.Error())
		}
		samples := make([]string, 0, doctorSampleLimit)
		for rows.Next() {
			var id int64
			var lemma string
			if err := rows.Scan(&id, &lemma); err != nil {
				_ = rows.Close()
				return diagnosticCheckError("word sources", "missing word source samples could not be scanned", err.Error())
			}
			samples = append(samples, fmt.Sprintf("%d:%s", id, lemma))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return diagnosticCheckError("word sources", "missing word source samples could not be iterated", err.Error())
		}
		_ = rows.Close()
		details = append(details, formatStringSamples("missing source rows", missingSourceCount, samples))
	}

	if sameSourceDuplicateCount > 0 {
		rows, err := s.db.QueryContext(ctx, `
SELECT source, lemma, IFNULL(pos, ''), COUNT(*)
FROM words
GROUP BY source, lemma, IFNULL(pos, '')
HAVING COUNT(*) > 1
ORDER BY source ASC, lemma ASC, IFNULL(pos, '') ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("word sources", "same-source duplicate samples could not be loaded", err.Error())
		}
		samples := make([]string, 0, doctorSampleLimit)
		for rows.Next() {
			var source string
			var lemma string
			var pos string
			var count int
			if err := rows.Scan(&source, &lemma, &pos, &count); err != nil {
				_ = rows.Close()
				return diagnosticCheckError("word sources", "same-source duplicate samples could not be scanned", err.Error())
			}
			if pos == "" {
				pos = "no-pos"
			}
			samples = append(samples, fmt.Sprintf("%s -> %s [%s] x%d", source, lemma, pos, count))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return diagnosticCheckError("word sources", "same-source duplicate samples could not be iterated", err.Error())
		}
		_ = rows.Close()
		details = append(details, formatStringSamples("same-source duplicates", sameSourceDuplicateCount, samples))
	}

	if duplicateCount > 0 {
		rows, err := s.db.QueryContext(ctx, `
SELECT lemma, IFNULL(pos, ''), GROUP_CONCAT(DISTINCT source)
FROM words
GROUP BY lemma, IFNULL(pos, '')
HAVING COUNT(DISTINCT source) > 1
ORDER BY lemma ASC, IFNULL(pos, '') ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("word sources", "cross-source duplicate samples could not be loaded", err.Error())
		}
		samples := make([]string, 0, doctorSampleLimit)
		for rows.Next() {
			var lemma string
			var pos string
			var sources string
			if err := rows.Scan(&lemma, &pos, &sources); err != nil {
				_ = rows.Close()
				return diagnosticCheckError("word sources", "cross-source duplicate samples could not be scanned", err.Error())
			}
			if pos == "" {
				pos = "no-pos"
			}
			samples = append(samples, fmt.Sprintf("%s [%s] -> %s", lemma, pos, sources))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return diagnosticCheckError("word sources", "cross-source duplicate samples could not be iterated", err.Error())
		}
		_ = rows.Close()
		details = append(details, formatStringSamples("cross-source duplicates", duplicateCount, samples))
	}

	switch {
	case missingSourceCount > 0:
		return diagnosticCheckError("word sources", fmt.Sprintf("%d word row(s) are missing a source", missingSourceCount), details...)
	case sameSourceDuplicateCount > 0:
		return diagnosticCheckError("word sources", fmt.Sprintf("%d lemma/pos pair(s) are duplicated within a source", sameSourceDuplicateCount), details...)
	case duplicateCount > 0:
		return diagnosticCheckWarning("word sources", fmt.Sprintf("%d lemma/pos pair(s) appear in multiple sources", duplicateCount), details...)
	default:
		return diagnosticCheckOK("word sources", "all words have sources and no duplicate lemma/pos pairs were found")
	}
}

func (s *Store) checkWordMetadata(ctx context.Context) DiagnosticCheck {
	type metadataIssue struct {
		label string
		count int
		query string
	}

	issues := []metadataIssue{
		{
			label: "missing pos",
			query: `
SELECT COUNT(*)
FROM words
WHERE TRIM(COALESCE(pos, '')) = ''
`,
		},
		{
			label: "missing level",
			query: `
SELECT COUNT(*)
FROM words
WHERE TRIM(COALESCE(level, '')) = ''
`,
		},
		{
			label: "missing frequency rank",
			query: `
SELECT COUNT(*)
FROM words
WHERE COALESCE(frequency_rank, 0) <= 0
`,
		},
		{
			label: "missing distractor group",
			query: `
SELECT COUNT(*)
FROM words
WHERE TRIM(COALESCE(distractor_group, '')) = ''
`,
		},
	}

	details := make([]string, 0, len(issues)+1)
	totalIssueCount := 0

	for i := range issues {
		count, err := s.countRows(ctx, issues[i].query)
		if err != nil {
			return diagnosticCheckError("word metadata", fmt.Sprintf("%s could not be counted", issues[i].label), err.Error())
		}
		issues[i].count = count
		if count == 0 {
			continue
		}
		totalIssueCount += count

		samples, err := s.sampleStringRows(ctx, fmt.Sprintf(`
SELECT lemma
FROM words
WHERE %s
ORDER BY id ASC
LIMIT ?
`, metadataConditionForLabel(issues[i].label)), doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("word metadata", fmt.Sprintf("%s samples could not be loaded", issues[i].label), err.Error())
		}
		details = append(details, formatStringSamples(issues[i].label, count, samples))
	}

	duplicateRankCount, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM (
  SELECT source, frequency_rank
  FROM words
  WHERE frequency_rank IS NOT NULL
  GROUP BY source, frequency_rank
  HAVING COUNT(*) > 1
)
`)
	if err != nil {
		return diagnosticCheckError("word metadata", "duplicate frequency ranks could not be counted", err.Error())
	}
	if duplicateRankCount > 0 {
		totalIssueCount += duplicateRankCount
		rows, err := s.db.QueryContext(ctx, `
SELECT source, frequency_rank, GROUP_CONCAT(lemma, ', ')
FROM (
  SELECT source, frequency_rank, lemma
  FROM words
  WHERE frequency_rank IS NOT NULL
  ORDER BY source ASC, frequency_rank ASC, lemma ASC
)
GROUP BY source, frequency_rank
HAVING COUNT(*) > 1
ORDER BY source ASC, frequency_rank ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("word metadata", "duplicate frequency rank samples could not be loaded", err.Error())
		}
		samples := make([]string, 0, doctorSampleLimit)
		for rows.Next() {
			var (
				source string
				rank   int
				lemmas string
			)
			if err := rows.Scan(&source, &rank, &lemmas); err != nil {
				_ = rows.Close()
				return diagnosticCheckError("word metadata", "duplicate frequency rank samples could not be scanned", err.Error())
			}
			samples = append(samples, fmt.Sprintf("%s -> %d (%s)", source, rank, lemmas))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return diagnosticCheckError("word metadata", "duplicate frequency rank samples could not be iterated", err.Error())
		}
		_ = rows.Close()
		details = append(details, formatStringSamples("duplicate frequency ranks", duplicateRankCount, samples))
	}

	if totalIssueCount > 0 {
		return diagnosticCheckWarning("word metadata", fmt.Sprintf("%d metadata issue(s) affect ranking or distractors", totalIssueCount), details...)
	}
	return diagnosticCheckOK("word metadata", "all words have metadata needed for ranking and distractors")
}

func metadataConditionForLabel(label string) string {
	switch label {
	case "missing pos":
		return "TRIM(COALESCE(pos, '')) = ''"
	case "missing level":
		return "TRIM(COALESCE(level, '')) = ''"
	case "missing frequency rank":
		return "COALESCE(frequency_rank, 0) <= 0"
	case "missing distractor group":
		return "TRIM(COALESCE(distractor_group, '')) = ''"
	default:
		panic("unsupported metadata label: " + label)
	}
}

func (s *Store) checkOrphanProgress(ctx context.Context) DiagnosticCheck {
	count, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM progress p
LEFT JOIN words w ON w.id = p.word_id
WHERE w.id IS NULL
`)
	if err != nil {
		return diagnosticCheckError("orphan progress", "progress rows could not be checked", err.Error())
	}
	if count == 0 {
		return diagnosticCheckOK("orphan progress", "all progress rows reference existing words")
	}

	ids, err := s.sampleInt64Rows(ctx, `
SELECT p.word_id
FROM progress p
LEFT JOIN words w ON w.id = p.word_id
WHERE w.id IS NULL
ORDER BY p.word_id ASC
LIMIT ?
`, doctorSampleLimit)
	if err != nil {
		return diagnosticCheckError("orphan progress", "orphan progress rows were found but samples could not be loaded", err.Error())
	}

	return diagnosticCheckError("orphan progress", fmt.Sprintf("%d row(s) reference missing words", count), formatInt64Samples("word_ids", count, ids))
}

func (s *Store) checkOrphanReviews(ctx context.Context) DiagnosticCheck {
	missingWords, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM reviews r
LEFT JOIN words w ON w.id = r.word_id
WHERE w.id IS NULL
`)
	if err != nil {
		return diagnosticCheckError("orphan reviews", "review word references could not be checked", err.Error())
	}

	missingSessions, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM reviews r
LEFT JOIN sessions s ON s.id = r.session_id
WHERE s.id IS NULL
`)
	if err != nil {
		return diagnosticCheckError("orphan reviews", "review session references could not be checked", err.Error())
	}

	if missingWords == 0 && missingSessions == 0 {
		return diagnosticCheckOK("orphan reviews", "all review rows reference existing words and sessions")
	}

	details := make([]string, 0, 2)
	if missingWords > 0 {
		ids, err := s.sampleInt64Rows(ctx, `
SELECT r.word_id
FROM reviews r
LEFT JOIN words w ON w.id = r.word_id
WHERE w.id IS NULL
ORDER BY r.word_id ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("orphan reviews", "review word samples could not be loaded", err.Error())
		}
		details = append(details, formatInt64Samples("missing words", missingWords, ids))
	}
	if missingSessions > 0 {
		ids, err := s.sampleStringRows(ctx, `
SELECT r.session_id
FROM reviews r
LEFT JOIN sessions s ON s.id = r.session_id
WHERE s.id IS NULL
ORDER BY r.session_id ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("orphan reviews", "review session samples could not be loaded", err.Error())
		}
		details = append(details, formatStringSamples("missing sessions", missingSessions, ids))
	}

	return diagnosticCheckError(
		"orphan reviews",
		fmt.Sprintf("review references are broken (missing words: %d, missing sessions: %d)", missingWords, missingSessions),
		details...,
	)
}

func (s *Store) checkOrphanSessionItems(ctx context.Context) DiagnosticCheck {
	missingWords, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM session_items si
LEFT JOIN words w ON w.id = si.word_id
WHERE w.id IS NULL
`)
	if err != nil {
		return diagnosticCheckError("orphan session items", "session item word references could not be checked", err.Error())
	}

	missingSessions, err := s.countRows(ctx, `
SELECT COUNT(*)
FROM session_items si
LEFT JOIN sessions s ON s.id = si.session_id
WHERE s.id IS NULL
`)
	if err != nil {
		return diagnosticCheckError("orphan session items", "session item session references could not be checked", err.Error())
	}

	if missingWords == 0 && missingSessions == 0 {
		return diagnosticCheckOK("orphan session items", "all session items reference existing words and sessions")
	}

	details := make([]string, 0, 2)
	if missingWords > 0 {
		ids, err := s.sampleInt64Rows(ctx, `
SELECT si.word_id
FROM session_items si
LEFT JOIN words w ON w.id = si.word_id
WHERE w.id IS NULL
ORDER BY si.word_id ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("orphan session items", "session item word samples could not be loaded", err.Error())
		}
		details = append(details, formatInt64Samples("missing words", missingWords, ids))
	}
	if missingSessions > 0 {
		ids, err := s.sampleStringRows(ctx, `
SELECT si.session_id
FROM session_items si
LEFT JOIN sessions s ON s.id = si.session_id
WHERE s.id IS NULL
ORDER BY si.session_id ASC
LIMIT ?
`, doctorSampleLimit)
		if err != nil {
			return diagnosticCheckError("orphan session items", "session item session samples could not be loaded", err.Error())
		}
		details = append(details, formatStringSamples("missing sessions", missingSessions, ids))
	}

	return diagnosticCheckError(
		"orphan session items",
		fmt.Sprintf("session item references are broken (missing words: %d, missing sessions: %d)", missingWords, missingSessions),
		details...,
	)
}

func (s *Store) checkActiveSessions(ctx context.Context) DiagnosticCheck {
	type activeSessionSnapshot struct {
		id                string
		mode              string
		totalQuestions    int
		answeredQuestions int
		finishedAt        sql.NullString
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, mode, total_questions, answered_questions, finished_at
FROM sessions
WHERE status = ?
ORDER BY started_at ASC
`, SessionStatusActive)
	if err != nil {
		return diagnosticCheckError("active sessions", "active sessions could not be read", err.Error())
	}

	sessions := make([]activeSessionSnapshot, 0, doctorSampleLimit)
	ids := make([]string, 0, doctorSampleLimit)
	for rows.Next() {
		var session activeSessionSnapshot
		if err := rows.Scan(&session.id, &session.mode, &session.totalQuestions, &session.answeredQuestions, &session.finishedAt); err != nil {
			return diagnosticCheckError("active sessions", "active sessions could not be scanned", err.Error())
		}
		sessions = append(sessions, session)
		if len(ids) < doctorSampleLimit {
			ids = append(ids, session.id)
		}
	}
	if err := rows.Err(); err != nil {
		return diagnosticCheckError("active sessions", "active sessions could not be iterated", err.Error())
	}
	if err := rows.Close(); err != nil {
		return diagnosticCheckError("active sessions", "active sessions could not be closed", err.Error())
	}

	sessionCount := len(sessions)
	if sessionCount == 0 {
		return diagnosticCheckOK("active sessions", "no active sessions")
	}

	issues := make([]string, 0)
	for _, session := range sessions {
		var (
			itemCount          int
			answeredItemCount  int
			pendingItemCount   int
			missingReviewCount int
			staleReviewCount   int
			invalidStatusCount int
		)

		// Read the active session headers first so the result set is closed
		// before the per-session aggregate query runs on our single SQLite connection.
		if err := s.db.QueryRowContext(ctx, `
SELECT
	COUNT(*),
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN status = ? AND answered_review_id IS NULL THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN status = ? AND answered_review_id IS NOT NULL THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN status NOT IN (?, ?) THEN 1 ELSE 0 END), 0)
FROM session_items
WHERE session_id = ?
`, ItemStatusAnswered, ItemStatusPending, ItemStatusAnswered, ItemStatusPending, ItemStatusPending, ItemStatusAnswered, session.id).Scan(
			&itemCount,
			&answeredItemCount,
			&pendingItemCount,
			&missingReviewCount,
			&staleReviewCount,
			&invalidStatusCount,
		); err != nil {
			return diagnosticCheckError("active sessions", fmt.Sprintf("session %s could not be inspected", session.id), err.Error())
		}

		if session.mode != ModeLearn && session.mode != ModeReview {
			issues = append(issues, fmt.Sprintf("session %s has unsupported mode %q", session.id, session.mode))
		}
		if session.answeredQuestions > session.totalQuestions {
			issues = append(issues, fmt.Sprintf("session %s has answered_questions %d > total_questions %d", session.id, session.answeredQuestions, session.totalQuestions))
		}
		if session.totalQuestions != itemCount {
			issues = append(issues, fmt.Sprintf("session %s stores total_questions=%d but has %d session items", session.id, session.totalQuestions, itemCount))
		}
		if session.answeredQuestions != answeredItemCount {
			issues = append(issues, fmt.Sprintf("session %s stores answered_questions=%d but has %d answered items", session.id, session.answeredQuestions, answeredItemCount))
		}
		if itemCount == 0 {
			issues = append(issues, fmt.Sprintf("session %s is active but has no session items", session.id))
		}
		if pendingItemCount == 0 {
			issues = append(issues, fmt.Sprintf("session %s is active but has no pending items", session.id))
		}
		if session.finishedAt.Valid {
			issues = append(issues, fmt.Sprintf("session %s is active but finished_at is set", session.id))
		}
		if missingReviewCount > 0 {
			issues = append(issues, fmt.Sprintf("session %s has %d answered item(s) without answered_review_id", session.id, missingReviewCount))
		}
		if staleReviewCount > 0 {
			issues = append(issues, fmt.Sprintf("session %s has %d pending item(s) with answered_review_id", session.id, staleReviewCount))
		}
		if invalidStatusCount > 0 {
			issues = append(issues, fmt.Sprintf("session %s has %d item(s) with unsupported status", session.id, invalidStatusCount))
		}
	}

	if sessionCount > 1 {
		issues = append([]string{formatStringSamples("active session ids", sessionCount, ids)}, issues...)
	}

	if len(issues) > 0 {
		summary := fmt.Sprintf("%d active session issue(s) detected across %d session(s)", len(issues), sessionCount)
		if sessionCount > 1 {
			summary = fmt.Sprintf("%d active session(s) detected; expected at most 1", sessionCount)
		}
		return diagnosticCheckError("active sessions", summary, issues...)
	}

	return diagnosticCheckOK("active sessions", fmt.Sprintf("%d active session is internally consistent", sessionCount))
}

func (s *Store) checkQuizability(ctx context.Context) DiagnosticCheck {
	words, err := s.listAllWords(ctx)
	if err != nil {
		return diagnosticCheckError("quizability", "words could not be loaded", err.Error())
	}
	if len(words) == 0 {
		return diagnosticCheckError("quizability", "words table is empty")
	}

	failures := make([]string, 0)
	for _, word := range words {
		pool, err := s.ListDistractorCandidates(ctx, word, doctorQuizPoolLimit, []int64{word.ID})
		if err != nil {
			return diagnosticCheckError("quizability", fmt.Sprintf("distractor pool for %s could not be loaded", word.Lemma), err.Error())
		}
		if countUniqueDistractorMeanings(word, pool) < doctorQuizChoiceSize-1 {
			failures = append(failures, describeWord(word))
		}
	}

	if len(failures) > 0 {
		return diagnosticCheckError(
			"quizability",
			fmt.Sprintf("%d word(s) cannot form %d-choice quizzes", len(failures), doctorQuizChoiceSize),
			formatStringSamples("words", len(failures), failures),
		)
	}

	return diagnosticCheckOK("quizability", fmt.Sprintf("all %d words can form %d-choice quizzes", len(words), doctorQuizChoiceSize))
}

func (s *Store) countRows(ctx context.Context, query string, args ...any) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) sampleInt64Rows(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := make([]int64, 0, doctorSampleLimit)
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) sampleStringRows(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	values := make([]string, 0, doctorSampleLimit)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) listAllWords(ctx context.Context) ([]Word, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, lemma, pos, meaning_ja, level, frequency_rank,
       distractor_group, example_en, example_ja, source, created_at
FROM words
ORDER BY COALESCE(frequency_rank, 999999) ASC, id ASC
`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	return scanWords(rows)
}

func countUniqueDistractorMeanings(correct Word, pool []Word) int {
	meanings := make(map[string]struct{}, len(pool))
	for _, candidate := range pool {
		if candidate.ID == correct.ID {
			continue
		}
		if candidate.MeaningJA == correct.MeaningJA {
			continue
		}
		meanings[candidate.MeaningJA] = struct{}{}
	}
	return len(meanings)
}

func describeWord(word Word) string {
	pos := word.Pos
	if pos == "" {
		pos = "no-pos"
	}
	return fmt.Sprintf("%s [%s]", word.Lemma, pos)
}

func diagnosticCheckOK(name, summary string, details ...string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: DiagnosticStatusOK, Summary: summary, Details: details}
}

func diagnosticCheckWarning(name, summary string, details ...string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: DiagnosticStatusWarning, Summary: summary, Details: details}
}

func diagnosticCheckError(name, summary string, details ...string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: DiagnosticStatusError, Summary: summary, Details: details}
}

func formatInt64Samples(label string, total int, values []int64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return formatSampleList(label, total, parts)
}

func formatStringSamples(label string, total int, values []string) string {
	return formatSampleList(label, total, values)
}

func formatSampleList(label string, total int, values []string) string {
	if len(values) == 0 {
		return fmt.Sprintf("%s: %d", label, total)
	}
	summary := strings.Join(values, ", ")
	if total > len(values) {
		return fmt.Sprintf("%s: %s (+%d more)", label, summary, total-len(values))
	}
	return fmt.Sprintf("%s: %s", label, summary)
}
