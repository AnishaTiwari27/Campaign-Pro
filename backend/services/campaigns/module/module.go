// Package module is the campaigns service's factory. It is the only way
// in: everything it constructs lives under the service's internal/, which
// the compiler forbids any sibling service or the composition root from
// importing. Callers get capabilities, never implementations.
package module

import (
	"context"
	"errors"
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/campaigns/internal/api"
	"campaigntrackerpro/services/campaigns/internal/service"
	"campaigntrackerpro/services/campaigns/internal/store"
	"campaigntrackerpro/services/identity"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Campaigns
	anomaly *service.AnomalyDetector
	handler *api.Handlers
	store   *store.Store
}

// New wires this service's own layers from the shared database.
func New(db *database.DB, logger *slog.Logger) *Module {
	st := store.New(db)
	svc := service.NewCampaigns(st)
	return &Module{
		svc:     svc,
		anomaly: service.NewAnomalyDetector(svc, logger),
		handler: api.New(svc, logger),
		store:   st,
	}
}

// Routes mounts the service's endpoints under the caller's router.
func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Reader is the read capability sibling services consume. Analytics,
// creators and reports each depend on this narrow interface rather than on
// the concrete service, so none of them can reach past what they need.
type Reader interface {
	List(ctx context.Context, p campaigns.ListParams) (campaigns.ListResult, error)
	Anomalies(ctx context.Context) ([]campaigns.Campaign, error)
	RowOf(ctx context.Context, c campaigns.Campaign) campaigns.CampaignRow
}

// Service hands back the concrete type for siblings that need more than
// Reader. Still unreachable without going through this package.
func (m *Module) Service() *service.Campaigns { return m.svc }

var _ Reader = (*service.Campaigns)(nil)

// Detector re-runs anomaly detection; the reports worker triggers it.
func (m *Module) Detector() *service.AnomalyDetector { return m.anomaly }

// Store exposes campaign-owned writes to the seeder, which loads fixtures
// across every service and is deliberately allowed past the service layer.
func (m *Module) Store() *store.Store { return m.store }

// LoadUser resolves the acting account for platform's auth middleware.
// Campaigns owns the users table, so it supplies the loader.
func (m *Module) LoadUser(ctx context.Context, email string) (any, error) {
	return m.store.GetUserByEmail(ctx, email)
}

// UserDirectory hands out the account-creation capability that identity's
// signup needs. Campaigns owns the users table, so the write lives here
// and identity asks for it through an interface of its own — neither
// service imports the other.
func (m *Module) UserDirectory() userDirectory { return userDirectory{m.store} }

// userDirectory is unexported on purpose: callers receive it, satisfy
// identity's Directory with it, and cannot reach past these two methods.
type userDirectory struct{ store *store.Store }

func (d userDirectory) EmailTaken(ctx context.Context, email string) (bool, error) {
	_, err := d.store.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, database.ErrNotFound):
		return false, nil
	default:
		return false, err
	}
}

// CreateUser writes the account and returns its ID. can_approve is derived
// from the role here rather than taken from the caller, so the column and
// identity's CanApprove() cannot be made to disagree about one account.
func (d userDirectory) CreateUser(ctx context.Context, email, name, role string, isAgency bool) (string, error) {
	// "admin" and "approver" are the user_role_t values that carry
	// sign-off; the enum is defined in the migrations and mirrored by
	// identity.Role.
	canApprove := isAgency && (role == "admin" || role == "approver")
	u, err := d.store.CreateUser(ctx, email, name, role, canApprove)
	if err != nil {
		return "", err
	}
	if err := d.store.SetUserAgency(ctx, u.ID, isAgency); err != nil {
		return "", err
	}
	return u.ID, nil
}

// ListUsers is the roster. The shape is identity's, filled from campaigns'
// own rows: the two services share a vocabulary of primitives rather than
// a type.
func (d userDirectory) ListUsers(ctx context.Context) ([]identity.DirectoryUser, error) {
	us, err := d.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]identity.DirectoryUser, 0, len(us))
	for _, u := range us {
		out = append(out, identity.DirectoryUser{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role,
			IsAgency: u.IsAgency, CanApprove: u.CanApprove, CreatedAt: u.CreatedAt,
		})
	}
	return out, nil
}

// SetRole changes a user's role. can_approve is derived from it inside the
// store, where it cannot drift from identity's CanApprove().
func (d userDirectory) SetRole(ctx context.Context, userID, role string, isAgency bool) error {
	return d.store.SetUserRole(ctx, userID, role, isAgency)
}
