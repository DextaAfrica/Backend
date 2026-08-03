CREATE TABLE enquiries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    email        TEXT NOT NULL,
    phone        TEXT NOT NULL DEFAULT '',
    subject      TEXT NOT NULL DEFAULT '',
    message      TEXT NOT NULL,
    source_page  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'new',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_enquiry_status CHECK (status IN ('new', 'read', 'archived'))
);

CREATE INDEX idx_enquiries_created_at ON enquiries (created_at DESC);
CREATE INDEX idx_enquiries_status ON enquiries (status);
