package service

import (
	"regexp"
	"strings"

	"campaigntrackerpro/services/insights"
)

// numberish finds anything a reader would take as a figure, together
// with whatever word follows it. The trailing run is captured rather
// than matched against the unit list directly because RE2 has no
// lookahead: without it, "the top 3 creatives" matched as "3 cr" and
// was reported as an invented crore. The suffix is sorted out below.
//
// The decimal part is (?:\.\d+)? and not \.?\d* for the same class of
// reason — the looser form swallowed the full stop in "running 18."
// and then failed to match the figure it had just read.
var numberish = regexp.MustCompile(`(?i)₹?\d[\d,]*(?:\.\d+)?\s*[a-z%]*`)

// units a figure may legitimately carry. Anything else trailing a
// number is the next word, not a unit.
var unitSet = map[string]bool{"": true, "cr": true, "l": true, "k": true, "x": true, "%": true}

// splitFigure separates a match into its number and its unit, dropping
// a trailing word that is not a unit.
func splitFigure(m string) string {
	i := len(m)
	for i > 0 {
		r := m[i-1]
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			break
		}
		i--
	}
	num, rest := m[:i], strings.ToLower(strings.TrimSpace(m[i:]))
	if unitSet[rest] {
		return normalise(num + rest)
	}
	return normalise(num)
}

// Verify reports whether every figure in the text was supplied in the
// evidence.
//
// This is the guard the whole feature rests on. A model writing about
// money is extremely good at producing a number that looks right, and
// a wrong one here would sit beside real figures and be believed. So
// nothing numeric survives that did not come from the database.
//
// Comparison is on normalised strings rather than parsed values: the
// model is instructed to quote the rendered figure exactly, so "₹98.0L"
// must appear as "₹98.0L" and not as "98 lakh". That is strict, and
// strict is the point — a model paraphrasing a number is a model
// one step from changing it.
func Verify(text string, ev insights.Evidence) (bad []string, ok bool) {
	allowed := map[string]bool{}
	for _, f := range ev.Facts {
		allowed[normalise(f.Display)] = true
		// A figure may legitimately be written without its unit when
		// the sentence supplies it ("reached 1.80Cr against a median
		// of 41.0L" — both forms are in the evidence anyway).
		allowed[normalise(stripUnits(f.Display))] = true
	}
	// Small integers are allowed unconditionally: "two creatives",
	// "day 18", "the top 3". These cannot misstate a monetary figure,
	// and banning them makes ordinary prose impossible.
	for _, m := range numberish.FindAllString(text, -1) {
		n := splitFigure(m)
		if n == "" || allowed[n] {
			continue
		}
		if bare, isInt := smallInteger(n); isInt && bare <= 31 {
			continue
		}
		bad = append(bad, strings.TrimSpace(m))
	}
	return bad, len(bad) == 0
}

func normalise(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "")
	return s
}

func stripUnits(s string) string {
	s = strings.TrimPrefix(strings.TrimSpace(s), "₹")
	for _, suffix := range []string{"Cr", "cr", "L", "l", "K", "k", "%", "x"} {
		s = strings.TrimSuffix(s, suffix)
	}
	return s
}

// smallInteger reports a bare integer with no unit attached.
func smallInteger(n string) (int, bool) {
	if n == "" {
		return 0, false
	}
	v := 0
	for _, r := range n {
		if r < '0' || r > '9' {
			return 0, false
		}
		v = v*10 + int(r-'0')
		if v > 10000 {
			return 0, false
		}
	}
	return v, true
}
