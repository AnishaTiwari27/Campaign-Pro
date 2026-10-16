// Package service decides entitlement and drives the provider.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/billing"
)

// Store is what this service needs of persistence.
type Store interface {
	Get(ctx context.Context) (billing.Subscription, error)
	GetFor(ctx context.Context, orgID string) (billing.Subscription, error)
	Upsert(ctx context.Context, sub billing.Subscription) (billing.Subscription, error)
	ApplyProviderStatus(ctx context.Context, providerSubID string, status billing.Status, periodEnd time.Time) (billing.Subscription, error)
}

// Provider is the payment processor, as this service needs it. An
// interface so the gate, the trial and the screens can all be built and
// tested without keys — and so a different processor is one
// implementation rather than a rewrite.
type Provider interface {
	Configured() bool
	Start(ctx context.Context, plan billing.Plan, totalCount int) (billing.StartedSubscription, error)
	VerifyWebhook(rawBody []byte, signature string) bool
}

// totalCount is the number of billing cycles Razorpay is asked for.
// Their API has no "until cancelled", so this is ten years of monthly
// charges — longer than the subscription will plausibly run.
const totalCount = 120

type Billing struct {
	store    Store
	provider Provider
	logger   *slog.Logger
	// now is injectable so the trial and expiry rules can be tested
	// without sleeping.
	now func() time.Time
}

func New(s Store, p Provider, logger *slog.Logger) *Billing {
	return &Billing{store: s, provider: p, logger: logger, now: time.Now}
}

// Payable reports whether a provider is configured. The billing screen
// asks, so it can say "payments are not set up yet" rather than offer
// a button that fails.
func (b *Billing) Payable() bool { return b.provider.Configured() }

// Current is the calling organisation's standing.
func (b *Billing) Current(ctx context.Context) (billing.Subscription, error) {
	return b.store.Get(ctx)
}

// Allows is the gate's question, asked once per request.
func (b *Billing) Allows(ctx context.Context, orgID string) (billing.Subscription, bool, error) {
	sub, err := b.store.GetFor(ctx, orgID)
	if err != nil {
		return billing.Subscription{}, false, err
	}
	return sub, sub.Allows(b.now()), nil
}

// StartTrial gives an organisation its trial, once.
//
// Refuses where a subscription already exists, so it cannot be used to
// extend a trial indefinitely by calling it again — which is the first
// thing anyone would try.
func (b *Billing) StartTrial(ctx context.Context, orgID string) (billing.Subscription, error) {
	existing, err := b.store.GetFor(ctx, orgID)
	if err != nil {
		return billing.Subscription{}, err
	}
	if existing.Status != billing.StatusNone {
		return existing, fmt.Errorf("this organisation has already had a trial")
	}
	return b.store.Upsert(ctx, billing.Subscription{
		OrgID:       orgID,
		Plan:        billing.Plans[0].ID,
		Status:      billing.StatusTrialing,
		TrialEndsAt: b.now().Add(billing.TrialPeriod),
	})
}

// Checkout creates the subscription at the provider and records it
// against the organisation, returning where to send the browser.
func (b *Billing) Checkout(ctx context.Context, orgID, planID string) (string, error) {
	plan, ok := billing.PlanByID(planID)
	if !ok {
		return "", fmt.Errorf("no such plan: %q", planID)
	}
	if !b.provider.Configured() {
		return "", billing.ErrNotConfigured
	}
	started, err := b.provider.Start(ctx, plan, totalCount)
	if err != nil {
		return "", err
	}

	// Recorded before the person pays, and deliberately not as active:
	// the webhook is what grants access. Writing the id now is what
	// lets that webhook find this row at all.
	current, err := b.store.GetFor(ctx, orgID)
	if err != nil {
		return "", err
	}
	if _, err := b.store.Upsert(ctx, billing.Subscription{
		OrgID:                  orgID,
		Plan:                   plan.ID,
		Status:                 statusWhileAwaitingPayment(current.Status),
		TrialEndsAt:            current.TrialEndsAt,
		CurrentPeriodEnd:       current.CurrentPeriodEnd,
		ProviderSubscriptionID: started.ID,
	}); err != nil {
		return "", err
	}
	return started.ShortURL, nil
}

// statusWhileAwaitingPayment keeps whatever access the organisation
// already had while the payment is in flight. Starting a checkout must
// not itself change entitlement in either direction: an org on day
// three of its trial keeps the trial, and an org with nothing gains
// nothing until the webhook says it paid.
func statusWhileAwaitingPayment(current billing.Status) billing.Status {
	if current == billing.StatusNone {
		return billing.StatusNone
	}
	return current
}

// ApplyWebhook records a provider event. The raw body and signature
// are checked here rather than in the handler so the rule travels with
// the decision it protects.
func (b *Billing) ApplyWebhook(ctx context.Context, rawBody []byte, signature string) error {
	if !b.provider.VerifyWebhook(rawBody, signature) {
		return fmt.Errorf("webhook signature does not verify")
	}
	ev, err := billing.ParseWebhook(rawBody)
	if err != nil {
		return err
	}
	status, changes := billing.StatusFromProvider(ev.Status)
	if !changes {
		// A step in a checkout, not a decision about access.
		b.logger.Info("billing: webhook ignored",
			"event", ev.Event, "providerStatus", ev.Status, "subscription", ev.SubscriptionID)
		return nil
	}
	sub, err := b.store.ApplyProviderStatus(ctx, ev.SubscriptionID, status, ev.CurrentPeriodEnd)
	if err != nil {
		if err == database.ErrNotFound {
			// An event for a subscription this app never created. Not
			// an error worth failing the delivery over — the provider
			// would retry it forever.
			b.logger.Warn("billing: webhook for an unknown subscription",
				"subscription", ev.SubscriptionID, "event", ev.Event)
			return nil
		}
		return err
	}
	b.logger.Info("billing: subscription updated",
		"org", sub.OrgID, "status", sub.Status, "event", ev.Event)
	return nil
}
