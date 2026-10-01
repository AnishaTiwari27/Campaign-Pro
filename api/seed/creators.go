package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A creator's campaigns are endorsements of brands, so each one is named
// "{Creator} × {Brand}". Giving every creator several campaigns is what
// makes tier index and consistency mean anything — a single data point
// tells you nothing about how reliably someone delivers.
type creatorCampaignSeed struct {
	brand         string
	adType        domain.AdType
	platform      string
	status        domain.Status
	daysRunning   int
	reach         float64
	spend, budget int64
	frequency     float64
	approval      domain.Approval
	curve         domain.CurveShape
	language      string
	hook          string
}

type creatorSeed struct {
	id, name, role, initials string
	category, region         string
	followers                int64
	platform                 string
	languages                []string
	campaigns                []creatorCampaignSeed
}

// Fictional people, deliberately. Attaching invented spend and performance
// figures to a real, named individual is a different kind of claim than
// illustrative numbers against a brand.
func creatorSeeds() []creatorSeed {
	return []creatorSeed{
		{
			id: "rohan-malhotra", name: "Rohan Malhotra", role: "Actor", initials: "RM",
			category: "Entertainment", region: "Mumbai", followers: 2_800_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Lakmé", domain.AdInfluencer, "Instagram", domain.StatusLive, 3, 2, 350000, 3000000, 1.2, domain.ApprovalPending, domain.CurveSlow, "Hindi", "story"},
				{"boAt", domain.AdInfluencer, "Instagram", domain.StatusEnded, 24, 41, 2900000, 3200000, 2.1, domain.ApprovalApproved, domain.CurveFast, "Hindi", "unboxing"},
				{"Myntra", domain.AdVideo, "YouTube", domain.StatusEnded, 18, 36, 2400000, 2800000, 1.9, domain.ApprovalApproved, domain.CurveSteady, "English", "demo"},
			},
		},
		{
			id: "kabir-sehgal", name: "Kabir Sehgal", role: "Cricketer", initials: "KS",
			category: "Sports", region: "Pan-India", followers: 14_200_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Dream11", domain.AdInfluencer, "Instagram", domain.StatusEnded, 22, 85, 6100000, 7000000, 2.4, domain.ApprovalRejected, domain.CurveFast, "Hindi", "announcement"},
				{"boAt", domain.AdVideo, "YouTube", domain.StatusLive, 14, 78, 5400000, 6500000, 2.2, domain.ApprovalApproved, domain.CurveFast, "Hindi", "demo"},
				{"CRED", domain.AdInfluencer, "Instagram", domain.StatusLive, 9, 92, 6800000, 8000000, 2.5, domain.ApprovalApproved, domain.CurveFast, "English", "testimonial"},
			},
		},
		{
			id: "aanya-verma", name: "Aanya Verma", role: "Singer", initials: "AV",
			category: "Music", region: "Mumbai", followers: 6_200_000,
			platform: "YouTube", languages: []string{"Hindi", "Punjabi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Nykaa", domain.AdVideo, "YouTube", domain.StatusEnded, 20, 58, 4900000, 5500000, 2.6, domain.ApprovalRejected, domain.CurveSteady, "Hindi", "story"},
				{"Tanishq", domain.AdVideo, "YouTube", domain.StatusLive, 11, 64, 4200000, 5000000, 2.3, domain.ApprovalApproved, domain.CurveSteady, "Punjabi", "story"},
			},
		},
		{
			id: "vikram-oberoi", name: "Vikram Oberoi", role: "CEO, Northgate", initials: "VO",
			category: "Business", region: "Bengaluru", followers: 180_000,
			platform: "LinkedIn", languages: []string{"English"},
			campaigns: []creatorCampaignSeed{
				{"PhonePe", domain.AdSocial, "LinkedIn", domain.StatusLive, 19, 26, 2100000, 3500000, 1.6, domain.ApprovalApproved, domain.CurveSteady, "English", "testimonial"},
				{"Urban Company", domain.AdSocial, "LinkedIn", domain.StatusEnded, 26, 22, 1700000, 2000000, 1.4, domain.ApprovalApproved, domain.CurveSlow, "English", "announcement"},
			},
		},
		{
			id: "meher-kapoor", name: "Meher Kapoor", role: "Creator", initials: "MK",
			category: "Influencer", region: "Delhi NCR", followers: 640_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Mamaearth", domain.AdInfluencer, "Instagram", domain.StatusLive, 7, 20, 1100000, 2500000, 1.5, domain.ApprovalPending, domain.CurveFast, "Hindi", "unboxing"},
				{"Zepto", domain.AdInfluencer, "Instagram", domain.StatusLive, 16, 31, 1900000, 2400000, 1.8, domain.ApprovalApproved, domain.CurveSteady, "Hindi", "offer"},
				{"Lenskart", domain.AdSocial, "Instagram", domain.StatusEnded, 21, 27, 1600000, 1900000, 1.7, domain.ApprovalApproved, domain.CurveSteady, "English", "demo"},
				{"Nykaa", domain.AdInfluencer, "Instagram", domain.StatusLive, 5, 24, 1400000, 2200000, 1.6, domain.ApprovalApproved, domain.CurveFast, "Hindi", "unboxing"},
			},
		},
		{
			id: "simran-bakshi", name: "Simran Bakshi", role: "Actor", initials: "SB",
			category: "Entertainment", region: "Delhi NCR", followers: 3_500_000,
			platform: "YouTube", languages: []string{"Hindi", "Punjabi"},
			campaigns: []creatorCampaignSeed{
				{"Myntra", domain.AdVideo, "YouTube", domain.StatusLive, 12, 50, 3400000, 5000000, 2.0, domain.ApprovalApproved, domain.CurveSteady, "Hindi", "story"},
				{"Tanishq", domain.AdVideo, "YouTube", domain.StatusEnded, 23, 47, 3100000, 3600000, 1.9, domain.ApprovalApproved, domain.CurveSlow, "Punjabi", "story"},
			},
		},
		{
			id: "dev-ahuja", name: "Dev Ahuja", role: "Founder, Finlytics", initials: "DA",
			category: "Business", region: "Pan-India", followers: 95_000,
			platform: "LinkedIn", languages: []string{"English"},
			campaigns: []creatorCampaignSeed{
				{"CRED", domain.AdSocial, "LinkedIn", domain.StatusScheduled, 0, 0, 0, 3000000, 0, domain.ApprovalApproved, domain.CurveSteady, "English", "announcement"},
				{"PhonePe", domain.AdSocial, "LinkedIn", domain.StatusLive, 13, 17, 1200000, 1800000, 1.3, domain.ApprovalApproved, domain.CurveSteady, "English", "demo"},
			},
		},
	}
}

var seedSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugOf(s string) string {
	return strings.Trim(seedSlugPattern.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// seedCreators writes creators and their campaigns, linking each campaign
// back to its creator so the roster can benchmark them.
func seedCreators(ctx context.Context, pool *pgxpool.Pool, st *store.Store) int {
	count := 0
	for _, cs := range creatorSeeds() {
		tier := domain.TierFor(cs.followers)
		if _, err := st.CreateCreator(ctx, domain.Creator{
			ID: cs.id, Name: cs.name, Role: cs.role, Initials: cs.initials,
			Category: cs.category, Region: cs.region, Tier: tier,
			Followers: cs.followers, PrimaryPlatform: cs.platform, Languages: cs.languages,
		}); err != nil {
			log.Fatalf("create creator %s: %v", cs.id, err)
		}

		for _, cc := range cs.campaigns {
			campaignID := fmt.Sprintf("%s-%s", cs.id, slugOf(cc.brand))
			name := fmt.Sprintf("%s × %s", cs.name, cc.brand)

			created, err := st.CreateCampaign(ctx, domain.Campaign{
				ID: campaignID, Name: name, SubjectType: domain.SubjectPerson, Role: cs.role,
				Initials: cs.initials, Category: cs.category, Region: cs.region,
				AdType: cc.adType, Platform: cc.platform, Status: cc.status,
				DaysRunning: cc.daysRunning, Reach: cc.reach, Spend: cc.spend, Budget: cc.budget,
				Frequency: cc.frequency, Approval: cc.approval, CurveShape: cc.curve,
			})
			if err != nil {
				log.Fatalf("create creator campaign %s: %v", campaignID, err)
			}
			if err := st.SetCampaignCreator(ctx, created.ID, cs.id); err != nil {
				log.Fatalf("link campaign %s to creator: %v", campaignID, err)
			}

			for _, cr := range creativesFor(string(cc.adType), cc.reach) {
				if _, err := st.CreateCreative(ctx, domain.Creative{
					CampaignID: created.ID, Headline: cr.headline, Kind: cr.kind,
					DurationLabel: cr.duration, Reach: cr.reach, CTR: cr.ctr,
					Language: cc.language, HookType: cc.hook,
					Claim: fmt.Sprintf("%s endorsement", cc.brand),
				}); err != nil {
					log.Fatalf("create creative for %s: %v", campaignID, err)
				}
			}

			startedAt := time.Now().AddDate(0, 0, -cc.daysRunning)
			events := []auditSpec{
				{"System", "Campaign created", "sys", startedAt},
				{"System", "Creative assets uploaded", "sys", startedAt.Add(time.Hour)},
			}
			switch cc.approval {
			case domain.ApprovalApproved:
				events = append(events, auditSpec{"You", "Approved", "user", startedAt.Add(2 * time.Hour)})
			case domain.ApprovalRejected:
				events = append(events, auditSpec{"You", "Rejected", "user", startedAt.Add(2 * time.Hour)})
			}
			for _, ev := range events {
				if err := insertAuditAt(ctx, pool, created.ID, ev.actor, ev.action, ev.kind, ev.at); err != nil {
					log.Fatalf("audit for %s: %v", campaignID, err)
				}
			}
			count++
		}
	}
	return count
}
