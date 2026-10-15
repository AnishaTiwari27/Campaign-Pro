// Command verifyingest makes one real call to an ad platform and
// reports what came back.
//
// The platform clients are written from published references and their
// mapping is unit-tested, but nothing here has been run against a live
// account — getting a Google developer token or Meta ads_read access
// takes an application and a review. This is the command that closes
// that gap: point it at a real account and it either prints campaigns
// or prints exactly why not.
//
//	make verify-ingest SOURCE=google-ads
//	make verify-ingest SOURCE=meta
//
// It reads only. Nothing is written to the database, so running it
// against production credentials cannot damage anything.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/services/ingest"
)

func main() {
	source := flag.String("source", "", "google-ads | meta")
	days := flag.Int("days", 30, "how far back to ask for")
	flag.Parse()

	cfg := config.Load()
	providers := map[string]ingest.Provider{
		"google-ads": ingest.GoogleAdsProvider{Config: ingest.GoogleAdsConfig{
			DeveloperToken:  cfg.GoogleAdsDeveloperToken,
			ClientID:        cfg.GoogleAdsClientID,
			ClientSecret:    cfg.GoogleAdsClientSecret,
			RefreshToken:    cfg.GoogleAdsRefreshToken,
			CustomerID:      cfg.GoogleAdsCustomerID,
			LoginCustomerID: cfg.GoogleAdsLoginCustomerID,
		}},
		"meta": ingest.MetaProvider{Config: ingest.MetaConfig{
			AccessToken: cfg.MetaAccessToken,
			AdAccountID: cfg.MetaAdAccountID,
		}},
	}

	p, ok := providers[*source]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown source %q. Try: google-ads, meta\n", *source)
		os.Exit(2)
	}

	if c, ok := p.(interface{ Configured() bool }); ok && !c.Configured() {
		fmt.Fprintf(os.Stderr, "%s is not configured. Set its credentials and try again:\n", *source)
		fmt.Fprintln(os.Stderr, envHelp(*source))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	started := time.Now()
	rows, err := p.Fetch(ctx, time.Now().AddDate(0, 0, -*days))
	elapsed := time.Since(started).Round(time.Millisecond)

	if err != nil {
		if errors.Is(err, ingest.ErrNotConfigured) {
			fmt.Fprintf(os.Stderr, "%s is not configured.\n%s\n", *source, envHelp(*source))
			os.Exit(1)
		}
		// The platform's own message, unwrapped, because that is what
		// tells you whether it is the token, the account id or the
		// permission that is wrong.
		fmt.Fprintf(os.Stderr, "FAILED after %s\n  %v\n", elapsed, err)
		os.Exit(1)
	}

	fmt.Printf("OK — %s returned %d campaigns in %s\n\n", *source, len(rows), elapsed)
	if len(rows) == 0 {
		fmt.Println("No campaigns in the window. Widen it with -days, or check the account has activity.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EXTERNAL ID\tNAME\tSTATUS\tAD TYPE\tREACH (L)\tSPEND\tBUDGET")
	for i, r := range rows {
		if i == 15 {
			fmt.Fprintf(w, "…\t(%d more)\t\t\t\t\t\n", len(rows)-15)
			break
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f\t%d\t%d\n",
			r.ExternalID, truncate(r.Name, 34), r.Status, r.AdType, r.ReachLakh, r.Spend, r.Budget)
	}
	w.Flush()

	fmt.Println("\nCheck these against the platform's own dashboard before trusting a sync:")
	fmt.Println("  - do the spend figures match, in rupees rather than micros or paise?")
	fmt.Println("  - is each campaign listed once, not once per day?")
	fmt.Println("  - are statuses and ad types mapped to this product's vocabulary?")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func envHelp(source string) string {
	switch source {
	case "google-ads":
		return "  GOOGLE_ADS_DEVELOPER_TOKEN  (applied for, issued per manager account)\n" +
			"  GOOGLE_ADS_CLIENT_ID\n  GOOGLE_ADS_CLIENT_SECRET\n" +
			"  GOOGLE_ADS_REFRESH_TOKEN\n  GOOGLE_ADS_CUSTOMER_ID\n" +
			"  GOOGLE_ADS_LOGIN_CUSTOMER_ID  (only when the account sits under an MCC)"
	case "meta":
		return "  META_ACCESS_TOKEN   (system user or page token with ads_read)\n" +
			"  META_AD_ACCOUNT_ID  (with or without the act_ prefix)"
	}
	return ""
}
