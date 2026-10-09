package main

import (
	"context"
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// mountable is what every service module offers the router: its own URL
// space. The composition root never names an individual path — services
// own their routes.
type mountable interface {
	Routes(r chi.Router)
}

// publicMountable serves routes that must work without a session — login
// being the obvious one, since requiring a session to sign in is circular.
type publicMountable interface {
	PublicRoutes(r chi.Router)
	Authenticator() httpx.SessionAuthenticator
}

func newRouter(health http.HandlerFunc, auth publicMountable, grants grantLookup,
	logger *slog.Logger, modules ...mountable) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpx.RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// Unauthenticated, so a readiness probe doesn't need credentials.
	r.Get("/health", health)

	r.Route("/api", func(r chi.Router) {
		// Login sits outside the guard; everything else sits behind it.
		auth.PublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireSession(auth.Authenticator(), logger))
			// Immediately after the session is resolved and before any
			// handler runs, so every read below inherits the caller's
			// visibility. A client with no grants sees nothing, which is
			// the safe end of the switch.
			r.Use(clientScope(grants, logger))
			r.Get("/health", health)
			for _, m := range modules {
				m.Routes(r)
			}
		})
	})

	// Everything that is not the API is the single-page app, served from the
	// same origin so the SameSite=Lax session cookie is sent with every
	// request. This is registered last and as the fallback, so it can never
	// shadow a route a service owns. /api and /health are passed through as
	// prefixes the SPA must not answer for, keeping their 404s JSON.
	r.NotFound(web.Handler("/api", "/health").ServeHTTP)

	return r
}

// grantLookup is what the scope middleware needs: the campaigns one client
// account may see. Declared here, in the composition root, so neither
// platform nor a service takes a dependency on the other for it.
type grantLookup interface {
	For(ctx context.Context, userID string) ([]string, error)
}

// clientPrincipal is the one thing the middleware asks of the session.
type clientPrincipal interface {
	IsClient() bool
	PrincipalID() string
}

// clientScope narrows what a client account can read, for the whole
// request, before any handler sees it.
//
// Agency staff pass through unrestricted. A client gets their granted
// campaign ids — and if the lookup fails, they get an empty restricted
// scope rather than an unrestricted one: a database hiccup must not be a
// way to see everybody's data.
func clientScope(grants grantLookup, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v, _ := httpx.UserFromContext(r.Context())
			p, ok := v.(clientPrincipal)
			if !ok || !p.IsClient() {
				next.ServeHTTP(w, r)
				return
			}
			ids, err := grants.For(r.Context(), p.PrincipalID())
			if err != nil {
				logger.Error("could not resolve client grants — showing nothing", "err", err)
				ids = nil
			}
			ctx := scope.WithCampaigns(r.Context(), scope.Campaigns{Restricted: true, IDs: ids})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
