package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/campaigns/internal/catalogclient"
	"campaigntrackerpro/services/campaigns/internal/models"
	"campaigntrackerpro/services/campaigns/internal/store"
)

// fakeStore is an in-memory CampaignStore — enough to exercise handler
// logic (validation, status codes) without a real Postgres.
type fakeStore struct {
	created       []models.Campaign
	byID          map[int64]models.Campaign
	anomalyReport models.AnomalyReport
	creatives     []models.Creative
}

func (f *fakeStore) Ping(context.Context) error { return nil }

func (f *fakeStore) List(context.Context, store.Filter, int, int, time.Time) ([]models.Campaign, int, error) {
	return nil, 0, nil
}

func (f *fakeStore) Export(context.Context, store.Filter, time.Time) ([]models.Campaign, error) {
	return nil, nil
}

func (f *fakeStore) ByID(_ context.Context, id int64, _ time.Time) (models.Campaign, bool, error) {
	c, ok := f.byID[id]
	return c, ok, nil
}

func (f *fakeStore) Create(_ context.Context, c models.Campaign, today time.Time) (models.Campaign, error) {
	c.ID = int64(len(f.created) + 1)
	c.Status = "Live"
	c.Pacing = models.PacingOf(c.Budget, c.Spend, c.Start, c.End, today)
	// Mirrors store.Store.Create: a caller that never sets ApprovalStatus
	// (the demo ticker, the seed script) gets "approved", same as
	// production — only handleCreateCampaign passes "pending" explicitly.
	if c.ApprovalStatus == "" {
		c.ApprovalStatus = "approved"
	}
	f.created = append(f.created, c)
	return c, nil
}

func (f *fakeStore) UpdateBudget(_ context.Context, id int64, budgetRupees *int64, today time.Time) (models.Campaign, bool, error) {
	c, ok := f.byID[id]
	if !ok {
		return models.Campaign{}, false, nil
	}
	c.Budget = budgetRupees
	c.Pacing = models.PacingOf(c.Budget, c.Spend, c.Start, c.End, today)
	f.byID[id] = c
	return c, true, nil
}

func (f *fakeStore) UpdateApprovalStatus(_ context.Context, id int64, status string, _ time.Time) (models.Campaign, bool, error) {
	c, ok := f.byID[id]
	if !ok {
		return models.Campaign{}, false, nil
	}
	c.ApprovalStatus = status
	f.byID[id] = c
	return c, true, nil
}

// The four aggregate methods aren't exercised by handlers_test.go's cases
// (those cover create/get, not the /internal/aggregates/* routes) — zero
// values satisfy the interface without pretending to test SQL that isn't
// running here. The SQL itself is verified against the live stack (see
// docs/ROADMAP.md's Phase A verification section), not a fake.
func (f *fakeStore) KPITotals(context.Context, store.Filter) (models.KPITotals, error) {
	return models.KPITotals{}, nil
}

func (f *fakeStore) WeeklyBuckets(context.Context, store.Filter, time.Time) ([]models.WeeklyBucket, error) {
	return nil, nil
}

func (f *fakeStore) TrendBuckets(context.Context, store.Filter) ([]models.TrendBucket, error) {
	return nil, nil
}

func (f *fakeStore) RegionCounts(context.Context, store.Filter) ([]models.RegionCount, error) {
	return nil, nil
}

func (f *fakeStore) BenchmarkRows(context.Context, store.Filter, string, bool) ([]models.BenchmarkAgg, error) {
	return nil, nil
}

func (f *fakeStore) AnomalyReport(context.Context, time.Time) (models.AnomalyReport, error) {
	return f.anomalyReport, nil
}

// SnapshotBenchmark isn't exercised by handlers_test.go's cases (no
// handler calls it — only the BENCHMARK_SNAPSHOT_ENABLED ticker does) — a
// nil return satisfies the interface without pretending to test SQL that
// isn't running here. Verified against the live stack instead.
func (f *fakeStore) SnapshotBenchmark(context.Context, time.Time) error {
	return nil
}

