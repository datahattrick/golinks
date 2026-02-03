CREATE TABLE IF NOT EXISTS click_history (
    link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    hour_bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    click_count INTEGER NOT NULL DEFAULT 1,
    UNIQUE (link_id, hour_bucket)
);

CREATE INDEX IF NOT EXISTS idx_click_history_link_hour ON click_history(link_id, hour_bucket);
