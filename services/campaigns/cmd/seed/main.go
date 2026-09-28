// Command seed populates campaigns.campaigns with an initial, realistic
// spread of campaigns across every brand and person catalog-service knows
// about. Run once after 02_campaigns.sql, with catalog-service already up:
//
//	go run ./cmd/seed
package main

import (
	"context"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/services/campaigns/internal/catalogclient"
	"campaigntrackerpro/services/campaigns/internal/models"
	"campaigntrackerpro/services/campaigns/internal/store"
)

var regions = []string{"Mumbai", "Delhi NCR", "Bengaluru", "South Zone", "West Zone", "Pan-India"}
var adTypePlatform = map[string]string{
	"Social Media": "Instagram", "Influencer": "YouTube Creators", "Google Ads": "Google Search",
	"Display": "Google Display", "Video": "YouTube", "Performance": "Meta Ads",
}

func main() {
	ctx := context.Background()
	today := time.Now()

	pool, err := pgxpool.New(ctx, envOr("DATABASE_URL", "postgres://campaigns_service:campaigns_dev_pw@localhost:5432/campaign_tracker?sslmode=disable"))
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	s := store.New(pool)

	catalog := catalogclient.New(envOr("CATALOG_URL", "http://localhost:8081"))
	subjects, err := catalog.ListAll(ctx)
	if err != nil {
		log.Fatalf("fetch subjects from catalog-service: %v (is it running?)", err)
	}
	if len(subjects) == 0 {
		log.Fatal("catalog-service returned zero subjects — nothing to seed against")
	}
	log.Printf("seeding against %d subjects (brands + people)", len(subjects))

	adTypes := make([]string, 0, len(adTypePlatform))
	for k := range adTypePlatform {
		adTypes = append(adTypes, k)
	}
	sort.Strings(adTypes) // deterministic order — map iteration isn't

	const n = 60
	created := 0
	for i := 0; i < n; i++ {
		subj := subjects[i%len(subjects)]
		// Step by 1, not 3: len(adTypes)==6 and gcd(3,6)==3 means "i*3 % 6"
		// only ever lands on two of the six ad types — a real bug carried
		// over from the original frontend mock's identical formula.
		adType := adTypes[i%len(adTypes)]
		region := regions[(i*5)%len(regions)]

		daysAgo := (i * 137) % 150
		start := today.AddDate(0, 0, -daysAgo)
		end := start.AddDate(0, 0, 14+(i%21))

		reach := int64(40000 + (i*91117)%900000)
		factor := 0.8 + float64((i*13)%7)/10.0
		spend := int64(math.Round(float64(reach) / 22.0 * factor))

		c := models.Campaign{
			Subject: subj.Name, SubjectType: subj.Type, Category: subj.Category,
			Region: region, AdType: adType, Platform: adTypePlatform[adType],
			Start: start, End: end, Reach: reach, Spend: spend,
		}
		if _, err := s.Create(ctx, c, today); err != nil {
			log.Printf("skip row %d (%s): %v", i, subj.Name, err)
			continue
		}
		created++
	}
	log.Printf("seeded %d campaigns", created)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
