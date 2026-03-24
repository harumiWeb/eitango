CREATE INDEX IF NOT EXISTS idx_progress_due_at ON progress(due_at);
CREATE INDEX IF NOT EXISTS idx_words_level_pos ON words(level, pos);
CREATE INDEX IF NOT EXISTS idx_words_distractor_group ON words(distractor_group);
CREATE INDEX IF NOT EXISTS idx_reviews_word_id_answered_at ON reviews(word_id, answered_at);
CREATE INDEX IF NOT EXISTS idx_reviews_session_id ON reviews(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status, started_at);
CREATE INDEX IF NOT EXISTS idx_session_items_status ON session_items(session_id, status, ordinal);
CREATE INDEX IF NOT EXISTS idx_session_items_word ON session_items(session_id, word_id, kind);
