package outreach_test

import (
	"strings"
	"testing"
	"time"

	"campaigntrackerpro/services/outreach"
)

var now = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// consented builds a contact that is eligible, so each test can spoil
// exactly one thing and the failure names that thing.
func consented() outreach.Contact {
	return outreach.Contact{
		ID:          "c1",
		OrgID:       "org-a",
		CreatorID:   "creator-1",
		Email:       "hello@example.com",
		Source:      outreach.SourceCreatorSignup,
		LawfulBasis: outreach.BasisConsent,
		ConsentedAt: now.Add(-30 * 24 * time.Hour),
		Purpose:     []string{"campaign_pitch"},
	}
}

func TestAConsentedContactIsEligible(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	ok, reason := s.Eligible(consented(), "campaign_pitch", now)
	if !ok {
		t.Fatalf("a freshly consented contact must be mailable, got %q", reason)
	}
}

// The default has to be refusal. Most of the rows that will ever reach
// this function come out of a vendor export or a hand-typed form, and
// the common failure is a basis nobody filled in.
func TestAnUnrecordedBasisIsRefused(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	for _, basis := range []string{"", "unknown", "opt_out", "Consent", "CONSENT"} {
		c := consented()
		c.LawfulBasis = basis
		ok, reason := s.Eligible(c, "campaign_pitch", now)
		if ok {
			t.Errorf("basis %q was accepted — the check must fail closed", basis)
		}
		if reason != outreach.SkipNoBasis {
			t.Errorf("basis %q: reason = %q, want %q", basis, reason, outreach.SkipNoBasis)
		}
	}
}

func TestConsentMustActuallyBeRecorded(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	c := consented()
	c.ConsentedAt = time.Time{}
	ok, reason := s.Eligible(c, "campaign_pitch", now)
	if ok || reason != outreach.SkipNoConsent {
		t.Fatalf("claiming consent without a date must be refused, got ok=%v reason=%q", ok, reason)
	}
}

func TestStaleConsentExpires(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	c := consented()
	c.ConsentedAt = now.Add(-outreach.ConsentValidity - time.Hour)
	ok, reason := s.Eligible(c, "campaign_pitch", now)
	if ok || reason != outreach.SkipStaleConsent {
		t.Fatalf("consent past the validity window must be refused, got ok=%v reason=%q", ok, reason)
	}

	// Just inside it still counts, or the boundary would silently
	// disqualify everyone at exactly the limit.
	c.ConsentedAt = now.Add(-outreach.ConsentValidity + time.Hour)
	if ok, reason := s.Eligible(c, "campaign_pitch", now); !ok {
		t.Fatalf("consent inside the window must hold, got %q", reason)
	}
}

// Consent is per purpose. Someone who agreed to be listed on a roster
// has not agreed to be pitched paid work.
func TestConsentIsScopedToItsPurpose(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	c := consented()
	c.Purpose = []string{"roster_listing"}
	ok, reason := s.Eligible(c, "campaign_pitch", now)
	if ok || reason != outreach.SkipWrongPurpose {
		t.Fatalf("a pitch to a roster-only consent must be refused, got ok=%v reason=%q", ok, reason)
	}
}

// The rule that makes a bought list unusable until consent is
// collected, which is the whole point of holding provenance.
func TestLegitimateInterestDoesNotLaunderAVendorList(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	for _, src := range []string{outreach.SourceVendor, outreach.SourceManual, outreach.SourceCreatorSignup, ""} {
		c := consented()
		c.LawfulBasis = outreach.BasisLegitimateInterest
		c.Source = src
		c.Purpose = nil
		ok, reason := s.Eligible(c, "campaign_pitch", now)
		if ok {
			t.Errorf("source %q under legitimate interest was accepted — a purchase is not a permission", src)
		}
		if reason != outreach.SkipBadSource {
			t.Errorf("source %q: reason = %q, want %q", src, reason, outreach.SkipBadSource)
		}
	}

	// An address the creator published for enquiries is the one case
	// it does cover, and it needs no purpose list.
	c := consented()
	c.LawfulBasis = outreach.BasisLegitimateInterest
	c.Source = outreach.SourcePublicBusinessProfile
	c.Purpose = nil
	if ok, reason := s.Eligible(c, "campaign_pitch", now); !ok {
		t.Fatalf("a published business address must be mailable, got %q", reason)
	}
}

func TestUnusableAddressesAreRefused(t *testing.T) {
	s := outreach.NewScreen("org-a", nil)
	// The things that actually turn up in vendor exports and pasted
	// columns, rather than exotic RFC 5322 edge cases.
	for _, addr := range []string{"", "   ", "not an email", "email not available", "@example.com", "hello@", "hello@localhost", "a b@example.com"} {
		c := consented()
		c.Email = addr
		if ok, _ := s.Eligible(c, "campaign_pitch", now); ok {
			t.Errorf("%q was accepted as an address", addr)
		}
	}
}

