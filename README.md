# tais-core

Shared Go library for the **TAIS** (Traffic Authority Information System — РБДА Tajikistan) platform.

> **Module:** `github.com/DC-TechHQ/tais-core`
> **Go version:** 1.24+
> **License:** Private — DC-TechHQ

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
```

**Fallback order:** requested lang → RU → error code itself (never panics).

---

### `errors` — AppError

```go
import pkgerr "github.com/DC-TechHQ/tais-core/errors"

// Common errors (pre-defined):
pkgerr.ErrNotFound        // 404
pkgerr.ErrAlreadyExists   // 409
pkgerr.ErrInvalidData     // 400
pkgerr.ErrForeignKey      // 400
pkgerr.ErrUnauthorized    // 401
pkgerr.ErrForbidden       // 403
pkgerr.ErrInvalidToken    // 401
pkgerr.ErrTokenExpired    // 401
pkgerr.ErrUserBlocked     // 403
pkgerr.ErrInvalidCredentials // 401
pkgerr.ErrDeadlock        // 409
pkgerr.ErrInternal        // 500

// Service-specific errors (defined in internal/errors/errors.go):
var ErrVehicleNotFound = pkgerr.New("ErrVehicleNotFound", 404)
```

**Rule:** never wrap `*AppError` across layers — it breaks `errors.As` matching.

---

### `logger` — Structured logger

Built on `go.uber.org/zap` + `gopkg.in/natefinch/lumberjack.v2`.

**Log files (JSON / production mode):**

| File | Content |
|---|---|
| `{dir}/info.log` | INFO and DEBUG messages |
| `{dir}/warn.log` | WARN messages |
| `{dir}/error.log` | ERROR and FATAL messages |
| `{dir}/gorm.log` | **Every** SQL query (elapsed, rows, slow marker) |

All levels also write to **stdout** so Docker log drivers and Grafana Loki collect them automatically without volume mounts.

```go
import pkglog "github.com/DC-TechHQ/tais-core/logger"

log, err := pkglog.New(pkglog.Config{
    Directory:  "/var/log/tais-vehicle", // required in json mode
    Level:      "info",                  // debug | info | warn | error
    Format:     "json",                  // json (prod) | console (dev)
    MaxSizeMB:  100,
    MaxBackups: 10,
    MaxAgeDays: 30,
    Compress:   true,
})

log.Info("vehicle created", "id", v.ID, "vin", v.VIN)
log.Error("db query failed", "error", err, "id", id)

// Scoped child logger — fields attached to every call:
repoLog := log.With("component", "vehicle-repo")
repoLog.Error("FindByID failed", "error", err, "id", id)

// Flush on graceful shutdown:
defer log.Sync()
```

---

### `config` — Docker secrets

```go
import pkgcfg "github.com/DC-TechHQ/tais-core/config"

// Reads /run/secrets/{name}, falls back to TAIS_{NAME} env var:
password := pkgcfg.ReadSecret("vehicle-db-password")

// Panics if not found (use for required secrets in production):
secret := pkgcfg.MustReadSecret("jwt-secret")
```

---

### `db` — PostgreSQL / GORM factory

```go
import pkgdb "github.com/DC-TechHQ/tais-core/db"

// Connection factory — each service gets its own *gorm.DB:
gdb, err := pkgdb.New(pkgdb.Config{
    DSN:             cfg.Postgres.DSN(), // includes search_path=schema
    MaxOpenConns:    25,
    MaxIdleConns:    10,
    ConnMaxLifetime: 300, // seconds
}, log)
```

**GORMLogger** — logs **every** SQL query to `gorm.log`:
- Normal queries → INFO with `sql`, `rows`, `elapsed_ms`, `slow_query: false`
- Slow queries (>200ms) → INFO with `slow_query: true`
- Error queries → ERROR with `error` field

**TranslateError** — maps raw GORM/pgx errors to `*AppError`. Must receive `log` for internal logging:

```go
// In repository methods:
return database.TranslateError(err, r.log)
// → pkgerr.ErrNotFound      (gorm.ErrRecordNotFound)
// → pkgerr.ErrAlreadyExists (23505 unique_violation)
// → pkgerr.ErrForeignKey    (23503 foreign_key_violation)
// → pkgerr.ErrInvalidData   (23502/23514 null/check violation)
// → pkgerr.ErrDeadlock      (40P01 deadlock_detected)
// → pkgerr.ErrInternal      (all others)
// → nil                     (nil error or context.Canceled)
```

**Builder** — fluent, nil-safe query builder. All conditions skip when value is zero:

```go
q := pkgdb.NewBuilder(r.db.WithContext(ctx).Model(&models.Vehicle{})).
    Where("status = ?", f.Status).           // skipped if f.Status == ""
    Where("type_id = ?", f.TypeID).          // skipped if f.TypeID == 0
    Search(f.Search, "vin", "plate_number"). // ILIKE on multiple columns
    DateRange("created_at", f.From, f.To)   // skipped if empty

