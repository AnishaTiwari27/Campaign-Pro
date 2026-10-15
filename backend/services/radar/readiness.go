package radar

import (
	"fmt"
	"strings"
	"time"
)

// Thresholds. Written here, in one place, as plain numbers rather than
// buried in the logic — they are judgement calls, not measurements,
// and the first thing anyone checking this estimate should be able to
// find and disagree with.
const (
	// freshDays is how recently something must have launched to count
	// as about to push. Four weeks is the window in which a launch is
	// still news.
	freshDays = 28
	// staleDays is when a launch stops being a prospect for a launch
	// campaign and becomes an ordinary company.
	staleDays = 120
	// tractionFloor is the traction below which the source is telling
	// us almost nothing. Deliberately low: it separates "nobody
	// noticed" from "some signal", not good from bad.
	tractionFloor = 25
)

// Assess bands a launch and says why.
//
// Deliberately a pure function of the stored signals: given the same
// launch it returns the same band, which is what makes the estimate
// auditable and lets it be recomputed when the rules change.
func Assess(l Launch, now time.Time) (Readiness, string) {
	age := int(now.Sub(l.LaunchedAt).Hours() / 24)
	if age < 0 {
		age = 0
	}

	var signals []string
	signals = append(signals, fmt.Sprintf("launched %s", humanAge(age)))
	if l.Traction > 0 {
		signals = append(signals, fmt.Sprintf("%d on %s", l.Traction, l.Source))
	}
	if l.HasSite {
		signals = append(signals, "has a site")
	} else {
		signals = append(signals, "no site found")
	}
	basis := strings.Join(signals, ", ")

	switch {
	case l.LaunchedAt.IsZero():
		// No launch date means the main signal is missing, and a band
		// computed without it would be a guess wearing a label.
		return ReadinessUnclear, "no launch date from the source"
	case age > staleDays:
		return ReadinessQuiet, basis + " — past the window for a launch push"
	case age <= freshDays && l.Traction >= tractionFloor && l.HasSite:
		return ReadinessSoon, basis
	case age <= freshDays:
		return ReadinessBuilding, basis
	case l.Traction >= tractionFloor:
		return ReadinessBuilding, basis
	default:
		return ReadinessQuiet, basis
	}
}

func humanAge(days int) string {
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 14:
		return fmt.Sprintf("%d days ago", days)
	case days < 60:
		return fmt.Sprintf("%d weeks ago", days/7)
	default:
		return fmt.Sprintf("%d months ago", days/30)
	}
}
