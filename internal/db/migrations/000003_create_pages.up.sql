-- pages holds every singleton/editorial page whose shape varies (home,
-- about, lifestyle, careers landing, and the static legal pages). JSONB
-- content keeps this table generic instead of needing a new migration for
-- every new page section the frontend adds.
CREATE TABLE pages (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key              TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL DEFAULT '',
    content          JSONB NOT NULL DEFAULT '{}'::jsonb,
    seo_title        TEXT NOT NULL DEFAULT '',
    seo_description  TEXT NOT NULL DEFAULT '',
    published        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pages_key ON pages (key);
