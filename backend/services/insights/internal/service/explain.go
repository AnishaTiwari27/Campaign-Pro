// Package service builds the evidence, asks the model, and refuses
// anything it cannot stand behind.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"campaigntrackerpro/platform/llm"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/insights"
)

// CampaignReader is what this service needs of campaigns. Narrow, so
// insights cannot wander into the rest of that service.
type CampaignReader interface {
	All(ctx context.Context) ([]campaigns.Campaign, error)
	Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow
}

type Insights struct {
	campaigns CampaignReader
	model     llm.Client
	logger    *slog.Logger

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	exp insights.Explanation
	at  time.Time
}

// cacheTTL. An explanation is a function of figures that change at most
// as often as the worker ticks, and the call is the slowest and most
// expensive thing on the page. Ten minutes keeps a dashboard someone is
// clicking around from re-asking the same question.
const cacheTTL = 10 * time.Minute

func New(c CampaignReader, model llm.Client, logger *slog.Logger) *Insights {
	return &Insights{campaigns: c, model: model, logger: logger, cache: map[string]cached{}}
}

const systemPrompt = `You explain why an advertising campaign has been flagged, for a media buyer who is about to decide what to do about it.

Rules, in order of importance:
1. Use ONLY the figures listed under FACTS. Never calculate, estimate, round, or introduce any number that is not there.
2. When you mention a figure, write it exactly as it appears in FACTS, including its symbol and unit.
3. Two or three sentences. Say what is unusual, what it is unusual against, and what the buyer's options are.
4. Plain professional English. No bullet points, no headings, no preamble, no restating the campaign name.
5. If the facts do not support a clear diagnosis, say so plainly rather than speculating.`

// Explain returns the diagnosis for one campaign.
func (s *Insights) Explain(ctx context.Context, campaignID string) (insights.Explanation, error) {
	if cachedExp, ok := s.fromCache(campaignID); ok {
		return cachedExp, nil
	}

	ev, err := s.evidence(ctx, campaignID)
	if err != nil {
		return insights.Explanation{}, err
	}

	exp := insights.Explanation{CampaignID: campaignID, Facts: ev.Facts}

	text, err := s.ask(ctx, ev)
	switch {
	case err == nil:
		exp.Text, exp.Source = text, insights.SourceModel
	case errors.Is(err, llm.ErrNotConfigured):
		// Not an error worth logging loudly: it means nobody has set up
		// a key, which is the normal state of an optional feature.
		exp.Text, exp.Source = fallback(ev), insights.SourceFallback
	default:
		s.logger.Error("insights: model call failed, using the assembled explanation",
			"campaign", campaignID, "err", err)
		exp.Text, exp.Source = fallback(ev), insights.SourceFallback
	}

	s.store(campaignID, exp)
	return exp, nil
}

func (s *Insights) ask(ctx context.Context, ev insights.Evidence) (string, error) {
	if s.model == nil {
		return "", llm.ErrNotConfigured
	}
	var b strings.Builder
	fmt.Fprintf(&b, "CAMPAIGN: %s\nCATEGORY: %s\nREGION: %s\nSTATUS: %s\nFLAG: %s\n\nFACTS:\n",
		ev.Name, ev.Category, ev.Region, ev.Status, ev.FlagReason)
	for _, f := range ev.Facts {
		fmt.Fprintf(&b, "- %s: %s\n", f.Label, f.Display)
	}

	text, err := s.model.Complete(ctx, systemPrompt, b.String())
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)

	// The guard. A model writing about money produces plausible numbers
	// very easily, and a wrong one here would sit next to real figures
	// and be believed.
	if bad, ok := Verify(text, ev); !ok {
		return "", fmt.Errorf("model cited figures that are not in the evidence: %s", strings.Join(bad, ", "))
	}
	return text, nil
}

// fallback assembles a true sentence with no model involved.
//
// It names the two or three figures the flag actually turns on rather
// than listing everything, because the full list is rendered directly
// beneath it — the first version restated all ten and read as a dump
// of the panel below it.
func fallback(ev insights.Evidence) string {
	byLabel := map[string]string{}
	for _, f := range ev.Facts {
		byLabel[f.Label] = f.Display
	}
	has := func(labels ...string) bool {
		for _, l := range labels {
			if byLabel[l] == "" {
				return false
			}
		}
		return true
	}

	var b strings.Builder
	if ev.FlagReason == "" {
		return "No anomaly is currently flagged for this campaign."
	}
	fmt.Fprintf(&b, "Flagged as %q.", strings.ToLower(ev.FlagReason))

	switch {
	case strings.Contains(strings.ToLower(ev.FlagReason), "budget"),
		strings.Contains(strings.ToLower(ev.FlagReason), "pace"):
		if has("Budget spent", "Budget the plan expects by now") {
			fmt.Fprintf(&b, " It has spent %s of budget against the %s its flight plan expects by now.",
				byLabel["Budget spent"], byLabel["Budget the plan expects by now"])
		} else if has("Budget spent") {
			fmt.Fprintf(&b, " It has spent %s of budget.", byLabel["Budget spent"])
		}
	default:
		// Both reach flags — outlier and under-delivering — turn on the
		// same comparison, in opposite directions.
		if has("Reach", "Category median reach", "Reach against category median") {
			fmt.Fprintf(&b, " Reach of %s is %s its category median of %s.",
				byLabel["Reach"], byLabel["Reach against category median"], byLabel["Category median reach"])
		}
	}
	b.WriteString(" The figures it was judged on are below.")
	return b.String()
}

func (s *Insights) fromCache(id string) (insights.Explanation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cache[id]
	if !ok || time.Since(c.at) > cacheTTL {
		return insights.Explanation{}, false
	}
	return c.exp, true
}

func (s *Insights) store(id string, exp insights.Explanation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[id] = cached{exp: exp, at: time.Now()}
}
