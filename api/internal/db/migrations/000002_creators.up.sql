-- Creators become first-class. Until now a person was only a subject_type
-- flag on a campaign, so there was no way to ask "how does this creator
-- perform across their campaigns" — which is the whole point of the
-- product. A creator now owns many campaigns and is benchmarked against
-- their own tier, not against brands with incomparable budgets.

CREATE TYPE creator_tier_t AS ENUM ('nano', 'micro', 'mid', 'macro', 'mega');

CREATE TABLE creators (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    role             TEXT NOT NULL,
    initials         TEXT NOT NULL,
    category         TEXT NOT NULL,
    region           TEXT NOT NULL,
    tier             creator_tier_t NOT NULL,
    followers        BIGINT NOT NULL DEFAULT 0 CHECK (followers >= 0),
    primary_platform TEXT NOT NULL,
    -- India runs campaigns in many languages; this is the axis global ad
    -- intelligence tools handle worst and the one worth indexing on.
    languages        TEXT[] NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_creators_tier ON creators (tier);
CREATE INDEX idx_creators_category ON creators (category);

ALTER TABLE campaigns ADD COLUMN creator_id TEXT REFERENCES creators(id) ON DELETE SET NULL;
CREATE INDEX idx_campaigns_creator_id ON campaigns (creator_id);

-- Creative attributes. These are populated by the CreativeAnalyzer
-- (internal/service/creative_analyzer.go): a real multimodal call when an
-- API key is configured, left NULL otherwise. Everything the app computes
-- from them — language lift, hook performance — is ordinary math over
-- whatever is present, so the analytics degrade rather than break.
CREATE TYPE hook_type_t AS ENUM ('demo', 'testimonial', 'offer', 'story', 'unboxing', 'announcement');

ALTER TABLE creatives ADD COLUMN language TEXT;
ALTER TABLE creatives ADD COLUMN hook_type hook_type_t;
ALTER TABLE creatives ADD COLUMN claim TEXT;
ALTER TABLE creatives ADD COLUMN festival TEXT;
ALTER TABLE creatives ADD COLUMN analyzed_at TIMESTAMPTZ;

CREATE INDEX idx_creatives_language ON creatives (language);
CREATE INDEX idx_creatives_hook_type ON creatives (hook_type);
