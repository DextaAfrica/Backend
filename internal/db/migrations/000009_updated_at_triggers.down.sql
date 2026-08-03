DROP TRIGGER IF EXISTS trg_admins_updated_at ON admins;
DROP TRIGGER IF EXISTS trg_pages_updated_at ON pages;
DROP TRIGGER IF EXISTS trg_portfolio_updated_at ON portfolio_developments;
DROP TRIGGER IF EXISTS trg_journal_updated_at ON journal_articles;
DROP TRIGGER IF EXISTS trg_career_listings_updated_at ON career_listings;
DROP FUNCTION IF EXISTS set_updated_at();
