package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// SMTPMailer sends the scheduled report digests for real.
//
// net/smtp rather than a provider SDK: the thing being sent is one small
// message with one CSV attached, every host worth using speaks SMTP, and
// a provider library would tie the deployment to an account that has to
// exist before the app will build.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	// Timeout bounds the whole conversation. Without it a wedged server
	// holds the report worker's goroutine open indefinitely, and the
	// worker is what fires every other tenant's reports.
	Timeout time.Duration
}

func (m SMTPMailer) addr() string { return m.Host + ":" + m.Port }

func (m SMTPMailer) Send(ctx context.Context, to []string, subject, attachmentName string, attachment []byte) error {
	if len(to) == 0 {
		// Not an error: a report with no recipients is a report nobody
		// asked to receive, and failing here would retry it forever.
		return nil
	}

	body, err := m.compose(to, subject, attachmentName, attachment)
	if err != nil {
		return fmt.Errorf("compose message: %w", err)
	}

	timeout := m.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- m.deliver(to, body) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// The goroutine is left to finish and discard its result rather
		// than leaking a half-open connection into the next send.
		return fmt.Errorf("smtp send to %s: %w", m.addr(), ctx.Err())
	}
}

func (m SMTPMailer) deliver(to []string, body []byte) error {
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	// smtp.SendMail issues STARTTLS whenever the server advertises it,
	// and PlainAuth refuses to hand over credentials on a connection
	// that is not encrypted — so an unencrypted server fails loudly
	// rather than leaking the password.
	if err := smtp.SendMail(m.addr(), auth, m.From, to, body); err != nil {
		return fmt.Errorf("smtp send to %s: %w", m.addr(), err)
	}
	return nil
}

// compose builds a multipart message: a short plain-text body and the
// report itself attached, base64-encoded with CRLF line endings because
// a bare LF is what makes a message arrive mangled or get scored as spam.
func (m SMTPMailer) compose(to []string, subject, attachmentName string, attachment []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	var head bytes.Buffer
	fmt.Fprintf(&head, "From: %s\r\n", m.From)
	fmt.Fprintf(&head, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&head, "Subject: %s\r\n", subject)
	fmt.Fprintf(&head, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	head.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&head, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", w.Boundary())

	text, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(text, "%s\r\n\r\nThe report is attached as %s.\r\n", subject, attachmentName)

	if len(attachment) > 0 {
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/csv; charset=utf-8"},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {fmt.Sprintf("attachment; filename=%q", attachmentName)},
		})
		if err != nil {
			return nil, err
		}
		if err := writeBase64Lines(part, attachment); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return append(head.Bytes(), buf.Bytes()...), nil
}

// writeBase64Lines wraps at 76 characters, which RFC 2045 requires and
// some servers enforce by rejecting the message outright.
func writeBase64Lines(w interface{ Write([]byte) (int, error) }, data []byte) error {
	const lineLen = 76
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 0 {
		n := lineLen
		if len(encoded) < n {
			n = len(encoded)
		}
		if _, err := w.Write([]byte(encoded[:n] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[n:]
	}
	return nil
}
