-- What an organisation is paying for, and whether it may use the app.
--
-- One row per organisation rather than a history of them: the question
-- the request path asks is "may this tenant in, right now", and that is
-- a property of the current subscription. Past periods live with the
-- payment provider, which is the system of record for money; this table
-- is a cache of entitlement, not an invoice ledger.
CREATE TABLE subscriptions (
    org_id TEXT PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    -- Which plan, by its id in services/billing. Free text rather than
    -- an enum so renaming or retiring a plan is a code change and not a
    -- migration against live rows.
    plan   TEXT NOT NULL,
    -- trialing | active | past_due | cancelled. Also free text: these
    -- mirror the provider's vocabulary, which is theirs to change.
    status TEXT NOT NULL,
    -- Both nullable, and the nulls mean different things. A null trial
    -- is "never had one"; a null period end is "does not expire", which
    -- is what the grandfathered rows below are.
    trial_ends_at      TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,

    provider                 TEXT NOT NULL DEFAULT 'razorpay',
    provider_customer_id     TEXT NOT NULL DEFAULT '',
    provider_subscription_id TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Reached on every request that gets past the session guard, to decide
-- whether the request may proceed at all.
CREATE INDEX subscriptions_status_idx ON subscriptions (status);

-- Every organisation that already exists keeps its access, in the same
-- transaction that creates the table.
--
-- This matters more than it looks. The gate locks the whole app when an
-- organisation has no active subscription, so a table that starts empty
-- would lock out every existing tenant the moment this migration ran —
-- including the live deployment, whose users never agreed to a paywall
-- and would simply find the product gone. Grandfathering is the only
-- version of this migration that is safe to run against real data.
--
-- current_period_end is null for these: they do not expire. Putting
-- anyone who was already using the product onto a countdown would be
-- the same outage, just delayed.
INSERT INTO subscriptions (org_id, plan, status, current_period_end)
SELECT id, 'grandfathered', 'active', NULL FROM organizations;
