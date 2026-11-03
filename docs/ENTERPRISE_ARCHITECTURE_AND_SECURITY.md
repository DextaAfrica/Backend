# Dexta Africa — Enterprise Backend Architecture, Security & Git Flow Specification

This document defines the high-end architectural standards, threat modeling, zero-trust security postures, container hardening, and Git Flow protocols implemented across the **Dexta Africa Enterprise Ecosystem** (`Go Core Backend` and `Next.js Operations Admin Portal`).

---

## 1. Threat Modeling & Penetration Defense Architecture

Engineered to withstand adversarial probing, penetration testing, and enterprise compliance audits modeled after top financial institutions and real estate asset management platforms.

```
                      INTERNET / ATTACK SURFACE
                                  │
                                  ▼
      ┌────────────────────────────────────────────────────────┐
      │               EDGE DEFENSE & REVERSE PROXY             │
      │  • SSL/TLS Termination (Strict HSTS 2-Year Preload)     │
      │  • Automated IP Sanitization & Header Spoof Guard      │
      └───────────────────────────┬────────────────────────────┘
                                  │
                                  ▼
      ┌────────────────────────────────────────────────────────┐
      │                 GO CORE BACKEND API GATE               │
      │  • Defensive Security Headers (OWASP ASVS 4.0)          │
      │  • Strict Brute-Force Rate Limiting (0.2 RPS, burst 5) │
      │  • Public Write Throttling (2 RPS, burst 10)           │
      │  • Payload Guard (1MB MaxBytes, DisallowUnknownFields) │
      │  • Panic Recovery Boundary & Structured Slog Engine    │
      └─────────────┬────────────────────────────┬─────────────┘
                    │                            │
                    ▼                            ▼
      ┌───────────────────────────┐┌───────────────────────────┐
      │   POSTGRESQL 16 CLUSTER   ││       REDIS 7 CACHE       │
      │  • 100% Parameterized SQL ││  • Distributed Locks     │
      │  • Transactional Outbox   ││  • Rate Limit State      │
      │  • Isolated Internal Net  ││  • MaxMemory LRU Evict   │
      └───────────────────────────┘└───────────────────────────┘
```

### 1.1 OWASP Top 10 Defenses Implemented

| Threat / Attack Vector | Mitigation in Dexta Africa Architecture | Verification Layer |
| :--- | :--- | :--- |
| **SQL Injection (SQLi)** | 100% parameterized queries using native `pgx/v5` prepared statements (`$1, $2, ...`). Zero raw string concatenation anywhere in repositories. | `internal/repository/postgres/*.go` |
| **Broken Authentication** | Bcrypt password hashing (`bcrypt.DefaultCost = 10`), single-use backup recovery codes, RFC 6238 TOTP authenticators, constant-time comparisons, and strict brute-force rate limiters on `/auth/login`. | `internal/service/auth_service.go`, `internal/transport/http/router.go` |
| **Sensitive Data Exposure** | Zero stack traces leaked to clients; generic authentication errors that never leak email registration existence; JWT secret strictly validated (≥ 32 chars). | `internal/authtoken/token.go`, `internal/transport/http/response/response.go` |
| **Denial of Service (DoS / Slowloris)** | `http.MaxBytesReader` capped at 1MB per request; `ReadHeaderTimeout` set to 10s; `IdleTimeout` set to 60s; per-IP token bucket rate limiting with automatic idle eviction. | `cmd/api/main.go`, `internal/transport/http/request/decode.go` |
| **Clickjacking & UI Redressing** | `X-Frame-Options: DENY` applied universally via `SecurityHeaders` middleware. | `internal/transport/http/middleware/security_headers.go` |
| **MIME Sniffing Vulnerabilities** | `X-Content-Type-Options: nosniff` enforced on every response. | `internal/transport/http/middleware/security_headers.go` |
| **Client IP Spoofing** | Rate limiter extracts the leftmost validated IP from `X-Forwarded-For` using `net.ParseIP()`, falling back to `RemoteAddr`. | `internal/transport/http/middleware/rate_limit.go` |
| **Data in Transit Security** | `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload` (2-year HSTS). | `internal/transport/http/middleware/security_headers.go` |

