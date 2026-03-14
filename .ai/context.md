# tais-core — AI Context

> **Source of truth:** `.ai/context.md`
> **Generated files:** `CLAUDE.md` (Claude Code) · `AGENTS.md` (OpenAI Codex) · `.gemini/GEMINI.md` (Gemini CLI)
> **To regenerate after editing:** `.ai/init all`

---

# ① THIS LIBRARY

## What It Does

`tais-core` is the **single shared Go library** for all 28 TAIS backend services.
It is **NOT a service** — no `main.go`, no HTTP server, no database, no port.
Every TAIS service imports it as a private Go module.

It provides: multilingual error/message registry (TJ+RU+EN), shared `AppError` type,
structured logger (zap+lumberjack), Docker secrets reader, GORM connection factory +
pg error translator + fluent query builder, Redis client factory, NATS JetStream factory,
JWT parsing, user context struct + Gin helpers, auth + permission middleware, standard HTTP
response envelope, pagination parser, and base NATS event struct.

## Identity

| | |
|---|---|
| Module | `github.com/DC-TechHQ/tais-core` |
| Type | Library — NO port, NO database, NO main.go |
| Go version | 1.26 |
| Used by | All 28 TAIS backend services |

## Directory Structure

```
tais-core/
├── go.mod
├── go.sum
├── i18n/
│   ├── i18n.go       # Register(data), Get(code, lang), LangTJ/LangRU/LangEN
│   └── common.go     # init() — common codes with TJ+RU+EN translations
├── errors/
│   └── errors.go     # AppError{Code, Status} + common error vars
├── logger/
│   └── logger.go     # New(cfg LogConfig) (*Logger, error)
├── config/
│   └── secrets.go    # ReadSecret(name) + MustReadSecret(name)
├── db/
│   ├── db.go         # New(cfg Config, log) (*gorm.DB, error)
│   ├── translator.go # TranslateError(err error) error — no logger
│   └── builder.go    # Builder — fluent nil-safe GORM query builder
├── redis/
│   └── redis.go      # New(cfg Config, log) (*redis.Client, error)
├── nats/
│   └── nats.go       # Connect(cfg, log) + Publish() + Subscribe()
├── jwt/
│   └── jwt.go        # Claims{Sub, Type, IpNet, JTI} + Parse(token, cfg)
├── context/
│   └── context.go    # UserCtx + GetUser(c) + MustGetUser(c) + HasPermission(u, code)
├── middleware/
│   ├── auth.go       # UserContextResolver interface + Required() + InternalOnly()
│   └── permission.go # Can(code) + CanAny(codes...) + CanAll(codes...)
├── response/
│   └── response.go   # OK + Created + Paginated + Error + NoContent
├── pagination/
│   └── pagination.go # Parse(c *gin.Context) Params{Page, Limit, Offset}
└── event/
    └── event.go      # BaseEvent + Subject(svc, entity, ev) + New(...)
```

## Package Build Order (dependency chain — implement in this order)

```
1.  i18n/       — zero deps
2.  errors/     — depends on i18n/
3.  logger/     — zero deps
4.  config/     — zero deps
5.  db/         — depends on logger/, errors/
6.  redis/      — depends on logger/
7.  nats/       — depends on logger/
8.  jwt/        — zero deps (+ jwt library)
9.  context/    — depends on errors/
10. response/   — depends on i18n/, errors/
11. pagination/ — zero deps
12. middleware/auth.go       — depends on jwt/, redis/, context/, response/, errors/
13. middleware/permission.go — depends on context/, response/
14. event/      — zero deps
```

## Full API Contracts

### `i18n/i18n.go`
```go
const (
    LangTJ = "tj"
    LangRU = "ru"
    LangEN = "en"
)

func Register(data map[string]map[string]string) // called from init()
func Get(code, lang string) string               // fallback: lang → RU → code itself
```

