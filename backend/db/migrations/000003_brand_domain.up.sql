-- The brand a campaign is for, as a domain. Used to resolve a real logo;
-- null for campaigns with no brand (or where no logo could be fetched),
-- in which case the UI falls back to a generated avatar.
ALTER TABLE campaigns ADD COLUMN brand_domain TEXT;
