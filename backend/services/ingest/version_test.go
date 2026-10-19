package ingest

import "testing"

// A pinned API version is right — a version change can move field
// semantics underneath a sync. A pin nobody rechecks is just a stale
// default, and both of these had gone stale unnoticed: Google sat on
// v18 and Meta on v21.0, either of which would have failed on the
// version in the URL at the first live call.
//
// The lists below are what each vendor served in October 2026. The
// test does not know what is current — it knows what it was told —
// so a failure here means "go and look", not "the code is wrong".
func TestPinnedAPIVersionsAreOnesTheVendorStillServes(t *testing.T) {
	t.Run("google ads", func(t *testing.T) {
		supported := map[string]bool{"v24": true, "v25": true, "v26": true}
		if got := (GoogleAdsProvider{}).apiVersion(); !supported[got] {
			t.Errorf("pinned at %q, which Google does not list as supported — "+
				"check developers.google.com/google-ads/api/docs/sunset-dates", got)
		}
	})

	t.Run("meta", func(t *testing.T) {
		// Meta guarantees a version for two years from release, so
		// anything older than about four versions back is at or past
		// its expiry.
		supported := map[string]bool{"v24.0": true, "v25.0": true, "v26.0": true}
		if got := (MetaProvider{}).apiVersion(); !supported[got] {
			t.Errorf("pinned at %q, which is outside the versions Meta still serves — "+
				"check developers.facebook.com/docs/graph-api/changelog", got)
		}
	})
}
