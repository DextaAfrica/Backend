# Architecture

## Layering

```
transport/http (handlers, DTOs, middleware, routing)
        │  depends on
        ▼
   service (business rules, orchestration)
        │  depends on
        ▼
   domain (models + repository interfaces — no framework/DB imports)
        ▲  implemented by
        │
repository/postgres (pgx queries)
```

`domain` defines every repository as an interface (`domain.DevelopmentRepository`,
`domain.ArticleRepository`, etc.) and never imports pgx, chi, or net/http.
Services depend on those interfaces, not on `repository/postgres` directly —
swapping Postgres for something else later means writing a new adapter, not
touching business logic. `main.go` is the only place that wires a concrete
`repository/postgres` type into a service.

## Error boundary

Every error in the system is normalized to `*apperror.Error` (`internal/apperror`)
before it reaches a client:

- Repositories return plain sentinel errors (`domain.ErrNotFound`, `domain.ErrConflict`)
  or wrapped driver errors — they know nothing about HTTP.
- Services translate those into a `*apperror.Error` with a stable `Code`
  (`NOT_FOUND`, `VALIDATION_ERROR`, `CONFLICT`, ...) and a client-safe message.
- `response.Error` (`internal/transport/http/response`) is the single place
  that turns a `*apperror.Error` into an HTTP status + JSON body. Handlers
  never write their own error JSON.
- `middleware.Recover` is the outermost boundary: any panic anywhere in the
  stack is caught, logged with a stack trace, and turned into a generic 500
  instead of crashing the process or leaking internals.

This means adding a new error case is one line in `apperror` plus a `case`
in `HTTPStatus()` — no handler needs to change.

## Configuration

Nothing is hardcoded. `internal/config` parses and validates every runtime
value (`DATABASE_URL`, `JWT_SECRET`, `CORS_ALLOWED_ORIGINS`, webhook URLs,
timeouts, rate limits) from the environment via struct tags, and fails fast
at boot if a required value is missing or invalid — never deep inside a
request handler. See `.env.example` for the full list.

## Admin auth

A single-role admin system: one or more rows in `admins`, bcrypt-hashed
passwords, JWT session tokens (`internal/authtoken`) verified by
`middleware.RequireAdmin` on every `/admin/*` route. There is no public
registration endpoint — the first admin is seeded from
`ADMIN_BOOTSTRAP_EMAIL`/`ADMIN_BOOTSTRAP_PASSWORD` on boot, only while the
`admins` table is empty. This is deliberately minimal: no roles, no
multi-tenant scoping, no password reset flow. If the product needs
multiple admins with different permission levels later, extend
`domain.Admin` and `authtoken.Claims` — the boundary is already in one
place.

## Content model

- **`pages`** — flexible JSONB documents for singleton editorial pages
  (home, about, lifestyle, careers landing, contact, and the static legal
  pages). Content shape is owned by the frontend contract for that key, not
  the backend; the API stores and serves it opaquely. Keyed by a stable
  string (`home`, `about`, ...), fetched publicly at `GET /content/{key}`.
- **`portfolio_developments`**, **`journal_articles`**, **`career_listings`**
  — structured, slug-addressed collections with their own columns because
  the frontend needs to list, filter, and paginate them, not just render an
  opaque blob.
- **`enquiries`**, **`newsletter_subscribers`** — form submissions. No
  content editing; only admin read/status-update.

## Revalidation and outbound webhooks

`internal/service/revalidate.go` and `internal/service/newsletter_forwarder.go`
are both best-effort, fire-and-forget HTTP clients:

- **Revalidator** notifies the Next.js frontend (`POST {FRONTEND_REVALIDATE_URL}`,
  `Authorization: Bearer <FRONTEND_REVALIDATE_SECRET>`) whenever an admin
  saves content, so the frontend's cache invalidates immediately instead of
  waiting out its TTL.
- **NewsletterForwarder** posts new subscriptions to an external email/CRM
  provider (`NEWSLETTER_WEBHOOK_URL`), if configured.

Both run in a detached goroutine and only log on failure — a flaky webhook
must never fail or roll back a write that already succeeded in Postgres.

## Why not sqlc / an ORM

Repository methods are hand-written pgx queries behind a narrow interface
per aggregate. For this project's size, that's less machinery than wiring
sqlc codegen or an ORM, and the interfaces already isolate persistence
choices from business logic — reintroducing sqlc later only touches
`repository/postgres`.
