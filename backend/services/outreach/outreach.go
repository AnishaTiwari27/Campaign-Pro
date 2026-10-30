// Package outreach is the public contract of the creator outreach
// service: who we may email, and what stops us.
//
// The service exists because "mail the whole list" is not a thing that
// can be built safely. The sending domain is shared across every
// tenant, so a single scraped-list blast gets the platform throttled
// and takes password resets and scheduled reports down with it. And an
// address whose origin nobody recorded cannot be defended under the
// DPDP Act, where the answer to "how did you get this" is the entire
// defence.
//
// So the decision of whether one address may be mailed is a pure
// function in this file, not a WHERE clause and not a convention. It
// fails closed on anything it does not recognise, and it reports why,
// because "412 sent, 88 skipped" is only useful if you can see that 61
// of the 88 had no recorded consent.
package outreach

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Lawful bases for holding and using an address.
//
// There is deliberately no value meaning "unknown". An unrecorded
// basis is a row that never gets mailed, rather than one that quietly
// does.
const (
	BasisConsent            = "consent"
	BasisLegitimateInterest = "legitimate_interest"
)

// Where an address came from. The source is not decoration: it decides
// which lawful basis is available, because the two are not
// interchangeable.
const (
	// SourceCreatorSignup — they typed it into our own form. The
	// strongest provenance there is.
	SourceCreatorSignup = "creator_signup"
	// SourcePublicBusinessProfile — published by the creator for
	// business enquiries, e.g. the contact address in their own bio or
	// media kit. This is the only source legitimate interest covers.
	SourcePublicBusinessProfile = "public_business_profile"
	// SourceVendor — bought from a discovery vendor. The vendor's
	// terms place the compliance obligation on us, and the creator
	// never published it to us, so this needs consent before use.
	SourceVendor = "vendor"
	// SourceManual — typed in by a user. Could be anything, so it is
	// treated as the weakest: consent required.
	SourceManual = "manual"
)

// ConsentValidity is how long a recorded consent is honoured before it
// has to be asked for again.
//
// This is a policy we chose, not a statutory deadline — the DPDP Act
// sets no expiry. Two years is here because a creator who agreed to be
// pitched in 2024 and has heard nothing since has not agreed to
// anything in 2026, and treating stale permission as live permission
// is how a defensible list becomes an indefensible one.
const ConsentValidity = 2 * 365 * 24 * time.Hour

// Reasons a recipient was skipped. Stored on the message row and shown
// in the batch report.
const (
	SkipNoBasis            = "no recorded lawful basis"
	SkipNoConsent          = "consent claimed but never recorded"
	SkipStaleConsent       = "consent older than our validity window"
	SkipWrongPurpose       = "not consented for this purpose"
	SkipBadSource          = "legitimate interest does not cover this source"
	SkipBadAddress         = "address is not a usable email"
	SkipSuppressed         = "suppressed"
	SkipSuppressedPlatform = "suppressed platform-wide"
)

// Contact is one address we hold for a creator.
type Contact struct {
	ID          string
	OrgID       string
	CreatorID   string
	Email       string
	Role        string
	Source      string
	LawfulBasis string
	// ConsentedAt is zero where the basis is not consent.
	ConsentedAt time.Time
	Purpose     []string
	VerifiedAt  time.Time
}

// Suppression is an address that must not be mailed again.
//
// OrgID empty means platform-wide, and that is the load-bearing case:
// hard bounces and spam complaints are facts about the shared sending
// reputation, so they bind every tenant. A tenant cannot clear one.
type Suppression struct {
	Email  string
	OrgID  string
	Reason string
}

// Screen answers whether an address may be mailed. Build it once per
// batch from the suppression rows, then ask it per recipient.
type Screen struct {
	// platform and perOrg hold lowercased addresses.
	platform map[string]string
	perOrg   map[string]string
}

// NewScreen indexes the suppressions that apply to one organisation.
func NewScreen(orgID string, rows []Suppression) *Screen {
	s := &Screen{platform: map[string]string{}, perOrg: map[string]string{}}
	for _, r := range rows {
		key := strings.ToLower(strings.TrimSpace(r.Email))
		switch {
		case r.OrgID == "":
			s.platform[key] = r.Reason
		case r.OrgID == orgID:
			s.perOrg[key] = r.Reason
		}
	}
	return s
}

// email is deliberately permissive — one @, something either side, a
// dot in the domain, no spaces. Validating an address properly is
// RFC 5322 and not worth it; this catches the cases that actually turn
// up in vendor exports and hand-typed rows, which are empty strings,
// names pasted into the wrong column, and "email not available".
var email = regexp.MustCompile(`^[^@\s]+@[^@\s.]+\.[^@\s]+$`)

