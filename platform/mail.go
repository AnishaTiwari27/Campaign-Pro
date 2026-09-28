package platform

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
)

// ErrMailerNotConfigured is returned by Mailer.Send when no SMTP host was
// given — no network attempt is made. Same "degrade, don't crash" contract
// Cache and EventBus already document for infrastructure this app can run
// without (see cache.go, events.go): a caller decides what to do with this
// (campaigns-service's email-report route reports {"sent": false,
// "reason": ...} rather than a 500 — this is an expected, not exceptional,
// state in an environment with no SMTP credentials configured).
var ErrMailerNotConfigured = errors.New("smtp not configured")

// Mailer sends a plain-text email with one attachment over SMTP. No new
// dependency — hand-builds a minimal multipart/mixed MIME message
// (mime/multipart, stdlib) and delivers it with net/smtp.SendMail, same
// restraint this codebase already shows elsewhere (hand-rolled CSV writing
// instead of a CSV library, hand-rolled CORS instead of a middleware one).
type Mailer struct {
	host, port, user, pass, from string
	configured                   bool
}

// NewMailer never fails synchronously — same reasoning ConnectCache
// documents for Redis: construction can't fail, only sending can, and
// even that degrades to ErrMailerNotConfigured rather than a runtime
// error when host is empty (the common case in dev, where no SMTP relay
// exists). Callers never need to nil-check a *Mailer — always construct
// one, even with an empty host.
func NewMailer(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from, configured: host != "" && port != ""}
}

// Send builds a multipart/mixed message (a short text body plus one
// attachment) and delivers it over SMTP. No TLS/STARTTLS is attempted —
// fine for a local/dev relay, not a real production one; auth is only
// attempted when user is non-empty. Returns ErrMailerNotConfigured
// immediately, before touching the network, when host is empty.
func (m *Mailer) Send(to, subject, textBody, attachmentName string, attachment []byte) error {
	if !m.configured {
		return ErrMailerNotConfigured
	}

	msg, err := buildMessage(m.from, to, subject, textBody, attachmentName, attachment)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(m.host+":"+m.port, auth, m.from, []string{to}, msg)
}

// buildMessage is split out from Send so the MIME-building logic is
// unit-testable without a network call (see mail_test.go) — the same
// "pure computation separate from I/O" split PacingOf/models.AnomalyReason
// already follow, just for a byte-building function instead of a formula.
func buildMessage(from, to, subject, textBody, attachmentName string, attachment []byte) ([]byte, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	textPart, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return nil, err
	}

	attachPart, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/csv"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename=%q`, attachmentName)},
		"Content-Transfer-Encoding": {"7bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := attachPart.Write(attachment); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%q\r\n", w.Boundary())
	fmt.Fprintf(&msg, "\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}
