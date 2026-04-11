ALTER TABLE words ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_words_source_active ON words(source, is_active);
CREATE INDEX IF NOT EXISTS idx_words_active_pos_rank ON words(is_active, pos, frequency_rank, id);
