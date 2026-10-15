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

// Real people, and only brand relationships they are actually
// documented as having. Virat Kohli fronts Puma; Alia Bhatt is Gucci's
// first Indian global ambassador; Salman Khan has endorsed Pepsi for
// years. Every pairing below is in the public record this product also
// serves at /endorsements, cited there to where it was reported.
//
// What is NOT real, and cannot be: the reach, spend, budget and pacing.
// Nobody publishes per-campaign performance — it is exactly what
// agencies are paid to know — so those figures are illustrative, and
// the dashboard says so where they appear.
//
// Which is why none of these campaigns carries a negative verdict. The
// numbers are set comfortably inside budget and at or above category
// median, so the anomaly detector has nothing to flag and no approval
// is left pending or rejected. An invented "under-delivering" against a
// named person's real commercial relationship is a reputational claim
// about them, and that is a different thing from an illustrative
// figure. The flags, rejections and over-budget cases all live on the
// brand campaigns, where the subject is a company and the claim is
// about a line of spend rather than somebody's professional standing.
func creatorSeeds() []creatorSeed {
	return []creatorSeed{
		{
			id: "virat-kohli", name: "Virat Kohli", role: "Cricketer", initials: "VK",
			category: "Sports", region: "Pan-India", followers: 270_000_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Puma", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 16, 96, 5_200_000, 9_000_000, 2.6, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "demo"},
				{"MRF", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 28, 88, 4_600_000, 8_000_000, 2.3, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "story"},
				{"Vivo", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 11, 74, 3_900_000, 7_500_000, 2.1, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "announcement"},
			},
		},
		{
			id: "alia-bhatt", name: "Alia Bhatt", role: "Actor", initials: "AB",
			category: "Entertainment", region: "Mumbai", followers: 85_000_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Gucci", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 13, 68, 4_100_000, 7_000_000, 2.2, campaigns.ApprovalApproved, campaigns.CurveFast, "English", "story"},
				{"L'Oréal Paris", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 9, 54, 2_700_000, 5_000_000, 1.9, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "demo"},
				{"Maybelline", campaigns.AdSocial, "Instagram", campaigns.StatusEnded, 21, 47, 2_300_000, 4_200_000, 1.8, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "demo"},
			},
		},
		{
			id: "salman-khan", name: "Salman Khan", role: "Actor", initials: "SK",
			category: "Entertainment", region: "Pan-India", followers: 75_000_000,
			platform: "Instagram", languages: []string{"Hindi"},
			campaigns: []creatorCampaignSeed{
				{"Pepsi", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 15, 82, 4_400_000, 8_200_000, 2.4, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "story"},
				{"Suzuki", campaigns.AdInfluencer, "Instagram", campaigns.StatusEnded, 26, 61, 3_100_000, 6_000_000, 2.0, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "demo"},
				{"Thums Up", campaigns.AdSocial, "Instagram", campaigns.StatusEnded, 19, 58, 2_800_000, 5_400_000, 1.9, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "announcement"},
			},
		},
		{
			id: "deepika-padukone", name: "Deepika Padukone", role: "Actor", initials: "DP",
			category: "Entertainment", region: "Mumbai", followers: 80_000_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Louis Vuitton", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 12, 71, 4_300_000, 8_000_000, 2.2, campaigns.ApprovalApproved, campaigns.CurveFast, "English", "story"},
				{"Cartier", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 7, 44, 2_200_000, 4_800_000, 1.7, campaigns.ApprovalApproved, campaigns.CurveSlow, "English", "story"},
				{"Tissot", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 23, 39, 1_900_000, 3_600_000, 1.6, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "demo"},
			},
		},
		{
			id: "ranveer-singh", name: "Ranveer Singh", role: "Actor", initials: "RS",
			category: "Entertainment", region: "Mumbai", followers: 45_000_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"Adidas", campaigns.AdInfluencer, "Instagram", campaigns.StatusLive, 14, 57, 3_000_000, 5_800_000, 2.0, campaigns.ApprovalApproved, campaigns.CurveFast, "Hindi", "demo"},
				{"Jack & Jones", campaigns.AdSocial, "Instagram", campaigns.StatusEnded, 20, 41, 2_000_000, 3_900_000, 1.7, campaigns.ApprovalApproved, campaigns.CurveSteady, "English", "story"},
				{"MakeMyTrip", campaigns.AdVideo, "YouTube", campaigns.StatusEnded, 25, 36, 1_700_000, 3_400_000, 1.6, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "offer"},
			},
		},
		{
			id: "ms-dhoni", name: "MS Dhoni", role: "Cricketer", initials: "MD",
			category: "Sports", region: "Pan-India", followers: 50_000_000,
			platform: "Instagram", languages: []string{"Hindi", "English"},
			campaigns: []creatorCampaignSeed{
				{"CARS24", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 10, 52, 2_600_000, 5_200_000, 1.9, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "testimonial"},
				{"SEVEN", campaigns.AdInfluencer, "Instagram", campaigns.StatusEnded, 22, 43, 2_100_000, 4_000_000, 1.7, campaigns.ApprovalApproved, campaigns.CurveSteady, "Hindi", "demo"},
				{"Shaka Harry", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 8, 31, 1_500_000, 3_200_000, 1.5, campaigns.ApprovalApproved, campaigns.CurveSlow, "English", "story"},
			},
		},
	}
}

