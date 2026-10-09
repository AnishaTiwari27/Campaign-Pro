package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"campaigntrackerpro/services/campaigns"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	"campaigntrackerpro/services/creators"
	creatorsmod "campaigntrackerpro/services/creators/module"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A creator's campaigns are endorsements of brands, so each one is named
// "{Creator} × {Brand}". Giving every creator several campaigns is what
// makes tier index and consistency mean anything — a single data point
// tells you nothing about how reliably someone delivers.
type creatorCampaignSeed struct {
	brand         string
	adType        campaigns.AdType
	platform      string
	status        campaigns.Status
	daysRunning   int
	reach         float64
	spend, budget int64
	frequency     float64
	approval      campaigns.Approval
	curve         campaigns.CurveShape
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
				{"Lakmé", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 3, 2, 350000, 3000000, 1.2, campaigns.ApprovalPending, campaigns.CurveSlow, "Hindi", "story"},
				{"boAt", campaigns.AdInfluencer, "Instagram", campaigns.StatusEnded, 24, 41, 2900000, 3200000, 2.1, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "unboxing"},
				{"Myntra", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 18, 36, 2400000, 2800000, 1.9, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "demo"},
			},
		},
		{
			id: "kabir-sehgal", name: "Kabir Sehgal", role: "Cricketer", initials: "KS",
			category: "Sports", region: "Pan-India", followers: 14_200_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Dream11", campaigns.AdInfluencer, "Instagram", campaigns.StatusEnded, 22, 85, 6100000, 7000000, 2.4, campaigns.ApprovalRejected, campaigns.CurveFast, "Hindi", "announcement"},
				{"boAt", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 14, 78, 5400000, 6500000, 2.2, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "demo"},
				{"CRED", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 9, 92, 6800000, 8000000, 2.5, campaigns.ApprovalApproved, campaigns.CurveFast, "English", "testimonial"},
			},
		},
		{
			id: "aanya-verma", name: "Aanya Verma", role: "Singer", initials: "AV",
			category: "Music", region: "Mumbai", followers: 6_200_000,
			platform: "YouTube", languages: []string{"Hindi", "Punjabi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Nykaa", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 20, 58, 4900000, 5500000, 2.6, campaigns.ApprovalRejected, campaigns.CurveSteady, "Hindi", "story"},
				{"Tanishq", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 11, 64, 4200000, 5000000, 2.3, campaigns.ApprovalApproved, campaigns.CurveSteady, "Punjabi", "story"},
			},
		},
		{
			id: "vikram-oberoi", name: "Vikram Oberoi", role: "CEO, Northgate", initials: "VO",
			category: "Business", region: "Bengaluru", followers: 180_000,
			platform: "LinkedIn", languages: []string{"English"},
			campaigns: []creatorCampaignSeed{
				{"PhonePe", campaigns.AdSocial, "LinkedIn", campaigns.StatusLive, 19, 26, 2100000, 3500000, 1.6, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "testimonial"},
				{"Urban Company", campaigns.AdSocial, "LinkedIn", campaigns.StatusEnded, 26, 22, 1700000, 2000000, 1.4, campaigns.ApprovalApproved, campaigns.CurveSlow, "English", "announcement"},
			},
		},
		{
			id: "meher-kapoor", name: "Meher Kapoor", role: "Creator", initials: "MK",
			category: "Influencer", region: "Delhi NCR", followers: 640_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Mamaearth", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 7, 20, 1100000, 2500000, 1.5, campaigns.ApprovalPending, campaigns.CurveFast, "Hindi", "unboxing"},
				{"Zepto", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 16, 31, 1900000, 2400000, 1.8, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "offer"},
				{"Lenskart", campaigns.AdSocial, "Instagram", campaigns.StatusEnded, 21, 27, 1600000, 1900000, 1.7, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "demo"},
				{"Nykaa", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 5, 24, 1400000, 2200000, 1.6, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "unboxing"},
			},
		},
		{
			id: "simran-bakshi", name: "Simran Bakshi", role: "Actor", initials: "SB",
			category: "Entertainment", region: "Delhi NCR", followers: 3_500_000,
			platform: "YouTube", languages: []string{"Hindi", "Punjabi"},
			campaigns: []creatorCampaignSeed{
				{"Myntra", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 12, 50, 3400000, 5000000, 2.0, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "story"},
				{"Tanishq", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 23, 47, 3100000, 3600000, 1.9, campaigns.ApprovalApproved, campaigns.CurveSlow, "Punjabi", "story"},
			},
		},
		{
			id: "dev-ahuja", name: "Dev Ahuja", role: "Founder, Finlytics", initials: "DA",
			category: "Business", region: "Pan-India", followers: 95_000,
			platform: "LinkedIn", languages: []string{"English"},
			campaigns: []creatorCampaignSeed{
				{"CRED", campaigns.AdSocial, "LinkedIn", campaigns.StatusScheduled, 0, 0, 0, 3000000, 0, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "announcement"},
				{"PhonePe", campaigns.AdSocial, "LinkedIn", campaigns.StatusLive, 13, 17, 1200000, 1800000, 1.3, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "demo"},
			},
		},
	}
}

