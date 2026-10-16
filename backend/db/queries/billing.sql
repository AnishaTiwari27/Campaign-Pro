-- name: GetSubscription :one
SELECT * FROM subscriptions WHERE org_id = @org_id::text;

-- name: UpsertSubscription :one
INSERT INTO subscriptions (
    org_id, plan, status, trial_ends_at, current_period_end,
    provider, provider_customer_id, provider_subscription_id
) VALUES (
    @org_id::text, @plan, @status, @trial_ends_at, @current_period_end,
    @provider, @provider_customer_id, @provider_subscription_id
)
ON CONFLICT (org_id) DO UPDATE SET
    plan = EXCLUDED.plan,
    status = EXCLUDED.status,
    trial_ends_at = EXCLUDED.trial_ends_at,
    current_period_end = EXCLUDED.current_period_end,
    provider = EXCLUDED.provider,
    provider_customer_id = EXCLUDED.provider_customer_id,
    provider_subscription_id = EXCLUDED.provider_subscription_id,
    updated_at = now()
RETURNING *;

-- SetSubscriptionStatus is what a provider webhook writes. It is keyed
-- by the provider's own subscription id rather than by an organisation,
-- because a webhook arrives with no session and no tenant — the id is
-- the only thing tying the event to a row, and it was stored on that
-- row when the subscription was created in-org.
-- name: SetSubscriptionStatus :one
UPDATE subscriptions
SET status = @status,
    current_period_end = @current_period_end,
    updated_at = now()
WHERE provider_subscription_id = @provider_subscription_id::text
  AND @provider_subscription_id::text <> ''
RETURNING *;
