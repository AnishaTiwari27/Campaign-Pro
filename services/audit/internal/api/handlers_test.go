package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"campaigntrackerpro/services/audit/internal/models"
)

// fakeStore is an in-memory AuditStore — enough to exercise handler logic
// (role gating, pagination/filter passthrough) without a real Postgres.
type fakeStore struct {
	entries []models.Entry

	lastPage, lastLimit int
	lastCampaignID      *int64
}

func (f *fakeStore) Ping(context.Context) error { return nil }

func (f *fakeStore) List(_ context.Context, page, limit int, campaignID *int64) ([]models.Entry, int, error) {
	f.lastPage, f.lastLimit, f.lastCampaignID = page, limit, campaignID
	return f.entries, len(f.entries), nil
}

func newTestServer() (*Server, *fakeStore) {
	fs := &fakeStore{}
	return &Server{Store: fs}, fs
}

func withRole(req *http.Request, role string) *http.Request {
	req.Header.Set("X-User-Role", role)
	return req
}

func TestHandleListAuditLog_RequiresAdminRole(t *testing.T) {
	s, _ := newTestServer()
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/audit-log", nil), "viewer")
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer: status = %d, want 403", rr.Code)
	}
}

func TestHandleListAuditLog_AdminSucceeds(t *testing.T) {
	s, fs := newTestServer()
	fs.entries = []models.Entry{{ID: 1, Action: "campaign.created", ActorEmail: "admin@campaigntracker.dev", ActorRole: "admin", Details: "created"}}
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/audit-log", nil), "admin")
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("admin: status = %d, body = %s, want 200", rr.Code, rr.Body.String())
	}
	var got struct {
		Data  []models.EntryJSON `json:"data"`
		Page  int                `json:"page"`
		Limit int                `json:"limit"`
		Total int                `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Data) != 1 || got.Data[0].Action != "campaign.created" {
		t.Fatalf("data = %+v, want the one seeded entry", got.Data)
	}
	if got.Page != 1 || got.Limit != 25 {
		t.Fatalf("page/limit = %d/%d, want defaults 1/25", got.Page, got.Limit)
	}
}

func TestHandleListAuditLog_PassesCampaignIDFilter(t *testing.T) {
	s, fs := newTestServer()
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/audit-log?campaignId=68&page=2&limit=10", nil), "admin")
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if fs.lastCampaignID == nil || *fs.lastCampaignID != 68 {
		t.Fatalf("campaignID passed to store = %v, want 68", fs.lastCampaignID)
	}
	if fs.lastPage != 2 || fs.lastLimit != 10 {
		t.Fatalf("page/limit passed to store = %d/%d, want 2/10", fs.lastPage, fs.lastLimit)
	}
}

func TestHandleListAuditLog_RejectsNonIntegerCampaignID(t *testing.T) {
	s, _ := newTestServer()
	req := withRole(httptest.NewRequest(http.MethodGet, "/api/v1/audit-log?campaignId=abc", nil), "admin")
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