### `i18n/common.go` — error/message codes
```go
const (
    MsgSuccess    = "MsgSuccess"
    MsgCreated    = "MsgCreated"
    MsgUpdated    = "MsgUpdated"
    MsgDeleted    = "MsgDeleted"

    ErrInternal           = "ErrInternal"
    ErrInvalidData        = "ErrInvalidData"
    ErrNotFound           = "ErrNotFound"
    ErrAlreadyExists      = "ErrAlreadyExists"
    ErrForeignKey         = "ErrForeignKey"
    ErrUnauthorized       = "ErrUnauthorized"
    ErrForbidden          = "ErrForbidden"
    ErrInvalidToken       = "ErrInvalidToken"
    ErrTokenExpired       = "ErrTokenExpired"
    ErrUserBlocked        = "ErrUserBlocked"
    ErrInvalidCredentials = "ErrInvalidCredentials"
    ErrDeadlock           = "ErrDeadlock"
)
```

### `errors/errors.go`
```go
type AppError struct {
    Code   string
    Status int
}
func (e *AppError) Error() string { return e.Code }
func New(code string, status int) *AppError

var (
    ErrInternal           = New(i18n.ErrInternal,           500)
    ErrInvalidData        = New(i18n.ErrInvalidData,        400)
    ErrNotFound           = New(i18n.ErrNotFound,           404)
    ErrAlreadyExists      = New(i18n.ErrAlreadyExists,      409)
    ErrForeignKey         = New(i18n.ErrForeignKey,         400)
    ErrUnauthorized       = New(i18n.ErrUnauthorized,       401)
    ErrForbidden          = New(i18n.ErrForbidden,          403)
    ErrInvalidToken       = New(i18n.ErrInvalidToken,       401)
    ErrTokenExpired       = New(i18n.ErrTokenExpired,       401)
    ErrUserBlocked        = New(i18n.ErrUserBlocked,        403)
    ErrInvalidCredentials = New(i18n.ErrInvalidCredentials, 401)
    ErrDeadlock           = New(i18n.ErrDeadlock,           409)
)
```

### `db/translator.go`
```go
// Maps raw DB errors to *AppError.
// Does NOT log — caller must log before calling this.
// Handles: 23505 unique, 23503 fk, 23502 null, 23514 check, 40P01 deadlock
// gorm.ErrRecordNotFound → ErrNotFound
// context.Canceled → nil (not an error)
func TranslateError(err error) error
```

### `db/builder.go`
```go
func NewBuilder(db *gorm.DB) *Builder
func (b *Builder) Where(query string, arg any) *Builder       // skips nil/zero pointer
func (b *Builder) Search(query *string, cols ...string) *Builder
func (b *Builder) DateRange(col string, from, to any) *Builder
func (b *Builder) OrderBy(col, dir string) *Builder
func (b *Builder) Pagination(p pagination.Params) *Builder
func (b *Builder) Build() *gorm.DB
```

### `response/response.go`
```go
// Named key is MANDATORY — never use positional data key.
// Examples:
//   OK(c, "vehicle", dto)           → {"success":true,"vehicle":{...}}
//   Created(c, "vehicle", dto)      → {"success":true,"vehicle":{...}}  HTTP 201
//   Paginated(c, "vehicles", items, total, page, limit)
//       → {"success":true,"vehicles":[...],"meta":{"total":N,"page":N,"limit":N,"total_pages":N}}
//   Error(c, err)                   → {"success":false,"error":{"code":"ErrNotFound","message":{"tj":"...","ru":"...","en":"..."}}}
//   NoContent(c)                    → HTTP 204

func OK(c *gin.Context, key string, data any)
func Created(c *gin.Context, key string, data any)
func NoContent(c *gin.Context)
func Error(c *gin.Context, err error)
func Paginated(c *gin.Context, key string, data any, total int64, page, limit int)
```

### `middleware/auth.go`
```go
// UserContextResolver — implemented per service in infra/resolver/identity.go
type UserContextResolver interface {
    Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error)
}

// Required(rdb, jwtCfg, resolver) — ALWAYS 3 params, resolver is mandatory
// Flow: Bearer → Parse JWT → ip_net check → blacklist check → user_ctx cache/resolve → IsActive
func Required(rdb *redis.Client, jwtCfg jwt.Config, resolver UserContextResolver) gin.HandlerFunc

func InternalOnly(token string) gin.HandlerFunc
func CitizenOnly() gin.HandlerFunc
func StaffOnly() gin.HandlerFunc
```

