ALTER TABLE words ADD COLUMN source TEXT NOT NULL DEFAULT 'core';

CREATE INDEX IF NOT EXISTS idx_words_source ON words(source);

CREATE INDEX IF NOT EXISTS idx_words_source_lemma_pos_key
  ON words(source, lemma, IFNULL(pos, ''));