func (f *fakeStore) ListCreatives(_ context.Context, campaignID int64) ([]models.Creative, error) {
	var out []models.Creative
	for _, c := range f.creatives {
		if c.CampaignID == campaignID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateCreative(_ context.Context, campaignID int64, headline, creativeType string, reach, spend int64) (models.Creative, error) {
	c := models.Creative{ID: int64(len(f.creatives) + 1), CampaignID: campaignID, Headline: headline, CreativeType: creativeType, Reach: reach, Spend: spend}
	f.creatives = append(f.creatives, c)
	return c, nil
}

// fakeCatalog is an in-memory CatalogLookup — subjects is the fixed set it
// "knows about", same role catalog-service plays for real over HTTP.
type fakeCatalog struct {
	subjects map[string]catalogclient.Subject // keyed by name
}

func (f *fakeCatalog) Lookup(_ context.Context, _ string, name string) (catalogclient.Subject, bool, error) {
	s, ok := f.subjects[name]
	return s, ok, nil
}

func (f *fakeCatalog) ListAll(context.Context) ([]catalogclient.Subject, error) {
	var out []catalogclient.Subject
	for _, s := range f.subjects {
		out = append(out, s)
	}
	return out, nil
}

// fakeMailer is a ReportMailer that never touches the network — records
// its last call so a test can assert the recipient/subject/attachment it
// was given, and can be told to fail like a real unconfigured/broken
// Mailer would.
type fakeMailer struct {
	err         error // nil = "succeeds"; set to platform.ErrMailerNotConfigured or any other error to simulate that
	lastTo      string
	lastSubject string
	lastAttach  []byte
	calls       int
}

func (f *fakeMailer) Send(to, subject, _ string, _ string, attachment []byte) error {
	f.calls++
	f.lastTo, f.lastSubject, f.lastAttach = to, subject, attachment
	return f.err
}

func newTestServer() (*Server, *fakeStore) {
	fs := &fakeStore{byID: map[int64]models.Campaign{}}
	fc := &fakeCatalog{subjects: map[string]catalogclient.Subject{
		"CRED": {Name: "CRED", Type: "brand", Category: "Fintech"},
	}}
	return &Server{Store: fs, Catalog: fc, InstanceID: "test", Mailer: &fakeMailer{err: platform.ErrMailerNotConfigured}}, fs
}

func createReq(t *testing.T, role string, body models.CreateRequest) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	return req
}

func TestHandleCreateCampaign_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer()
	req := createReq(t, "viewer", models.CreateRequest{Subject: "CRED", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31"})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer create: status = %d, want 403", rr.Code)
	}
}