---

## 2. Resiliency & Error Boundary Architecture

Top financial backends cannot afford unhandled crashes or stack leakage. The ecosystem uses a three-tier error isolation model:

```
                    Level 1: Next.js Client Route Boundary
                         (src/app/error.tsx)
          [Gracefully traps runtime UI faults, isolates session]
                                  │
                                  ▼
                   Level 2: Next.js Global Root Catch
                      (src/app/global-error.tsx)
         [Quarantines root layout crashes with incident digest]
                                  │
                                  ▼
                     Level 3: Go Kernel Panic Boundary
             (internal/transport/http/middleware/recover.go)
          [Recovers any panics, outputs structured JSON 500,
           prevents container crashes, correlates via X-Request-ID]
```

1. **Client Route Isolation**: If any operational module fails, `src/app/(admin)/error.tsx` keeps the Sidebar and Header intact, allowing the admin to navigate away without session loss.
2. **Panic Recovery**: Any panic inside the Go backend is caught by `middleware.Recover`, logged with full debug stack trace to server logs along with the unique `X-Request-ID`, and masked to the client as an opaque `500 Internal Server Error`.

---

## 3. Production Dockerization & Container Hardening

### 3.1 Principles Applied
- **Minimal Attack Surface (Distroless)**: Production images use `gcr.io/distroless/static-debian12:nonroot` for Go and `node:20-alpine` non-root for Next.js. No bash, no package managers (`apt`, `apk`), no build compilers in production containers.
- **Unprivileged Execution**:
  - Go Backend runs as `nonroot:nonroot` (UID 65532).
  - Admin Portal runs as `nextjs:nodejs` (UID 1001).
- **Network Segmentation**:
  - `dexta_internal`: Dedicated isolated network without public gateway access for Postgres and Redis.
  - `dexta_public`: Gateway network for ingress traffic to the Go API and Next.js portal.
- **Resource Constraints**: Capped CPU and Memory allocations on all services to prevent memory exhaustion from bringing down the host machine.
- **Security Options**: `no-new-privileges:true` prevents container processes from gaining elevated permissions.

---

## 4. Enterprise Git Flow Specification

All development follows a disciplined Git Flow strategy ensuring zero broken builds in production branches.

```
  main ──────●──────────────────────────────●────────────── (Production Releases)
             ▲                              ▲
             │ (Release Merge)              │ (Hotfix Merge)
  staging ───●──────────────────────────────┼────────────── (Pre-Production / UAT)
             ▲                              │
             │ (Staging PR)                 │
  develop ───●──────────●──────────●────────┼────────────── (Active Integration)
             ▲          ▲          ▲        │
             │          │          │        ▼
  feature/───┘          │          └─ hotfix/security-patch
  portfolio-v2          └── feature/crm-pipeline
```

### 4.1 Branch Taxonomy

| Branch | Purpose | Protection Rules |
| :--- | :--- | :--- |
| **`main`** | Production-ready stable releases. Deployed to production environments. | Protected. Direct pushes disabled. Requires 2 peer approvals, passing CI (tests, SAST, linting), and linear commit history. |
| **`staging`** | Pre-production UAT environment for client demonstrations and regression audits. | Protected. Automated deployment to staging cluster on merge. |
| **`develop`** | Primary integration branch for completed features. | Protected. Requires passing CI checks before merge. |
| **`feature/*`** | Feature-specific development branches branched from `develop` (`feature/crm-pipeline`). | Local branching. Merged via Pull Request into `develop`. |
| **`hotfix/*`** | Critical security patches branched directly from `main` (`hotfix/token-rotation`). | Merged back into both `main` and `develop`. |

### 4.2 Pull Request Checklist
Before any PR is approved:
1. `golangci-lint run ./...` passes with zero issues.
2. `go vet ./...` and `go test ./...` pass with zero failures.
3. Next.js `npm run typecheck` and `npm run build` pass with zero errors.
4. No sensitive secrets, credentials, or mock OTP codes in code.
5. All new database migrations are accompanied by matching down migrations (`00000X_schema.down.sql`).

