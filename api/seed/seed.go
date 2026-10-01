// Command seed truncates and repopulates the database with 24 campaigns,
// their creatives and audit events, 3 scheduled reports with run history,
// and the seeded admin user — then runs anomaly detection once so flags
// reflect the same formulas the running app uses, not hardcoded guesses.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"campaigntrackerpro/internal/config"
	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/service"
	"campaigntrackerpro/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignSeed struct {
	id, name                 string
	subjectType              domain.SubjectType
	role, initials, category string
	region                   string
	adType                   domain.AdType
	platform                 string
	status                   domain.Status
	daysRunning              int
	reach                    float64
	spend, budget            int64
	frequency                float64
	approval                 domain.Approval
	curveShape               domain.CurveShape
}

func main() {
	ctx := context.Background()
	cfg := config.Load()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := reset(ctx, pool); err != nil {
		log.Fatalf("reset: %v", err)
	}

	st := store.New(pool)

	if _, err := st.CreateUser(ctx, cfg.AdminEmail, "Anisha T.", "admin", true); err != nil {
		log.Fatalf("create admin user: %v", err)
	}

	campaigns := campaignSeeds()
	for _, c := range campaigns {
		created, err := st.CreateCampaign(ctx, domain.Campaign{
			ID: c.id, Name: c.name, SubjectType: c.subjectType, Role: c.role, Initials: c.initials,
			Category: c.category, Region: c.region, AdType: c.adType, Platform: c.platform,
			Status: c.status, DaysRunning: c.daysRunning, Reach: c.reach, Spend: c.spend, Budget: c.budget,
			Frequency: c.frequency, Approval: c.approval, CurveShape: c.curveShape,
		})
		if err != nil {
			log.Fatalf("create campaign %s: %v", c.id, err)
		}

		for _, cr := range creativesFor(string(c.adType), c.reach) {
			if _, err := st.CreateCreative(ctx, domain.Creative{
				CampaignID: created.ID, Headline: cr.headline, Kind: cr.kind,
				DurationLabel: cr.duration, Reach: cr.reach, CTR: cr.ctr,
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

	if err := seedReports(ctx, pool, st); err != nil {
		log.Fatalf("seed reports: %v", err)
	}

	campaignsSvc := service.NewCampaigns(st)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	detector := service.NewAnomalyDetector(campaignsSvc, logger)
	flagged, err := detector.Run(ctx)
	if err != nil {
		log.Fatalf("anomaly detection: %v", err)
	}
	fmt.Printf("Seeded %d campaigns, %d reports. %d campaign(s) newly flagged by anomaly detection.\n",
		len(campaigns), 3, len(flagged))
}

func reset(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE audit_events, creatives, report_runs, reports, campaigns, users RESTART IDENTITY CASCADE`)
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
	case domain.ApprovalApproved:
		events = append(events, auditSpec{"You", "Approved", "user", startedAt.Add(2 * time.Hour)})
	case domain.ApprovalRejected:
		events = append(events, auditSpec{"You", "Rejected", "user", startedAt.Add(2 * time.Hour)})
	}
	return events
}

func seedReports(ctx context.Context, pool *pgxpool.Pool, st *store.Store) error {
	type reportSeed struct {
		name         string
		enabled      bool
		cadence      domain.Cadence
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
			name: "Weekly performance digest", enabled: true, cadence: domain.CadenceWeeklyMon9,
			scopeFilters: map[string]any{}, scopeLabel: "All campaigns", columns: service.ReportColumns,
			runs: []struct {
				daysAgo int
				result  string
				rows    int
			}{{14, "Delivered", 22}, {7, "Delivered", 23}},
		},
		{
			name: "Daily approvals reminder", enabled: true, cadence: domain.CadenceWeekday830,
			scopeFilters: map[string]any{"approval": "pending"}, scopeLabel: "Pending approvals",
			columns: []string{"Subject", "Category", "Region", "Approval", "Spend", "Waiting since"},
			runs: []struct {
				daysAgo int
				result  string
				rows    int
			}{{2, "Delivered", 4}, {1, "Delivered", 6}, {0, "Delivered", 5}},
		},
		{
			name: "Anomaly alerts", enabled: false, cadence: domain.CadenceOnFlag,
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
		rep, err := st.CreateReport(ctx, domain.Report{
			Name: spec.name, Enabled: spec.enabled, Cadence: string(spec.cadence),
			Recipients: []string{"anishatiwari695@gmail.com"}, ScopeFilters: spec.scopeFilters,
			ScopeLabel: spec.scopeLabel, Columns: spec.columns,
		})
		if err != nil {
			return fmt.Errorf("create report %q: %w", spec.name, err)
		}
		for _, run := range spec.runs {
			at := time.Now().AddDate(0, 0, -run.daysAgo)
			if err := insertReportRunAt(ctx, pool, rep.ID, run.result, run.rows, at); err != nil {
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
		{"dream11", "Dream11", domain.SubjectBrand, "", "D1", "Sports", "Pan-India", domain.AdVideo, "YouTube", domain.StatusLive, 18, 180, 9800000, 12000000, 2.6, domain.ApprovalApproved, domain.CurveFast},
		{"phonepe", "PhonePe", domain.SubjectBrand, "", "PP", "Fintech", "Pan-India", domain.AdGoogleAds, "Google Search", domain.StatusLive, 20, 65, 4200000, 6000000, 1.8, domain.ApprovalApproved, domain.CurveSteady},
		{"cred", "CRED", domain.SubjectBrand, "", "CR", "Fintech", "Delhi NCR", domain.AdSocial, "Instagram", domain.StatusLive, 6, 22, 1800000, 4000000, 1.5, domain.ApprovalPending, domain.CurveFast},
		{"amul", "Amul", domain.SubjectBrand, "", "AM", "FMCG", "Pan-India", domain.AdDisplay, "Google Display", domain.StatusEnded, 30, 95, 5200000, 6000000, 3.1, domain.ApprovalApproved, domain.CurveSlow},
		{"zepto", "Zepto", domain.SubjectBrand, "", "ZP", "E-commerce", "Mumbai", domain.AdPerformance, "Google Search", domain.StatusLive, 10, 16, 900000, 2500000, 2.0, domain.ApprovalApproved, domain.CurveSteady},
		{"myntra", "Myntra", domain.SubjectBrand, "", "MY", "Fashion", "Bengaluru", domain.AdSocial, "Instagram", domain.StatusLive, 14, 48, 4300000, 4500000, 2.2, domain.ApprovalApproved, domain.CurveSteady},
		{"boat", "boAt", domain.SubjectBrand, "", "BO", "Technology", "Pan-India", domain.AdVideo, "YouTube", domain.StatusLive, 16, 72, 4600000, 6500000, 2.5, domain.ApprovalApproved, domain.CurveFast},
		{"swiggy", "Swiggy", domain.SubjectBrand, "", "SW", "Services", "Bengaluru", domain.AdPerformance, "Google Search", domain.StatusLive, 12, 55, 8500000, 8000000, 2.1, domain.ApprovalApproved, domain.CurveFast},
		{"tanishq", "Tanishq", domain.SubjectBrand, "", "TQ", "Luxury", "Delhi NCR", domain.AdDisplay, "Google Display", domain.StatusEnded, 25, 40, 3200000, 4000000, 1.9, domain.ApprovalRejected, domain.CurveSlow},
		{"licious", "Licious", domain.SubjectBrand, "", "LC", "FMCG", "South Zone", domain.AdSocial, "Instagram", domain.StatusLive, 9, 18, 1200000, 3000000, 1.7, domain.ApprovalApproved, domain.CurveSteady},
		{"blinkit", "Blinkit", domain.SubjectBrand, "", "BL", "E-commerce", "Delhi NCR", domain.AdGoogleAds, "Google Search", domain.StatusLive, 11, 150, 5000000, 9000000, 2.3, domain.ApprovalPending, domain.CurveFast},
		{"lenskart", "Lenskart", domain.SubjectBrand, "", "LK", "Fashion", "West Zone", domain.AdVideo, "YouTube", domain.StatusLive, 17, 38, 9700000, 10000000, 2.0, domain.ApprovalApproved, domain.CurveSteady},
		{"mamaearth", "Mamaearth", domain.SubjectBrand, "", "MA", "Beauty", "Mumbai", domain.AdSocial, "Instagram", domain.StatusLive, 8, 15, 700000, 2000000, 1.4, domain.ApprovalPending, domain.CurveSlow},
		{"nykaa", "Nykaa", domain.SubjectBrand, "", "NK", "Beauty", "Mumbai", domain.AdSocial, "Meta Ads", domain.StatusLive, 13, 60, 4400000, 6000000, 2.1, domain.ApprovalApproved, domain.CurveSteady},
		{"rapido", "Rapido", domain.SubjectBrand, "", "RP", "Services", "South Zone", domain.AdDisplay, "Google Display", domain.StatusLive, 15, 30, 2600000, 4000000, 1.6, domain.ApprovalApproved, domain.CurveSteady},
		{"urban-company", "Urban Company", domain.SubjectBrand, "", "UC", "Services", "Bengaluru", domain.AdPerformance, "Meta Ads", domain.StatusEnded, 28, 52, 4700000, 5500000, 2.0, domain.ApprovalApproved, domain.CurveSlow},
		{"ethos-watches", "Ethos Watches", domain.SubjectBrand, "", "EW", "Luxury", "Delhi NCR", domain.AdDisplay, "Google Display", domain.StatusScheduled, 0, 0, 0, 5000000, 0, domain.ApprovalApproved, domain.CurveSteady},
		{"rohan-malhotra", "Rohan Malhotra", domain.SubjectPerson, "Actor", "RM", "Entertainment", "Mumbai", domain.AdInfluencer, "Instagram", domain.StatusLive, 3, 2, 350000, 3000000, 1.2, domain.ApprovalPending, domain.CurveSlow},
		{"kabir-sehgal", "Kabir Sehgal", domain.SubjectPerson, "Cricketer", "KS", "Sports", "Pan-India", domain.AdInfluencer, "Instagram", domain.StatusEnded, 22, 85, 6100000, 7000000, 2.4, domain.ApprovalRejected, domain.CurveFast},
		{"aanya-verma", "Aanya Verma", domain.SubjectPerson, "Singer", "AV", "Music", "Mumbai", domain.AdVideo, "YouTube", domain.StatusEnded, 20, 58, 4900000, 5500000, 2.6, domain.ApprovalRejected, domain.CurveSteady},
		{"vikram-oberoi", "Vikram Oberoi", domain.SubjectPerson, "CEO, Northgate", "VO", "Business", "Bengaluru", domain.AdSocial, "LinkedIn", domain.StatusLive, 19, 26, 2100000, 3500000, 1.6, domain.ApprovalApproved, domain.CurveSteady},
		{"meher-kapoor", "Meher Kapoor", domain.SubjectPerson, "Creator", "MK", "Influencer", "Delhi NCR", domain.AdInfluencer, "Instagram", domain.StatusLive, 7, 20, 1100000, 2500000, 1.5, domain.ApprovalPending, domain.CurveFast},
		{"simran-bakshi", "Simran Bakshi", domain.SubjectPerson, "Actor", "SB", "Entertainment", "Delhi NCR", domain.AdVideo, "YouTube", domain.StatusLive, 12, 50, 3400000, 5000000, 2.0, domain.ApprovalApproved, domain.CurveSteady},
		{"dev-ahuja", "Dev Ahuja", domain.SubjectPerson, "Founder, Finlytics", "DA", "Business", "Pan-India", domain.AdSocial, "LinkedIn", domain.StatusScheduled, 0, 0, 0, 3000000, 0, domain.ApprovalApproved, domain.CurveSteady},
	}
}
