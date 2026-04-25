# syntax=docker/dockerfile:1.7

FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server


FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S -G app app

COPY --from=build /out/server /app/server
COPY --from=build /src/migrations /app/migrations

RUN mkdir -p /app/storage && chown -R app:app /app

USER app

ENV APP_PORT=8080 \
    APP_ENV=production \
    STORAGE_DIR=/app/storage \
    MIGRATIONS_DIR=/app/migrations \
    AUTO_MIGRATE=true

EXPOSE 8080
VOLUME ["/app/storage"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
