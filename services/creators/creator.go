package creators

import (
	"campaigntrackerpro/services/campaigns"
	"math"
	"sort"
	"time"
)

// Tier buckets creators by audience size. Comparing a nano creator's reach
// against a mega creator's is meaningless, so every creator metric in this
// file is normalised against the creator's own tier rather than against the
// whole fleet — this is the difference between "big account did big
// numbers" and "this creator over-delivered for what they are".
type Tier string

const (
	TierNano  Tier = "nano"
	TierMicro Tier = "micro"
	TierMid   Tier = "mid"
	TierMacro Tier = "macro"
	TierMega  Tier = "mega"
)

var TierOrder = []Tier{TierNano, TierMicro, TierMid, TierMacro, TierMega}

// TierFor buckets by follower count using the bands the Indian creator
// market generally uses.
func TierFor(followers int64) Tier {
	switch {
	case followers >= 10_000_000:
		return TierMega
	case followers >= 1_000_000:
		return TierMacro
	case followers >= 250_000:
		return TierMid
	case followers >= 50_000:
		return TierMicro
	default:
		return TierNano
	}
}

func TierLabel(t Tier) string {
	switch t {
	case TierNano:
		return "Nano"
	case TierMicro:
		return "Micro"
	case TierMid:
		return "Mid"
	case TierMacro:
		return "Macro"
	case TierMega:
		return "Mega"
	default:
		return string(t)
	}
}

type Creator struct {
	ID              string
	Name            string
	Role            string
	Initials        string
	Category        string
	Region          string
	Tier            Tier
	Followers       int64
	PrimaryPlatform string
	Languages       []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreatorPerformance is everything the roster and detail views need, all
// derived from the creator's own campaigns.
type CreatorPerformance struct {
	Creator       Creator
	Campaigns     int
	LiveCampaigns int
	TotalReach    float64
	TotalSpend    int64
	AvgReach      float64
	TierMedian    float64 // median campaign reach across this creator's tier
	TierIndex     float64 // AvgReach / TierMedian — "1.6x for their tier"
	CostPerLakh   float64 // ₹ per 1L reach — comparable across tiers
	Consistency   float64 // 0-1; how repeatable their delivery is
	// ConsistencyN is how many campaigns that score came from. Below 2 the
	// score is not meaningful and callers should present it as unknown
	// rather than as a perfect 1.
	ConsistencyN     int
	AudienceReachPct float64 // avg reach as a share of follower base
	Flagged          int
}

// TierMedianReach is the median campaign reach within a tier, the baseline
// every creator in that tier is measured against.
func TierMedianReach(campaignsByTier []float64) float64 {
	return campaigns.Median(campaignsByTier)
}

// Consistency scores how repeatable a creator's delivery is: 1 means every
// campaign landed at the same reach, approaching 0 means wild swings. It's
// 1 - the coefficient of variation, clamped — a creator who reliably does
// 20L is worth more to a planner than one averaging 20L via 2L and 38L.
func Consistency(reaches []float64) float64 {
	if len(reaches) < 2 {
		return 1
	}
	var sum float64
	for _, r := range reaches {
		sum += r
	}
	mean := sum / float64(len(reaches))
	if mean == 0 {
		return 0
	}
	var variance float64
	for _, r := range reaches {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(reaches))
	cv := math.Sqrt(variance) / mean
	score := 1 - cv
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// CostPerLakhReach is spend per 1 lakh of reach — the one efficiency number
// that compares a nano creator to a mega one honestly.
func CostPerLakhReach(spend int64, reachLakh float64) float64 {
	if reachLakh <= 0 {
		return 0
	}
	return float64(spend) / reachLakh
}

// AudienceReachPct is average campaign reach against the creator's follower
// base. Over 100% means they're reaching well beyond their own following —
// the content travelled.
func AudienceReachPct(avgReachLakh float64, followers int64) float64 {
	if followers <= 0 {
		return 0
	}
	return (avgReachLakh * 100000) / float64(followers) * 100
}

// RankCreators orders a roster by tier index descending — best performers
// relative to their own size first, not simply the biggest accounts.
func RankCreators(perf []CreatorPerformance) {
	sort.SliceStable(perf, func(i, j int) bool { return perf[i].TierIndex > perf[j].TierIndex })
}
