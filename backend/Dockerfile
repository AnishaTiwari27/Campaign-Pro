# Unverified alongside docker-compose.yml — no Docker on the build machine.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/server"]
