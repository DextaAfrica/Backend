-- Reverses 000001_schema.up.sql in full, in reverse dependency order.

DROP TRIGGER IF EXISTS trg_visitor_report_schedules_updated_at ON visitor_report_schedules;
DROP TRIGGER IF EXISTS trg_visitors_updated_at ON visitors;
DROP TRIGGER IF EXISTS trg_dashboard_access_requests_updated_at ON dashboard_access_requests;
DROP TRIGGER IF EXISTS trg_production_release_requests_updated_at ON production_release_requests;
DROP TRIGGER IF EXISTS trg_custom_forms_updated_at ON custom_forms;
DROP TRIGGER IF EXISTS trg_career_listings_updated_at ON career_listings;
DROP TRIGGER IF EXISTS trg_journal_updated_at ON journal_articles;
DROP TRIGGER IF EXISTS trg_portfolio_updated_at ON portfolio_developments;
DROP TRIGGER IF EXISTS trg_pages_updated_at ON pages;
DROP TRIGGER IF EXISTS trg_admin_sessions_updated_at ON admin_sessions;
DROP TRIGGER IF EXISTS trg_admins_updated_at ON admins;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS visitor_report_deliveries;
DROP TABLE IF EXISTS visitor_report_schedules;
DROP TABLE IF EXISTS visitors;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS dashboard_access_requests;
DROP TABLE IF EXISTS production_release_requests;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS form_submissions;
DROP TABLE IF EXISTS custom_forms;
DROP TABLE IF EXISTS career_inquiries;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS newsletter_subscribers;
DROP TABLE IF EXISTS enquiries;
DROP TABLE IF EXISTS career_listings;
DROP TABLE IF EXISTS journal_articles;
DROP TABLE IF EXISTS portfolio_developments;
DROP TABLE IF EXISTS pages;
DROP TABLE IF EXISTS admins;

DROP EXTENSION IF EXISTS pgcrypto;
