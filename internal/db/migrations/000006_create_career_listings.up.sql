CREATE TABLE career_listings (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug             TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    department       TEXT NOT NULL DEFAULT '',
    location         TEXT NOT NULL DEFAULT '',
    employment_type  TEXT NOT NULL DEFAULT 'full_time',
    description      JSONB NOT NULL DEFAULT '{}'::jsonb,
    published        BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_career_listings_slug ON career_listings (slug);
CREATE INDEX idx_career_listings_published ON career_listings (published);
