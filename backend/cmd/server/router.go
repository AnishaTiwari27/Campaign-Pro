package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/billing"
	"campaigntrackerpro/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// rateLimits is the per-IP request budget, named here so the router does
// not import the whole config.
//
// Two jobs, so two limits. The API budget is deliberately generous: it
// exists to stop one client saturating a small instance, not to shape
// ordinary use — a page of this dashboard fires several reads at once.
//
// The auth budget is the strict one, and it is the one that matters
// here: this deployment's fixture passwords are published in a public
// repository, which makes credential stuffing realistic rather than
// theoretical. It keys on address alone rather than address+email,
// because keying on the email would let an attacker rotate addresses and
// draw a fresh budget for every guess.
type rateLimits struct {
	APIPerMinute  int
	AuthPerMinute int
}

// Both limiters answer in the API's own error shape. The default is
// plain text, which would be the only non-JSON body the API can return
// and would land in the client's parser as a surprise.
func limitJSON(message string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteTooManyRequests(w, message)
	}
}

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

// subscriptionGate answers whether an organisation may use the app.
// Declared here, in the composition root, so platform does not learn
// about billing and billing does not learn about the router.
type subscriptionGate func(ctx context.Context, orgID string) (billing.Subscription, bool, error)

func newRouter(health http.HandlerFunc, auth publicMountable, grants grantLookup,
	gate subscriptionGate, ungatedRoutes func(chi.Router), billingWebhook func(chi.Router),
	limits rateLimits, logger *slog.Logger, modules ...mountable) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// Must come before anything that keys on the caller's address. This
	// deployment is only reachable through Render's proxy, so RemoteAddr
	// is the proxy on every request and the rate limiters below would
	// otherwise put the entire internet in one bucket — a single noisy
	// client would lock everyone out. RealIP reads X-Forwarded-For, which
	// is trustworthy exactly because the proxy is the only way in; an
	// origin exposed directly would let a client forge it.
	r.Use(middleware.RealIP)
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
		r.Use(httprate.Limit(
			limits.APIPerMinute, time.Minute,
			httprate.WithKeyByIP(),
			httprate.WithLimitHandler(limitJSON("too many requests — slow down and try again shortly")),
		))

		// Login sits outside the session guard, which is exactly why it
		// needs a limit of its own: it is the only authenticated-adjacent
		// endpoint an anonymous caller can reach, and it is the one that
		// tells you whether a password was right.
		r.Group(func(r chi.Router) {
			r.Use(httprate.Limit(
				limits.AuthPerMinute, time.Minute,
				httprate.WithKeyByIP(),
				httprate.WithLimitHandler(limitJSON("too many sign-in attempts — wait a minute and try again")),
			))
			auth.PublicRoutes(r)
		})

		// The provider's webhook. Outside the session guard because
		// Razorpay has no session, and outside the gate because the
		// event that says an organisation paid obviously cannot
		// require that it has. Its signature is what authenticates it.
		if billingWebhook != nil {
			billingWebhook(r)
		}

		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireSession(auth.Authenticator(), logger))
			// Immediately after the session is resolved and before any
			// handler runs, so every read below inherits the caller's
			// visibility. A client with no grants sees nothing, which is
			// the safe end of the switch.
			// The tenant wall goes on before the client narrowing, and
			// before any handler: everything below reads inside one
			// organisation or reads nothing at all.
			r.Use(orgScope(logger))
			r.Use(clientScope(grants, logger))
			r.Get("/health", health)

			// Signed in, but in front of the paywall. Two kinds of
			// thing belong here.
			//
			// Billing, because an organisation that cannot use the app
			// is exactly the one that needs to reach it — behind the
			// gate it would be a door that locks from the inside.
			//
			// And identity, because /me is how the app learns who is
			// signed in. Behind the gate it answers 402, the session
			// check fails, and the browser concludes nobody is logged
			// in and shows the sign-in screen — so a lapsed customer
			// is bounced to a login form instead of a way to pay.
			if ungatedRoutes != nil {
				ungatedRoutes(r)
			}

			r.Group(func(r chi.Router) {
				if gate != nil {
					r.Use(requireSubscription(gate, logger))
				}
				for _, m := range modules {
					m.Routes(r)
				}
			})
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

// orgPrincipal is the one thing the tenant middleware asks of the session.
type orgPrincipal interface {
	Organisation() string
}

// orgScope confines the request to the signed-in user's organisation.
//
// A principal that cannot name one is a bug rather than a permission
// problem — every user row carries org_id NOT NULL — so it refuses the
// request instead of carrying on unscoped. Letting it through would mean
// a context with no organisation, and while the queries treat that as
// "match nothing", relying on that as the only defence puts the whole
// boundary on one empty string.
func orgScope(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v, _ := httpx.UserFromContext(r.Context())
			p, ok := v.(orgPrincipal)
			if !ok || p.Organisation() == "" {
				logger.Error("request principal carries no organisation", "path", r.URL.Path)
				httpx.WriteForbidden(w, "this account is not attached to an organisation")
				return
			}
			ctx := scope.WithOrg(r.Context(), scope.Org{ID: p.Organisation()})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireSubscription refuses the whole API when an organisation has no
// active subscription.
//
// It runs after orgScope, because it needs the tenant, and before any
// handler, because the alternative is remembering to check in each of
// them. 402 rather than 403: forbidden tells the client this account
// may never do this and the feature should be hidden, where the truth
// is that it could, as soon as someone pays.
func requireSubscription(gate subscriptionGate, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			org := scope.OrgFrom(r.Context())
			sub, ok, err := gate(r.Context(), org.ID)
			if err != nil {
				// Fail closed. A database that cannot answer "has this
				// tenant paid" is not grounds for assuming it has.
				logger.Error("subscription check failed", "org", org.ID, "err", err)
				httpx.WriteServiceError(w, logger, err)
				return
			}
			if !ok {
				httpx.WritePaymentRequired(w, refusalFor(sub))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// refusalFor says which way the subscription failed, because "trial
// ended" and "payment failed" need different things from the reader
// and a single generic message would send both to the wrong place.
func refusalFor(sub billing.Subscription) string {
	switch sub.Status {
	case billing.StatusTrialing:
		return "Your trial has ended. Choose a plan to carry on."
	case billing.StatusPastDue:
		return "We could not take the last payment. Update your card to restore access."
	case billing.StatusCancelled:
		return "This subscription was cancelled. Choose a plan to start again."
	default:
		return "This organisation does not have a subscription yet."
	}
}
