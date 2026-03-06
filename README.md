# tais-core

Shared Go library for the **TAIS** (Traffic Authority Information System — РБДА Tajikistan) platform.

> **Module:** `github.com/DC-TechHQ/tais-core`
> **Go version:** 1.26
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
// → "Транспортное средство не найдено"
```

---

### `errors` — AppError type

HTTP-aware error type. Always carries an i18n code + HTTP status. Never wrap across layers.

```go
import pkgerr "github.com/DC-TechHQ/tais-core/errors"

// Service-level custom error
var ErrVehicleNotFound = pkgerr.New(i18n.ErrVehicleNotFound, 404)

// Common errors available:
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
defer nc.Drain()

// Publish — marshals to JSON internally
err = pkgnats.Publish(js, "tais.vehicle.vehicle.created", payload, log)

// Subscribe — durable consumer with panic recovery
pkgnats.Subscribe(js, "tais.vehicle.>", "tais-audit-consumer", func(msg *nats.Msg) {
    // process msg.Data
    msg.Ack()
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

pkgresp.OK(c, data)                              // 200 {"success":true,"data":{...}}
pkgresp.Created(c, data)                         // 201 {"success":true,"data":{...}}
pkgresp.NoContent(c)                             // 204

pkgresp.Paginated(c, items, total, page, limit)  // 200 {"success":true,"data":[...],"meta":{...}}
// meta: {"total":500,"page":2,"limit":20,"total_pages":25}

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
type identityResolver struct {
    url   string
    token string
}

func (r *identityResolver) Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error) {
    // GET {r.url}/internal/users/{userID}/context
    // X-Internal-Token: {r.token}
    // → unmarshal into *pkgctx.UserCtx
}
```

---

### `event` — Domain events

```go
import pkgevent "github.com/DC-TechHQ/tais-core/event"

// Build subject
subj := pkgevent.Subject("registration", "vehicle", "registered")
// → "tais.registration.vehicle.registered"

// Create event — embed in service-level struct
type VehicleRegisteredEvent struct {
    pkgevent.BaseEvent
    VehicleID uint   `json:"vehicle_id"`
    VIN       string `json:"vin"`
}

actorID := u.ID
ev := VehicleRegisteredEvent{
    BaseEvent: pkgevent.New(subj, "tais-registration", &actorID, nil),
    VehicleID: v.ID,
    VIN:       v.VIN,
}

// Publish via nats
pkgnats.Publish(js, subj, ev, log)
```

`BaseEvent` fields: `id` (UUID), `subject`, `service`, `actor_id` (`nil` for system events), `occurred_at`, `payload`.

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

## Development usage (before tagging v0.1.0)

```go
// In service go.mod — use replace directive while developing locally
require github.com/DC-TechHQ/tais-core v0.0.0

replace github.com/DC-TechHQ/tais-core => ../tais-core
```

After tagging:
```bash
go get github.com/DC-TechHQ/tais-core@v0.1.0
```

Private module — all services must set:
```bash
export GOPRIVATE=github.com/DC-TechHQ/*
```

---

## Tests

```bash
cd tais-core
go test ./...
```

All packages with business logic have unit tests. Integration tests (real PostgreSQL, Redis, NATS)
are run in each service's own test suite.
