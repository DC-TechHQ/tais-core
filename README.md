# tais-core

Shared Go library for the **TAIS** (Traffic Authority Information System — РБДА Tajikistan) platform.

> **Module:** `github.com/DC-TechHQ/tais-core`
> **Go version:** 1.26

This library is the **single source of infrastructure truth** for all 28 TAIS microservices.
Every service imports it; nothing is copy-pasted between services.

---

## Packages

### Build order (dependency graph)

```
i18n  ──►  errors  ──►  logger  ──►  config
                                        │
                    ┌───────────────────┘
                    ▼
              db · redis · nats · jwt
                    │
                    ▼
             context · pagination
                    │
                    ▼
        response · middleware · event
```

---

### `i18n` — Translation registry

Global TJ / RU / EN registry. Services register their own codes in `internal/i18n/` via `init()`.

```go
import pkgi18n "github.com/DC-TechHQ/tais-core/i18n"

// Register translations (called once from init())
pkgi18n.Register(map[string]map[string]string{
    "ErrVehicleNotFound": {
        pkgi18n.LangTJ: "Нақлиёт ёфт нашуд",
        pkgi18n.LangRU: "Транспортное средство не найдено",
        pkgi18n.LangEN: "Vehicle not found",
    },
})

// Retrieve translation
msg := pkgi18n.Get("ErrVehicleNotFound", pkgi18n.LangRU)
// → "Транспортное средство не найдено"
```

---

### `errors` — AppError type

HTTP-aware error type. Always carries an i18n code + HTTP status. Never wrap across layers.

Domain errors are **pure string constants** — no HTTP status in the domain layer.
HTTP status is registered at the delivery layer via `RegisterStatus`.

```go
import pkgerr "github.com/DC-TechHQ/tais-core/errors"

// internal/domain/errors.go — string constants ONLY, zero imports
const ErrVehicleNotFound = "ErrVehicleNotFound"

// internal/i18n/vehicle.go — HTTP status + translation registered in init()
pkgerr.RegisterStatus(domain.ErrVehicleNotFound, http.StatusNotFound)
pkgi18n.Register(map[string]map[string]string{
    domain.ErrVehicleNotFound: {
        pkgi18n.LangTJ: "Нақлиёт ёфт нашуд",
        pkgi18n.LangRU: "Транспортное средство не найдено",
        pkgi18n.LangEN: "Vehicle not found",
    },
})

// Common errors (tais-core built-ins, HTTP status already embedded):
pkgerr.ErrInternal          // 500
pkgerr.ErrInvalidData       // 400
pkgerr.ErrNotFound          // 404
pkgerr.ErrAlreadyExists     // 409
pkgerr.ErrForeignKey        // 400
pkgerr.ErrUnauthorized      // 401
pkgerr.ErrForbidden         // 403
pkgerr.ErrInvalidToken      // 401
pkgerr.ErrTokenExpired      // 401
pkgerr.ErrUserBlocked       // 403
pkgerr.ErrInvalidCredentials // 401
pkgerr.ErrDeadlock          // 409
```

---

### `logger` — Structured logger

zap + lumberjack. Per-level files: `info.log`, `warn.log`, `error.log`, `debug.log`, `gorm.log`.
All levels tee to stdout (Docker log drivers / Grafana Loki pick it up).
`gorm.log` is **file-only** (too verbose for stdout) and captures every SQL query — slow queries (`>200ms`) are marked with `slow_query:true`.

```go
import pkglog "github.com/DC-TechHQ/tais-core/logger"

log, err := pkglog.New(pkglog.Config{
    Directory:  "./logs",
    Level:      "info",        // debug | info | warn | error
    Format:     "json",        // json | console
    MaxSizeMB:  100,
    MaxBackups: 10,
    MaxAgeDays: 30,
    Compress:   true,
})

log.Info("vehicle created", "id", 42, "vin", "WVWZZZ1KZ")
log.Error("db error", "error", err)

// Child logger with persistent fields
svcLog := log.With("service", "tais-vehicle", "request_id", reqID)
```

---

### `config` — Docker secrets

```go
import pkgcfg "github.com/DC-TechHQ/tais-core/config"

// Reads /run/secrets/tais_{name}, falls back to TAIS_{NAME} env var
secret := pkgcfg.ReadSecret("vehicle_db_password")

// Panics if empty (use for required production secrets)
secret := pkgcfg.MustReadSecret("jwt_secret")
```

---

### `db` — GORM factory + query builder

