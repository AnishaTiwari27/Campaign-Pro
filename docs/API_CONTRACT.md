# API Contract — v1

Base URL: `/api/v1` off the gateway (`http://localhost:8080` in dev). JSON
in, JSON out, except `/campaigns/export` (CSV). All list endpoints share the
same filter query params.

## Shared query params (list + aggregate endpoints)

| Param         | Type   | Example       | Notes                                     |
|----------------|--------|----------------|---------------------------------------------|
| `category`    | string | `Fintech`      | omit or `All` for no filter                |
| `region`      | string | `Bengaluru`    | omit or `All`                              |
| `adType`      | string | `Performance`  | omit or `All`                              |
| `subjectType` | string | `brand`        | `brand` \| `person`, omit or `All`         |
| `subject`     | string | `CRED`         | exact subject name (drawer "focus subject")|
| `q`           | string | `myn`          | case-insensitive substring match on subject|
| `days`        | int    | `30`           | campaigns started within the last N days   |
| `page`        | int    | `1`            | default `1`                                |
| `limit`       | int    | `25`           | default `25`, max `100`                    |

Auth: `Authorization: Bearer <access_token>` is required on every route
below except `/auth/*` and `/healthz` — see the Auth section and
[`ARCHITECTURE.md`](ARCHITECTURE.md#auth) for the trust model.

---

## `GET /campaigns`

List campaigns, newest `start_date` first.

```json
{
  "data": [
    {
      "id": 1,
      "subject": "CRED",
      "subjectType": "brand",
      "category": "Fintech",
      "region": "Bengaluru",
      "adType": "Performance",
      "platform": "Meta Ads",
      "start": "2026-07-02",
      "end": "2026-07-23",
      "reach": 412000,
      "spend": 187818,
      "budget": 200000,
      "status": "Completed",
      "pacing": "on",
      "approvalStatus": "approved"
    }
  ],
  "page": 1,
  "limit": 25,
  "total": 48
}
```

`spend` is rupees (integer), already divided down from `spend_paise`.
`subjectType` is `"brand"` or `"person"` — a campaign can be about either.
`budget` and `pacing` are both omitted entirely (not `null`) when no budget
is set — most campaigns today. `pacing` (`"under"` \| `"on"` \| `"over"`) is
derived at read time from `budget`/`spend`/how much of the campaign's
start→end flight has elapsed; see [`ARCHITECTURE.md`](ARCHITECTURE.md) for
the exact formula and thresholds. `approvalStatus`
(`"pending"` \| `"approved"` \| `"rejected"`) is always present, unlike
`budget`/`pacing` — every campaign has a real value here, not a "maybe
absent" one; see `PATCH /campaigns/{id}/approval`.

## `GET /campaigns/{id}`

Single campaign — same shape as one row of `/campaigns`, or `404`.

## `POST /campaigns`

`editor` or `admin` (`403` for `viewer`/`approver` — see the Auth
section). Body:

```json
{
  "subject": "CRED", "subjectType": "brand", "region": "Bengaluru",
  "adType": "Performance", "platform": "Meta Ads",
  "start": "2026-08-01", "end": "2026-08-30", "reach": 100000, "spend": 5000
}
```

`subject`/`subjectType` are validated against catalog-service (`404`-shaped
`400` if no such subject exists); `category` is snapshotted from there, not
sent by the caller. `budget` (rupees) is optional. `approvalStatus` always
starts `"pending"` — there's no request field that sets it otherwise, so a
real admin-submitted campaign can't skip review (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#campaign-approval)). Returns `201`
with the created campaign (same shape as `GET /campaigns/{id}`).

## `PATCH /campaigns/{id}`

`editor` or `admin`. Sets or clears a campaign's budget — there's no
campaign-creation UI yet (see `docs/ROADMAP.md`), so this is how a budget
actually gets set day to day, not just at creation. Body:

```json
{ "budget": 200000 }
```

`budget: null` clears it. Negative values are `400`. Returns `200` with
the updated campaign (`pacing` freshly recomputed), or `404` if the id
doesn't exist.

## `PATCH /campaigns/{id}/approval`

`approver` or `admin` — a different role tier than the budget PATCH above
(`editor`/`admin`), real separation of duties (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#rbac--roles-beyond-vieweradmin)), and
a separate *route* too (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#campaign-approval) for why one request
body covering two independent optional fields is a real correctness risk
this avoids). Body:

```json
{ "status": "approved" }
```

`status` must be `"pending"`, `"approved"`, or `"rejected"` — `400`
otherwise. Returns `200` with the updated campaign, or `404` if the id
doesn't exist. Not a visibility gate: a `"pending"` campaign is still
fully visible in every list/export/aggregate — this only flags it for
review.

## `GET /campaigns/export`

Same filters as `/campaigns`, no pagination — streams the full filtered set as
`text/csv`, `Content-Disposition: attachment; filename="campaign-report.csv"`.
Columns: `Subject,Subject Type,Category,Region,Ad Type,Platform,Start,Reach,Spend,Budget,Approval`
(`Budget` blank when unset).

## `GET /campaigns/anomalies`

A global "what needs attention" signal — deliberately **ignores** every
filter param above (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#anomaly-detection) for why). No admin
gate (read-only, same openness as `/campaigns`/`/campaigns/export`).

```json
{
  "campaigns": [
    { "id": 58, "subject": "CRED", "category": "Fintech", "reason": "spend 6.0x category median" }
  ],
  "staleCategories": [
    { "category": "Beauty & Cosmetics", "lastActivity": "2026-08-25", "daysSinceActivity": 17 }
  ]
}
```

`campaigns` flags a campaign whose reach or spend is a statistical outlier
against its own category's peers (only when that category has enough peers
to make "median" meaningful). `staleCategories` flags a category with real
history that's gone quiet — no new campaign in 14+ days. Either array may
be empty (never omitted).

## `POST /campaigns/email-report`

Any authenticated role — same openness as export. Same filters as
`/campaigns`/`/campaigns/export`, via the query string (no request body).
Emails the filtered CSV export to the *caller's own* account email (the
trusted `X-User-Email` the gateway sets — never a client-supplied
recipient), rather than downloading it.

```json
{ "sent": true, "reason": "" }
```

Always `200` — this reports a degraded capability honestly, it never fails
the request just because SMTP isn't configured
(see [`ARCHITECTURE.md`](ARCHITECTURE.md#scheduledexported-reports)):

```json
{ "sent": false, "reason": "email delivery is not configured in this environment" }
```

## `GET /campaigns/{id}/creatives`

Any authenticated role — same openness as reading the campaign itself.

```json
{ "data": [{ "id": 1, "campaignId": 71, "headline": "50% off this weekend", "creativeType": "video", "reach": 25000, "spend": 1200, "createdAt": "2026-09-12T21:30:03+05:30" }] }
```

## `POST /campaigns/{id}/creatives`

`editor` or `admin` — same tier as creating a campaign. Body:

```json
{ "headline": "50% off this weekend", "creativeType": "video", "reach": 25000, "spend": 1200 }
```

`creativeType` must be `"image"`, `"video"`, `"carousel"`, or `"text"` —
`400` otherwise. `headline` is required. `reach`/`spend` default to `0`,
`400` if negative. `404` if the campaign id doesn't exist. **Additive,
not decomposed** — creating a creative never changes the parent
campaign's own `reach`/`spend` (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#creative-level-tracking)). Returns
`201` with the created creative.

## `GET /kpis`

Powers the four KPI tiles + their sparklines.

```json
{
  "activeCampaigns": { "value": 48, "delta": "+12%", "up": true, "spark": [3,5,4,6,7,5,8,6] },
  "subjectsTracked": { "value": 12, "delta": "+3",   "up": true, "spark": [1,1,2,2,2,3,3,3] },
  "estimatedReach":  { "value": 8412000, "delta": "+8.4%", "up": true, "spark": [...] },
  "estimatedSpend":  { "value": 3819200, "delta": "+4 new", "up": true, "spark": [...] }
}
```

`value` for reach/spend is a raw number; the client formats it (`8.4L`, `₹38.2L`).
`spark` is 8 weekly buckets, oldest → newest.

## `GET /trend`

Powers the "Campaign activity by format" chart. One row per month present in
the filtered set, one field per ad type.

```json
[
  { "month": "Apr", "Social Media": 4, "Performance": 2, "Video": 1 },
  { "month": "May", "Google Ads": 3, "Display": 2 }
]
```

## `GET /regions`

Powers the region bar chart. Always all 6 regions, zero-filled.

```json
[
  { "region": "Pan-India", "campaigns": 14 },
  { "region": "South Zone", "campaigns": 9 }
]
```

## `GET /benchmark`

Powers the competitor benchmarking table. Top 10 subjects by the sort key.

Extra params: `sort` (`subject`|`count`|`reach`|`last`, default `count`), `dir` (`asc`|`desc`, default `desc`).

```json
[
  {
    "subject": "Zepto",
    "subjectType": "brand",
    "category": "E-commerce",
    "count": 6,
    "reach": 2140000,
    "platforms": ["Meta Ads", "Google Search"],
    "last": "2026-08-20",
    "trend": [1820000, 1820000, 1940000, 1940000, 1940000, 2140000, 2140000, 2140000]
  }
]
```

`trend` is up to 8 recent snapshot points (reach), oldest → newest —
omitted entirely (not an empty array) until at least one
`campaigns.benchmark_snapshots` row exists for that subject; see
[`ARCHITECTURE.md`](ARCHITECTURE.md#benchmark-trending).

## `GET /meta`

Static vocabularies for the filter dropdowns — fetched once on load.
`categories` is split by which subject type each applies to (most
categories apply to only one; `"Luxury"` is the one exception, valid for
both) so the frontend's Type filter shows the right list.

```json
{
  "categories": {
    "brand": ["E-commerce", "Fashion", "Food Delivery", "Technology", "Fintech", "FMCG", "Beauty & Cosmetics", "Footwear", "Luxury"],
    "person": ["Influencer", "Music", "Entertainment", "Business", "Luxury"]
  },
  "regions": ["Mumbai", "Delhi NCR", "Bengaluru", "South Zone", "West Zone", "Pan-India"],
  "adTypes": ["Social Media", "Influencer", "Google Ads", "Display", "Video", "Performance"]
}
```

## `GET /subjects`

Brands and/or people matching `type`/`category`/`q` — powers subject
search/autocomplete. Flattened shape:

```json
[{ "name": "CRED", "type": "brand", "category": "Fintech" }]
```

---

## Auth

Served by `auth-service` via the gateway, unauthenticated (these are how
you *get* a token).

| Route                 | Method | Body                          | Response                            |
|------------------------|--------|--------------------------------|--------------------------------------|
| `/auth/register`       | POST   | `{ "email", "password" }`      | `201` `{ "accessToken", "refreshToken" }` |
| `/auth/login`          | POST   | `{ "email", "password" }`      | `{ "accessToken", "refreshToken" }` |
| `/auth/refresh`        | POST   | `{ "refreshToken" }`           | `{ "accessToken", "refreshToken" }` |
| `/auth/logout`         | POST   | `{ "refreshToken" }`           | `204`                                |

`/auth/register` has no `role` field — every self-registered account is a
`viewer`; every other role (`editor`, `approver`, `admin`) only exists via
the seeded dev users or an admin's `PATCH /users/{id}/role` (see
[`ARCHITECTURE.md`](ARCHITECTURE.md#auth)). Password must be at least 8
characters. `400` on invalid email/short password, `409` `email_taken` if
the email is already registered. Registering logs the account straight in
(same response shape as `/auth/login`).

`refreshToken` is single-use: every `/auth/refresh` call invalidates the
token it was given and returns a new one, so the client must persist the
new `refreshToken` from each response. `accessToken` is a JWT, 15 minutes.
`role` (`"viewer"` | `"editor"` | `"approver"` | `"admin"`) is carried in
its claims; see [`ARCHITECTURE.md`](ARCHITECTURE.md#rbac--roles-beyond-vieweradmin)
for exactly what each role can do.

## User management

Served by `auth-service`, under `/api/v1/` (not `/auth/`) since these need
an already-verified admin session — see
[`ARCHITECTURE.md`](ARCHITECTURE.md#rbac--roles-beyond-vieweradmin) for
why that's a deliberate placement, not `/auth/users`.

### `GET /users`

Admin only. `{ "data": [{ "id": "...", "email": "...", "role": "..." }] }`
— never includes a password hash.

### `PATCH /users/{id}/role`

Admin only. Body: `{ "role": "editor" }` — must be `"viewer"`, `"editor"`,
`"approver"`, or `"admin"` (`400` otherwise). `400` if `id` is the
caller's own id (ask another admin). `404` if no such user. Returns `200`
with the updated user.

## `GET /audit-log`

Admin only, served by `audit-service`. Paginated (`page`/`limit`, same
defaults as everything else), optional `?campaignId=` filter.

```json
{
  "data": [
    { "id": 3, "action": "campaign.approval_updated", "actorId": "...", "actorEmail": "approver@campaigntracker.dev", "actorRole": "approver", "campaignId": 71, "details": "approval status set to \"approved\"", "createdAt": "2026-09-12T21:30:03+05:30" }
  ],
  "page": 1, "limit": 25, "total": 3
}
```

`action` is one of `"campaign.created"`, `"campaign.budget_updated"`,
`"campaign.approval_updated"`. Rows arrive from campaigns-service's
`campaign.audit` NATS events, not written directly by any client — see
[`ARCHITECTURE.md`](ARCHITECTURE.md#audit-log). A demo-ticker-created
campaign's row carries `actorId: "system"`, `actorRole: "system"` (no real
caller to attribute it to).

## Errors

Every non-2xx body:

```json
{ "error": { "code": "invalid_filter", "message": "unknown region: Chennai" } }
```

| Status | When                                      |
|--------|--------------------------------------------|
| 400    | bad query param / body                      |
| 401    | missing/invalid access token                |
| 403    | authenticated, but not allowed (e.g. a `viewer` POSTing a campaign, an `editor` trying to approve one) |
| 404    | campaign id not found                       |
| 409    | email already registered (`/auth/register`) |
| 429    | rate limited                                |
| 500    | unexpected — logged with a request id       |
