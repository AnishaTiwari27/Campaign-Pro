# Deployment

Running this somewhere other than a laptop: what the image does, what
has to be configured, and the free hosting path it was built against.

## Deployment

One container serves both halves. `npm run build` output is embedded into the
Go binary (`backend/web`), which serves the API on `/api`, the health probe on
`/health`, and the React app on everything else.

This is not just tidy packaging. The session cookie is `SameSite=Lax` and the
fetch client sends no cross-origin credentials, so hosting the frontend on a
separate origin would break sign-in. Making that work would mean
`SameSite=None` (weaker — that is what CSRF protection is), `AllowCredentials:
true`, an origin allowlist and `credentials: "include"`. Same origin needs none
of it, and because `client.ts` calls `/api` as a relative path, the frontend
needs no build-time configuration at all.

```
make build          # frontend -> backend/web/dist -> ./bin/server
./bin/server        # the whole app on one port
```

The same shape runs on any container host, so moving from a free tier to AWS
(App Runner, ECS/Fargate, Elastic Beanstalk) is a hosting change, not a
rewrite.

### What the image does on boot

1. **Checks configuration it cannot run without.** If `DATABASE_URL` is unset
   on a managed host, the process logs that one sentence and exits 1 rather
   than falling back to the development database and dialling a `localhost`
   Postgres that is not there. A platform marker (`RENDER`, `FLY_APP_NAME`,
   `DYNO`, `K_SERVICE`, and friends) is what distinguishes a deployment from
   a laptop, where the fallback is a convenience and stays.
2. **Applies migrations** from the embedded SQL, then serves. A deploy to a
   blank database comes up with a schema and no separate tooling in the image.
   It uses golang-migrate, the same tool as `make migrate`, so both share its
   `schema_migrations` bookkeeping and cannot disagree. It takes a Postgres
   advisory lock, so two instances starting together cannot both migrate.
   This suits single-instance deployment; a multi-instance rollout should move
   migration to a release-phase command so schema changes land before new code
   serves traffic.
3. **Logs the database it reached** — host, port, database and user, never the
   password. A deploy quietly pointing at the wrong database is expensive to
   notice any later than this.
4. **Logs whether a frontend is embedded** (`"frontend":true`). An image built
   without one serves a notice page on every route, which is far easier to
   diagnose from one log line than from a blank tab.

### Required configuration

| Env var | Production value |
|---|---|
| `DATABASE_URL` | Your managed Postgres URL, including `sslmode=require`. Either of Neon's strings works: migrations are routed around a pooled host automatically (see below), while the app's pool uses the URL as given. Enter it bare — surrounding quotes and stray whitespace are stripped on load, so a value pasted out of a shell command still works |
| `SECURE_COOKIES` | **`true`** — without it the session cookie ships without `Secure` |
| `TZ` | `Asia/Kolkata` — every report cadence is computed in IST |
| `PORT` | Whatever the platform injects; defaults to `8090` |
| `ALLOW_SIGNUP` | `true` (the default) exposes the public signup page and `POST /api/auth/signup`. Set it to `false` to close registration once the accounts that should exist do — the route is then not registered at all, rather than registered and refusing |
| `SIGNUP_ORG_ID` | The organisation a self-registered account joins; defaults to `default`. Signup is a public route, so there is no session to read a tenant from and one has to be named. It must be a row in `organizations` — `users.org_id` is a foreign key, so a value that is not there makes every signup fail |

### Sign in with Google (OAuth / OIDC)

Optional. Without it the password form is the only way in; with it the
sign-in and signup screens both offer **Continue with Google**, and an
address Google vouches for gets an analyst account on first use without
choosing a password at all.

Any OIDC-compliant provider works — the four variables below are all
that differs between Google, Microsoft Entra, Okta and Auth0. The button
names whichever one the issuer identifies, derived from the issuer so
the label cannot name a provider other than the one it reaches.

In Google Cloud Console: **APIs & Services → Credentials → Create
credentials → OAuth client ID → Web application**. Add one authorised
redirect URI, `https://your-host/api/auth/sso/callback`. Then:

| Env var | Value |
|---|---|
| `OIDC_ISSUER` | `https://accounts.google.com` |
| `OIDC_CLIENT_ID` | From the console |
| `OIDC_CLIENT_SECRET` | From the console. A secret: set it on the service, never in the repo |
| `OIDC_REDIRECT_URL` | `https://your-host/api/auth/sso/callback` — must match what you registered character for character, or Google refuses the exchange |
| `OIDC_ORG_ID` | The organisation SSO users join; defaults to `default` |

All four are required together. A partial set leaves SSO off, because
offering a button that the provider will reject is worse than not
offering one. Discovery runs once at boot: if the issuer is unreachable
the log says so and the app continues with password sign-in rather than
failing to start.

Everyone who can sign in with the provider gets an analyst account —
read everything in scope, decide nothing. Restrict who that is on the
provider's side, which for a Google Workspace domain is the OAuth
client's own publishing setting, not something this app can enforce.

What the flow checks, because each is a real attack if skipped: `state`
(CSRF), `nonce` (replay), PKCE `S256` (code interception), the RS256
signature against the provider's JWKS, and `iss`/`aud`/`exp`/`iat`. An
unverified email is refused outright. See `platform/oidc`.

### Connecting a real ad account

Optional, and the only way the per-campaign figures become real. Until
one of these is connected the numbers in the app are fixtures; no
amount of public data substitutes for them, because a campaign's reach
and spend are not published by anyone.

Neither API costs money. What they need is authorisation on the account
whose data you are reading — which is the client's to give.

**Google Ads.** Create a Google Cloud project, enable the Google Ads
API on it, and generate an OAuth client plus a refresh token for a user
with access to the Ads account.