// Eligible reports whether this contact may receive a message sent for
// the given purpose, and why not when it may not.
//
// Order matters: suppression is checked last, not first, so that a
// contact which is both unconsented and suppressed reports the
// permission problem. The suppression will still stop it; the reason a
// human needs to see is the one they can act on.
func (s *Screen) Eligible(c Contact, purpose string, now time.Time) (bool, string) {
	if !email.MatchString(strings.TrimSpace(c.Email)) {
		return false, SkipBadAddress
	}

	switch c.LawfulBasis {
	case BasisConsent:
		if c.ConsentedAt.IsZero() {
			return false, SkipNoConsent
		}
		if now.Sub(c.ConsentedAt) > ConsentValidity {
			return false, SkipStaleConsent
		}
		if !hasPurpose(c.Purpose, purpose) {
			return false, SkipWrongPurpose
		}
	case BasisLegitimateInterest:
		// Only an address the creator published themselves, for
		// business enquiries. A vendor-supplied personal address is
		// the case this rule exists to refuse: the vendor's licence
		// does not make it lawful for us to cold-mail it, and treating
		// a purchase as a permission is the single most common way
		// these lists become a liability.
		if c.Source != SourcePublicBusinessProfile {
			return false, SkipBadSource
		}
	default:
		return false, SkipNoBasis
	}

	key := strings.ToLower(strings.TrimSpace(c.Email))
	if _, ok := s.platform[key]; ok {
		return false, SkipSuppressedPlatform
	}
	if _, ok := s.perOrg[key]; ok {
		return false, SkipSuppressed
	}
	return true, ""
}

func hasPurpose(have []string, want string) bool {
	for _, p := range have {
		if p == want {
			return true
		}
	}
	return false
}

// Policy bounds how fast and how much we send.
type Policy struct {
	// MaxBatchSize caps one send run. Small on purpose: a pitch to 150
	// creators chosen for a brief is outreach, and the same message to
	// 15,000 is the thing that gets reported. The cap is the difference
	// expressed as code rather than as advice in a document.
	MaxBatchSize int
	// MinInterval is the pause between messages. Providers accept far
	// faster than this — the limit is reputational, not technical.
	MinInterval time.Duration
	// MaxPerDay is the ceiling once the domain is warm.
	MaxPerDay int
}

func DefaultPolicy() Policy {
	return Policy{MaxBatchSize: 200, MinInterval: 2 * time.Second, MaxPerDay: 1000}
}

// WarmupCap is how many messages a domain may send on a given day of
// its life, counting from 1.
//
// A new domain that sends a thousand cold emails on its first day is
// not treated as a new sender, it is treated as a compromised one.
// Volume has to climb slowly enough that the receiving side builds a
// reputation for it: 50 a day in the first week, doubling weekly until
// it reaches the policy ceiling.
func (p Policy) WarmupCap(day int) int {
	if day < 1 {
		day = 1
	}
	week := (day - 1) / 7
	// Shifting by an unbounded week overflows and comes back as
	// nonsense that happens to be under the ceiling, so the ramp is
	// clamped before the shift rather than after it.
	if week > 20 {
		return p.MaxPerDay
	}
	allowed := 50 << week // 50, 100, 200, 400, ...
	if allowed > p.MaxPerDay || allowed <= 0 {
		return p.MaxPerDay
	}
	return allowed
}

// Batch is one send run.
type Batch struct {
	ID         string
	OrgID      string
	CampaignID string
	Subject    string
	Body       string
	Purpose    string
	Status     string
	CreatedBy  string
	CreatedAt  time.Time
}

// Message is one intended recipient of a batch.
type Message struct {
	ID               string
	BatchID          string
	CreatorID        string
	Email            string
	Status           string
	SkipReason       string
	UnsubscribeToken string
	SentAt           time.Time
}

// Message statuses.
const (
	StatusQueued  = "queued"
	StatusSent    = "sent"
	StatusSkipped = "skipped"
	StatusFailed  = "failed"
	StatusBounced = "bounced"
	StatusReplied = "replied"
)

var placeholder = regexp.MustCompile(`\{\{[^}]*\}\}`)

// Render fills a template's {{placeholders}} from vars.
//
// It fails rather than sending when one is left over. An unfilled
// placeholder is not a cosmetic bug: "Hi {{name}}," is the single most
// recognisable tell of a mass mail, and one missing key would stamp it
// across the whole batch. Catching it here costs one send; catching it
// in the replies costs the list.
func Render(tmpl string, vars map[string]string) (string, error) {
	out := tmpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	if left := placeholder.FindString(out); left != "" {
		return "", fmt.Errorf("outreach: template still contains %s — refusing to send", left)
	}
	return out, nil
}

// NewToken mints an unsubscribe secret.
//
// 32 bytes from crypto/rand, one per message: a token that leaks
// through a forward or a link-rewriting proxy can then only unsubscribe
// the recipient it was issued to, and cannot be walked to reach
// anybody else's.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("outreach: mint unsubscribe token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