### `context/context.go`
```go
type UserCtx struct {
    ID            uint
    Type          string   // "staff" | "citizen"
    IsSuperAdmin  bool
    IsActive      bool
    Roles         []string
    Permissions   []string
    DeptID        *uint
    RegionID      *uint
    DLAuthorityID *uint
    IpNet         string
    JTI           string
}
const KeyUser = "tais_user"
func GetUser(c *gin.Context) (*UserCtx, bool)
func MustGetUser(c *gin.Context) *UserCtx
func HasPermission(u *UserCtx, code string) bool
// super_admin → true always. "*" in permissions → true always.
```

### `event/event.go`
```go
type BaseEvent struct {
    ID         string    // uuid
    Subject    string
    Service    string
    ActorID    *uint     // nil = system action
    OccurredAt time.Time
    Payload    any
}

func Subject(service, entity, event string) string
    // returns "tais.{service}.{entity}.{event}"
func New(subject, service string, actorID *uint, payload any) BaseEvent
```

### `nats/nats.go`
```go
func Connect(cfg Config, log *logger.Logger) (*nats.Conn, nats.JetStreamContext, error)

// Marshals payload to JSON and publishes. Use for ad-hoc publishes.
// For critical ops, use Transactional Outbox instead.
func Publish(js nats.JetStreamContext, subject string, payload any, log *logger.Logger) error

func Subscribe(js nats.JetStreamContext, subject, consumer string,
    handler func(*nats.Msg), log *logger.Logger)
```

## Rules for THIS Repo (tais-core specific)

```
1. No main.go — library only
2. No circular imports — follow build order strictly
3. No service-specific logic — if it only makes sense for one service, it doesn't belong here
4. No hardcoded URLs or service names — UserContextResolver interface exists for this
5. TranslateError has NO logger param — pure translation, one responsibility
6. All Builder methods skip nil — nil-safe by design, no panics
7. Every package must have _test.go
8. response.OK / Created / Paginated ALWAYS take a named key (2nd param) — never generic "data"
9. Required() ALWAYS takes 3 params: rdb, jwtCfg, resolver — resolver is never optional
```

---

# ② ARCHITECTURE — THE LAW

**This section is identical across ALL 28 TAIS service repos.**

## Clean Architecture

```
delivery/http/  →  app/  →  domain/  ←  infra/
(handlers,          (use      (entities,    (DB, Redis,
 middleware,         cases)    interfaces)   NATS, ...)
 router)
```

**Dependency rule:** `delivery` → `app` → `domain` ← `infra`

## Layer Rules

| Layer | MUST | MUST NOT |
|---|---|---|
| `domain/entity/` | Pure Go structs | Import GORM, gin, any framework |
| `domain/repository/` | Define interfaces only | Implement anything |
| `app/` | Business logic, domain interfaces | Import GORM, gin, Redis directly |
| `dto/request/`, `dto/response/` | HTTP boundary types | Contain business logic |
| `delivery/http/handler/` | Parse → call app → respond | Touch DB or Redis directly |
| `infra/database/models/` | GORM structs with tags | Leak outside `infra/` |
| `infra/repository/postgres/` | Implement domain interfaces + mappers | Contain business logic |
| `infra/cache/` | Redis GET/SET — cache miss returns `nil, nil` | Contain business logic |
| `infra/broker/` | Publish (via app interface) + idempotent consumer | Contain business logic |
| `infra/resolver/` | Implement `UserContextResolver` — HTTP call to tais-identity | Contain business logic |
| `infra/container/` | Wire all dependencies + start outbox goroutine | Contain business logic |
| `migrations/` | Versioned goose SQL files (Up + Down) | Contain application logic |

## 13 Non-Negotiable Rules

