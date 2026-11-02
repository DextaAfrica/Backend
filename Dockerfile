# Production image: multi-stage build producing a minimal, non-root,
# distroless runtime. No shell, no package manager, no build tooling in the
# final image — smallest possible attack surface for what actually runs in
# production. For local development with hot reload, use Dockerfile.dev.

# --- Build stage ---
FROM golang:1.23-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app

COPY --from=build /out/api ./api
COPY --from=build /src/internal/db/migrations ./internal/db/migrations

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/api"]
