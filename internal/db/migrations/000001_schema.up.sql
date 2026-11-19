-- Full database schema for the Dexta Africa backend, applied as a single
-- atomic migration. Kept as one file deliberately: this project ships schema
-- changes as one reviewed, atomic script rather than a long chain of
-- incremental migrations, so the whole data model can be read top to bottom
-- in one place.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Admins & Team Members ──────────────────────────────────────────────────
-- Comprehensive administrator and employee identity store with role-based
-- routing, departmental tagging, and operational telemetry.
CREATE TABLE admins (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    full_name      TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL DEFAULT 'ADMIN',
    department     TEXT NOT NULL DEFAULT 'OPERATIONS',
    job_title      TEXT NOT NULL DEFAULT '',
    phone          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_admins_role ON admins (role);
CREATE INDEX idx_admins_department ON admins (department);

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

-- ── Portfolio Developments ───────────────────────────────────────────────
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

-- ── Journal Articles ─────────────────────────────────────────────────────
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

-- ── Career Listings ──────────────────────────────────────────────────────
CREATE TABLE career_listings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                  TEXT NOT NULL UNIQUE,
    title                 TEXT NOT NULL,
    department            TEXT NOT NULL DEFAULT '',
    location              TEXT NOT NULL DEFAULT '',
    employment_type       TEXT NOT NULL DEFAULT 'full_time',
    description           JSONB NOT NULL DEFAULT '{}'::jsonb,
    status                TEXT NOT NULL DEFAULT 'open',
    applicant_click_count INTEGER NOT NULL DEFAULT 0,
    published             BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_career_listings_slug ON career_listings (slug);
CREATE INDEX idx_career_listings_published ON career_listings (published);
CREATE INDEX idx_career_listings_status ON career_listings (status);

-- ── Career Inquiries & Applications ──────────────────────────────────────
CREATE TABLE career_inquiries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    career_listing_id  UUID NOT NULL REFERENCES career_listings(id) ON DELETE CASCADE,
    applicant_name     TEXT NOT NULL,
    applicant_email    TEXT NOT NULL,
    applicant_phone    TEXT NOT NULL DEFAULT '',
    resume_url         TEXT NOT NULL DEFAULT '',
    cover_letter       TEXT NOT NULL DEFAULT '',
    source             TEXT NOT NULL DEFAULT 'website',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_career_inquiries_listing_id ON career_inquiries (career_listing_id);
CREATE INDEX idx_career_inquiries_created_at ON career_inquiries (created_at DESC);

-- ── Enquiries & Pipeline Inquiries ───────────────────────────────────────
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
    CONSTRAINT chk_enquiry_status CHECK (status IN ('new', 'read', 'in_progress', 'resolved', 'archived'))
);

CREATE INDEX idx_enquiries_created_at ON enquiries (created_at DESC);
CREATE INDEX idx_enquiries_status ON enquiries (status);

-- ── Newsletter Subscribers ───────────────────────────────────────────────
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

-- ── Custom Forms and Registration Portals ─────────────────────────────────
CREATE TABLE custom_forms (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug             TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    form_type        TEXT NOT NULL DEFAULT 'event',
    status           TEXT NOT NULL DEFAULT 'active',
    cover_image_url  TEXT NOT NULL DEFAULT '',
    event_date       TIMESTAMPTZ,
    event_location   TEXT NOT NULL DEFAULT '',
    fields           JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_custom_forms_slug ON custom_forms (slug);
CREATE INDEX idx_custom_forms_type ON custom_forms (form_type);

CREATE TABLE form_submissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id          UUID NOT NULL REFERENCES custom_forms(id) ON DELETE CASCADE,
    full_name        TEXT NOT NULL,
    email            TEXT NOT NULL,
    phone            TEXT NOT NULL DEFAULT '',
    company          TEXT NOT NULL DEFAULT '',
    job_title        TEXT NOT NULL DEFAULT '',
    category         TEXT NOT NULL DEFAULT 'Attendee',
    attendance_mode  TEXT NOT NULL DEFAULT 'in-person',
    answers          JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes            TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'registered',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_form_submissions_form_id ON form_submissions (form_id);
CREATE INDEX idx_form_submissions_email ON form_submissions (email);
CREATE INDEX idx_form_submissions_category ON form_submissions (category);
CREATE INDEX idx_form_submissions_created_at ON form_submissions (created_at);

-- ── Transactional Outbox (Zero Data Loss) ──────────────────────────────────
CREATE TABLE outbox_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type     TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    payload        JSONB NOT NULL DEFAULT '{}'::jsonb,
    status         TEXT NOT NULL DEFAULT 'PENDING',
    retry_count    INTEGER NOT NULL DEFAULT 0,
    max_retries    INTEGER NOT NULL DEFAULT 5,
    error_message  TEXT,
    processed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_outbox_status CHECK (status IN ('PENDING', 'PROCESSING', 'PROCESSED', 'FAILED'))
);

CREATE INDEX idx_outbox_status ON outbox_events (status, created_at);

-- ── Immutable Enterprise Audit Logs ──────────────────────────────────────
CREATE TABLE audit_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id       TEXT,
    actor_email    TEXT,
    action         TEXT NOT NULL,
    entity         TEXT NOT NULL,
    entity_id      TEXT,
    ip_address     TEXT,
    user_agent     TEXT,
    metadata       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_action ON audit_logs (action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id);

-- ── updated_at Maintenance ───────────────────────────────────────────────
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

CREATE TRIGGER trg_custom_forms_updated_at BEFORE UPDATE ON custom_forms
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
