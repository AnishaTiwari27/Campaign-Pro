package reports

import "time"

// Cadence is one of the 4 fixed report schedules — not a cron expression.
// Four presets cover every case the spec's Reports screen offers, so real
// calendar-aware "next occurrence" math per preset is enough; a general
// cron parser would be unused generality.
type Cadence string

const (
	CadenceWeeklyMon9 Cadence = "weekly_mon_9"
	CadenceWeekday830 Cadence = "weekday_830"
	CadenceMonthly19  Cadence = "monthly_1_9"
	CadenceOnFlag     Cadence = "on_flag"
)

var ValidCadences = []Cadence{CadenceWeeklyMon9, CadenceWeekday830, CadenceMonthly19, CadenceOnFlag}

func IsValidCadence(c string) bool {
	for _, v := range ValidCadences {
		if string(v) == c {
			return true
		}
	}
	return false
}

// CadenceLabel is the human label the Reports screen's cadence select shows.
func CadenceLabel(c Cadence) string {
	switch c {
	case CadenceWeeklyMon9:
		return "Every Monday 9:00 AM IST"
	case CadenceWeekday830:
		return "Every weekday 8:30 AM IST"
	case CadenceMonthly19:
		return "First of the month 9:00 AM IST"
	case CadenceOnFlag:
		return "Within 15 min of a flag"
	default:
		return string(c)
	}
}

var istLocation = loadIST()

func loadIST() *time.Location {
	if loc, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		return loc
	}
	// tzdata may be unavailable in a minimal container; IST has no DST so a
	// fixed UTC+5:30 offset is exact, not an approximation.
	return time.FixedZone("IST", 5*3600+30*60)
}

// NextOccurrence returns the next IST instant, strictly after `after`, that
// cadence should fire. on_flag has no calendar schedule — the worker fires
// it reactively when a flag is written, so this returns the zero time.
func NextOccurrence(c Cadence, after time.Time) time.Time {
	ist := after.In(istLocation)
	switch c {
	case CadenceWeeklyMon9:
		return nextWeekdayAt(ist, time.Monday, 9, 0)
	case CadenceWeekday830:
		return nextBusinessDayAt(ist, 8, 30)
	case CadenceMonthly19:
		return nextMonthlyAt(ist, 1, 9, 0)
	default:
		return time.Time{}
	}
}

func nextWeekdayAt(from time.Time, weekday time.Weekday, hour, min int) time.Time {
	daysUntil := (int(weekday) - int(from.Weekday()) + 7) % 7
	candidate := time.Date(from.Year(), from.Month(), from.Day()+daysUntil, hour, min, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

func nextBusinessDayAt(from time.Time, hour, min int) time.Time {
	candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, min, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	for isWeekend(candidate.Weekday()) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

func isWeekend(d time.Weekday) bool { return d == time.Saturday || d == time.Sunday }

func nextMonthlyAt(from time.Time, day, hour, min int) time.Time {
	candidate := time.Date(from.Year(), from.Month(), day, hour, min, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = time.Date(from.Year(), from.Month()+1, day, hour, min, 0, 0, from.Location())
	}
	return candidate
}
