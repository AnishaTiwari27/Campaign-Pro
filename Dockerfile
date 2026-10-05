# The production image: one container serving both the API and the app.
#
# Same origin is not a convenience here. The session cookie is SameSite=Lax
# and the fetch client sends no cross-origin credentials, so hosting the
# frontend on a different origin would break sign-in. Serving both from one
# binary is what the rest of the code already assumes.
#
# Build from the repository root, because the build needs both halves:
#   docker build -t campaign-tracker-pro .

# ---- 1. Build the frontend -------------------------------------------------
FROM node:22-alpine AS web
WORKDIR /web
# Dependencies first, so a source-only change does not reinstall them.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# `npm run build` type-checks (tsc -b) before bundling, so a type error fails
# the image rather than shipping.
RUN npm run build

# ---- 2. Build the server, with the frontend embedded -----------------------
FROM golang:1.25-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# backend/web embeds ./dist, so the frontend has to be in the Go tree before
# the compile. Only .gitkeep is committed there; this is what fills it.
COPY --from=web /web/dist/ ./web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
# Shipped alongside the server so an operator can create the first real admin
# without `make seed`, which begins with TRUNCATE.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/createuser ./cmd/createuser

# ---- 3. Runtime ------------------------------------------------------------
FROM alpine:3.21
# tzdata because every report cadence is computed in Asia/Kolkata, and
# ca-certificates because a managed Postgres is reached over TLS.
RUN apk add --no-cache tzdata ca-certificates && adduser -D -u 10001 app
COPY --from=api /out/server /usr/local/bin/server
COPY --from=api /out/createuser /usr/local/bin/createuser
USER app
ENV PORT=8090 TZ=Asia/Kolkata
EXPOSE 8090
# No shell form: the server handles SIGTERM itself and shuts down gracefully,
# which only works if it is PID 1 and receives the signal directly.
ENTRYPOINT ["/usr/local/bin/server"]
