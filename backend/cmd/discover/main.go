// Command discover searches YouTube for creators matching a brief and
// records what it finds as candidates for review.
//
// YouTube is here because it is the only platform that answers "who
// makes this kind of thing". Instagram has no creator search at any
// price, and its hashtag endpoint returns posts without saying who
// posted them — so this is the one free route to a creator nobody on
// the team had heard of.
//
// Nothing it finds reaches the roster. Every baseline the product
// computes is drawn from the roster, so unreviewed search results
// joining it would move those figures using channels nobody has
// looked at. A person promotes; discovery only proposes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	"campaigntrackerpro/services/creators"
	creatorsmod "campaigntrackerpro/services/creators/module"
)

func main() {
	var briefFile string
	var units int
	flag.StringVar(&briefFile, "briefs", "", "file of search briefs, one per line")
	flag.IntVar(&units, "quota", creators.DailyQuota, "quota units this run may spend")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	briefs, err := readBriefs(briefFile, flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(briefs) == 0 {
		fmt.Fprint(os.Stderr, `Nothing to search for. Give briefs as arguments or in a file:

  go run ./cmd/discover "dance creators bangalore" "fitness reels hindi"
  go run ./cmd/discover -briefs briefs.txt

Each brief costs 100 of the day's 10,000 quota units, so a default day
is 100 searches. Filling in the figures for the channels they return
costs 1 unit per fifty channels.
`)
		os.Exit(1)
	}

	budget := creators.NewBudget(units)
	yt := creators.NewYouTube(creators.YouTubeConfig{
		APIKey:   cfg.YouTubeAPIKey,
		Region:   cfg.YouTubeRegion,
		Language: cfg.YouTubeLanguage,
	}, budget)
	if !yt.Configured() {
		fmt.Fprint(os.Stderr, `YouTube is not configured. Set a Data API v3 key and try again:

  YOUTUBE_API_KEY       a Data API v3 key
  YOUTUBE_REGION        optional, defaults to IN
  YOUTUBE_LANGUAGE      optional, e.g. hi

The key is yours and the channels searched grant nothing — only their
public channel data is returned.
`)
		os.Exit(1)
	}

	if want := len(briefs) * creators.QuotaSearch; want > budget.Remaining {
		logger.Warn("more briefs than the budget covers; the run will stop part way",
			"briefs", len(briefs), "searchesAffordable", budget.SearchesLeft())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("cannot connect to the database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := database.New(pool)

	campaignsMod := campaignsmod.New(db, logger)
	creatorsMod := creatorsmod.New(db, campaignsMod.Service(), logger)
	ctx = scope.SystemOrgIn(ctx, scope.DefaultOrg)

	res, err := creatorsMod.Discover(ctx, yt, scope.DefaultOrg, briefs, logger)
	if err != nil {
		logger.Error("discovery failed", "err", err, "saved", res.Saved)
		os.Exit(1)
	}

	logger.Info("discovery complete",
		"briefs", res.Queries, "channels", res.Found, "saved", res.Saved,
		"quotaLeft", budget.Remaining, "searchesLeft", budget.SearchesLeft())
	if res.Exhausted {
		logger.Warn("stopped on the daily quota — briefs that did not run are not briefs that found nothing")
	}
}

// readBriefs takes them from a file or the command line. Blank lines
// and # comments are skipped so a brief list can be annotated.
func readBriefs(path string, args []string) ([]string, error) {
	if path == "" {
		return args, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return append(out, args...), nil
}
