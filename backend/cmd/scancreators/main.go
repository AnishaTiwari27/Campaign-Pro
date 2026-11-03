// Command scancreators reads each creator's recent public posts and
// records what they establish: today's follower and engagement
// snapshot, and the evidence of any disclosed brand work.
//
// Separate from syncfollowers on purpose. A follower sync reads two
// counters; this reads people's posts, which is a bigger ask and
// should be a deliberate act rather than a side effect of something
// smaller.
//
// Worth running on a schedule, because the snapshot table is worth
// exactly what its history is worth, and history only accrues to
// whoever started taking it. A discovery vendor's growth trend is
// snapshots they began before we did; there is no other difference.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	"campaigntrackerpro/services/creators"
	creatorsmod "campaigntrackerpro/services/creators/module"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken:       cfg.InstagramAccessToken,
		BusinessAccountID: cfg.InstagramBusinessAccount,
	})
	if !ig.Configured() {
		fmt.Fprint(os.Stderr, `Instagram is not configured. Set both and try again:

  INSTAGRAM_ACCESS_TOKEN         a token for an account you control
  INSTAGRAM_BUSINESS_ACCOUNT_ID  that account's Instagram professional account id

Optionally:

  BRAND_HANDLES                  comma-separated accounts worth naming in a
                                 caption, e.g. nykaa,boat.lifestyle

A handle not on that list still counts its disclosure; it just records
no brand against it, which is honest where guessing one would not be.
`)
		os.Exit(1)
	}

	// Longer than the follower sync's budget: this reads posts for
	// every creator, and the endpoint is rate-limited.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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

	// Not a request, so it spans organisations — the same stance the
	// seeder and the report worker take. Each write still names the
	// creator's own organisation rather than a guessed one.
	ctx = scope.SystemOrgIn(ctx, scope.DefaultOrg)

	res, err := creatorsMod.ScanCreators(ctx, ig, cfg.BrandHandles, time.Now().UTC(), logger)
	if err != nil {
		if errors.Is(err, creators.ErrInstagramNotConfigured) {
			logger.Error("instagram is not configured")
		} else {
			logger.Error("scan failed", "err", err, "scanned", res.Scanned)
		}
		os.Exit(1)
	}

	logger.Info("scan complete",
		"checked", res.Checked, "scanned", res.Scanned,
		"brandPosts", res.BrandPosts, "mismatched", res.Mismatched, "failed", res.Failed)
	if res.Mismatched > 0 {
		logger.Warn("handles that did not match the creator recorded against them — nothing was written for these",
			"count", res.Mismatched)
	}
}
