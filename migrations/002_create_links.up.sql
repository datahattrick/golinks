CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keyword TEXT UNIQUE NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    created_by UUID REFERENCES users(id),
    click_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_links_keyword ON links(keyword);
CREATE INDEX IF NOT EXISTS idx_links_created_by ON links(created_by);
CREATE INDEX IF NOT EXISTS idx_links_keyword_trgm ON links USING gin (keyword gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_links_url_trgm ON links USING gin (url gin_trgm_ops);
