package creators

import "testing"

func TestTierFor(t *testing.T) {
	cases := []struct {
		followers int64
		want      Tier
	}{
		{5_000, TierNano},
		{49_999, TierNano},
		{50_000, TierMicro},
		{249_999, TierMicro},
		{250_000, TierMid},
		{999_999, TierMid},
		{1_000_000, TierMacro},
		{9_999_999, TierMacro},
		{10_000_000, TierMega},
	}
	for _, c := range cases {
		if got := TierFor(c.followers); got != c.want {
			t.Fatalf("TierFor(%d) = %v, want %v", c.followers, got, c.want)
		}
	}
}

func TestConsistency(t *testing.T) {
	// Identical delivery every time -> perfectly consistent.
	if got := Consistency([]float64{20, 20, 20}); got != 1 {
		t.Fatalf("identical reaches = %v, want 1", got)
	}
	// A single campaign can't vary.
	if got := Consistency([]float64{20}); got != 1 {
		t.Fatalf("single campaign = %v, want 1", got)
	}
	// Same mean, wild swings -> markedly lower than the steady creator.
	steady := Consistency([]float64{19, 20, 21})
	swingy := Consistency([]float64{2, 20, 38})
	if swingy >= steady {
		t.Fatalf("swingy (%v) should score below steady (%v)", swingy, steady)
	}
	if swingy < 0 || swingy > 1 {
		t.Fatalf("consistency out of range: %v", swingy)
	}
	// Zero mean is undefined rather than divide-by-zero.
	if got := Consistency([]float64{0, 0}); got != 0 {
		t.Fatalf("zero mean = %v, want 0", got)
	}
}

func TestCostPerLakhReach(t *testing.T) {
	// ₹10,00,000 for 20L reach -> ₹50,000 per lakh.
	if got := CostPerLakhReach(1000000, 20); got != 50000 {
		t.Fatalf("CostPerLakhReach = %v, want 50000", got)
	}
	if got := CostPerLakhReach(1000, 0); got != 0 {
		t.Fatalf("zero reach = %v, want 0", got)
	}
}

func TestAudienceReachPct(t *testing.T) {
	// 5L reach against 500k followers = exactly 100%.
	if got := AudienceReachPct(5, 500_000); got != 100 {
		t.Fatalf("AudienceReachPct = %v, want 100", got)
	}
	// Reaching beyond the follower base reads above 100.
	if got := AudienceReachPct(10, 500_000); got != 200 {
		t.Fatalf("AudienceReachPct = %v, want 200", got)
	}
	if got := AudienceReachPct(5, 0); got != 0 {
		t.Fatalf("zero followers = %v, want 0", got)
	}
}

func TestRankCreatorsOrdersByTierIndex(t *testing.T) {
	perf := []CreatorPerformance{
		{Creator: Creator{ID: "a"}, TierIndex: 0.8},
		{Creator: Creator{ID: "b"}, TierIndex: 2.1},
		{Creator: Creator{ID: "c"}, TierIndex: 1.4},
	}
	RankCreators(perf)
	if perf[0].Creator.ID != "b" || perf[1].Creator.ID != "c" || perf[2].Creator.ID != "a" {
		t.Fatalf("unexpected order: %v %v %v", perf[0].Creator.ID, perf[1].Creator.ID, perf[2].Creator.ID)
	}
}
