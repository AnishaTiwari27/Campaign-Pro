package platform

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestMailer_SendUnconfiguredReturnsSentinelWithNoNetworkAttempt(t *testing.T) {
	m := NewMailer("", "", "", "", "reports@campaigntracker.dev")
	err := m.Send("someone@example.com", "subject", "body", "report.csv", []byte("a,b\n1,2\n"))
	if !errors.Is(err, ErrMailerNotConfigured) {
		t.Fatalf("err = %v, want ErrMailerNotConfigured", err)
	}
}

func TestMailer_ConfiguredIffHostAndPortSet(t *testing.T) {
	cases := []struct {
		name, host, port string
		want             bool
	}{
		{"both set", "localhost", "2525", true},
		{"no host", "", "2525", false},
		{"no port", "localhost", "", false},
		{"neither", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewMailer(c.host, c.port, "", "", "from@x.dev")
			if m.configured != c.want {
				t.Fatalf("configured = %v, want %v", m.configured, c.want)
			}
		})
	}
}

func TestBuildMessage_HasHeadersAndAttachment(t *testing.T) {
	msg, err := buildMessage("reports@campaigntracker.dev", "user@example.com", "Your campaign report",
		"Attached: 3 campaigns matching your filters.", "campaign-report.csv", []byte("Subject,Reach\nCRED,1000\n"))
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	s := string(msg)

	for _, want := range []string{
		"From: reports@campaigntracker.dev",
		"To: user@example.com",
		"Subject: Your campaign report",
		"Content-Type: multipart/mixed;",
		`filename="campaign-report.csv"`,
		"Attached: 3 campaigns matching your filters.",
		"CRED,1000",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("message missing %q\nfull message:\n%s", want, s)
		}
	}
}

func TestBuildMessage_AttachmentBytesRoundTrip(t *testing.T) {
	csv := []byte("Subject,Reach\nZepto,500000\n")
	msg, err := buildMessage("from@x.dev", "to@x.dev", "s", "body", "report.csv", csv)
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	if !bytes.Contains(msg, csv) {
		t.Fatalf("attachment bytes not found verbatim in built message")
	}
}
