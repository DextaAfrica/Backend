CREATE TABLE portfolio_developments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL UNIQUE,
    name              TEXT NOT NULL,
    summary           TEXT NOT NULL DEFAULT '',
    body              JSONB NOT NULL DEFAULT '{}'::jsonb,
    hero_image_url    TEXT NOT NULL DEFAULT '',
    gallery           JSONB NOT NULL DEFAULT '[]'::jsonb,
    location          TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'planning',
    featured          BOOLEAN NOT NULL DEFAULT false,
    seo_title         TEXT NOT NULL DEFAULT '',
    seo_description   TEXT NOT NULL DEFAULT '',
    published         BOOLEAN NOT NULL DEFAULT false,
    published_at      TIMESTAMPTZ,
    display_order     INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_portfolio_status CHECK (status IN ('planning', 'under_construction', 'completed'))
);

CREATE INDEX idx_portfolio_slug ON portfolio_developments (slug);
CREATE INDEX idx_portfolio_published ON portfolio_developments (published, display_order);
