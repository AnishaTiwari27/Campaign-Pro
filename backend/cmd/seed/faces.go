package main

import (
	"context"
	"log"
)

// creatorFaces attaches a portrait to each seeded creator.
//
// The images are synthetic — StyleGAN faces of people who do not exist,
// committed under frontend/public/faces and served by the app itself
// rather than hotlinked. That is the one source that is honest here: the
// creators in this data are invented, so a real person's photograph beside
// an invented campaign, an invented spend figure and an approval trail
// would read as a record of an endorsement that never happened. A
// generated face carries the same recognition value and misrepresents
// nobody.
//
// Swap any file for a real creator's own photograph when the agency has
// one it may publish; the path is all this map holds.
var creatorFaces = map[string]string{
	"aanya-verma":    "/faces/aanya-verma.jpg",
	"dev-ahuja":      "/faces/dev-ahuja.jpg",
	"kabir-sehgal":   "/faces/kabir-sehgal.jpg",
	"meher-kapoor":   "/faces/meher-kapoor.jpg",
	"rohan-malhotra": "/faces/rohan-malhotra.jpg",
	"simran-bakshi":  "/faces/simran-bakshi.jpg",
	"vikram-oberoi":  "/faces/vikram-oberoi.jpg",
}

// applyCreatorFaces is called after the roster is written. A creator with
// no entry keeps the generated avatar, which is the correct rendering for
// someone whose likeness the agency cannot publish.
func applyCreatorFaces(ctx context.Context, set func(ctx context.Context, id, url string) error) {
	for id, url := range creatorFaces {
		if err := set(ctx, id, url); err != nil {
			log.Fatalf("set photo for %s: %v", id, err)
		}
	}
}
