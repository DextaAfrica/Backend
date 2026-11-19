-- Reverses 000002_forms.up.sql

DROP TRIGGER IF EXISTS trg_custom_forms_updated_at ON custom_forms;
DROP TABLE IF EXISTS form_submissions;
DROP TABLE IF EXISTS custom_forms;
