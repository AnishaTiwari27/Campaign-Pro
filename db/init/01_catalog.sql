-- Run as catalog_service against campaign_tracker:
--   psql -U catalog_service -d campaign_tracker -f db/init/01_catalog.sql
-- (requires 00_roles.sql to have run first)

CREATE TABLE catalog.categories (
    id         SMALLSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    -- which kind of subject this category can apply to — drives the
    -- frontend's category dropdown per Type (Brand/Person) filter.
    applies_to TEXT NOT NULL DEFAULT 'brand' CHECK (applies_to IN ('brand', 'person', 'both'))
);

CREATE TABLE catalog.regions (
    id   SMALLSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE catalog.ad_types (
    id               SMALLSERIAL PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,
    default_platform TEXT NOT NULL,
    color_hex        TEXT NOT NULL
);

CREATE TABLE catalog.brands (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    category_id SMALLINT NOT NULL REFERENCES catalog.categories(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE catalog.people (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,
    category_id      SMALLINT NOT NULL REFERENCES catalog.categories(id),
    occupation       TEXT NOT NULL, -- 'Influencer' | 'Musician' | 'Actor' | 'CEO, Business Leader' ...
    primary_platform TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Seed vocabulary + subjects
-- ---------------------------------------------------------------------------

INSERT INTO catalog.categories (name, applies_to) VALUES
    ('E-commerce', 'brand'), ('Fashion', 'brand'), ('Food Delivery', 'brand'),
    ('Technology', 'brand'), ('Fintech', 'brand'), ('FMCG', 'brand'),
    ('Beauty & Cosmetics', 'brand'), ('Footwear', 'brand'), ('Luxury', 'both'),
    ('Influencer', 'person'), ('Music', 'person'), ('Entertainment', 'person'), ('Business', 'person');

INSERT INTO catalog.regions (name) VALUES
    ('Mumbai'), ('Delhi NCR'), ('Bengaluru'), ('South Zone'), ('West Zone'), ('Pan-India');

INSERT INTO catalog.ad_types (name, default_platform, color_hex) VALUES
    ('Social Media', 'Instagram',        '#C97A1A'),
    ('Influencer',   'YouTube Creators', '#3B7A57'),
    ('Google Ads',   'Google Search',    '#B23A2A'),
    ('Display',      'Google Display',   '#8A6D3B'),
    ('Video',         'YouTube',         '#5A6ACF'),
    ('Performance',  'Meta Ads',         '#1F6F6F');

INSERT INTO catalog.brands (name, category_id) VALUES
    ('Zepto',              (SELECT id FROM catalog.categories WHERE name = 'E-commerce')),
    ('Myntra',             (SELECT id FROM catalog.categories WHERE name = 'Fashion')),
    ('Swiggy',             (SELECT id FROM catalog.categories WHERE name = 'Food Delivery')),
    ('Tata CLiQ',          (SELECT id FROM catalog.categories WHERE name = 'E-commerce')),
    ('boAt',               (SELECT id FROM catalog.categories WHERE name = 'Technology')),
    ('Nykaa',              (SELECT id FROM catalog.categories WHERE name = 'Beauty & Cosmetics')),
    ('CRED',               (SELECT id FROM catalog.categories WHERE name = 'Fintech')),
    ('Flipkart',           (SELECT id FROM catalog.categories WHERE name = 'E-commerce')),
    ('Dream11',            (SELECT id FROM catalog.categories WHERE name = 'Technology')),
    ('Amul',               (SELECT id FROM catalog.categories WHERE name = 'FMCG')),
    ('Lenskart',           (SELECT id FROM catalog.categories WHERE name = 'Fashion')),
    ('PhonePe',            (SELECT id FROM catalog.categories WHERE name = 'Fintech')),
    ('Tanishq',            (SELECT id FROM catalog.categories WHERE name = 'Luxury')),
    ('Forest Essentials',  (SELECT id FROM catalog.categories WHERE name = 'Beauty & Cosmetics')),
    ('Metro Shoes',        (SELECT id FROM catalog.categories WHERE name = 'Footwear')),
    ('Bata',               (SELECT id FROM catalog.categories WHERE name = 'Footwear')),
    ('Ethos Watches',      (SELECT id FROM catalog.categories WHERE name = 'Luxury'));

-- Fictional personas, deliberately not real named individuals — see
-- docs/ARCHITECTURE.md for why (fabricated paid-campaign/spend figures
-- attributed to a real, named person is a different kind of claim than the
-- same treatment applied to a company).
INSERT INTO catalog.people (name, category_id, occupation, primary_platform) VALUES
    ('Aria Kapoor',   (SELECT id FROM catalog.categories WHERE name = 'Influencer'),   'Influencer',              'Instagram'),
    ('Rohan Bhatia',  (SELECT id FROM catalog.categories WHERE name = 'Business'),     'CEO, Business Leader',   'LinkedIn'),
    ('Meera Chopra',  (SELECT id FROM catalog.categories WHERE name = 'Entertainment'),'Actor',                   'Instagram'),
    ('Dev Malhotra',  (SELECT id FROM catalog.categories WHERE name = 'Music'),        'Musician',                'YouTube'),
    ('Ishaan Verma',  (SELECT id FROM catalog.categories WHERE name = 'Influencer'),   'Influencer',              'Instagram'),
    ('Priya Nair',    (SELECT id FROM catalog.categories WHERE name = 'Business'),     'Founder, Business Leader','LinkedIn'),
    ('Kabir Sethi',   (SELECT id FROM catalog.categories WHERE name = 'Entertainment'),'Actor',                   'Instagram'),
    ('Ananya Rao',    (SELECT id FROM catalog.categories WHERE name = 'Music'),        'Singer',                  'YouTube'),
    ('Vivaan Oberoi', (SELECT id FROM catalog.categories WHERE name = 'Influencer'),   'Influencer',              'Instagram'),
    ('Sana Iyer',     (SELECT id FROM catalog.categories WHERE name = 'Business'),     'CEO, Business Leader',   'LinkedIn');
