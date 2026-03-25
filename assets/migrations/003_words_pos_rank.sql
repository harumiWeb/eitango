CREATE INDEX IF NOT EXISTS idx_words_pos_rank ON words(pos, frequency_rank, id);
