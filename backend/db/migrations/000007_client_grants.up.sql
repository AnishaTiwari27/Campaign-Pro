-- Which campaigns a client account may see.
--
-- is_agency has always said "this person is not staff", and IsClient() has
-- always been computed from it, but nothing consumed either: a client saw
-- all 35 campaigns exactly as an admin did. This is the relation that was
-- missing — without it there was no answer to "their own data" to enforce.
--
-- Grants are per campaign rather than per brand or account, because this
-- schema has no brand or account entity: campaigns carry a category, a
-- region and a brand domain, none of which is an ownership boundary.
-- Naming a campaign is the only unambiguous grant available.
--
-- An agency user never appears here. Their visibility is "everything", and
-- recording that as 35 rows that must be kept in step with every new
-- campaign is a bug waiting to happen.
CREATE TABLE client_campaign_grants (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, campaign_id)
);

-- The lookup is always "what may this user see", on every request they make.
CREATE INDEX client_campaign_grants_user_idx ON client_campaign_grants (user_id);