| Env var | Value |
|---|---|
| `GOOGLE_ADS_CLIENT_ID` / `GOOGLE_ADS_CLIENT_SECRET` | From the Cloud project's OAuth client |
| `GOOGLE_ADS_REFRESH_TOKEN` | For a user authorised on the Ads account |
| `GOOGLE_ADS_CUSTOMER_ID` | The account to read, hyphens optional |
| `GOOGLE_ADS_LOGIN_CUSTOMER_ID` | The manager (MCC) account, when the customer sits under one |
| `GOOGLE_ADS_DEVELOPER_TOKEN` | Optional, and only if you already hold one — see below |

**Developer tokens were sunset on 9 September 2026.** Access levels
(Test, Explorer, Basic, Standard) now attach to the Google Cloud
project that issued the OAuth credentials, and you request a level on
that project's Google Ads API page in the Cloud console; Basic is
reviewed automatically once the project is brand-verified. A token sent
in a request is accepted and ignored, so an older integration keeps
working — but a setup starting today cannot obtain one, which is why
this app does not require it.

New projects start at Test access, which reaches **test accounts only**.
A call against a production account will fail until the project is
raised to at least Basic.

The API version is pinned in `services/ingest/googleads.go` rather than
tracking "latest", because a version change can move field semantics
underneath a sync. The flip side is that a pin goes stale: check it
against Google's deprecation and sunset page before a first run.

**Meta.** A system user access token on the ad account, plus the
account id:

| Env var | Value |
|---|---|
| `META_ACCESS_TOKEN` | A system user token with `ads_read` |
| `META_AD_ACCOUNT_ID` | `act_<id>` |

The token is the credential itself — there is no refresh exchange — so
it expires on Meta's schedule and has to be reissued rather than
renewed.

**Then prove it before trusting it:**

```bash
make verify-ingest SOURCE=google-ads
make verify-ingest SOURCE=meta
```

Both clients are written from the published references and covered by
tests against recorded response shapes, but neither has been run
against a live account. Treat them as unproven until that command
passes.

A note on what these do *not* unlock: your own accounts, not anyone
else's. Competitors' real spend and reach are not purchasable — Meta's
Ad Library archive API covers only political and issue ads in India,
and what the competitive-intelligence vendors sell is modelled
estimates, not actuals.

### Creating the first account

**Never run `make seed` against a deployment.** It begins with
`TRUNCATE audit_events, creatives, report_runs, reports, campaigns, creators,
users` — it is a fixture loader, and it would erase production.

There is no user-management endpoint yet, so accounts are made with
`cmd/createuser`, which creates exactly one and touches nothing else. Run it
from your machine against the deployment's database:

```bash
DATABASE_URL='postgres://...' go run ./cmd/createuser \
  -email you@example.com -name 'Your Name' -role admin
```

It prints a generated password once. Re-running it for an existing email
re-passwords that account, which is also the only way back in if a password is
lost, since there is no reset flow. The binary is in the image as
`createuser` too, for hosts that give you a shell.

### Render + Neon (the free path)

[`render.yaml`](render.yaml) declares the web service. The database is
deliberately **not** on Render: its free Postgres expires 30 days after
creation and is then deleted. Neon's free plan is permanent, so
`DATABASE_URL` points there.

**Setting `DATABASE_URL`, the one manual step.** The key is declared
`sync: false`, which means Render never reads its value from `render.yaml` —
a secret in the repository is worse than a step in a runbook. Render prompts
for it once, when the Blueprint first creates the service. It is *not* filled
in by a later Blueprint sync, so a service that predates the file, or one
where the prompt was skipped, runs with it empty:

```
ERROR  invalid configuration  err="DATABASE_URL is not set, and this process
       is running on a managed host where the local development fallback
       cannot work. ..."
==> Exited with status 1
```

The fix is the service's **Environment** tab → add `DATABASE_URL` → paste the
Neon string (no quotes) → save, which redeploys. A healthy boot says so:

```
INFO  schema up to date      version=5
INFO  database connected     host=ep-….aws.neon.tech port=5432 database=neondb user=…
INFO  server listening       port=10000 frontend=true
```

If `host` there is `localhost`, the variable did not take effect; nothing else
in the log is worth reading first.

**On connection pooling.** golang-migrate serialises migrations with a
session-scoped `pg_advisory_lock`, and Neon's pooled endpoint is PgBouncer in
transaction-pooling mode, which does not keep a session pinned to one backend
connection — the lock and its unlock can land on different ones, leaving it
held by an idle connection while migrations hang. Pooling is still right for
ordinary traffic, so `migrateURL` rewrites only the migration connection onto
Neon's direct host and leaves the app's pool using `DATABASE_URL` as given.
That means either Neon string can be pasted in without it being a trap.

Known trade-off: Render's free tier sleeps after about 15 minutes idle and
takes roughly a minute to wake. While it sleeps the in-process worker is not
running, so scheduled reports do not fire on time and anomaly detection only
runs while someone is using the app.

[`.github/workflows/keep-awake.yml`](.github/workflows/keep-awake.yml) pings
`/health` every 10 minutes to hold it open, which also serves as a crude
uptime check. Treat it as a workaround rather than a fix: GitHub's scheduled
runs are frequently 5-15 minutes late, so the service still naps sometimes;
GitHub disables the schedule after 60 days without a commit; and holding one
service awake uses about 730 of the free plan's 750 instance-hours a month,
which fits for exactly one service. The actual fix is a paid instance.

### Before scaling past one instance

The anomaly detector and report scheduler run in-process on a 30-second
ticker in `cmd/server/main.go`. Two instances means every scheduled report
sends twice. Keep this at one instance until that work moves behind a lock or
into a separate scheduler.

---