```go
import pkgdb "github.com/DC-TechHQ/tais-core/db"

// Open connection
gdb, err := pkgdb.New(pkgdb.Config{
    DSN:             cfg.Postgres.DSN(),
    MaxOpenConns:    25,
    MaxIdleConns:    10,
    ConnMaxLifetime: 300,
}, log)

// Translate errors — call in every repository method AFTER logging the raw error.
// Pure translation — no logging. Repositories log context (operation, entity ID)
// themselves before calling TranslateError.
err = pkgdb.TranslateError(err)
// nil → nil, ErrRecordNotFound → *AppError(404), pg23505 → *AppError(409), etc.

// Fluent query builder — all column names are validated against
// ^[a-zA-Z_][a-zA-Z0-9_.]*$ to prevent SQL injection via column interpolation.
q := pkgdb.NewBuilder(gdb.WithContext(ctx).Model(&models.Vehicle{})).
    Where("status = ?", filter.Status).          // skipped if Status == ""
    Search(filter.Search, "vin", "plate_number"). // ILIKE on multiple columns
    DateRange("created_at", filter.From, filter.To)

var total int64
q.Build().Count(&total)

var ms []models.Vehicle
q.Pagination(filter.Params).OrderBy("created_at", "desc").Build().Find(&ms)
```

---

### `redis` — go-redis client

```go
import pkgredis "github.com/DC-TechHQ/tais-core/redis"

rdb, err := pkgredis.New(pkgredis.Config{
    Addr:     cfg.Redis.Addr,
    Password: cfg.Redis.Password,
    DB:       cfg.Redis.DB,
}, log)
```

Key namespace convention: `tais:{service}:{entity}:{id}`

---

### `nats` — JetStream

tais-core only parses/publishes. tais-auth owns stream configuration.

```go
import pkgnats "github.com/DC-TechHQ/tais-core/nats"

// Connect
nc, js, err := pkgnats.Connect(pkgnats.Config{URL: cfg.NATS.URL}, log)
// Shutdown: call nc.Drain() explicitly in main.go — do NOT defer.
// Order: HTTP stop → outbox flush → nc.Drain()

// Publish — marshals payload to JSON, uses JetStream acknowledged delivery.
// For critical events (DB write + publish atomic), use the Transactional Outbox
// pattern instead (see SERVICE-ARCHITECTURE.md ⑮). Direct publish is acceptable
// for informational events where loss is tolerable.
err = pkgnats.Publish(js, "tais.vehicle.vehicle.created", evt, log)

// Subscribe — durable consumer with panic recovery and at-least-once delivery.
// Consumer name convention: "{subscribing-service}.{subject-with-dots-as-hyphens}"
pkgnats.Subscribe(js, "tais.>", "tais-audit.tais-all", func(msg *nats.Msg) {
    // Handlers MUST be idempotent (JetStream delivers at-least-once).
    // Check processed_events table by BaseEvent.ID before processing.
    // msg.Ack()  — success or duplicate
    // msg.Nak()  — transient error, redeliver
    // msg.Term() — poison message, never redeliver
    _ = msg.Ack()
}, log)
```

---

### `jwt` — Token parsing

tais-core **only parses** JWT tokens. Signing is done exclusively by `tais-auth`.

```go
import pkgjwt "github.com/DC-TechHQ/tais-core/jwt"

cfg := pkgjwt.Config{Secret: cfg.JWT.Secret}

// Parse + validate HS256 signature and expiry
claims, err := pkgjwt.Parse(tokenStr, cfg)

// claims.Sub    — user ID
// claims.Type   — "staff" | "citizen"
// claims.IpNet  — "10.200.1" (staff only)
// claims.JTI    — unique token ID

// Check IP /24 subnet binding (staff only)
if !pkgjwt.CheckIPNet(claims, clientIP) {
    // reject
}
```

---

### `context` — User context

```go
import pkgctx "github.com/DC-TechHQ/tais-core/context"

// In handler — retrieve authenticated user
u, ok := pkgctx.GetUser(c)       // returns (nil, false) if unauthenticated
u := pkgctx.MustGetUser(c)       // panics if not set — use only after Required middleware

// Permission check (super_admin always passes, "*" wildcard for admin)
if pkgctx.HasPermission(u, "vehicle:read") { ... }
```

`UserCtx` fields:

| Field           | Type      | Description                           |
|-----------------|-----------|---------------------------------------|
| `ID`            | `uint`    | User ID                               |
| `Type`          | `string`  | `"staff"` \| `"citizen"`             |
| `IsSuperAdmin`  | `bool`    | Bypasses all permission checks        |
| `IsActive`      | `bool`    | False → 403 ErrUserBlocked            |
| `Roles`         | `[]string`| Named roles (e.g. `"inspector"`)      |
| `Permissions`   | `[]string`| Permission codes + `"*"` wildcard     |
| `DeptID`        | `*uint`   | Department scope (optional)           |
| `RegionID`      | `*uint`   | Region scope (optional)               |
| `DLAuthorityID` | `*uint`   | DL authority scope (optional)         |
| `IpNet`         | `string`  | JWT /24 subnet (`"10.200.1"`)         |
| `JTI`           | `string`  | JWT ID (blacklist reference)          |

