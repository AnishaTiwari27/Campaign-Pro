-- Outreach: who we may email, who we may never email again, and a row
-- per message actually sent.
--
-- The shape here is driven by one fact: the sending domain is shared
-- infrastructure. Every organisation's outreach leaves from the same
-- IP and the same DKIM key, so one tenant mailing a scraped list gets
-- the whole platform throttled — and takes password resets and
-- scheduled reports down with it, because those ride the same
-- reputation. That is why consent is a column with a NOT NULL basis
-- rather than a convention, and why suppression has a platform-wide
-- scope that no tenant can override.

-- An address we hold for a creator, and the provenance that says
-- whether we may use it.
--
-- Separate from creators rather than a column on it: a creator can
-- have a personal address and a manager's, they arrive from different
-- sources with different permissions, and one can be suppressed while
-- the other is still good. A single creators.email could not express
-- any of that.
CREATE TABLE creator_contacts (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    -- Who this address reaches: the creator, or the person who handles
    -- their business. Worth knowing before writing the message.
    role       TEXT NOT NULL DEFAULT 'creator',

    -- Where the address came from. Not decoration: under the DPDP Act
    -- the answer to "how did you get this" is the whole defence, and
    -- an address whose origin nobody recorded is one that cannot be
    -- defended. Free text over an enum so a new vendor is a code
    -- change, with source_detail naming the specific vendor or page.
    source        TEXT NOT NULL,
    source_detail TEXT NOT NULL DEFAULT '',

    -- consent | legitimate_interest. There is deliberately no third
    -- value meaning "unknown": the eligibility check refuses anything
    -- it does not recognise, so an unrecorded basis is a row that
    -- never gets mailed rather than one that quietly does.
    lawful_basis TEXT NOT NULL,
    -- When they agreed, and to what. Null where the basis is not
    -- consent. purpose is an array because "you may pitch me paid
    -- campaigns" and "you may add me to a brand's roster" are
    -- different permissions and get asked separately.
    consented_at TIMESTAMPTZ,
    purpose      TEXT[] NOT NULL DEFAULT '{}',

    -- Deliverability, not permission: the last time this address was
    -- confirmed to accept mail. An unverified address in a batch is
    -- what turns into the bounce rate that gets a sender suspended.
    verified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One row per address per creator per tenant. Re-importing a
    -- vendor export updates rather than duplicating, so a creator
    -- cannot be mailed twice by one batch.
    UNIQUE (org_id, creator_id, email)
);
CREATE INDEX creator_contacts_org_idx     ON creator_contacts (org_id);
CREATE INDEX creator_contacts_creator_idx ON creator_contacts (creator_id);

-- Addresses that must not be mailed, ever again.
--
-- org_id is nullable on purpose, and the null is the important case:
--
--   a row WITH an org  -- this tenant may not mail them. An
--                         unsubscribe is addressed to the sender it
--                         came from, and another agency's standing
--                         relationship is not ours to cancel.
--   a row WITHOUT one  -- nobody may mail them, on any tenant. Hard
--                         bounces and spam complaints are facts about
--                         the shared sending reputation, so they bind
--                         the whole platform. A tenant cannot clear
--                         one of these by deleting their own row.
CREATE TABLE suppressions (
    id      TEXT PRIMARY KEY,
    org_id  TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    email   TEXT NOT NULL,
    -- unsubscribed | hard_bounce | complaint | manual
    reason  TEXT NOT NULL,
    note    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Lowercased because the local part is technically case-sensitive and
-- no real mail host treats it that way — matching exactly would let
-- Foo@x.com through a suppression on foo@x.com. coalesce keeps the
-- platform-wide row distinct from each tenant's own.
CREATE UNIQUE INDEX suppressions_email_scope_idx
    ON suppressions (lower(email), coalesce(org_id, ''));
-- The eligibility check reads this on every recipient of every batch.
CREATE INDEX suppressions_email_idx ON suppressions (lower(email));

-- One send run: a message, and the creators it is going to.
CREATE TABLE outreach_batches (
    id      TEXT PRIMARY KEY,
    org_id  TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    -- What this is pitching. Null for a batch not tied to one, but the
    -- named case is the one that performs: a pitch that says which
    -- brand and which deliverable is outreach, and the same message
    -- sent to everyone is the thing that gets reported as spam.
    campaign_id TEXT REFERENCES campaigns(id) ON DELETE SET NULL,
    subject     TEXT NOT NULL,
    -- Plain text with {{name}}-style placeholders, filled per
    -- recipient. Stored as sent so an audit can reproduce the message
    -- rather than guess at the template that was live that day.
    body        TEXT NOT NULL,
    -- draft | sending | sent | paused | cancelled
    status      TEXT NOT NULL DEFAULT 'draft',
    created_by  TEXT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);
CREATE INDEX outreach_batches_org_idx ON outreach_batches (org_id, created_at DESC);

-- One row per intended recipient, written before anything is sent.
--
-- Queued-then-marked rather than send-then-record: a batch that dies
-- half way through has to be resumable, and the only way to know who
-- already received it is to have written the row first. Sending first
-- and recording after means a crash re-mails everyone on retry, which
-- is both the complaint rate and the reason they stop replying.
CREATE TABLE outreach_messages (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    batch_id   TEXT NOT NULL REFERENCES outreach_batches(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    -- queued | sent | skipped | failed | bounced | complained | replied
    status      TEXT NOT NULL DEFAULT 'queued',
    -- Why this recipient was not mailed. The useful half of a batch
    -- report: "412 sent, 88 skipped" is only actionable if you can see
    -- that 61 of the 88 had no recorded consent.
    skip_reason TEXT NOT NULL DEFAULT '',
    -- The one-click unsubscribe secret, unique per message rather than
    -- per contact: a token that leaks — forwarded mail, a proxy that
    -- rewrites links — can then only unsubscribe the recipient it was
    -- issued to, and cannot be used to enumerate anyone else.
    unsubscribe_token TEXT NOT NULL UNIQUE,
    sent_at     TIMESTAMPTZ,
    replied_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- A creator appears at most once in a batch, enforced rather than
    -- filtered in code, because the duplicate is the thing that reads
    -- as spam from the recipient's side.
    UNIQUE (batch_id, creator_id)
);
CREATE INDEX outreach_messages_batch_idx ON outreach_messages (batch_id, status);
CREATE INDEX outreach_messages_org_idx   ON outreach_messages (org_id);
