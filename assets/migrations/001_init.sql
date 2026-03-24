CREATE TABLE IF NOT EXISTS words (
  id                INTEGER PRIMARY KEY,
  lemma             TEXT NOT NULL,
  pos               TEXT,
  meaning_ja        TEXT NOT NULL,
  level             TEXT,
  frequency_rank    INTEGER,
  distractor_group  TEXT,
  example_en        TEXT,
  example_ja        TEXT,
  created_at        TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS progress (
  word_id           INTEGER PRIMARY KEY,
  state             TEXT NOT NULL,
  due_at            TEXT,
  interval_days     REAL NOT NULL DEFAULT 0,
  ease_factor       REAL NOT NULL DEFAULT 2.5,
  last_seen_at      TEXT,
  streak_correct    INTEGER NOT NULL DEFAULT 0,
  total_correct     INTEGER NOT NULL DEFAULT 0,
  total_wrong       INTEGER NOT NULL DEFAULT 0,
  lapses            INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(word_id) REFERENCES words(id)
);

CREATE TABLE IF NOT EXISTS sessions (
  id                  TEXT PRIMARY KEY,
  started_at          TEXT NOT NULL,
  finished_at         TEXT,
  mode                TEXT NOT NULL,
  total_questions     INTEGER NOT NULL,
  answered_questions  INTEGER NOT NULL DEFAULT 0,
  status              TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reviews (
  id                  INTEGER PRIMARY KEY,
  word_id             INTEGER NOT NULL,
  session_id          TEXT NOT NULL,
  answered_at         TEXT NOT NULL,
  selected_choice     INTEGER NOT NULL,
  correct_choice      INTEGER NOT NULL,
  is_correct          INTEGER NOT NULL,
  response_ms         INTEGER,
  rating              TEXT,
  FOREIGN KEY(word_id) REFERENCES words(id),
  FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS session_items (
  session_id          TEXT NOT NULL,
  ordinal             INTEGER NOT NULL,
  word_id             INTEGER NOT NULL,
  kind                TEXT NOT NULL,
  status              TEXT NOT NULL DEFAULT 'pending',
  source_ordinal      INTEGER,
  answered_review_id  INTEGER,
  created_at          TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY(session_id, ordinal),
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE,
  FOREIGN KEY(word_id) REFERENCES words(id),
  FOREIGN KEY(answered_review_id) REFERENCES reviews(id)
);

CREATE TABLE IF NOT EXISTS app_meta (
  key         TEXT PRIMARY KEY,
  value       TEXT NOT NULL,
  updated_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
