CREATE TABLE IF NOT EXISTS keyword_lookups (
    keyword TEXT NOT NULL,
    outcome VARCHAR(20) NOT NULL,
    count BIGINT NOT NULL DEFAULT 0,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (keyword, outcome)
);
