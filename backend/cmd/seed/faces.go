package main

import (
	"context"
	"log"
)

// creatorFaces attaches a portrait to each person in the roster.
//
// These are real photographs of the real people they depict, the same
// freely-licensed images the public record uses — CC BY, CC BY-SA and
// GODL-India from Wikimedia Commons, downloaded and served by this app
// rather than hotlinked. Every one requires attribution, which is
// rendered on that person's public-record page.
//
// This replaced a set of synthetic StyleGAN faces, which were the
// right answer while the roster was invented people: a real face
// beside an invented endorsement would have claimed something that
// never happened. Now the roster is real people and their documented
// brand relationships, so the real photograph is the honest one. What
// stays illustrative is the performance data, and the dashboard says
// so where those figures appear.
var creatorFaces = map[string]string{
	"virat-kohli":      "/celebs/virat-kohli.jpg",
	"alia-bhatt":       "/celebs/alia-bhatt.jpg",
	"salman-khan":      "/celebs/salman-khan.jpg",
	"deepika-padukone": "/celebs/deepika-padukone.jpg",
	"ranveer-singh":    "/celebs/ranveer-singh.jpg",
	"ms-dhoni":         "/celebs/ms-dhoni.jpg",
	"bhuvan-bam":       "/celebs/bhuvan-bam.jpg",
	"prajakta-koli":    "/celebs/prajakta-koli.jpg",
	"kusha-kapila":     "/celebs/kusha-kapila.jpg",
}

// applyCreatorFaces sets each creator's photograph after the roster is
// written. A creator with no entry keeps the generated avatar.
func applyCreatorFaces(ctx context.Context, setPhoto func(context.Context, string, string) error) {
	for id, url := range creatorFaces {
		if err := setPhoto(ctx, id, url); err != nil {
			log.Fatalf("set photo for %s: %v", id, err)
		}
	}
}
