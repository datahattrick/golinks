-- Restore the original unique constraint (this would prevent org keywords from shadowing global ones)
-- Note: This may fail if there are duplicate keywords across scopes
ALTER TABLE links ADD CONSTRAINT links_keyword_key UNIQUE (keyword);
