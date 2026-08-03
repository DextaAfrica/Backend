# Dexta Africa — Backend

A production-grade Go API serving the [Dexta Africa](../Frontend) Next.js
frontend: a lightweight CMS for editorial content, a portfolio/journal/careers
catalog, and the enquiry/newsletter integration boundary — with typed errors,
env-driven configuration, and no hardcoded values anywhere in the request path.

## Stack

- Go 1.23, [chi](https://github.com/go-chi/chi) router on the standard `net/http`
- PostgreSQL via [pgx](https://github.com/jackc/pgx), schema migrations via [golang-migrate](https://github.com/golang-migrate/migrate)
- JWT admin auth ([golang-jwt](https://github.com/golang-jwt/jwt)) + bcrypt
- Struct validation via [go-playground/validator](https://github.com/go-playground/validator)
- Structured logging via the standard library's `log/slog`

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the layering, error
boundary, and content model in depth.

## Start developing

Requires Docker. A local Go 1.23+ install is optional — every command below
also works inside `golang:1.23` via Docker (see "Running without a local Go
install" below).

```bash
cp .env.example .env   # fill in JWT_SECRET, ADMIN_BOOTSTRAP_*, etc.
make dev                # hot-reloading API (air) + Postgres, source bind-mounted
```

The API listens on `http://localhost:8080`. Log in as the bootstrap admin:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@dexta.africa","password":"<ADMIN_BOOTSTRAP_PASSWORD>"}'
```

### Two Docker setups: development vs. production

| | Dockerfile | Compose file | Behavior |
|---|---|---|---|
| **Development** | `Dockerfile.dev` | `docker-compose.dev.yml` | Full Go toolchain, [air](https://github.com/air-verse/air) hot reload, source bind-mounted, Postgres port exposed on the host |
| **Production**  | `Dockerfile`     | `docker-compose.yml`     | Multi-stage build → minimal non-root distroless runtime, no source mount, Postgres not exposed publicly |

```bash
make dev        # docker compose -f docker-compose.dev.yml up --build
make prod       # docker compose up --build (the same image used in deployment)
```

### Running without a local Go install

```bash
docker run --rm -v "$(pwd):/app" -w /app golang:1.23 go build ./...
docker run --rm -v "$(pwd):/app" -w /app golang:1.23 go test ./...
```

## Commands

```bash
make run              # go run ./cmd/api
make build            # compile to bin/api
make test             # go test ./... -race -cover
make vet              # go vet ./...
make lint             # golangci-lint (matches the CI lint job)
make dev / make dev-down     # hot-reloading local stack
make prod / make prod-down   # production-image local stack
make docker-build-dev        # build Dockerfile.dev standalone
make docker-build-prod       # build Dockerfile standalone
```

## Configuration

Every tunable — database DSN, JWT secret, CORS origins, rate limits, the
frontend revalidation webhook, the newsletter forwarding webhook — is read
from the environment by `internal/config`, validated at startup, and never
defaulted for anything sensitive. See [.env.example](.env.example) for the
full, documented list.

## Project layout

```text
cmd/api/                     entrypoint: wires config → db → repos → services → handlers → server
internal/
├── config/                  env-driven configuration, validated at boot
├── apperror/                 the single error vocabulary (Code, HTTP status mapping)
├── domain/                  models + repository interfaces (no framework imports)
├── repository/postgres/     pgx implementations of the domain repository interfaces
├── service/                 business logic: validation rules, publish transitions,
│                             revalidation triggers, webhook forwarding
├── authtoken/                admin session JWT issue/verify
├── validator/                struct-tag validation → apperror.Error
├── db/                       connection pool + migrations runner
│   └── migrations/           000001_schema.up.sql / .down.sql — the whole schema, one file each way
│                             (see the file's header comment for how to add the *next* migration)
├── logging/                  slog setup (JSON in prod, text in dev)
└── transport/http/
    ├── router.go              the full route table — public, rate-limited, admin
    ├── handlers/               one file per resource; request → service → response
    ├── request/                inbound DTOs + validation tags
    ├── response/               success/error envelope, the only place that writes JSON errors
    └── middleware/             request ID, logging, panic recovery, CORS, rate limiting, JWT auth
```

## API surface

All responses are JSON, wrapped as `{"data": ...}` on success or
`{"error": {"code", "message", "fields"}, "request_id"}` on failure.

### Public

| Method | Path                          | Purpose                              |
|--------|-------------------------------|---------------------------------------|
| GET    | `/healthz` / `/readyz`        | Liveness / readiness probes           |
| POST   | `/auth/login`                 | Admin login → JWT                     |
| GET    | `/content/{key}`               | Published page content (`home`, `about`, `lifestyle`, `contact`, `careers`, `privacy`, `terms`, `cookies`, `accessibility`) |
| GET    | `/portfolio`, `/portfolio/{slug}` | Published developments (list/detail) |
| GET    | `/journal`, `/journal/{slug}`  | Published articles (list/detail)      |
| GET    | `/careers/listings`, `/careers/listings/{slug}` | Published job listings |
| POST   | `/enquiries`                   | Submit the contact form (rate-limited)|
| POST   | `/newsletter/subscribe`        | Subscribe (rate-limited)              |
| POST   | `/newsletter/unsubscribe`      | Unsubscribe (rate-limited)            |

### Admin (`Authorization: Bearer <token>` from `/auth/login`)

| Method | Path                                    | Purpose                        |
|--------|------------------------------------------|---------------------------------|
| GET/PUT| `/admin/content/{key}`                   | Read/write a page's content     |
| GET    | `/admin/content`                         | List every page                 |
| CRUD   | `/admin/portfolio[/{id}]`                | Manage developments             |
| CRUD   | `/admin/journal[/{id}]`                  | Manage articles                 |
| CRUD   | `/admin/careers/listings[/{id}]`         | Manage job listings             |
| GET    | `/admin/enquiries[/{id}]`                | Review submissions              |
| PATCH  | `/admin/enquiries/{id}/status`           | Mark new/read/archived          |
| GET    | `/admin/newsletter/subscribers`          | List subscribers                |

Saving content, portfolio, journal, or career entries fires a best-effort
`POST` to `FRONTEND_REVALIDATE_URL` so the Next.js frontend's cache updates
immediately — see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#revalidation-and-outbound-webhooks).

## CI/CD

| Workflow | Trigger | Does |
|---|---|---|
| `.github/workflows/ci.yml` | every push/PR to `main`/`develop` | `go vet` + `golangci-lint`; `go build` + unit tests (race + coverage); a real-Postgres migration-reversibility check (`internal/db/migrate_integration_test.go`, tag `integration`); a `Dockerfile` build smoke test |
| `.github/workflows/docker-publish.yml` | push to `main`, tags `v*.*.*` | builds `Dockerfile` and pushes to `ghcr.io/<org>/backend`, tagged by branch, semver, and commit SHA |
| `.github/workflows/codeql.yml` | push/PR + weekly schedule | CodeQL static analysis for Go |
| `.github/dependabot.yml` | weekly | dependency update PRs for Go modules, Docker base images, and Actions versions |

## Wiring the frontend

Point the frontend's `CONTENT_API_URL` at this API's base URL, and set its
`CONTENT_REVALIDATION_SECRET` to the same value as this service's
`FRONTEND_REVALIDATE_SECRET`. The frontend fetches published content from
`/content/{key}`, `/portfolio`, `/journal`, and `/careers/listings`, and this
API calls back into the frontend's `POST /api/revalidate` whenever an admin
publishes a change.
