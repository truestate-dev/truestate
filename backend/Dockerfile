# ── Build stage ───────────────────────────────────────────────────────────────
FROM harbor.apps.vjinx.de/dockerhub/library/golang:1.25-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/truestate-api ./backend/cmd/api

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM harbor.apps.vjinx.de/dockerhub/library/debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /out/truestate-api /usr/local/bin/truestate-api

EXPOSE 8080

ENV LISTEN_ADDR=:8080
ENV LOG_LEVEL=info

ENTRYPOINT ["/usr/local/bin/truestate-api"]
