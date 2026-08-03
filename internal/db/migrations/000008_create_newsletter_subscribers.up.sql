CREATE TABLE newsletter_subscribers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'subscribed',
    subscribed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    unsubscribed_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_newsletter_status CHECK (status IN ('subscribed', 'unsubscribed'))
);

CREATE INDEX idx_newsletter_email ON newsletter_subscribers (email);
