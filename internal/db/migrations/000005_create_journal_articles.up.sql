CREATE TABLE journal_articles (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL UNIQUE,
    title             TEXT NOT NULL,
    excerpt           TEXT NOT NULL DEFAULT '',
    body              JSONB NOT NULL DEFAULT '{}'::jsonb,
    cover_image_url   TEXT NOT NULL DEFAULT '',
    author            TEXT NOT NULL DEFAULT '',
    category          TEXT NOT NULL DEFAULT '',
    seo_title         TEXT NOT NULL DEFAULT '',
    seo_description   TEXT NOT NULL DEFAULT '',
    published         BOOLEAN NOT NULL DEFAULT false,
    published_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_journal_slug ON journal_articles (slug);
CREATE INDEX idx_journal_published ON journal_articles (published, published_at DESC);
