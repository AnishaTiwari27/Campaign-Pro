# Campaign Tracker Pro — Frontend

Vite + React dashboard, wired to the gateway (`../services/gateway`), which
fronts the rest of the microservices — see
[`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md).

## Run

```bash
npm install
cp .env.example .env.local   # only if the gateway isn't at the default URL
npm run dev
```

Opens on `http://localhost:5173` — the gateway's default `ALLOWED_ORIGIN`.
Start the backend first: at minimum the gateway, auth-service,
catalog-service, campaigns-service, and analytics-service (each
`go run ./cmd/server` under `../services/*`), plus Postgres/Redis/NATS —
or the dashboard's login screen and KPI tiles/charts will show the red
error banner instead of data.

The dashboard is behind a login gate (`src/AuthGate.jsx`) — see
[`../docs/API_CONTRACT.md`](../docs/API_CONTRACT.md)'s Auth section for the
seeded dev credentials.

## Layout

- `CampaignTrackerDashboard.jsx` — the shell: left nav, top bar, the shared
  filter row, and the KPI ticker.
- `tabs/*.jsx` — one component per nav tab (Overview, Campaigns, Region,
  Reports), each fetching only its own data on activation rather than the
  shell fetching everything for every tab up front.
- `api/client.js` / `api/auth.js` — the only files that talk to the
  network. `auth/session.js` holds the access token (memory) and refresh
  token (`localStorage`) and notifies `AuthGate` on login/logout.
- `components/` — pieces shared across tabs (`CampaignDrawer`,
  `SubjectTypeIcon`).

Full response shapes: [`../docs/API_CONTRACT.md`](../docs/API_CONTRACT.md).