q.Build().Count(&total)
q.OrderBy("created_at", "desc").Pagination(f.Params).Build().Find(&list)
```

---

### `redis` — Redis client factory

```go
import pkgredis "github.com/DC-TechHQ/tais-core/redis"

rdb, err := pkgredis.New(pkgredis.Config{
    Addr:     "redis:6379",
    Password: pkgcfg.ReadSecret("redis-password"),
    DB:       0,
}, log)
```

Key namespace convention: `tais:{service}:{entity}:{id}`

---

### `nats` — NATS JetStream

```go
import pkgnats "github.com/DC-TechHQ/tais-core/nats"

nc, js, err := pkgnats.Connect(pkgnats.Config{
    URL:   "nats://nats:4222",
    Token: pkgcfg.ReadSecret("nats-token"),
}, log)
defer nc.Close()

// Fire-and-forget publish (use js.Publish for durable):
pkgnats.Publish(nc, "tais.vehicle.vehicle.created", payload)

// Subscribe:
pkgnats.Subscribe(nc, "tais.vehicle.>", handler)
```

Subject convention: `tais.{service}.{entity}.{event}`
Stream: `TAIS_EVENTS`
tais-audit subscribes `tais.>` to capture all events.

---

### `jwt` — Token parsing

```go
import pkgjwt "github.com/DC-TechHQ/tais-core/jwt"

// Parse and validate (tais-auth issues, all other services parse):
claims, err := pkgjwt.Parse(tokenStr, pkgjwt.Config{
    Secret: pkgcfg.MustReadSecret("jwt-secret"),
})

// Sign (tais-auth only):
token, err := pkgjwt.Sign(&pkgjwt.Claims{
    Sub:   user.ID,
    Type:  pkgjwt.TypeStaff,
    IpNet: "10.200.1",  // first 3 octets — staff only
    JTI:   uuid.NewString(),
}, cfg)

// IP subnet check (staff tokens only, citizens always pass):
if !pkgjwt.CheckIPNet(claims, c.ClientIP()) { ... }
```

**Claims:** `Sub` (user ID), `Type` (staff|citizen), `IpNet` (/24 subnet), `JTI` (blacklist key).
**Algorithm:** HS256 only.

---

### `context` — UserCtx

```go
import pkgctx "github.com/DC-TechHQ/tais-core/context"

// In middleware (after auth):
pkgctx.SetUser(c, userCtx)

// In handlers and use cases:
u, ok := pkgctx.GetUser(c)     // returns (nil, false) if unauthenticated
u    := pkgctx.MustGetUser(c)  // panics if not set — only on protected routes

// Permission check:
if pkgctx.HasPermission(u, "vehicle:read") { ... }
// super_admin → always true
// admin (permissions: ["*"]) → always true
// others → exact match in u.Permissions slice
```

---

### `middleware` — HTTP middleware

```go
import pkgmw "github.com/DC-TechHQ/tais-core/middleware"

// Router setup:
r.Use(pkgmw.Recovery(log))           // panic recovery → 500
r.Use(pkgmw.RequestLogger(log))      // request logging with request_id
r.Use(pkgmw.CORS(cfg.CORSOrigins))   // CORS headers

auth := pkgmw.Required(rdb, cfg.JWT, resolver) // JWT auth + blacklist + IP check + user ctx

