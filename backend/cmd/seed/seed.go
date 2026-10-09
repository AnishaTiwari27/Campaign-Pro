// Command seed truncates and repopulates the database with 24 campaigns,
// their creatives and audit events, 3 scheduled reports with run history,
// and the seeded admin user — then runs anomaly detection once so flags
// reflect the same formulas the running app uses, not hardcoded guesses.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	creatorsmod "campaigntrackerpro/services/creators/module"
	identitymod "campaigntrackerpro/services/identity/module"
	"campaigntrackerpro/services/reports"
	reportsmod "campaigntrackerpro/services/reports/module"

	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignSeed struct {
	id, name                 string
	subjectType              campaigns.SubjectType
	role, initials, category string
	region                   string
	adType                   campaigns.AdType
	platform                 string
	status                   campaigns.Status
	daysRunning              int
	reach                    float64
	spend, budget            int64
	frequency                float64
	approval                 campaigns.Approval
	curveShape               campaigns.CurveShape
	brandDomain              string
}

func main() {
	ctx := context.Background()
	cfg := config.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := reset(ctx, pool); err != nil {
		log.Fatalf("reset: %v", err)
	}

	db := database.New(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Fixtures are written through each service's own store, reached via
	// its factory — the seeder gets no more access than a service has.
	camp := campaignsmod.New(db, logger)
	cre := creatorsmod.New(db, camp.Service(), logger)
	rep := reportsmod.New(db, camp.Service(), camp.Detector(), nil, logger, time.Minute)
	ident := identitymod.New(db, noopAuditor{}, nil, identitymod.Options{}, logger)

	// Fixture logins. The password is generated per run and printed once.
	//
	// It used to be a constant in this file, which meant it was published:
	// this repository is public, so anyone who read it could sign in to any
	// deployment that had ever been seeded — including as an approver, who
	// signs off on spend. A loud placeholder is still a credential.
	//
	// SEED_PASSWORD overrides it for the workflows that need a known value,
	// which is the e2e suite and nothing else. See the Makefile.
	demoPassword := os.Getenv("SEED_PASSWORD")
	generatedPassword := demoPassword == ""
	if generatedPassword {
		var err error
		if demoPassword, err = randomPassword(); err != nil {
			log.Fatalf("generate seed password: %v", err)
		}
	}
	for _, u := range []struct {
		email, name, role string
		agency, approve   bool
	}{
		{cfg.AdminEmail, "Anisha T.", "admin", true, true},
		{"approver@campaigntracker.test", "Rhea Nair", "approver", true, true},
		{"analyst@campaigntracker.test", "Arjun Rao", "analyst", true, false},
	} {
		created, err := camp.Store().CreateUser(ctx, u.email, u.name, u.role, u.approve)
		if err != nil {
			log.Fatalf("create user %s: %v", u.email, err)
		}
		if err := camp.Store().SetUserAgency(ctx, created.ID, u.agency); err != nil {
			log.Fatalf("set agency flag for %s: %v", u.email, err)
		}
		if err := ident.SetPassword(ctx, created.ID, demoPassword); err != nil {
			log.Fatalf("set password for %s: %v", u.email, err)
		}
	}

	brandCampaigns := campaignSeeds()
	for _, c := range brandCampaigns {
		created, err := camp.Store().CreateCampaign(ctx, campaigns.Campaign{
			ID: c.id, Name: c.name, SubjectType: c.subjectType, Role: c.role, Initials: c.initials,
			Category: c.category, Region: c.region, AdType: c.adType, Platform: c.platform,
			Status: c.status, DaysRunning: c.daysRunning, FlightDays: plannedFlight(c.id, c.daysRunning, c.status), Reach: c.reach, Spend: c.spend, Budget: c.budget,
			Frequency: c.frequency, Approval: c.approval, CurveShape: c.curveShape,
			BrandDomain: c.brandDomain,
		})
		if err != nil {
			log.Fatalf("create campaign %s: %v", c.id, err)
		}

		for ci, cr := range creativesFor(string(c.adType), c.reach) {
			if _, err := camp.Store().CreateCreative(ctx, campaigns.Creative{
				CampaignID: created.ID, Headline: cr.headline, Kind: cr.kind,
				DurationLabel: cr.duration, Reach: cr.reach, CTR: cr.ctr,
				Language: pickFor(created.ID, ci, seedLanguages),
				HookType: pickFor(created.ID, ci, seedHooks),
				Claim:    c.category + " brand spot",
				Festival: festivalFor(created.ID, ci),
				// Seeded creatives are analysed by construction, so the column
				// that records when reflects that rather than reading as a
				// backlog of unprocessed assets.
				AnalyzedAt: time.Now(),
			}); err != nil {
				log.Fatalf("create creative for %s: %v", c.id, err)
			}
		}

		for _, ev := range auditEventsFor(c) {
			if err := insertAuditAt(ctx, pool, created.ID, ev.actor, ev.action, ev.kind, ev.at); err != nil {
				log.Fatalf("create audit event for %s: %v", c.id, err)
			}
		}
	}

	creatorCampaigns := seedCreators(ctx, pool, camp, cre)

	if err := seedReports(ctx, pool, rep); err != nil {
		log.Fatalf("seed reports: %v", err)
	}

	// Flags are produced by the real detector, never seeded by hand.
	detector := camp.Detector()
	flagged, err := detector.Run(ctx)
	if err != nil {
		log.Fatalf("anomaly detection: %v", err)
	}
	if generatedPassword {
		fmt.Printf("\nFixture accounts were given this generated password, shown once:\n\n  %s\n\n", demoPassword)
	}
	fmt.Printf("Seeded %d brand + %d creator campaigns across %d creators, %d reports. %d flagged by anomaly detection.\n",
		len(brandCampaigns), creatorCampaigns, len(creatorSeeds()), 3, len(flagged))
}

func reset(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE audit_events, creatives, report_runs, reports, campaigns, creators, users RESTART IDENTITY CASCADE`)
	return err
}

func insertAuditAt(ctx context.Context, pool *pgxpool.Pool, campaignID, actor, action, kind string, at time.Time) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO audit_events (campaign_id, actor, action, kind, created_at) VALUES ($1,$2,$3,$4,$5)`,
		campaignID, actor, action, kind, at)
	return err
}

