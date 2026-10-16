// Package api is the subscription service's HTTP surface.
package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/billing"
	"campaigntrackerpro/services/billing/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc    *service.Billing
	logger *slog.Logger
}

func New(svc *service.Billing, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// Routes mount behind the session guard but in front of the
// subscription gate — an organisation that cannot use the app is
// exactly the one that needs to reach the billing screen.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/billing", h.Current)
	r.Post("/billing/trial", h.StartTrial)
	r.Post("/billing/checkout", h.Checkout)
}

// WebhookRoutes mount outside the session guard. A provider has no
// session; the signature is what authenticates it.
func (h *Handlers) WebhookRoutes(r chi.Router) {
	r.Post("/billing/webhook", h.Webhook)
}

type planDTO struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Amount   int64    `json:"amountPaise"`
	Interval string   `json:"interval"`
	Blurb    string   `json:"blurb"`
	Includes []string `json:"includes"`
}

type subscriptionDTO struct {
	Plan   string `json:"plan"`
	Status string `json:"status"`
	// Allowed is the answer the client actually needs: whether the app
	// is usable. Sent rather than re-derived in the browser, so the
	// rule lives in one place and cannot drift.
	Allowed   bool   `json:"allowed"`
	DaysLeft  int    `json:"daysLeft"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	// Payable reports whether a provider is configured at all. Without
	// it the screen offers a button that cannot work.
	Payable bool      `json:"payable"`
	Plans   []planDTO `json:"plans"`
}

func (h *Handlers) Current(w http.ResponseWriter, r *http.Request) {
	sub, err := h.svc.Current(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDTO(sub, h.svc.Payable(), time.Now()))
}

func toDTO(sub billing.Subscription, payable bool, now time.Time) subscriptionDTO {
	out := subscriptionDTO{
		Plan:     sub.Plan,
		Status:   string(sub.Status),
		Allowed:  sub.Allows(now),
		DaysLeft: sub.DaysLeft(now),
		Payable:  payable,
	}
	if e := sub.Expires(); !e.IsZero() {
		out.ExpiresAt = e.Format(time.RFC3339)
	}
	for _, p := range billing.Plans {
		out.Plans = append(out.Plans, planDTO{
			ID: p.ID, Name: p.Name, Amount: p.AmountPaise,
			Interval: p.Interval, Blurb: p.Blurb, Includes: p.Includes,
		})
	}
	return out
}

func (h *Handlers) StartTrial(w http.ResponseWriter, r *http.Request) {
	org := scope.OrgFrom(r.Context())
	sub, err := h.svc.StartTrial(r.Context(), org.ID)
	if err != nil {
		httpx.WriteConflict(w, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDTO(sub, h.svc.Payable(), time.Now()))
}

type checkoutRequest struct {
	Plan string `json:"plan"`
}

func (h *Handlers) Checkout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	org := scope.OrgFrom(r.Context())
	url, err := h.svc.Checkout(r.Context(), org.ID, req.Plan)
	if err != nil {
		if err == billing.ErrNotConfigured {
			httpx.WriteConflict(w, "payments are not configured on this deployment yet")
			return
		}
		httpx.WriteValidationError(w, err.Error(), "plan")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"checkoutUrl": url})
}

// Webhook takes a provider event.
//
// The raw bytes are what the signature covers, so the body is read
// whole and handed over unparsed — decoding it first and re-encoding
// would change the bytes and nothing would ever verify.
func (h *Handlers) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.WriteValidationError(w, "could not read body", "")
		return
	}
	if err := h.svc.ApplyWebhook(r.Context(), body, r.Header.Get("X-Razorpay-Signature")); err != nil {
		// Deliberately terse. A webhook endpoint is unauthenticated and
		// reachable by anyone, so the response says nothing about why
		// it was refused.
		h.logger.Error("billing: webhook rejected", "err", err)
		httpx.WriteUnauthorized(w, "rejected")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