---

### `pagination` — Query pagination

```go
import pkgpage "github.com/DC-TechHQ/tais-core/pagination"

// In handler — parse ?page=&limit= from query string
params := pkgpage.Parse(c)
// params.Page, params.Limit, params.Offset
// Defaults: page=1, limit=20. Max limit: 100.
```

---

### `response` — HTTP envelopes

All responses follow the same JSON structure. All error messages contain TJ + RU + EN.

```go
import pkgresp "github.com/DC-TechHQ/tais-core/response"

pkgresp.OK(c, "user", dto)                               // 200 {"success":true,"user":{...}}
pkgresp.Created(c, "vehicle", dto)                       // 201 {"success":true,"vehicle":{...}}
pkgresp.NoContent(c)                                     // 204

pkgresp.Paginated(c, "users", items, total, page, limit) // 200
// {"success":true,"users":[...],"meta":{"total":500,"page":2,"limit":20,"total_pages":25}}

pkgresp.Error(c, err)                            // auto-mapped status
// {"success":false,"error":{"code":"ErrNotFound","message":{"tj":"...","ru":"...","en":"..."}}}

pkgresp.ErrorWithData(c, err, validationErrs)    // same + "data" field in error
```

---

### `middleware` — HTTP middleware

```go
import pkgmw "github.com/DC-TechHQ/tais-core/middleware"

// In router.go — wire once
auth := pkgmw.Required(ctn.Redis, ctn.Config.JWT, ctn.Resolver)

v1 := r.Group("/api/v1")

// Staff routes
vehicles := v1.Group("/vehicles")
vehicles.Use(auth, pkgmw.StaffOnly())
vehicles.GET("/:id",  pkgmw.Can("vehicle:read"),     handler.Get)
vehicles.POST("",     pkgmw.Can("vehicle:register"), handler.Create)

// Citizen routes
portal := v1.Group("/portal")
portal.Use(auth, pkgmw.CitizenOnly())
portal.GET("/my-vehicles", handler.GetMine)

// Internal service-to-service routes (NOT exposed via Traefik)
internal := r.Group("/internal")
internal.Use(pkgmw.InternalOnly(ctn.Config.InternalToken))
internal.GET("/vehicles/:id", handler.GetInternal)

// Global middleware (register before routes)
r.Use(pkgmw.Recovery(log))
r.Use(pkgmw.RequestLogger(log))
r.Use(pkgmw.CORS(cfg.HTTP.CORSOrigins))
```

**`Required` middleware flow:**
1. Extract `Authorization: Bearer {token}`
2. Parse + validate JWT (HS256 signature + expiry)
3. Check `ip_net` claim vs client /24 subnet (staff only)
4. Check Redis blacklist: `tais:blacklist:{jti}` → 401 if found
5. Load `user_ctx:{sub}` from Redis (cache miss → `resolver.Resolve` → cache SET EX 300)
6. Check `is_active` → 403 ErrUserBlocked if false
7. `c.Set(pkgctx.KeyUser, userCtx)`

**`UserContextResolver`** — implemented per-service in `infra/resolver/identity.go`:

```go
// internal/infra/resolver/identity.go
type Identity struct {
    baseURL     string       // e.g. "http://identity:8002"
    token       string       // X-Internal-Token value
    serviceName string       // e.g. "tais-vehicle" — from cfg.App.Name, not hardcoded
    client      *http.Client // explicit timeout, never http.DefaultClient
}

func NewIdentity(baseURL, token, serviceName string) *Identity { ... }

func (r *Identity) Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error) {
    // GET {r.baseURL}/internal/users/{userID}/context
    // Headers: X-Internal-Token, X-Service-Name
    // → unmarshal into *pkgctx.UserCtx
}
```

---

### `event` — Domain events

```go
import pkgevent "github.com/DC-TechHQ/tais-core/event"

// Build NATS subject — always use Subject(), never hardcode strings.
subj := pkgevent.Subject("registration", "vehicle", "registered")
// → "tais.registration.vehicle.registered"

// Create event envelope.
// actorID is nil for system-generated events (migrations, scheduled jobs).
actorID := u.ID
evt := pkgevent.New(subj, "tais-registration", &actorID, payload)
// evt.ID → UUID v4 — used by consumers for deduplication (processed_events table)
```

**Publish modes — choose based on criticality:**

