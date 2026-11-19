DROP TABLE IF EXISTS career_inquiries;
ALTER TABLE career_listings DROP COLUMN IF EXISTS status;
ALTER TABLE career_listings DROP COLUMN IF EXISTS applicant_click_count;

ALTER TABLE enquiries DROP CONSTRAINT IF EXISTS chk_enquiry_status;
ALTER TABLE enquiries ADD CONSTRAINT chk_enquiry_status 
    CHECK (status IN ('new', 'read', 'archived'));
