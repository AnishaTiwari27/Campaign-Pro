package billing_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"campaigntrackerpro/services/billing"
)

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// Anyone who learns the webhook URL can post to it, so the signature is
// the only thing standing between a stranger and a free subscription.
func TestVerifyWebhook(t *testing.T) {
	const secret = "whsec-test"
	body := `{"event":"subscription.charged"}`
	r := billing.NewRazorpay(billing.RazorpayConfig{
		KeyID: "k", KeySecret: "s", WebhookSecret: secret,
	})

	if !r.VerifyWebhook([]byte(body), sign(secret, body)) {
		t.Error("a correctly signed payload was rejected")
	}
	if r.VerifyWebhook([]byte(body), sign("another-secret", body)) {
		t.Error("a payload signed with the wrong secret was accepted")
	}
	if r.VerifyWebhook([]byte(body+" "), sign(secret, body)) {
		t.Error("a tampered body was accepted against the original signature")
	}
	if r.VerifyWebhook([]byte(body), "") {
		t.Error("an unsigned payload was accepted")
	}

	// No secret configured: nothing can be checked, so nothing is
	// trusted. Failing open here is the whole vulnerability.
	open := billing.NewRazorpay(billing.RazorpayConfig{KeyID: "k", KeySecret: "s"})
	if open.VerifyWebhook([]byte(body), sign(secret, body)) {
		t.Error("with no webhook secret set, a payload was accepted — this lets anyone " +
			"who finds the URL grant themselves a subscription")
	}
}

func TestParseWebhook(t *testing.T) {
	body := `{"event":"subscription.charged","payload":{"subscription":{"entity":{
		"id":"sub_123","status":"active","current_end":1797000000}}}}`
	ev, err := billing.ParseWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if ev.SubscriptionID != "sub_123" || ev.Status != "active" {
		t.Fatalf("parsed %+v", ev)
	}
	if !ev.CurrentPeriodEnd.Equal(time.Unix(1797000000, 0).UTC()) {
		t.Errorf("period end parsed as %v", ev.CurrentPeriodEnd)
	}

	// An event with no subscription cannot be applied to a row.
	if _, err := billing.ParseWebhook([]byte(`{"event":"ping","payload":{}}`)); err == nil {
		t.Error("an event carrying no subscription parsed without error")
	}
}

// The half-finished states are the dangerous ones: they must neither
// grant access nor revoke it.
func TestStatusFromProviderLeavesCheckoutStatesAlone(t *testing.T) {
	for _, s := range []string{"created", "authenticated", "pending", "", "something_new"} {
		if _, ok := billing.StatusFromProvider(s); ok {
			t.Errorf("%q was treated as an entitlement change; it would either sell access "+
				"for an uncharged subscription or lock someone out mid-checkout", s)
		}
	}
	for in, want := range map[string]billing.Status{
		"active":    billing.StatusActive,
		"halted":    billing.StatusPastDue,
		"cancelled": billing.StatusCancelled,
		"completed": billing.StatusCancelled,
		"expired":   billing.StatusCancelled,
	} {
		got, ok := billing.StatusFromProvider(in)
		if !ok || got != want {
			t.Errorf("%q -> (%v, %v), want (%v, true)", in, got, ok, want)
		}
	}
}