var brandDomains = map[string]string{
	// Brand campaigns.
	"Lakmé": "lakmeindia.com", "boAt": "boat-lifestyle.com", "Myntra": "myntra.com",
	"Dream11": "dream11.com", "CRED": "cred.club", "Nykaa": "nykaa.com",
	"Tanishq": "tanishq.co.in", "PhonePe": "phonepe.com", "Urban Company": "urbancompany.com",
	"Mamaearth": "mamaearth.in", "Zepto": "zeptonow.com", "Lenskart": "lenskart.com",
	// Brands the people in the roster are documented as fronting.
	"Puma": "puma.com", "MRF": "mrftyres.com", "Vivo": "vivo.com",
	"Gucci": "gucci.com", "L'Oréal Paris": "lorealparis.co.in", "Maybelline": "maybelline.co.in",
	"Pepsi": "pepsi.com", "Suzuki": "marutisuzuki.com", "Thums Up": "coca-colaindia.com",
	"Louis Vuitton": "louisvuitton.com", "Cartier": "cartier.com", "Tissot": "tissotwatches.com",
	"Adidas": "adidas.co.in", "Jack & Jones": "jackjones.in", "MakeMyTrip": "makemytrip.com",
	"CARS24": "cars24.com", "SEVEN": "seven.co.in", "Shaka Harry": "shakaharry.com",
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

			// An endorsement is one asset in the person's own feed, so it
			// is written from the brand and their role rather than drawn
			// from the brand's own creative set.
			ad := creatorCreative(cc.brand, campaignID)
			for ci, cr := range []creativeSpec{{
				headline: ad.headline, kind: ad.kind, duration: ad.duration,
				reach: cc.reach * ad.reachMul, ctr: baseCTR(string(cc.adType)) * ad.ctrMul,
				hook: ad.hook,
			}} {
				if _, err := camp.Store().CreateCreative(ctx, campaigns.Creative{
					CampaignID: created.ID, Headline: cr.headline, Kind: cr.kind,
					DurationLabel: cr.duration, Reach: cr.reach, CTR: cr.ctr,
					Language: cc.language, HookType: cr.hook,
					Claim:    fmt.Sprintf("%s endorsement", cc.brand),
					Festival: festivalFor(campaignID, cs.category, ci),
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

// seedCreatorFaces runs after the roster is written.
func seedCreatorFaces(ctx context.Context, cre *creatorsmod.Module) {
	applyCreatorFaces(ctx, cre.Store().SetPhoto)
}
