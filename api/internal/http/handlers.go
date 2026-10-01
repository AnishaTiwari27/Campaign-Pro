package httpapi

import (
	"log/slog"

	"campaigntrackerpro/internal/service"
)

// Handlers holds every dependency the route handlers need. One struct kept
// deliberately small: services already encapsulate their own logic, so
// handlers stay thin — parse request, call service, write response.
type Handlers struct {
	Campaigns *service.Campaigns
	Overview  *service.OverviewService
	Reports   *service.Reports
	Creators  *service.Creators
	Anomaly   *service.AnomalyDetector
	Logger    *slog.Logger
}
