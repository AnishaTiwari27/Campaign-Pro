package models

import "testing"

func TestAnomalyReason(t *testing.T) {
	cases := []struct {
		name                                   string
		reach, medianReach, spend, medianSpend int64
		want                                   string
	}{
		{
			name: "high reach dominates", reach: 310, medianReach: 100, spend: 100, medianSpend: 100,
			want: "reach 3.1x category median",
		},
		{
			name: "low reach dominates", reach: 20, medianReach: 100, spend: 90, medianSpend: 100,
			want: "reach 0.2x category median",
		},
		{
			name: "high spend dominates", reach: 100, medianReach: 100, spend: 400, medianSpend: 100,
			want: "spend 4.0x category median",
		},
		{
			name: "low spend dominates", reach: 105, medianReach: 100, spend: 10, medianSpend: 100,
			want: "spend 0.1x category median",
		},
		{
			name: "tie-break picks the further-from-1.0 candidate", reach: 300, medianReach: 100, spend: 50, medianSpend: 100,
			want: "reach 3.0x category median", // reach distance 2.0 > spend distance 0.5
		},
		{
			name: "no usable median returns empty", reach: 100, medianReach: 0, spend: 100, medianSpend: 0,
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AnomalyReason(c.reach, c.medianReach, c.spend, c.medianSpend)
			if got != c.want {
				t.Fatalf("AnomalyReason(%d,%d,%d,%d) = %q, want %q", c.reach, c.medianReach, c.spend, c.medianSpend, got, c.want)
			}
		})
	}
}
