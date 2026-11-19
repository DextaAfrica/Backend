-- ── Update enquiries check constraint to support full pipeline statuses ──
ALTER TABLE enquiries DROP CONSTRAINT IF EXISTS chk_enquiry_status;
ALTER TABLE enquiries ADD CONSTRAINT chk_enquiry_status 
    CHECK (status IN ('new', 'read', 'in_progress', 'resolved', 'archived'));

-- ── Enhance career listings with workflow status & applicant telemetry ──
ALTER TABLE career_listings ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'open';
ALTER TABLE career_listings ADD COLUMN IF NOT EXISTS applicant_click_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_career_listings_status ON career_listings(status);

-- ── Candidate applications & website click inquiries ─────────────────────
CREATE TABLE IF NOT EXISTS career_inquiries (
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

CREATE INDEX IF NOT EXISTS idx_career_inquiries_listing_id ON career_inquiries(career_listing_id);
CREATE INDEX IF NOT EXISTS idx_career_inquiries_created_at ON career_inquiries(created_at DESC);