func TestHandleCreateCampaign_MissingRoleHeaderIsRejected(t *testing.T) {
	s, _ := newTestServer()
	req := createReq(t, "", models.CreateRequest{Subject: "CRED", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31"})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("no X-User-Role: status = %d, want 403 (fail closed, not open)", rr.Code)
	}
}

func TestHandleCreateCampaign_UnknownSubjectIsBadRequest(t *testing.T) {
	s, _ := newTestServer()
	req := createReq(t, "admin", models.CreateRequest{Subject: "NoSuchBrand", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31"})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unknown subject: status = %d, want 400", rr.Code)
	}
}

func TestHandleCreateCampaign_AdminSucceeds(t *testing.T) {
	s, fs := newTestServer()
	req := createReq(t, "admin", models.CreateRequest{Subject: "CRED", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31", Reach: 1000, Spend: 100})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("admin create: status = %d, body = %s, want 201", rr.Code, rr.Body.String())
	}
	if len(fs.created) != 1 || fs.created[0].Subject != "CRED" || fs.created[0].Category != "Fintech" {
		t.Fatalf("expected one stored campaign with the catalog-snapshotted category, got %+v", fs.created)
	}
	if fs.created[0].ApprovalStatus != "pending" {
		t.Fatalf("approvalStatus = %q, want %q — a real create must start as a review checkpoint, not client-controlled", fs.created[0].ApprovalStatus, "pending")
	}
}

func TestHandleCreateCampaign_EditorSucceeds(t *testing.T) {
	s, fs := newTestServer()
	req := createReq(t, "editor", models.CreateRequest{Subject: "CRED", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31", Reach: 1000, Spend: 100})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("editor create: status = %d, body = %s, want 201", rr.Code, rr.Body.String())
	}
	if len(fs.created) != 1 {
		t.Fatalf("expected one stored campaign, got %d", len(fs.created))
	}
}

func TestHandleCreateCampaign_ApproverIsForbidden(t *testing.T) {
	s, _ := newTestServer()
	req := createReq(t, "approver", models.CreateRequest{Subject: "CRED", SubjectType: "brand", Region: "Mumbai", AdType: "Performance", Start: "2026-01-01", End: "2026-01-31"})
	rr := httptest.NewRecorder()
	s.handleCreateCampaign(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("approver create: status = %d, want 403 — creating is editor/admin's tier, not approver's", rr.Code)
	}
}

func TestHandleGetCampaign_NotFound(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/999", nil)
	req.SetPathValue("id", "999")
	rr := httptest.NewRecorder()
	s.handleGetCampaign(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleGetCampaign_Found(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", Status: "Live"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/42", nil)
	req.SetPathValue("id", "42")
	rr := httptest.NewRecorder()
	s.handleGetCampaign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestHandleGetCampaign_NonIntegerID(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	s.handleGetCampaign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func patchBudgetReq(t *testing.T, id, role string, budget *int64) *http.Request {
	t.Helper()
	b, err := json.Marshal(models.UpdateBudgetRequest{Budget: budget})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.SetPathValue("id", id)
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	return req
}

func TestHandleUpdateCampaignBudget_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer()
	budget := int64(50000)
	req := patchBudgetReq(t, "42", "viewer", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer PATCH: status = %d, want 403", rr.Code)
	}
}

func TestHandleUpdateCampaignBudget_NotFound(t *testing.T) {
	s, _ := newTestServer()
	budget := int64(50000)
	req := patchBudgetReq(t, "999", "admin", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleUpdateCampaignBudget_RejectsNegativeBudget(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	budget := int64(-1)
	req := patchBudgetReq(t, "42", "admin", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleUpdateCampaignBudget_AdminSucceedsAndPacingIsComputed(t *testing.T) {
	s, fs := newTestServer()
	// handleUpdateCampaignBudget uses time.Now() internally (not an
	// injectable "today"), so this campaign's flight is pinned safely in
	// the past — whatever the real current time is, elapsedFraction
	// clamps to 1 (see models.PacingOf), and spend so far outstrips this
	// budget at any elapsed fraction from >0 onward, so "over" is the
	// only possible outcome without needing to control "now".
	fs.byID[42] = models.Campaign{
		ID: 42, Subject: "CRED",
		Start: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2020, 1, 30, 0, 0, 0, 0, time.UTC),
		Spend: 90000,
	}
	budget := int64(10000)
	req := patchBudgetReq(t, "42", "admin", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("admin PATCH: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		Budget *int64 `json:"budget"`
		Pacing string `json:"pacing"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Budget == nil || *got.Budget != 10000 {
		t.Fatalf("budget = %v, want 10000", got.Budget)
	}
	if got.Pacing != "over" {
		t.Fatalf("pacing = %q, want %q (spend 90000 against budget 10000)", got.Pacing, "over")
	}
}

func TestHandleUpdateCampaignBudget_EditorSucceeds(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	budget := int64(50000)
	req := patchBudgetReq(t, "42", "editor", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("editor PATCH budget: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateCampaignBudget_ApproverIsForbidden(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	budget := int64(50000)
	req := patchBudgetReq(t, "42", "approver", &budget)
	rr := httptest.NewRecorder()
	s.handleUpdateCampaignBudget(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("approver PATCH budget: status = %d, want 403 — budget is editor/admin's tier, not approver's", rr.Code)
	}
}

func patchApprovalReq(t *testing.T, id, role, status string) *http.Request {
	t.Helper()
	b, err := json.Marshal(models.UpdateApprovalStatusRequest{Status: status})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+id+"/approval", bytes.NewReader(b))
	req.SetPathValue("id", id)
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	return req
}

func TestHandleUpdateApprovalStatus_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer()
	req := patchApprovalReq(t, "42", "viewer", "approved")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer PATCH: status = %d, want 403", rr.Code)
	}
}

func TestHandleUpdateApprovalStatus_NotFound(t *testing.T) {
	s, _ := newTestServer()
	req := patchApprovalReq(t, "999", "admin", "approved")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleUpdateApprovalStatus_RejectsInvalidStatus(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", ApprovalStatus: "pending"}
	req := patchApprovalReq(t, "42", "admin", "on-hold")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleUpdateApprovalStatus_AdminSucceeds(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", ApprovalStatus: "pending"}
	req := patchApprovalReq(t, "42", "admin", "approved")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("admin PATCH: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		ApprovalStatus string `json:"approvalStatus"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ApprovalStatus != "approved" {
		t.Fatalf("approvalStatus = %q, want %q", got.ApprovalStatus, "approved")
	}
}

func TestHandleUpdateApprovalStatus_ApproverSucceeds(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", ApprovalStatus: "pending"}
	req := patchApprovalReq(t, "42", "approver", "approved")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("approver PATCH approval: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateApprovalStatus_EditorIsForbidden(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", ApprovalStatus: "pending"}
	req := patchApprovalReq(t, "42", "editor", "approved")
	rr := httptest.NewRecorder()
	s.handleUpdateApprovalStatus(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("editor PATCH approval: status = %d, want 403 — approval is approver/admin's tier, not editor's", rr.Code)
	}
}

func TestHandleListAnomalies_ReturnsReport(t *testing.T) {
	s, fs := newTestServer()
	fs.anomalyReport = models.AnomalyReport{
		Campaigns:       []models.CampaignAnomaly{{ID: 7, Subject: "Zepto", Category: "E-commerce", Reason: "reach 3.1x category median"}},
		StaleCategories: []models.StaleCategory{{Category: "Beauty & Cosmetics", LastActivity: "2026-08-10", DaysSinceActivity: 27}},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/anomalies", nil)
	rr := httptest.NewRecorder()
	s.handleListAnomalies(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got models.AnomalyReport
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Campaigns) != 1 || got.Campaigns[0].Subject != "Zepto" {
		t.Fatalf("campaigns = %+v, want one flagged campaign for Zepto", got.Campaigns)
	}
	if len(got.StaleCategories) != 1 || got.StaleCategories[0].Category != "Beauty & Cosmetics" {
		t.Fatalf("staleCategories = %+v, want one stale category", got.StaleCategories)
	}
}

func emailReportReq(email string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/email-report", nil)
	if email != "" {
		req.Header.Set("X-User-Email", email)
	}
	return req
}

func TestHandleEmailReport_RequiresAuthenticatedUser(t *testing.T) {
	s, _ := newTestServer()
	req := emailReportReq("")
	rr := httptest.NewRecorder()
	s.handleEmailReport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no X-User-Email: status = %d, want 401", rr.Code)
	}
}

func TestHandleEmailReport_UnconfiguredMailerReturns200NotSent(t *testing.T) {
	s, _ := newTestServer() // newTestServer's fakeMailer defaults to ErrMailerNotConfigured
	req := emailReportReq("viewer@campaigntracker.dev")
	rr := httptest.NewRecorder()
	s.handleEmailReport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 (a degraded capability, not a server error)", rr.Code, rr.Body.String())
	}
	var got struct {
		Sent   bool   `json:"sent"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Sent {
		t.Fatalf("sent = true, want false (mailer not configured)")
	}
	if got.Reason == "" {
		t.Fatalf("reason is empty — must tell the caller honestly why nothing went out")
	}
}

func TestHandleEmailReport_ConfiguredMailerSendsToCallerEmail(t *testing.T) {
	s, _ := newTestServer()
	fm := &fakeMailer{}
	s.Mailer = fm
	req := emailReportReq("someone@example.com")
	rr := httptest.NewRecorder()
	s.handleEmailReport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		Sent bool `json:"sent"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Sent {
		t.Fatalf("sent = false, want true")
	}
	if fm.calls != 1 {
		t.Fatalf("mailer called %d times, want 1", fm.calls)
	}
	if fm.lastTo != "someone@example.com" {
		t.Fatalf("recipient = %q, want the caller's own X-User-Email, not a client-supplied one", fm.lastTo)
	}
}

func creativeReq(t *testing.T, method, campaignID, role string, body *models.CreateCreativeRequest) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req = httptest.NewRequest(method, "/api/v1/campaigns/"+campaignID+"/creatives", bytes.NewReader(b))
	} else {
		req = httptest.NewRequest(method, "/api/v1/campaigns/"+campaignID+"/creatives", nil)
	}
	req.SetPathValue("id", campaignID)
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	return req
}

func TestHandleListCreatives_NotFound(t *testing.T) {
	s, _ := newTestServer()
	req := creativeReq(t, http.MethodGet, "999", "viewer", nil)
	rr := httptest.NewRecorder()
	s.handleListCreatives(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleListCreatives_AnyAuthenticatedRoleSucceeds(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	fs.creatives = []models.Creative{{ID: 1, CampaignID: 42, Headline: "50% off", CreativeType: "image", Reach: 1000, Spend: 100}}
	req := creativeReq(t, http.MethodGet, "42", "viewer", nil)
	rr := httptest.NewRecorder()
	s.handleListCreatives(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("viewer list creatives: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		Data []models.CreativeJSON `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Data) != 1 || got.Data[0].Headline != "50% off" {
		t.Fatalf("data = %+v, want the one seeded creative", got.Data)
	}
}

func TestHandleCreateCreative_RequiresEditorOrAdminRole(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	req := creativeReq(t, http.MethodPost, "42", "viewer", &models.CreateCreativeRequest{Headline: "50% off", CreativeType: "image"})
	rr := httptest.NewRecorder()
	s.handleCreateCreative(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer create creative: status = %d, want 403", rr.Code)
	}
}

func TestHandleCreateCreative_NotFound(t *testing.T) {
	s, _ := newTestServer()
	req := creativeReq(t, http.MethodPost, "999", "editor", &models.CreateCreativeRequest{Headline: "50% off", CreativeType: "image"})
	rr := httptest.NewRecorder()
	s.handleCreateCreative(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleCreateCreative_RejectsInvalidType(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	req := creativeReq(t, http.MethodPost, "42", "editor", &models.CreateCreativeRequest{Headline: "50% off", CreativeType: "billboard"})
	rr := httptest.NewRecorder()
	s.handleCreateCreative(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleCreateCreative_RejectsMissingHeadline(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED"}
	req := creativeReq(t, http.MethodPost, "42", "editor", &models.CreateCreativeRequest{CreativeType: "image"})
	rr := httptest.NewRecorder()
	s.handleCreateCreative(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleCreateCreative_EditorSucceedsAndCampaignUntouched(t *testing.T) {
	s, fs := newTestServer()
	fs.byID[42] = models.Campaign{ID: 42, Subject: "CRED", Reach: 500000, Spend: 20000}
	req := creativeReq(t, http.MethodPost, "42", "editor", &models.CreateCreativeRequest{Headline: "50% off", CreativeType: "image", Reach: 1000, Spend: 100})
	rr := httptest.NewRecorder()
	s.handleCreateCreative(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("editor create creative: status = %d, body = %s, want 201", rr.Code, rr.Body.String())
	}
	var got models.CreativeJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.CampaignID != 42 || got.Headline != "50% off" || got.CreativeType != "image" {
		t.Fatalf("created creative = %+v, unexpected shape", got)
	}
	// The parent campaign's own reach/spend are untouched — creatives are
	// additive, never decomposed back into the campaign total.
	if fs.byID[42].Reach != 500000 || fs.byID[42].Spend != 20000 {
		t.Fatalf("parent campaign was modified: %+v", fs.byID[42])
	}
}
