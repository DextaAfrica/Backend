-- Full database schema for the Dexta Africa backend, applied as a single
-- migration. Kept as one file deliberately: this project ships schema
-- changes as one reviewed, atomic script rather than a long chain of
-- incremental migrations, so the whole data model can be read top to bottom
-- in one place. Future schema changes should add a new
-- NNNNNN_description.up.sql / .down.sql pair rather than editing this file,
-- once it has shipped to any real environment.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Admins ───────────────────────────────────────────────────────────────
-- Single-role CMS admin accounts. See docs/ARCHITECTURE.md for why there is
-- no roles/permissions model yet.
CREATE TABLE admins (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Pages ────────────────────────────────────────────────────────────────
-- Flexible, JSONB-bodied content for every singleton editorial page (home,
-- about, lifestyle, careers landing, contact, and the static legal pages).
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

-- ── Portfolio developments ───────────────────────────────────────────────
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

-- ── Journal articles ─────────────────────────────────────────────────────
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

-- ── Career listings ──────────────────────────────────────────────────────
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

-- ── Enquiries ────────────────────────────────────────────────────────────
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

-- ── Newsletter subscribers ───────────────────────────────────────────────
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

-- ── updated_at maintenance ───────────────────────────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_admins_updated_at BEFORE UPDATE ON admins
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pages_updated_at BEFORE UPDATE ON pages
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_portfolio_updated_at BEFORE UPDATE ON portfolio_developments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_journal_updated_at BEFORE UPDATE ON journal_articles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_career_listings_updated_at BEFORE UPDATE ON career_listings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