func TestSuppressionStopsASend(t *testing.T) {
	s := outreach.NewScreen("org-a", []outreach.Suppression{
		{Email: "hello@example.com", OrgID: "org-a", Reason: "unsubscribed"},
	})
	ok, reason := s.Eligible(consented(), "campaign_pitch", now)
	if ok || reason != outreach.SkipSuppressed {
		t.Fatalf("a suppressed address must be refused, got ok=%v reason=%q", ok, reason)
	}
}

// Case is the part that gets this wrong in practice: the local part is
// technically case-sensitive and no real host treats it that way, so
// matching exactly would let Hello@ through a suppression on hello@.
func TestSuppressionIgnoresCaseAndSurroundingSpace(t *testing.T) {
	s := outreach.NewScreen("org-a", []outreach.Suppression{
		{Email: "  HELLO@Example.COM ", OrgID: "org-a", Reason: "complaint"},
	})
	c := consented()
	c.Email = "hello@example.com"
	if ok, _ := s.Eligible(c, "campaign_pitch", now); ok {
		t.Fatal("a differently-cased suppression must still match")
	}
}

// The tenancy rule, in both directions. An unsubscribe belongs to the
// sender it was sent to; a bounce belongs to everyone, because the
// sending reputation is shared.
func TestSuppressionScopeSeparatesTenantsButNotThePlatform(t *testing.T) {
	s := outreach.NewScreen("org-a", []outreach.Suppression{
		{Email: "hello@example.com", OrgID: "org-b", Reason: "unsubscribed"},
	})
	if ok, reason := s.Eligible(consented(), "campaign_pitch", now); !ok {
		t.Fatalf("another tenant's unsubscribe is not ours to honour, got %q", reason)
	}

	s = outreach.NewScreen("org-a", []outreach.Suppression{
		{Email: "hello@example.com", Reason: "hard_bounce"},
	})
	ok, reason := s.Eligible(consented(), "campaign_pitch", now)
	if ok {
		t.Fatal("a platform-wide suppression must bind every tenant")
	}
	if reason != outreach.SkipSuppressedPlatform {
		t.Errorf("reason = %q, want %q — the report has to distinguish the two", reason, outreach.SkipSuppressedPlatform)
	}
}

// A contact that is both unconsented and suppressed reports the
// permission problem, because that is the one a human can act on.
func TestThePermissionProblemIsReportedOverTheSuppression(t *testing.T) {
	s := outreach.NewScreen("org-a", []outreach.Suppression{
		{Email: "hello@example.com", Reason: "hard_bounce"},
	})
	c := consented()
	c.LawfulBasis = ""
	_, reason := s.Eligible(c, "campaign_pitch", now)
	if reason != outreach.SkipNoBasis {
		t.Fatalf("reason = %q, want %q", reason, outreach.SkipNoBasis)
	}
}

func TestRenderRefusesToShipAnUnfilledPlaceholder(t *testing.T) {
	_, err := outreach.Render("Hi {{name}}, {{brand}} is looking for creators.", map[string]string{
		"name": "Prajakta",
	})
	if err == nil {
		t.Fatal("a template with an unfilled placeholder must not send")
	}
	if !strings.Contains(err.Error(), "{{brand}}") {
		t.Errorf("the error must name the placeholder that was missing, got %v", err)
	}
}

func TestRenderFillsEveryPlaceholder(t *testing.T) {
	out, err := outreach.Render("Hi {{name}}, {{brand}} wants {{deliverable}}.", map[string]string{
		"name": "Prajakta", "brand": "Boat", "deliverable": "two reels",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hi Prajakta, Boat wants two reels." {
		t.Errorf("got %q", out)
	}
}

func TestWarmupRampsThenHoldsAtTheCeiling(t *testing.T) {
	p := outreach.DefaultPolicy()
	for _, tc := range []struct{ day, want int }{
		{1, 50}, {7, 50}, // first week
		{8, 100}, {14, 100}, // second
		{15, 200}, {22, 400}, {29, 800},
		{36, p.MaxPerDay}, // would be 1600, so the ceiling holds
		{400, p.MaxPerDay},
		{0, 50}, {-5, 50}, // a nonsense day is the start, not unlimited
	} {
		if got := p.WarmupCap(tc.day); got != tc.want {
			t.Errorf("day %d: cap = %d, want %d", tc.day, got, tc.want)
		}
	}
}

// An absurd day used to overflow the shift and come back as a number
// that happened to be under the ceiling.
func TestWarmupDoesNotOverflow(t *testing.T) {
	p := outreach.DefaultPolicy()
	for _, day := range []int{1 << 20, 1 << 40, 1<<62 - 1} {
		if got := p.WarmupCap(day); got != p.MaxPerDay {
			t.Errorf("day %d: cap = %d, want the ceiling %d", day, got, p.MaxPerDay)
		}
	}
}

func TestTokensAreUniqueAndUrlSafe(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		tok, err := outreach.NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if seen[tok] {
			t.Fatalf("token %q repeated — it is the unsubscribe secret", tok)
		}
		seen[tok] = true
		if strings.ContainsAny(tok, "+/= ") {
			t.Fatalf("token %q needs escaping in a URL", tok)
		}
		if len(tok) < 40 {
			t.Fatalf("token %q is short enough to guess", tok)
		}
	}
}
