# Production image: multi-stage build producing a hardened, minimal,
# non-root Alpine runtime. Fully compatible with container health checks (wget),
# automated migrations, and enterprise cloud ingress.

# --- Build stage ---
FROM golang:1.27-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime stage ---
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S -g 10001 appgroup && \
    adduser -S -u 10001 -G appgroup -h /app -s /sbin/nologin appuser

WORKDIR /app

COPY --from=build --chown=appuser:appgroup /out/api ./api
COPY --from=build --chown=appuser:appgroup /src/internal/db/migrations ./internal/db/migrations

USER appuser:appgroup
EXPOSE 8080

ENTRYPOINT ["/app/api"]
