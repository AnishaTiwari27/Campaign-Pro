package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"campaigntrackerpro/services/creators"
)

type stubLookup struct {
	configured bool
	byHandle   map[string]creators.Profile
	errs       map[string]error
}

func (s stubLookup) Configured() bool { return s.configured }
func (s stubLookup) Lookup(_ context.Context, handle string) (creators.Profile, error) {
	if err, ok := s.errs[handle]; ok {
		return creators.Profile{}, err
	}
	p, ok := s.byHandle[handle]
	if !ok {
		return creators.Profile{}, errors.New("not found")
	}
	return p, nil
}

func TestSyncFollowersRefusesAMismatchedHandle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Two creators. The second's handle belongs to somebody else —
	// the ordinary consequence of a typo, and the one that matters:
	// it would attach another person's audience to a real name.
	lookup := stubLookup{
		configured: true,
		byHandle: map[string]creators.Profile{
			"bhuvan.bam22": {Username: "bhuvan.bam22", Name: "Bhuvan Bam", Followers: 17_200_000},
			"kushakapila":  {Username: "kushakapila", Name: "Kusha Kapila", Followers: 4_200_000},
		},
	}

	if !creators.SamePerson("Bhuvan Bam", lookup.byHandle["bhuvan.bam22"]) {
		t.Fatal("the matching case does not match — the rest of this test is meaningless")
	}
	if creators.SamePerson("Prajakta Koli", lookup.byHandle["kushakapila"]) {
		t.Error("a handle belonging to another person was accepted")
	}
	_ = logger
}

// With nothing configured the sync must say so rather than report a
// successful run over zero creators.
func TestSyncFollowersUnconfigured(t *testing.T) {
	c := &Creators{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := c.SyncFollowers(context.Background(), stubLookup{configured: false}, logger)
	if !errors.Is(err, creators.ErrInstagramNotConfigured) {
		t.Errorf("got %v, want ErrInstagramNotConfigured", err)
	}
}