```go
// Mode A: Informational event — direct publish via broker interface (infra/broker/).
// Acceptable when losing the event is tolerable (e.g. cache invalidation).
err = pkgnats.Publish(b.js, subj, evt, b.log)

// Mode B: Critical event (DB write + publish atomic) — Transactional Outbox.
// Marshal once → store as TEXT in outbox table → background goroutine publishes
// raw bytes via js.Publish() directly (avoids double-serialisation).
// See SERVICE-ARCHITECTURE.md ⑮ for complete implementation.
payload, _ := json.Marshal(evt)        // serialise once
tx.Outbox.Enqueue(ctx, subj, payload)  // inside DB transaction
// Publisher goroutine: js.Publish(row.Subject, row.Payload) — NOT pkgnats.Publish
```

**Architecture rule:** The `app/` layer never holds `nats.JetStreamContext` directly.
All NATS interaction goes through a `BrokerInterface` defined in `app/` and implemented in `infra/broker/`.

`BaseEvent` fields: `id` (UUID v4), `subject`, `service`, `actor_id` (`*uint`, nil=system), `occurred_at`, `payload`.

---

## Cross-Cutting Patterns

### Transactional Outbox (for critical events)

The outbox guarantees that a DB write and its corresponding NATS event are **atomic** — neither is lost if the service crashes between the two operations.

```
Service-level (NOT in tais-core — each service owns its own outbox):

domain/repository/outbox.go      → OutboxRepository interface
domain/repository/unit_of_work.go → UnitOfWork interface + per-service Tx struct
infra/database/uow.go            → UnitOfWork impl: gorm.DB.Transaction(...)
infra/repository/postgres/outbox_repo.go → Enqueue / FetchUnpublished / MarkPublished
infra/outbox/publisher.go        → background goroutine, polls every 500ms
```

**Key rules:**
- Payload is serialised **once** (`json.Marshal(evt)`) and stored as `TEXT` in the outbox table.
- The publisher uses `js.Publish(subject, []byte)` directly — **never** `pkgnats.Publish` (which would marshal again).
- Single publisher goroutine — sequential flush per tick, no concurrent goroutines sharing a batch.
- Shutdown order: HTTP stop → `cancelOutbox()` (final flush) → `nc.Drain()`.

### Idempotent NATS consumers

JetStream delivers **at-least-once**. Every consumer MUST be idempotent.

```
Service-level:

migrations/00003_processed_events.sql  → processed_events(event_id TEXT PK, subject, created_at)
infra/broker/{entity}_consumer.go      → handler: check → process → mark → Ack
```

**Handler steps:**
1. Unmarshal `BaseEvent` — on error: `msg.Term()` (poison, no redeliver)
2. `EventProcessed(ctx, evt.ID)` — if true: `msg.Ack()` and return
3. Execute business logic — on transient error: `msg.Nak()` (redeliver after AckWait)
4. `MarkEventProcessed(ctx, evt.ID)` — best-effort
5. `msg.Ack()`

### Graceful shutdown order (non-negotiable)

```go
// 1. Stop HTTP — no new requests
srv.Shutdown(httpCtx)

// 2. Cancel outbox publisher — triggers final flush while NATS is still up
ctn.CancelOutbox()

// 3. Drain NATS — flushes pending publishes + in-flight acks, then closes
ctn.NATS.Drain()

// 4. Flush logger buffers
log.Sync()
```

---

## Import aliases (use consistently across all services)

```go
pkgi18n  "github.com/DC-TechHQ/tais-core/i18n"
pkgerr   "github.com/DC-TechHQ/tais-core/errors"
pkglog   "github.com/DC-TechHQ/tais-core/logger"
pkgcfg   "github.com/DC-TechHQ/tais-core/config"
pkgdb    "github.com/DC-TechHQ/tais-core/db"
pkgredis "github.com/DC-TechHQ/tais-core/redis"
pkgnats  "github.com/DC-TechHQ/tais-core/nats"
pkgjwt   "github.com/DC-TechHQ/tais-core/jwt"
pkgctx   "github.com/DC-TechHQ/tais-core/context"
pkgmw    "github.com/DC-TechHQ/tais-core/middleware"
pkgresp  "github.com/DC-TechHQ/tais-core/response"
pkgpage  "github.com/DC-TechHQ/tais-core/pagination"
pkgevent "github.com/DC-TechHQ/tais-core/event"
```

---

## Usage

```bash
go get github.com/DC-TechHQ/tais-core@latest
```

```go
// In service go.mod — use replace directive for local development
require github.com/DC-TechHQ/tais-core v0.0.0

replace github.com/DC-TechHQ/tais-core => ../tais-core
```

---

## Tests

```bash
cd tais-core
go test ./...
```

All packages with business logic have unit tests. Integration tests (real PostgreSQL, Redis, NATS)
are run in each service's own test suite.
