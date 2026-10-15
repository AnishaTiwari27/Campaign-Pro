package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"campaigntrackerpro/platform/llm"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/insights"
)

type stubModel struct {
	reply string
	err   error
	calls int
}

func (s *stubModel) Complete(context.Context, string, string) (string, error) {
	s.calls++
	return s.reply, s.err
}

type stubCampaigns struct{}

func (stubCampaigns) All(context.Context) ([]campaigns.Campaign, error) {
	return []campaigns.Campaign{{
		ID: "c1", Name: "Blinkit", Category: "E-commerce", Region: "Delhi NCR",
		Status: campaigns.StatusLive, FlagReason: "Reach outlier",
		Reach: 150, Spend: 5_000_000, Budget: 8_000_000, DaysRunning: 12, FlightDays: 30,
	}}, nil
}
func (stubCampaigns) Enrich(context.Context, []campaigns.Campaign) map[string]campaigns.CampaignRow {
	return map[string]campaigns.CampaignRow{"c1": {Pace: 62, ExpectedPace: 40, CategoryIndex: 1.8, CPM: 210}}
}

func svc(t *testing.T, m llm.Client) *Insights {
	t.Helper()
	return New(stubCampaigns{}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// A model that quotes only supplied figures is used as written.
func TestAGroundedExplanationIsUsed(t *testing.T) {
	m := &stubModel{reply: "Reach is running at 1.8x the category median on 62% of budget, well ahead of the 40% the plan expects by now."}
	exp, err := svc(t, m).Explain(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if exp.Source != insights.SourceModel {
		t.Errorf("source %q, want model — the text was grounded", exp.Source)
	}
}

// The case the whole feature turns on. An invented figure must never
// reach the reader, however fluent the sentence around it.
func TestAnInventedFigureFallsBackToTheAssembledText(t *testing.T) {
	m := &stubModel{reply: "Spend has reached ₹61.4L, which is ahead of plan."}
	exp, err := svc(t, m).Explain(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if exp.Source != insights.SourceFallback {
		t.Fatalf("source %q — an unverified figure was shown to the reader", exp.Source)
	}
	if strings.Contains(exp.Text, "61.4") {
		t.Error("the invented figure survived into the fallback")
	}
}

// A provider outage degrades to the assembled sentence rather than an
// error page: the figures are on the screen either way.
func TestAProviderFailureDegradesRatherThanFailing(t *testing.T) {
	exp, err := svc(t, &stubModel{err: errors.New("502 from provider")}).Explain(context.Background(), "c1")
	if err != nil {
		t.Fatalf("a model outage should not fail the request: %v", err)
	}
	if exp.Source != insights.SourceFallback || exp.Text == "" {
		t.Errorf("expected an assembled explanation, got %+v", exp)
	}
}

// With no key at all the panel still says something true.
func TestNoModelConfiguredStillExplains(t *testing.T) {
	exp, err := svc(t, nil).Explain(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if exp.Source != insights.SourceFallback || !strings.Contains(exp.Text, "reach outlier") {
		t.Errorf("got %+v", exp)
	}
}

// The evidence is rendered by the app's own formatters, so the model
// never formats a number and the figures match what is on screen.
func TestEvidenceCarriesRenderedFiguresNotRawNumbers(t *testing.T) {
	ev, err := svc(t, nil).evidence(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Budget spent": "62%", "Spend": "₹50.0L", "Reach": "1.50Cr"}
	got := map[string]string{}
	for _, f := range ev.Facts {
		got[f.Label] = f.Display
	}
	for label, display := range want {
		if got[label] != display {
			t.Errorf("%s = %q, want %q", label, got[label], display)
		}
	}
}

// Asking twice does not pay for the model twice.
func TestTheSecondAskIsServedFromCache(t *testing.T) {
	m := &stubModel{reply: "Reach is 1.8x the category median."}
	s := svc(t, m)
	for i := 0; i < 3; i++ {
		if _, err := s.Explain(context.Background(), "c1"); err != nil {
			t.Fatal(err)
		}
	}
	if m.calls != 1 {
		t.Errorf("called the model %d times for the same campaign, want 1", m.calls)
	}
}

// A campaign the caller cannot see is simply absent from All(), so the
// answer is not-found rather than a leak.
func TestAnUnknownCampaignIsNotFound(t *testing.T) {
	if _, err := svc(t, nil).Explain(context.Background(), "nope"); err == nil {
		t.Error("expected not-found for a campaign outside the caller's visibility")
	}
}
