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
