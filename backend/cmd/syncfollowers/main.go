// Command syncfollowers refreshes creator follower counts from
// Instagram, replacing the seeded press-article figures with what the
// platform reports.
//
// A command rather than a background worker, deliberately. Follower
// counts move slowly, the endpoint is rate-limited, and the first
// thing anyone needs from this is to watch it run once and see which
// handles were right — which a loop on a six-hour tick does not give
// you.
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

Business discovery is queried through your own account. The creators
looked up grant nothing and are not involved; only their public
professional profile is returned.
`)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
	// seeder and the report worker take.
	res, err := creatorsMod.SyncFollowers(scope.SystemOrgIn(ctx, scope.DefaultOrg), ig, logger)
	if err != nil {
		if errors.Is(err, creators.ErrInstagramNotConfigured) {
			logger.Error("instagram is not configured")
		} else {
			logger.Error("follower sync failed", "err", err)
		}
		os.Exit(1)
	}

	fmt.Printf("\nChecked %d, updated %d.\n", res.Checked, res.Updated)
	if len(res.Mismatched) > 0 {
		fmt.Printf("\nNot written — the account did not look like the person it is recorded against:\n")
		for _, id := range res.Mismatched {
			fmt.Printf("  %s\n", id)
		}
		fmt.Print("\nThat is the guard working. Check the handle before trusting the figure.\n")
	}
	if len(res.Failed) > 0 {
		fmt.Printf("\nCould not be read (personal, private, age-gated, or the handle is wrong):\n")
		for _, id := range res.Failed {
			fmt.Printf("  %s\n", id)
		}
	}
	if res.Checked == 0 {
		fmt.Print("\nNo creator has an Instagram handle set, so there was nothing to look up.\n" +
			"See cmd/seed/handles.go for why most are deliberately unset.\n")
	}
}
