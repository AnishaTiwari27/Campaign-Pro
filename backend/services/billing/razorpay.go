package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNotConfigured is returned when no keys are set. Distinct from a
// transport failure: "you have not set this up" and "Razorpay is down"
// need different responses, and reporting the second when it is the
// first sends people to look at the wrong thing.
var ErrNotConfigured = errors.New("razorpay is not configured")

// RazorpayConfig is what the API needs before it will answer.
//
// Razorpay rather than Stripe because this product prices in rupees for
// Indian agencies, and UPI and netbanking are how those invoices
// actually get paid.
type RazorpayConfig struct {
	KeyID     string
	KeySecret string
	// WebhookSecret verifies that a webhook came from Razorpay. Without
	// it anyone who learns the URL can mark any subscription active, so
	// the handler refuses every event when it is unset rather than
	// trusting the body.
	WebhookSecret string
	// PlanIDs maps a plan in this app to the plan created in the
	// Razorpay dashboard. Their ids are generated, so they cannot be
	// derived and have to be configured.
	PlanIDs map[string]string
	BaseURL string
	HTTP    *http.Client
}

func (c RazorpayConfig) complete() bool { return c.KeyID != "" && c.KeySecret != "" }

// Razorpay talks to the subscriptions API.
//
// NOT YET RUN AGAINST THE LIVE API. The request shapes and the
// signature check are written from Razorpay's published reference and
// covered by tests against recorded payloads, but nobody has pointed
// this at a real account, because that needs keys. Treat it as
// unproven until it has taken one real payment.
type Razorpay struct{ cfg RazorpayConfig }

func NewRazorpay(cfg RazorpayConfig) *Razorpay { return &Razorpay{cfg: cfg} }

func (r *Razorpay) Configured() bool { return r.cfg.complete() }

func (r *Razorpay) baseURL() string {
	if r.cfg.BaseURL != "" {
		return r.cfg.BaseURL
	}
	return "https://api.razorpay.com/v1"
}

func (r *Razorpay) client() *http.Client {
	if r.cfg.HTTP != nil {
		return r.cfg.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// StartedSubscription is what the browser needs to open the checkout.
type StartedSubscription struct {
	ID       string
	ShortURL string
	Status   string
	PlanID   string
}

type razorpaySubscription struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ShortURL   string `json:"short_url"`
	PlanID     string `json:"plan_id"`
	CurrentEnd int64  `json:"current_end"`
}

// Start creates a subscription and returns the hosted checkout link.
//
// The hosted page rather than an embedded form on purpose: it keeps
// card details off this origin entirely, which is the difference
// between being in PCI scope and not.
func (r *Razorpay) Start(ctx context.Context, plan Plan, totalCount int) (StartedSubscription, error) {
	if !r.cfg.complete() {
		return StartedSubscription{}, ErrNotConfigured
	}
	providerPlan := r.cfg.PlanIDs[plan.ID]
	if providerPlan == "" {
		return StartedSubscription{}, fmt.Errorf("no Razorpay plan id configured for %q", plan.ID)
	}

	body, err := json.Marshal(map[string]any{
		"plan_id": providerPlan,
		// Razorpay requires a bound count rather than "until
		// cancelled"; 120 months is a subscription nobody outlives.
		"total_count":     totalCount,
		"customer_notify": 1,
	})
	if err != nil {
		return StartedSubscription{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL()+"/subscriptions", bytes.NewReader(body))
	if err != nil {
		return StartedSubscription{}, err
	}
	req.SetBasicAuth(r.cfg.KeyID, r.cfg.KeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client().Do(req)
	if err != nil {
		return StartedSubscription{}, fmt.Errorf("razorpay: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return StartedSubscription{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StartedSubscription{}, fmt.Errorf("razorpay: subscription create returned %s: %s", resp.Status, payload)
	}
	var out razorpaySubscription
	if err := json.Unmarshal(payload, &out); err != nil {
		return StartedSubscription{}, fmt.Errorf("razorpay: malformed response: %w", err)
	}
	return StartedSubscription{ID: out.ID, ShortURL: out.ShortURL, Status: out.Status, PlanID: out.PlanID}, nil
}

// VerifyWebhook reports whether a payload really came from Razorpay.
//
// HMAC-SHA256 of the raw body under the webhook secret, compared in
// constant time. The raw body matters: re-marshalling the JSON first
// would change the bytes and the signature would never match.
func (r *Razorpay) VerifyWebhook(rawBody []byte, signature string) bool {
	if r.cfg.WebhookSecret == "" {
		// No secret means nothing can be verified, so nothing is
		// trusted. Failing open here would let anyone who finds the
		// URL grant themselves a subscription.
		return false
	}
	mac := hmac.New(sha256.New, []byte(r.cfg.WebhookSecret))
	mac.Write(rawBody)
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(signature))
}

// WebhookEvent is the part of a Razorpay event this app acts on.
type WebhookEvent struct {
	Event            string
	SubscriptionID   string
	Status           string
	CurrentPeriodEnd time.Time
}

type razorpayWebhook struct {
	Event   string `json:"event"`
	Payload struct {
		Subscription struct {
			Entity razorpaySubscription `json:"entity"`
		} `json:"subscription"`
	} `json:"payload"`
}

// ParseWebhook pulls out what matters. It does not verify; call
// VerifyWebhook on the raw bytes first.
func ParseWebhook(rawBody []byte) (WebhookEvent, error) {
	var w razorpayWebhook
	if err := json.Unmarshal(rawBody, &w); err != nil {
		return WebhookEvent{}, err
	}
	e := w.Payload.Subscription.Entity
	if e.ID == "" {
		return WebhookEvent{}, fmt.Errorf("webhook %q carries no subscription", w.Event)
	}
	ev := WebhookEvent{Event: w.Event, SubscriptionID: e.ID, Status: e.Status}
	if e.CurrentEnd > 0 {
		ev.CurrentPeriodEnd = time.Unix(e.CurrentEnd, 0).UTC()
	}
	return ev, nil
}

// StatusFromProvider maps Razorpay's vocabulary onto this app's,
// reporting whether the event changes entitlement at all.
//
// The false case is the important one. "created", "authenticated" and
// "pending" mean a subscription exists but has not been charged — they
// are steps in a checkout, not a decision about access. Mapping them to
// any status would be wrong in both directions: to an active-ish one
// and an unpaid subscription buys entry, to a dead one and an
// organisation gets locked out halfway through paying. Returning false
// leaves the row exactly as it was, which for someone mid-checkout is
// their trial, and for everyone else is nothing gained.
func StatusFromProvider(s string) (Status, bool) {
	switch s {
	case "active":
		return StatusActive, true
	case "halted":
		return StatusPastDue, true
	case "cancelled", "completed", "expired":
		return StatusCancelled, true
	default:
		return StatusNone, false
	}
}