var brandDomains = map[string]string{
	"Lakmé": "lakmeindia.com", "boAt": "boat-lifestyle.com", "Myntra": "myntra.com",
	"Dream11": "dream11.com", "CRED": "cred.club", "Nykaa": "nykaa.com",
	"Tanishq": "tanishq.co.in", "PhonePe": "phonepe.com", "Urban Company": "urbancompany.com",
	"Mamaearth": "mamaearth.in", "Zepto": "zeptonow.com", "Lenskart": "lenskart.com",
}

var seedSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugOf(s string) string {
	return strings.Trim(seedSlugPattern.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// seedCreators writes creators and their campaigns, linking each campaign
// back to its creator so the roster can benchmark them.
func seedCreators(ctx context.Context, pool *pgxpool.Pool, camp *campaignsmod.Module, cre *creatorsmod.Module) int {
	count := 0
	for _, cs := range creatorSeeds() {
		tier := creators.TierFor(cs.followers)
		if _, err := cre.Store().CreateCreator(ctx, creators.Creator{
			ID: cs.id, Name: cs.name, Role: cs.role, Initials: cs.initials,
			Category: cs.category, Region: cs.region, Tier: tier,
			Followers: cs.followers, PrimaryPlatform: cs.platform, Languages: cs.languages,
		}); err != nil {
			log.Fatalf("create creator %s: %v", cs.id, err)
		}

		for _, cc := range cs.campaigns {
			campaignID := fmt.Sprintf("%s-%s", cs.id, slugOf(cc.brand))
			name := fmt.Sprintf("%s × %s", cs.name, cc.brand)

			created, err := camp.Store().CreateCampaign(ctx, campaigns.Campaign{
				ID: campaignID, Name: name, SubjectType: campaigns.SubjectPerson, Role: cs.role,
				Initials: cs.initials, Category: cs.category, Region: cs.region,
				AdType: cc.adType, Platform: cc.platform, Status: cc.status,
				DaysRunning: cc.daysRunning, FlightDays: plannedFlight(campaignID, cc.daysRunning, cc.status), Reach: cc.reach, Spend: cc.spend, Budget: cc.budget,
				Frequency: cc.frequency, Approval: cc.approval, CurveShape: cc.curve,
				BrandDomain: brandDomains[cc.brand],
			})
			if err != nil {
				log.Fatalf("create creator campaign %s: %v", campaignID, err)
			}
			if err := camp.Store().SetCampaignCreator(ctx, created.ID, cs.id); err != nil {
				log.Fatalf("link campaign %s to creator: %v", campaignID, err)
			}

			for ci, cr := range creativesFor(string(cc.adType), cc.reach) {
				if _, err := camp.Store().CreateCreative(ctx, campaigns.Creative{
					CampaignID: created.ID, Headline: cr.headline, Kind: cr.kind,
					DurationLabel: cr.duration, Reach: cr.reach, CTR: cr.ctr,
					Language: cc.language, HookType: cc.hook,
					Claim:    fmt.Sprintf("%s endorsement", cc.brand),
					Festival: festivalFor(campaignID, ci),
					// Seeded creatives are analysed by construction, so the column
					// that records when reflects that rather than reading as a
					// backlog of unprocessed assets.
					AnalyzedAt: time.Now(),
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
			case campaigns.ApprovalApproved:
				events = append(events, auditSpec{"You", "Approved", "user", startedAt.Add(2 * time.Hour)})
			case campaigns.ApprovalRejected:
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
