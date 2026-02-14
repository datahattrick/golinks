CREATE TABLE shared_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    keyword VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT no_self_share CHECK (sender_id != recipient_id),
    CONSTRAINT unique_pending_share UNIQUE (sender_id, recipient_id, keyword)
);

CREATE INDEX idx_shared_links_recipient ON shared_links(recipient_id);
CREATE INDEX idx_shared_links_sender ON shared_links(sender_id);
