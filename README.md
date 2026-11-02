# Dexta Africa — Enterprise Real Estate Platform Backend

High-performance, zero-data-loss backend engine engineered for **Dexta Africa**. This platform powers both internal continental operations (Admin Portal, back-office workflows, inventory, financials, and CRM) and the customer-facing dynamic web portal.

---

## 🏛️ Architecture Highlights

- **Clean / Hexagonal Architecture (DDD)**: Domain models isolated from external frameworks.
- **Zero-Data-Loss Guarantee**: Transactional Outbox pattern ensures event delivery (emails, notifications, calendar bookings, invoices) even during network drops.
- **Concurrency & Double-Booking Protection**: Row-level pessimistic locking (`SELECT FOR UPDATE`) prevents simultaneous allocation collisions on property units.
- **Sliding-Window Rate Limiting**: Redis-backed distributed rate limiting with automated client throttling and fallback.
- **Unified JSON Response Envelope**: Standardized across all endpoints (`success`, `status_code`, `message`, `data`, `meta`, `error`).
- **OpenAPI 3.1 Spec**: Ready for contract-driven type generation across the Web and Admin repos (`docs/openapi.yaml`).

---

## 📁 Project Structure

```
Backend/
├── cmd/
│   ├── api/                    # HTTP Server & Router Entrypoint
│   └── seeder/                 # Continental Data Seeder (Real portfolio data)
├── internal/
│   ├── application/            # Outbox Worker & Use-case orchestrators
│   ├── config/                 # Environment & Config loader
│   ├── domain/                 # Core Domain Entities (Property, CRM, Identity, Finance, Outbox)
│   ├── infrastructure/         # Postgres (GORM/pgx), Redis, Logger (Zerolog)
│   └── interfaces/             # HTTP Handlers, Middlewares (Auth, CORS, Rate Limit, Recover), Responses
├── docs/                       # OpenAPI 3.1 Specification
├── Dockerfile                  # Multi-stage optimized Alpine container
├── docker-compose.yml          # PostgreSQL 16, Redis 7.2, Mailpit, Dexta API
└── Makefile                    # Standardized development commands
```

---

## 🚀 Quick Start (Local Development)

### 1. Boot Services with Docker Compose
```bash
docker compose up -d
```
This starts:
- **PostgreSQL 16** on `localhost:5432`
- **Redis 7.2** on `localhost:6379`
- **Mailpit** (Local Email Sandbox) on `http://localhost:8025` (SMTP: `1025`)

### 2. Run Database Seeder
Seed realistic continental portfolio developments (Lagos & Nairobi), units, construction milestones, and initial staff credentials:
```bash
make seeder
```

### 3. Run the Backend Server
```bash
make run
```
Server runs on `http://localhost:8080`.

---

## 🧪 Testing & Verification
```bash
make test
```
All unit and integration tests execute with race detection enabled.

---

## 🔗 Frontend Integration (Web & Admin Repos)
The API contract is defined in `docs/openapi.yaml`. In your Web and Admin frontends, generate TypeScript client SDKs directly:
```bash
npx @hey-api/openapi-ts -i ../Backend/docs/openapi.yaml -o src/api/generated
```
