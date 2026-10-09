package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
	"campaigntrackerpro/services/identity/internal/service"

	"github.com/go-chi/chi/v5"
)

// UserRoutes mount behind the session guard. Every one of them re-checks
// CanManageUsers in the service, so hiding the screen in the UI is a
// convenience and never the control.
func (h *Handlers) UserRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)
		r.Post("/", h.CreateUser)
		r.Patch("/{id}", h.SetUserRole)
		r.Post("/{id}/reset-password", h.ResetUserPassword)
		r.Get("/{id}/grants", h.GetGrants)
		r.Put("/{id}/grants", h.SetGrants)
	})
}

type directoryUserDTO struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	IsAgency   bool      `json:"isAgency"`
	CanApprove bool      `json:"canApprove"`
	CreatedAt  time.Time `json:"createdAt"`
	// IsSelf lets the screen disable the controls an admin must not use on
	// their own account, without the client having to compare ids itself.
	IsSelf bool `json:"isSelf"`
}

func toDirectoryDTO(u identity.DirectoryUser, actorID string) directoryUserDTO {
	return directoryUserDTO{
		ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role,
		IsAgency: u.IsAgency, CanApprove: u.CanApprove, CreatedAt: u.CreatedAt,
		IsSelf: u.ID == actorID,
	}
}

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	users, err := h.auth.Users(r.Context(), actor)
	if err != nil {
		h.writeUserError(w, err)
		return
	}
	out := make([]directoryUserDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toDirectoryDTO(u, actor.ID))
	}
	roles := make([]string, 0, len(service.ManageableRoles))
	for _, role := range service.ManageableRoles {
		roles = append(roles, string(role))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out, "roles": roles})
}

type newUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsAgency bool   `json:"isAgency"`
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	var req newUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	user, password, err := h.auth.CreateUser(r.Context(), actor, service.NewUserInput{
		Name: req.Name, Email: req.Email, Role: req.Role, IsAgency: req.IsAgency,
	})
	if err != nil {
		h.writeUserError(w, err)
		return
	}
	// The password is in this response and nowhere else, ever again.
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"user": toDirectoryDTO(user, actor.ID), "password": password,
	})
}

type setRoleRequest struct {
	Role     string `json:"role"`
	IsAgency bool   `json:"isAgency"`
}

func (h *Handlers) SetUserRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	var req setRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	if err := h.auth.SetRole(r.Context(), actor, chi.URLParam(r, "id"), req.Role, req.IsAgency); err != nil {
		h.writeUserError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handlers) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	password, err := h.auth.ResetPassword(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		h.writeUserError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"password": password})
}

// writeUserError maps the shared failures of these four handlers once.
func (h *Handlers) writeUserError(w http.ResponseWriter, err error) {
	if msg, field, ok := service.AsValidationError(err); ok {
		httpx.WriteValidationError(w, msg, field)
		return
	}
	if errors.Is(err, service.ErrEmailTaken) {
		httpx.WriteConflict(w, err.Error())
		return
	}
	httpx.WriteServiceError(w, h.logger, err)
}

type campaignChoiceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Meta string `json:"meta"`
}

// GetGrants is which campaigns a client may see, and everything they could
// be given.
func (h *Handlers) GetGrants(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	granted, choices, err := h.auth.Grants(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		h.writeUserError(w, err)
		return
	}
	opts := make([]campaignChoiceDTO, 0, len(choices))
	for _, c := range choices {
		opts = append(opts, campaignChoiceDTO{ID: c.ID, Name: c.Name, Meta: c.Meta})
	}
	if granted == nil {
		granted = []string{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"granted": granted, "campaigns": opts})
}

type setGrantsRequest struct {
	CampaignIDs []string `json:"campaignIds"`
}

func (h *Handlers) SetGrants(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(r)
	if !ok {
		httpx.WriteUnauthorized(w, "sign in to continue")
		return
	}
	var req setGrantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidationError(w, "invalid JSON body", "")
		return
	}
	if err := h.auth.SetGrants(r.Context(), actor, chi.URLParam(r, "id"), req.CampaignIDs); err != nil {
		h.writeUserError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
