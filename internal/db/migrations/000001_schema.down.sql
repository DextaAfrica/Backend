-- Reverses 000001_schema.up.sql in full, in reverse dependency order.

DROP TRIGGER IF EXISTS trg_career_listings_updated_at ON career_listings;
DROP TRIGGER IF EXISTS trg_journal_updated_at ON journal_articles;
DROP TRIGGER IF EXISTS trg_portfolio_updated_at ON portfolio_developments;
DROP TRIGGER IF EXISTS trg_pages_updated_at ON pages;
DROP TRIGGER IF EXISTS trg_admins_updated_at ON admins;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS newsletter_subscribers;
DROP TABLE IF EXISTS enquiries;
DROP TABLE IF EXISTS career_listings;
DROP TABLE IF EXISTS journal_articles;
DROP TABLE IF EXISTS portfolio_developments;
DROP TABLE IF EXISTS pages;
DROP TABLE IF EXISTS admins;

DROP EXTENSION IF EXISTS pgcrypto;
