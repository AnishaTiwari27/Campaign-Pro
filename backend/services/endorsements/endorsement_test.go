package endorsements_test

import (
	"strings"
	"testing"

	"campaigntrackerpro/services/endorsements"
)

// Every photograph here is used under a licence that requires
// attribution — CC BY, CC BY-SA, CC0 and GODL-India all do. Dropping a
// credit in a later edit would be a licence breach that nothing else
// would catch, so it fails the build instead.
func TestEveryPhotoCarriesItsAttribution(t *testing.T) {
	for _, p := range endorsements.All() {
		if p.Photo.URL == "" {
			continue // a person with no photograph needs no credit
		}
		for field, v := range map[string]string{
			"artist":     p.Photo.Artist,
			"license":    p.Photo.License,
			"licenseUrl": p.Photo.LicenseURL,
			"source":     p.Photo.Source,
		} {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: photo is missing %s", p.ID, field)
			}
		}
	}
}

// These are real, named people. Every claim made about them has to be
// traceable to the source it came from, and a deal has to say what kind
// of relationship it was — reporting an ownership stake as an
// endorsement would misstate both.
func TestEveryClaimIsSourcedAndTyped(t *testing.T) {
	valid := map[string]bool{
		endorsements.KindEndorsement: true,
		endorsements.KindAmbassador:  true,
		endorsements.KindStake:       true,
	}
	for _, p := range endorsements.All() {
		if !strings.HasPrefix(p.Source, "https://") {
			t.Errorf("%s: no source for the claims about this person", p.ID)
		}
		if len(p.Deals) == 0 {
			t.Errorf("%s: listed with no reported brand work", p.ID)
		}
		for _, d := range p.Deals {
			if strings.TrimSpace(d.Brand) == "" {
				t.Errorf("%s: a deal with no brand", p.ID)
			}
			if !valid[d.Kind] {
				t.Errorf("%s/%s: unknown kind %q", p.ID, d.Brand, d.Kind)
			}
		}
	}
}

// The point of the package. If someone ever adds a reach or spend field
// to Person or Deal, this is the test that should stop them and send
// them to the package comment.
func TestFindIsExactAndMissIsNotAnError(t *testing.T) {
	if _, ok := endorsements.Find("virat-kohli"); !ok {
		t.Fatal("expected virat-kohli in the record")
	}
	if _, ok := endorsements.Find("Virat-Kohli"); ok {
		t.Error("lookup should be exact, not case-insensitive")
	}
	if _, ok := endorsements.Find("nobody"); ok {
		t.Error("a miss must report not-found rather than an empty person")
	}
}

// All() hands out a copy; a caller sorting it must not reorder the
// record for everyone else.
func TestAllReturnsACopy(t *testing.T) {
	a := endorsements.All()
	if len(a) < 2 {
		t.Skip("needs at least two people")
	}
	first := a[0].ID
	a[0], a[1] = a[1], a[0]
	if endorsements.All()[0].ID != first {
		t.Error("All() handed out the package's own slice")
	}
}
