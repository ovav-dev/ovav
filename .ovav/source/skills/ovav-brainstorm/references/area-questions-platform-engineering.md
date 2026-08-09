# Platform Engineering — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when backend/Go/runtime/security request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## Platform Engineering Questions

### PEQ1: Module Strategy
"¿Go modules separados o single module?"
- Multi-module (repo con `modules/` conteniendo N módulos go-publicados)
- Single module (todo en un `go.mod`, simplicidad)
- Micro-modules (1 repo por paquete,激进 pero máximo reuso)

Why: Multi-module = separate versionado + `go get` independiente. Single = simplicidad de CI + refactors cross-cutting.

### PEQ2: Database Strategy
"¿Motor de base de datos y driver?"
- PostgreSQL 16+ + `lib/pq` o `pgx` (ACID, JSONB, full-text search)
- SQLite + `modernc.org/sqlite` (MVP, zero-config, embedded)
- MySQL 8+ + `go-sql-driver/mysql` (si MySQL es constraint)
- MongoDB + `mongo-driver` (si schema flexible es requisito)

Why: PostgreSQL + pgx = standard profesional. SQLite = velocidad de setup para MVP sin red.

### PEQ3: API Style
"¿Estilo de API del backend?"
- REST (HTTP + JSON, simple, cacheable, estándar)
- gRPC + Protobuf (streaming, контракт primeiro, tooling pesado)
- GraphQL (queries complejas, over-fetching resuelto)
- tRPC (end-to-end type safety, mismo repo frontend/backend)

Why: REST = universally understood. gRPC = performance crítico. tRPC = DX premium en TypeScript monorepos.

### PEQ4: Authentication Model
"¿Cómo autentican?"
- JWT RS256 (firma asimétrica, scalable)
- PASETO (nuevo, simplicidad, sin RFC 7519 legacy)
- Session cookies + server-side sessions (stateful, rotateable)
- OAuth2/OIDC (Google/Apple login)
- API Keys (para servicios machines-to-machine)

Why: JWT RS256 = standard. PASETO = simpler. Sessions = revoke instant. OAuth = login social.

### PEQ5: Observability Stack
"¿Logging, metrics, tracing?"
- Estructurado: `slog` (stdlib Go 1.21+) → JSON → stdout → Prometheus/Grafana
- Metrics: Prometheus client_golang
- Tracing: OpenTelemetry (Jaeger/Zipkin compatible)
- Sin observabilidad (logs print, no metrics)

Why: `slog` + Prometheus es el stack moderno en Go. OpenTelemetry es para debugging distribuido.

## Deliverable
After all questions answered → Thavren adds `## 6. Backend Architecture` and `## 7. Data Model` sections to DESIGN.md.