```
1.  Clean Architecture — layers only talk through interfaces, never skip layers
2.  GORM models NEVER leave infra/ — mapper converts to/from domain entity
3.  Cache miss → nil, nil — NOT an error. Real Redis failure → actual error.
4.  Never wrap *AppError across layers — breaks errors.As matching
5.  All errors return TJ + RU + EN in every response
6.  goose migrations — no AutoMigrate, no manual SQL outside migrations/
7.  database.TranslateError(err) — NO log parameter. Pure translation only.
8.  Repository logs raw error BEFORE calling TranslateError (op name + entity ID)
9.  tais-core is the ONLY shared library — no copy-paste between services
10. App layer NEVER holds nats.JetStreamContext — only infra/broker/ structs do
11. Critical operations use Transactional Outbox — DB write + outbox in ONE transaction
12. Shutdown order: HTTP stop → CancelOutbox (final flush) → NATS.Drain → log.Sync
13. NATS consumers are idempotent — check processed_events before processing
```

## Import Aliases (ALWAYS use these — never deviate)

```go
import (
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
)
```

## Key Code Templates

### Response format (MUST use named key)
```go
pkgresp.OK(c, "vehicle", dto)                                 // {"success":true,"vehicle":{...}}
pkgresp.Created(c, "vehicle", dto)                            // HTTP 201
pkgresp.Paginated(c, "vehicles", items, total, page, limit)   // + "meta": {total, page, limit, total_pages}
pkgresp.Error(c, err)                                         // {"success":false,"error":{"code":"...","message":{"tj":"...","ru":"...","en":"..."}}}
pkgresp.NoContent(c)                                          // HTTP 204
```

### Auth middleware (router.go)
```go
auth := pkgmw.Required(ctn.Redis, ctn.Config.JWT, ctn.Resolver)

v1 := r.Group("/api/v1")
{
    res := v1.Group("/{resource}")
    res.Use(auth)
    {
        res.POST("",       pkgmw.Can("{resource}:create"), h.Create)
        res.GET("",        pkgmw.Can("{resource}:read"),   h.List)
        res.GET("/:id",    pkgmw.Can("{resource}:read"),   h.GetByID)
        res.PUT("/:id",    pkgmw.Can("{resource}:update"), h.Update)
        res.DELETE("/:id", pkgmw.Can("{resource}:delete"), h.Delete)
    }
}

internal := r.Group("/internal")
internal.Use(pkgmw.InternalOnly(ctn.Config.InternalToken))
internal.GET("/{resources}/:id", h.GetByIDInternal)
```

### Repository pattern
```go
func (r *repo) FindOne(ctx context.Context, f entity.Filter) (*entity.Entity, error) {
    var m models.Entity
    q := pkgdb.NewBuilder(r.db.WithContext(ctx).Model(&models.Entity{})).
        Where("id = ?", f.ID)
    if err := q.Build().First(&m).Error; err != nil {
        r.log.Error("EntityRepo.FindOne", "filter", f, "err", err)
        return nil, database.TranslateError(err)
    }
    return toEntityDomain(&m), nil
}
```

### Cache-aside (app layer)
```go
func (a *App) GetByID(ctx context.Context, id uint) (*entity.Entity, error) {
    if v, _ := a.cache.Get(ctx, id); v != nil {
        return v, nil
    }
    v, err := a.repo.FindOne(ctx, entity.Filter{ID: &id})
    if err != nil {
        return nil, err
    }
    _ = a.cache.Set(ctx, v) // best-effort
    return v, nil
}
```

### Shutdown sequence (MUST follow this order exactly)
```go
srv.Shutdown(httpCtx)   // 1. stop HTTP — no new requests
ctn.CancelOutbox()      // 2. final outbox flush while NATS still up
ctn.NATS.Drain()        // 3. flush in-flight NATS — LAST, after HTTP + outbox
log.Sync()              // 4. flush log buffers
```

### i18n registration (service-specific)
```go
// internal/i18n/{entity}.go
func init() {
    pkgi18n.Register(map[string]map[string]string{
        "ErrEntityNotFound": {
            pkgi18n.LangTJ: "...",
            pkgi18n.LangRU: "...",
            pkgi18n.LangEN: "...",
        },
    })
}
// imported as _ in main.go: import _ "tais-{service}/internal/i18n"
```

