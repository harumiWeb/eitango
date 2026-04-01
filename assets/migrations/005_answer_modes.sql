ALTER TABLE sessions ADD COLUMN answer_mode TEXT NOT NULL DEFAULT 'choice';
ALTER TABLE reviews ADD COLUMN answer_mode TEXT NOT NULL DEFAULT 'choice';