func insertReportRunAt(ctx context.Context, pool *pgxpool.Pool, reportID, result string, rowCount int, at time.Time) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO report_runs (report_id, ran_at, result, row_count) VALUES ($1,$2,$3,$4)`,
		reportID, at, result, rowCount)
	return err
}

type creativeSpec struct {
	headline, kind, duration string
	reach, ctr               float64
}

// creativesFor gives every campaign 1-2 creatives with a kind and CTR that
// fit its ad type.
func creativesFor(adType string, reach float64) []creativeSpec {
	switch adType {
	case "Video":
		return []creativeSpec{{"Brand film — 30s cut", "video", "0:30", reach * 0.65, 2.1}}
	case "Display":
		return []creativeSpec{
			{"Static banner — hero visual", "banner", "static", reach * 0.5, 0.8},
			{"Static banner — offer callout", "banner", "static", reach * 0.35, 0.6},
		}
	case "Social":
		return []creativeSpec{
			{"Feed post — launch creative", "text", "static", reach * 0.55, 1.4},
			{"Reel — behind the scenes", "video", "0:15", reach * 0.4, 2.0},
		}
	case "Influencer":
		return []creativeSpec{{"Sponsored reel", "video", "0:20", reach * 0.7, 2.6}}
	case "Google Ads", "Performance":
		return []creativeSpec{{"Search ad — brand keywords", "text", "article", reach * 0.6, 3.1}}
	default:
		return []creativeSpec{{"Primary creative", "banner", "static", reach * 0.5, 1.0}}
	}
}

type auditSpec struct {
	actor, action, kind string
	at                  time.Time
}

// auditEventsFor gives every campaign 2-3 audit events: creation, an
// asset-upload step, and (once reviewed) the approve/reject decision.
func auditEventsFor(c campaignSeed) []auditSpec {
	startedAt := time.Now().AddDate(0, 0, -c.daysRunning)
	events := []auditSpec{
		{"System", "Campaign created", "sys", startedAt},
		{"System", "Creative assets uploaded", "sys", startedAt.Add(1 * time.Hour)},
	}
	switch c.approval {
	case campaigns.ApprovalApproved:
		events = append(events, auditSpec{"You", "Approved", "user", startedAt.Add(2 * time.Hour)})
	case campaigns.ApprovalRejected:
		events = append(events, auditSpec{"You", "Rejected", "user", startedAt.Add(2 * time.Hour)})
	}
	return events
}

func seedReports(ctx context.Context, pool *pgxpool.Pool, rep *reportsmod.Module) error {
	type reportSeed struct {
		name         string
		enabled      bool
		cadence      reports.Cadence
		scopeFilters map[string]any
		scopeLabel   string
		columns      []string
		runs         []struct {
			daysAgo int
			result  string
			rows    int
		}
	}

	specs := []reportSeed{
		{
			name: "Weekly performance digest", enabled: true, cadence: reports.CadenceWeeklyMon9,
			scopeFilters: map[string]any{}, scopeLabel: "All campaigns", columns: campaigns.ReportColumns,
			runs: []struct {
				daysAgo int
				result  string
				rows    int
			}{{14, "Delivered", 22}, {7, "Delivered", 23}},
		},
		{
			name: "Daily approvals reminder", enabled: true, cadence: reports.CadenceWeekday830,
			scopeFilters: map[string]any{"approval": "pending"}, scopeLabel: "Pending approvals",
			columns: []string{"Subject", "Category", "Region", "Approval", "Spend", "Waiting since"},
			runs: []struct {
				daysAgo int
				result  string
				rows    int
			}{{2, "Delivered", 4}, {1, "Delivered", 6}, {0, "Delivered", 5}},
		},
		{
			name: "Anomaly alerts", enabled: false, cadence: reports.CadenceOnFlag,
			scopeFilters: map[string]any{}, scopeLabel: "Flagged campaigns",
			columns: []string{"Subject", "Category", "Flag reason", "Index", "Budget pace"},
			runs: []struct {
				daysAgo int
				result  string
				rows    int
			}{{5, "Delivered", 2}},
		},
	}

	for _, spec := range specs {
		created, err := rep.Store().CreateReport(ctx, reports.Report{
			Name: spec.name, Enabled: spec.enabled, Cadence: string(spec.cadence),
			Recipients: []string{"anishatiwari695@gmail.com"}, ScopeFilters: spec.scopeFilters,
			ScopeLabel: spec.scopeLabel, Columns: spec.columns,
		})
		if err != nil {
			return fmt.Errorf("create report %q: %w", spec.name, err)
		}
		for _, run := range spec.runs {
			at := time.Now().AddDate(0, 0, -run.daysAgo)
			if err := insertReportRunAt(ctx, pool, created.ID, run.result, run.rows, at); err != nil {
				return fmt.Errorf("run for report %q: %w", spec.name, err)
			}
		}
	}
	return nil
}

// campaignSeeds is the app's 24-campaign fixture: 17 real Indian brands and
// 7 fictional people (never real named individuals — attributing fabricated
// spend/performance figures to an actual person is a different, more
// sensitive claim than illustrative brand-campaign numbers). All 13
// categories, all 6 regions, all 6 ad types and all 6 platforms appear at
// least once.
func campaignSeeds() []campaignSeed {
	return []campaignSeed{
		{"dream11", "Dream11", campaigns.SubjectBrand, "", "D1", "Sports", "Pan-India", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 18, 180, 9800000, 12000000, 2.6, campaigns.ApprovalApproved, campaigns.CurveFast, "dream11.com"},
		{"phonepe", "PhonePe", campaigns.SubjectBrand, "", "PP", "Fintech", "Pan-India", campaigns.AdGoogleAds, "Google Search", campaigns.StatusLive, 20, 65, 4200000, 6000000, 1.8, campaigns.ApprovalApproved, campaigns.CurveSteady, "phonepe.com"},
		{"cred", "CRED", campaigns.SubjectBrand, "", "CR", "Fintech", "Delhi NCR", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 6, 22, 1800000, 4000000, 1.5, campaigns.ApprovalPending, campaigns.CurveFast, "cred.club"},
		{"amul", "Amul", campaigns.SubjectBrand, "", "AM", "FMCG", "Pan-India", campaigns.AdDisplay, "Google Display", campaigns.StatusEnded, 30, 95, 5200000, 6000000, 3.1, campaigns.ApprovalApproved, campaigns.CurveSlow, "amul.com"},
		{"zepto", "Zepto", campaigns.SubjectBrand, "", "ZP", "E-commerce", "Mumbai", campaigns.AdPerformance, "Google Search", campaigns.StatusLive, 10, 16, 900000, 2500000, 2.0, campaigns.ApprovalApproved, campaigns.CurveSteady, "zeptonow.com"},
		{"myntra", "Myntra", campaigns.SubjectBrand, "", "MY", "Fashion", "Bengaluru", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 14, 48, 4300000, 4500000, 2.2, campaigns.ApprovalApproved, campaigns.CurveSteady, "myntra.com"},
		{"boat", "boAt", campaigns.SubjectBrand, "", "BO", "Technology", "Pan-India", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 16, 72, 4600000, 6500000, 2.5, campaigns.ApprovalApproved, campaigns.CurveFast, "boat-lifestyle.com"},
		{"swiggy", "Swiggy", campaigns.SubjectBrand, "", "SW", "Services", "Bengaluru", campaigns.AdPerformance, "Google Search", campaigns.StatusLive, 12, 55, 8500000, 8000000, 2.1, campaigns.ApprovalApproved, campaigns.CurveFast, "swiggy.com"},
		{"tanishq", "Tanishq", campaigns.SubjectBrand, "", "TQ", "Luxury", "Delhi NCR", campaigns.AdDisplay, "Google Display", campaigns.StatusEnded, 25, 40, 3200000, 4000000, 1.9, campaigns.ApprovalRejected, campaigns.CurveSlow, "tanishq.co.in"},
		{"licious", "Licious", campaigns.SubjectBrand, "", "LC", "FMCG", "South Zone", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 9, 18, 1200000, 3000000, 1.7, campaigns.ApprovalApproved, campaigns.CurveSteady, "licious.in"},
		{"blinkit", "Blinkit", campaigns.SubjectBrand, "", "BL", "E-commerce", "Delhi NCR", campaigns.AdGoogleAds, "Google Search", campaigns.StatusLive, 11, 150, 5000000, 9000000, 2.3, campaigns.ApprovalPending, campaigns.CurveFast, "blinkit.com"},
		{"lenskart", "Lenskart", campaigns.SubjectBrand, "", "LK", "Fashion", "West Zone", campaigns.AdVideo, "YouTube", campaigns.StatusLive, 17, 38, 9700000, 10000000, 2.0, campaigns.ApprovalApproved, campaigns.CurveSteady, "lenskart.com"},
		{"mamaearth", "Mamaearth", campaigns.SubjectBrand, "", "MA", "Beauty", "Mumbai", campaigns.AdSocial, "Instagram", campaigns.StatusLive, 8, 15, 700000, 2000000, 1.4, campaigns.ApprovalPending, campaigns.CurveSlow, "mamaearth.in"},
		{"nykaa", "Nykaa", campaigns.SubjectBrand, "", "NK", "Beauty", "Mumbai", campaigns.AdSocial, "Meta Ads", campaigns.StatusLive, 13, 60, 4400000, 6000000, 2.1, campaigns.ApprovalApproved, campaigns.CurveSteady, "nykaa.com"},
		{"rapido", "Rapido", campaigns.SubjectBrand, "", "RP", "Services", "South Zone", campaigns.AdDisplay, "Google Display", campaigns.StatusLive, 15, 30, 2600000, 4000000, 1.6, campaigns.ApprovalApproved, campaigns.CurveSteady, "rapido.bike"},
		{"urban-company", "Urban Company", campaigns.SubjectBrand, "", "UC", "Services", "Bengaluru", campaigns.AdPerformance, "Meta Ads", campaigns.StatusEnded, 28, 52, 4700000, 5500000, 2.0, campaigns.ApprovalApproved, campaigns.CurveSlow, "urbancompany.com"},
		{"ethos-watches", "Ethos Watches", campaigns.SubjectBrand, "", "EW", "Luxury", "Delhi NCR", campaigns.AdDisplay, "Google Display", campaigns.StatusScheduled, 0, 0, 0, 5000000, 0, campaigns.ApprovalApproved, campaigns.CurveSteady, "ethoswatches.com"},
	}
}

// noopAuditor satisfies identity's auditor dependency during seeding;
// there is no point recording "signed in" for fixture password writes.
type noopAuditor struct{}

func (noopAuditor) Record(ctx context.Context, userID, actor, action, entityType, entityID string) error {
	return nil
}

// randomPassword returns 24 bytes of entropy, URL-safe so it survives being
// copied through a terminal and a browser form. Matches cmd/createuser.
func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// plannedFlight gives a seeded campaign a believable planned length from
// what it already has. Derived rather than typed into every row: 35
// hand-picked numbers would be 35 chances to make the demo incoherent, and
// the shape of the spread is what matters, not any single value.
//
// An ended campaign has finished its flight, so its planned length is what
// it ran. A live one is somewhere inside it — the id's hash picks how far,
// between 55% and 95% through, so the Signals quadrant gets campaigns on
// both sides of plan instead of a single stripe.
func plannedFlight(id string, daysRunning int, status campaigns.Status) int {
	if daysRunning <= 0 {
		return 0
	}
	if status == campaigns.StatusEnded {
		return daysRunning
	}
	var h uint32 = 2166136261
	for i := 0; i < len(id); i++ {
		h = (h ^ uint32(id[i])) * 16777619
	}
	// 55%..95% of the way through, in 5-point steps.
	pct := 55 + int(h%9)*5
	flight := daysRunning * 100 / pct
	if flight <= daysRunning {
		flight = daysRunning + 1
	}
	return flight
}

// Indian festivals and tentpoles that campaigns actually plan around. The
// festival column existed in the schema from the start and was never
// filled, which left the most India-specific cut of the data unopened:
// what a Diwali creative delivers against an untagged one.
var seedFestivals = []string{
	"Diwali", "Holi", "IPL", "Navratri", "Eid", "Raksha Bandhan", "Pongal", "Independence Day",
}

// festivalFor tags roughly half the creatives, which is the honest shape:
// a brand runs festival work around the calendar and evergreen work the
// rest of the year. Keyed off the campaign id so a reseed is stable.
func festivalFor(campaignID string, n int) string {
	var h uint32 = 2166136261
	for i := 0; i < len(campaignID); i++ {
		h = (h ^ uint32(campaignID[i])) * 16777619
	}
	h = (h ^ uint32(n)) * 16777619
	if h%10 < 5 {
		return ""
	}
	return seedFestivals[int(h/10)%len(seedFestivals)]
}

// seedLanguages and seedHooks fill the two analyzer columns for brand
// creatives, which previously carried neither — only creator campaigns set
// them, so half the roster looked unanalysed.
var seedLanguages = []string{"Hindi", "English", "Tamil", "Telugu", "Marathi", "Bengali"}
var seedHooks = []string{"announcement", "demo", "offer", "story", "testimonial", "unboxing"}

func pickFor(campaignID string, n int, from []string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(campaignID); i++ {
		h = (h ^ uint32(campaignID[i])) * 16777619
	}
	h = (h ^ uint32(n*7+13)) * 16777619
	return from[int(h)%len(from)]
}
