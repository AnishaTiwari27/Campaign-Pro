// Package store persists one subscription per organisation.
package store

import (
	"context"
	"errors"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/billing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct{ db *database.DB }

func New(db *database.DB) *Store { return &Store{db: db} }

func ts(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toDomain(r gen.Subscription) billing.Subscription {
	return billing.Subscription{
		OrgID:                  r.OrgID,
		Plan:                   r.Plan,
		Status:                 billing.Status(r.Status),
		TrialEndsAt:            database.TimeOf(r.TrialEndsAt),
		CurrentPeriodEnd:       database.TimeOf(r.CurrentPeriodEnd),
		ProviderSubscriptionID: r.ProviderSubscriptionID,
	}
}

// Get reads the calling organisation's subscription.
//
// A missing row is StatusNone rather than an error: an organisation
// that has never subscribed is a normal state the gate has to answer
// for, not a failure.
func (s *Store) Get(ctx context.Context) (billing.Subscription, error) {
	org := scope.OrgFrom(ctx)
	r, err := s.db.Queries.GetSubscription(ctx, org.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return billing.Subscription{OrgID: org.ID, Status: billing.StatusNone}, nil
		}
		return billing.Subscription{}, err
	}
	return toDomain(r), nil
}

// GetFor reads a named organisation's subscription. The gate uses this:
// it runs before any handler and is handed the tenant directly.
func (s *Store) GetFor(ctx context.Context, orgID string) (billing.Subscription, error) {
	r, err := s.db.Queries.GetSubscription(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return billing.Subscription{OrgID: orgID, Status: billing.StatusNone}, nil
		}
		return billing.Subscription{}, err
	}
	return toDomain(r), nil
}

func (s *Store) Upsert(ctx context.Context, sub billing.Subscription) (billing.Subscription, error) {
	r, err := s.db.Queries.UpsertSubscription(ctx, gen.UpsertSubscriptionParams{
		OrgID:                  sub.OrgID,
		Plan:                   sub.Plan,
		Status:                 string(sub.Status),
		TrialEndsAt:            ts(sub.TrialEndsAt),
		CurrentPeriodEnd:       ts(sub.CurrentPeriodEnd),
		Provider:               "razorpay",
		ProviderCustomerID:     "",
		ProviderSubscriptionID: sub.ProviderSubscriptionID,
	})
	if err != nil {
		return billing.Subscription{}, err
	}
	return toDomain(r), nil
}

// ApplyProviderStatus is the webhook's write, keyed by the provider's
// subscription id because a webhook carries no tenant.
func (s *Store) ApplyProviderStatus(ctx context.Context, providerSubID string, status billing.Status, periodEnd time.Time) (billing.Subscription, error) {
	r, err := s.db.Queries.SetSubscriptionStatus(ctx, gen.SetSubscriptionStatusParams{
		ProviderSubscriptionID: providerSubID,
		Status:                 string(status),
		CurrentPeriodEnd:       ts(periodEnd),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return billing.Subscription{}, database.ErrNotFound
		}
		return billing.Subscription{}, err
	}
	return toDomain(r), nil
}