// Route guards:
v.GET("/:id", auth, pkgmw.Can("vehicle:read"),     handler.Get)
v.POST("",    auth, pkgmw.Can("vehicle:register"), handler.Create)
v.GET("",     auth, pkgmw.CanAny("vehicle:read", "vehicle:register"), handler.List)

// Internal service-to-service routes (not exposed via Traefik):
internal.Use(pkgmw.InternalOnly(cfg.InternalToken))
```

**`UserContextResolver` interface** — implemented per-service in `infra/resolver/identity.go`:

```go
type UserContextResolver interface {
    Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error)
}
// Implementation calls: GET {identityURL}/internal/users/{id}/context
// Result cached in Redis: user_ctx:{user_id} TTL 5 min
```

---

### `response` — HTTP response helpers

All responses follow a single envelope format.

```go
import pkgresp "github.com/DC-TechHQ/tais-core/response"

pkgresp.OK(c, dto)                        // 200 { success: true, data: ... }
pkgresp.Created(c, dto)                   // 201 { success: true, data: ... }
pkgresp.NoContent(c)                      // 204
pkgresp.Paginated(c, list, total, params) // 200 + pagination meta
pkgresp.Error(c, err)                     // maps *AppError → HTTP status + TJ+RU+EN
pkgresp.ErrorWithData(c, err, data)       // same + extra data (e.g. validation errors)
```

**Error envelope:**
```json
{
  "success": false,
  "error": {
    "code": "ErrNotFound",
    "message": { "tj": "...", "ru": "...", "en": "..." },
    "data": null
  }
}
```

**Validation error example:**
```go
pkgresp.ErrorWithData(c, pkgerr.ErrInvalidData, []pkgresp.ValidationError{
    {Field: "vin",          Message: "invalid format"},
    {Field: "plate_number", Message: "already registered"},
})
```

**Paginated response:**
```json
{
  "success": true,
  "data": [...],
  "pagination": { "page": 1, "limit": 20, "total": 500, "total_pages": 25 }
}
```

---

### `pagination` — Query parameter parsing

```go
import pkgpage "github.com/DC-TechHQ/tais-core/pagination"

params := pkgpage.Parse(c) // ?page=2&limit=10
// params.Page   = 2
// params.Limit  = 10
// params.Offset = 10
// Defaults: page=1, limit=20. Cap: limit=100. Invalid values → defaults.
```

---

### `event` — NATS event envelope

```go
import pkgevent "github.com/DC-TechHQ/tais-core/event"

type VehicleCreatedEvent struct {
    pkgevent.BaseEvent
    VehicleID uint   `json:"vehicle_id"`
    VIN       string `json:"vin"`
}

ev := VehicleCreatedEvent{
    BaseEvent: pkgevent.New("tais-vehicle", "vehicle", "vehicle", "created", actorID),
    VehicleID: v.ID,
    VIN:       v.VIN,
}
// ev.Subject = "tais.vehicle.vehicle.created"
// ev.ID      = unique nano-timestamp ID
// ev.OccurredAt = time.Now().UTC()
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

## Rules (non-negotiable)

1. **Never copy-paste** code from tais-core into services — import it.
2. **Never import service code** into tais-core — this is a leaf dependency.
3. **TranslateError always receives `log`** — logging is done inside, never before calling it.
4. **Never wrap `*AppError`** across layers — `errors.As` matching breaks.
5. **Cache miss returns `nil, nil`** — never an error.
6. **All error responses include TJ + RU + EN** translations.
7. **Log at repository layer** — handlers and use cases call `pkgresp.Error(c, err)`, never log there.
8. **GORM models never leave `infra/`** — mapper converts to/from domain entity.

---

## Running tests

```bash
go test ./...

# With verbose output:
go test ./... -v

# Single package:
go test ./db/... -v
go test ./jwt/... -v
go test ./response/... -v
```

> Tests requiring a real PostgreSQL or Redis instance are integration tests and are skipped in CI unless the `INTEGRATION` env var is set.

---

## Local development

This is a **library** — it has no `main.go`, no HTTP server, no port.
To use locally in a service during development:

```bash
# In the service's go.mod:
replace github.com/DC-TechHQ/tais-core => ../tais-core
```

Remove the `replace` directive before pushing to CI.
