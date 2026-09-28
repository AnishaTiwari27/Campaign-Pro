package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/catalog/internal/store"
)

const metaCacheKey = "catalog:meta"
const metaCacheTTL = 5 * time.Minute // vocabulary changes rarely — safe to hold longer than analytics' 30s

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		platform.Upstream(w, "database unreachable")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.Cache != nil {
		var cached any
		if s.Cache.Get(ctx, metaCacheKey, &cached) {
			platform.WriteJSON(w, http.StatusOK, cached)
			return
		}
	}

	meta, err := s.Store.Meta(ctx)
	if err != nil {
		slog.Error("load meta", "err", err)
		platform.Internal(w, "failed to load catalog metadata")
		return
	}
	if s.Cache != nil {
		s.Cache.Set(ctx, metaCacheKey, meta, metaCacheTTL)
	}
	platform.WriteJSON(w, http.StatusOK, meta)
}

func (s *Server) handleListSubjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subjects, err := s.Store.ListSubjects(r.Context(), store.SubjectFilter{
		Type:     q.Get("type"),
		Category: q.Get("category"),
		Query:    q.Get("q"),
	})
	if err != nil {
		slog.Error("list subjects", "err", err)
		platform.Internal(w, "failed to list subjects")
		return
	}
	platform.WriteJSON(w, http.StatusOK, subjects)
}

// handleLookupSubject is called by campaigns-service at campaign-create
// time — not meant for the frontend/gateway to expose publicly.
func (s *Server) handleLookupSubject(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subjectType, name := q.Get("type"), q.Get("name")
	if subjectType == "" || name == "" {
		platform.BadRequest(w, "type and name are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	subj, found, err := s.Store.LookupSubject(ctx, subjectType, name)
	if err != nil {
		platform.BadRequest(w, err.Error())
		return
	}
	if !found {
		platform.NotFound(w, "no such "+subjectType)
		return
	}
	platform.WriteJSON(w, http.StatusOK, subj)
}