### Transactional Outbox (critical ops)
```go
// domain/repository/unit_of_work.go
type UnitOfWork interface {
    RunInTx(ctx context.Context, fn func(tx TxRepos) error) error
}
type TxRepos struct {
    Entity EntityRepository
    Outbox OutboxRepository
}

// app layer:
err = a.uow.RunInTx(ctx, func(tx repository.TxRepos) error {
    if err := tx.Entity.Save(ctx, &e); err != nil { return err }
    return tx.Outbox.Enqueue(ctx, outbox.Message{Subject: subj, Payload: payload})
})
```

### NATS consumer (idempotent)
```go
func (c *Consumer) handle(msg *nats.Msg) {
    var event pkgevent.BaseEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        msg.Term() // poison message — don't retry
        return
    }
    // idempotency check
    if processed, _ := c.repo.IsProcessed(ctx, event.ID); processed {
        msg.Ack()
        return
    }
    if err := c.process(ctx, event); err != nil {
        msg.Nak() // transient error — retry
        return
    }
    _ = c.repo.MarkProcessed(ctx, event.ID)
    msg.Ack()
}
```

---

# ③ FORBIDDEN PATTERNS

**Never do these — they are bugs, not style choices.**

```
❌ gorm.AutoMigrate(...)
✅ goose SQL migrations in migrations/ directory

❌ fmt.Errorf("wrap: %w", appErr)   // breaks errors.As
✅ return appErr                     // return AppError directly

❌ r.db.First(&model, id)  // GORM model leaked to caller
✅ return toEntityDomain(&m) // always convert via mapper

❌ database.TranslateError(err, r.log)  // no log param
✅ r.log.Error("op", "id", id, "err", err); return database.TranslateError(err)

❌ pkgresp.OK(c, dto)        // no named key
✅ pkgresp.OK(c, "vehicle", dto)

❌ pkgmw.Required(rdb, jwtCfg)  // missing resolver
✅ pkgmw.Required(rdb, jwtCfg, ctn.Resolver)

❌ js.Publish(subject, data)  // direct NATS from app layer
✅ a.broker.Publish(ctx, event)  // via BrokerInterface

❌ resolver.NewIdentity("http://identity:8002", token)  // hardcoded URL
✅ resolver.NewIdentity(cfg.Services.IdentityURL, cfg.InternalToken, cfg.App.Name)

❌ copy-paste business logic between services
✅ add to tais-core if it's truly shared

❌ pgbouncer:5432 or postgres:5432
✅ pgbouncer:6432

❌ cache miss returning an error
✅ if errors.Is(err, redis.Nil) { return nil, nil }

❌ NATS.Drain() called before HTTP shutdown
✅ srv.Shutdown() → ctn.CancelOutbox() → ctn.NATS.Drain() → log.Sync()

❌ processing NATS event without idempotency check
✅ check processed_events(event_id) before any processing
```

---

# ④ ENVIRONMENT

| | |
|---|---|
| Dev server | `45.94.216.40` |
| Domain | `dc-techhq.tj` |
| Staff panel | `staff.dc-techhq.tj` (VPN required) |
| Citizen portal | `portal.dc-techhq.tj` (public) |
| API | `api.dc-techhq.tj` |
| Swagger | `swagger.dc-techhq.tj` |
| PgBouncer | `pgbouncer:6432` (NOT postgres:5432) |
| Deploy | `docker stack deploy -c {service}-stack.yml tais` |
| Private modules | `GOPRIVATE=github.com/DC-TechHQ/*` |

---

# ⑤ FULL DOCUMENTATION

| Document | What's in it |
|---|---|
| `tais-docs/CLAUDE.md` | Full project context — all services, all rules, all patterns |
| `tais-docs/SERVICE-ARCHITECTURE.md` | THE LAW — complete code examples for every layer |
| `tais-docs/IMPLEMENTATION-PLAN.md` | Roadmap + tais-core package specs + full startup sequence |
| `tais-docs/SERVICES.md` | Per-service: entities, NATS events, responsibilities |
| `tais-docs/PERMISSIONS.md` | All 32 roles, ~120 permission codes, RBAC logic |
| `tais-docs/DATABASE.md` | All 16 databases, schemas, tables |
| `tais-docs/DEPLOYMENT.md` | Docker Swarm stacks, CI/CD, Dockerfile, Swagger |
