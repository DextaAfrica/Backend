-- ── Custom Forms and Registration Portals ─────────────────────────────────
-- Enables dynamic short-link onboarding and event registrations with
-- rich demographic telemetry, attendance modes, and custom field capture.

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

CREATE TRIGGER trg_custom_forms_updated_at BEFORE UPDATE ON custom_forms
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
