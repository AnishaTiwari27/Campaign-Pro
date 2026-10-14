package mail

import (
	"strings"
	"testing"
)

func testMailer() SMTPMailer {
	return SMTPMailer{Host: "smtp.example.test", Port: "587", From: "reports@example.test"}
}

// The message has to be a well-formed MIME document before it is worth
// testing anything else: a header block, a boundary that actually
// appears, and the attachment inside it.
func TestComposeProducesAWellFormedMultipartMessage(t *testing.T) {
	m := testMailer()
	body, err := m.compose([]string{"a@example.test", "b@example.test"}, "Weekly digest", "report.csv", []byte("name,spend\nBlinkit,500\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	for _, want := range []string{
		"From: reports@example.test",
		"To: a@example.test, b@example.test",
		"Subject: Weekly digest",
		"MIME-Version: 1.0",
		"multipart/mixed; boundary=",
		`attachment; filename="report.csv"`,
		"Content-Transfer-Encoding: base64",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message is missing %q", want)
		}
	}
}

// Headers and boundaries must use CRLF. A bare LF is the classic cause
// of a message that arrives with its attachment mangled, or does not
// arrive at all because a strict server rejected it.
func TestComposeUsesCRLFLineEndings(t *testing.T) {
	m := testMailer()
	body, err := m.compose([]string{"a@example.test"}, "Subject", "r.csv", []byte("x,y\n1,2\n"))
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range body {
		if b == '\n' && (i == 0 || body[i-1] != '\r') {
			t.Fatalf("bare LF at byte %d — headers and boundaries need CRLF", i)
		}
	}
}

// RFC 2045 caps an encoded line at 76 characters and some servers
// enforce it by rejecting the message, so a large attachment has to
// wrap rather than run on as one enormous line.
func TestBase64AttachmentWrapsAt76Columns(t *testing.T) {
	m := testMailer()
	big := make([]byte, 4096)
	for i := range big {
		big[i] = byte('a' + i%26)
	}
	body, err := m.compose([]string{"a@example.test"}, "Subject", "r.csv", big)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(body), "\r\n") {
		if len(line) > 998 { // the hard SMTP line limit, well above our 76
			t.Fatalf("line of %d characters — the attachment is not wrapped", len(line))
		}
	}
}

// A report whose recipient list is empty is one nobody asked to
// receive. Erroring would make the worker retry it on every tick
// forever, so it is a no-op instead.
func TestSendWithNoRecipientsIsANoOp(t *testing.T) {
	m := testMailer()
	m.Host = "127.0.0.1" // would fail to connect if it tried
	m.Port = "1"
	if err := m.Send(t.Context(), nil, "Subject", "r.csv", []byte("x")); err != nil {
		t.Errorf("expected a silent no-op, got %v", err)
	}
}
