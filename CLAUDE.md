# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Start dependencies (PostgreSQL 15 + MailHog SMTP)
docker-compose up -d
 
# Run the application
go run cmd/server/main.go
 
# Run all tests
go test ./... -v
 
# Run a single package's tests
go test ./internal/service/... -v
 
# Run linter
task ops:lint
 
# Run linter with auto-fix
task ops:lint:fix
 
# Install linter
task ops:lint:install
```

## Architecture

**CaseTaskNotifier** is a Go service that lets users subscribe to email notifications for new GitHub repository releases, using a polling model rather than webhooks.

### Layer structure

```
HTTP Layer (chi router)  →  Service Layer  →  Repository Layer  →  PostgreSQL
                                ↓
                    GitHub Client / SMTP Mailer / URLBuilder
```

- **`internal/http/`** — Chi router and handlers for `POST /api/subscribe`, `GET /api/confirm/{token}`, `GET /api/unsubscribe/{token}`, `GET /api/subscriptions`
- **`internal/service/`** — Business logic; `subscription_service.go` orchestrates subscribe/confirm/unsubscribe flows; `scanner_service.go` contains the release detection logic with `ReleaseScanner` interface
- **`internal/scanner/`** — Background goroutine that ticks on `SCAN_INTERVAL` and delegates to `service.ReleaseScanner`
- **`internal/repository/`** — SQL access for `repositories` and `subscriptions` tables
- **`internal/github/`** — Thin HTTP wrapper over GitHub API v3 (repo existence check + latest release)
- **`internal/mailer/`** — SMTP mailer for confirmation and release notification emails
- **`internal/urlbuilder/`** — Builds confirm/unsubscribe URLs from base URL; injected as `*urlbuilder.Builder`
- **`internal/config/`** — Loads all config from environment variables
- **`internal/validator/`** — Email and `owner/repo` format validation
- **`migrations/`** — SQL migration files, auto-applied on startup via `golang-migrate`
### Key design points

- **Interfaces everywhere** — `github.Client`, `mailer.Mailer`, `service.ReleaseScanner`, `service.SubscriptionService`, `repository.SubscriptionRepository`, `repository.GitHubRepository` are all interfaces; each layer depends only on interfaces, never on concrete types.
- **Interface compliance checks** — `var _ ReleaseScanner = (*ReleaseScannerImpl)(nil)` and `var _ SubscriptionService = (*SubscriptionServiceImpl)(nil)` verify implementations at compile time; always add these for new interface implementations.
- **FindOrCreate pattern** — `GitHubRepository.FindOrCreate` ensures one `repositories` row per `owner/repo`; no duplicate GitHub API calls per scan cycle.
- **URLBuilder** — URL construction is isolated in `internal/urlbuilder`; never build URLs with `fmt.Sprintf` directly in service or handler code.
- **Scanner decoupling** — `scanner.Scanner` depends only on `service.ReleaseScanner` interface; it knows nothing about repositories, GitHub, or email.
- **Scanner first-scan baseline** — When `last_seen_tag` is `NULL`, the scanner sets it without sending notifications; only subsequent new tags trigger emails.
- **Double opt-in** — Subscriptions start `confirmed=false`; a UUID confirmation token is emailed; the scanner only processes `confirmed=true AND active=true` subscriptions.
- **Sentinel errors** — Service layer exposes typed errors (`ErrInvalidEmail`, `ErrInvalidRepo`, `ErrRepoNotFound`, `ErrAlreadySubscribed`, `ErrInvalidToken`, `ErrTokenNotFound`); handlers switch on these with `errors.Is`.
- **slog for logging** — Use `log/slog` everywhere; `log.Print*` and `fmt.Print*` are forbidden by the linter.
- **HTTP server timeouts** — Server is configured with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` as named constants in `main.go`.
- **`run()` pattern** — `main()` only calls `run()` and handles exit; all startup logic lives in `run() error`.
### Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `GITHUB_TOKEN` | — | Raises GitHub rate limit from 60→5,000 req/hour |
| `SMTP_HOST` | — | SMTP hostname |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_USER` / `SMTP_PASS` | — | SMTP credentials |
| `BASE_URL` | `http://localhost:8080` | Used in confirmation/unsubscribe links |
| `SCAN_INTERVAL` | `5m` | Release polling interval (e.g. `10s`, `5m`) |
| `PORT` | `8080` | HTTP server port |

### Local development

Docker Compose provides PostgreSQL 15 (port 5432) and MailHog (SMTP on 1025, web UI on 8025). Typical env setup:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/notifier?sslmode=disable"
export BASE_URL="http://localhost:8080"
export SMTP_HOST=localhost
export SMTP_PORT=1025
export SCAN_INTERVAL=10s
```

Inspect emails at `http://localhost:8025`. Architecture diagrams and ADRs are in `docs/`.

## SOLID & GRASP principles

The codebase follows these principles — new code must maintain them:

**SOLID:**
- **SRP** — Each struct has one responsibility: `Scanner` only schedules ticks; `ReleaseScannerImpl` only checks releases; `SubscriptionServiceImpl` only handles subscription flows; `urlbuilder.Builder` only builds URLs.
- **OCP** — New notification channels (e.g. Telegram) should be added by implementing the `mailer.Mailer` interface, not by modifying existing code.
- **LSP** — All interface implementations are verified at compile time via `var _ Interface = (*Impl)(nil)`; always add this for new implementations.
- **ISP** — Interfaces are narrow and role-specific: `ReleaseScanner` has only `CheckReleases`; `SubscriptionService` has only the 4 subscription methods.
- **DIP** — All dependencies are injected via constructors; no package-level singletons; upper layers depend on interfaces, not concrete types.
  **GRASP:**
- **Information Expert** — Logic lives where the data is: URL building in `urlbuilder`, validation in `validator`, SQL in `repository`.
- **Creator** — Services create domain objects (`model.Subscription`) before passing to repository.
- **Controller** — HTTP handlers are thin controllers; they delegate all decisions to the service layer.
- **Low Coupling** — `scanner.Scanner` depends only on `service.ReleaseScanner`; it has no knowledge of DB, GitHub, or SMTP.
- **High Cohesion** — Each package contains only what belongs to its responsibility.
## What Claude should NOT do

- Do not add any comments to the code — the codebase is intentionally comment-free
- Do not add global variables or package-level mutable state
- Do not bypass the interface layer — each layer must depend only on interfaces, never on concrete structs from another layer
- Do not build URLs with `fmt.Sprintf` in services or handlers — use `urlbuilder.Builder`
- Do not add `init()` functions — forbidden by linter (`gochecknoinits`)
- Do not use `err.Error()` string comparison for error handling — use sentinel errors and `errors.Is`
- Do not use `fmt.Print*`, `log.Print*`, or `print` builtins — use `log/slog`
- Do not skip `fmt.Errorf("context: %w", err)` wrapping when propagating errors
- Do not add business logic inside HTTP handlers — handlers do only: decode request → call service → encode response
- Do not create new migration files that modify already-applied ones — always create a new numbered migration
- Do not add a new interface implementation without the compile-time check: `var _ Interface = (*Impl)(nil)`
## Linter rules (golangci-lint via ses-6-ops)

The project uses a shared linter config downloaded at CI time from `vladyslavpavlenko/ses-6-ops`. Always produce code that passes it.

**Forbidden patterns (forbidigo):**
- `fmt.Print*` — use `slog` instead
- `log.Print*` — use `slog` instead
- `print*` builtins — forbidden
- `spew.Dump` — forbidden
  **Key rules to follow:**
- `errcheck` — every error return must be checked, including type assertions
- `gochecknoinits` — no `init()` functions
- `noctx` — never make HTTP requests without a `context.Context`
- `mnd` — no magic numbers; define named constants instead (see timeout constants in `main.go`)
- `gocognit` / `gocyclo` — keep functions simple (complexity limit: 25)
- `revive: function-result-limit` — functions return at most 3 values
- `revive: line-length-limit` — max 110 characters per line
- `revive: error-strings` — error strings must not be capitalized and must not end with punctuation
- `revive: use-errors-new` — prefer `errors.New` over `fmt.Errorf` when there is no formatting
- `govet: shadow` — avoid variable shadowing (except `err` which is excluded by config)
- `unused` / `unparam` — no unused code or parameters
- `testpackage` — tests must be in `_test` package (e.g. `package service_test`, not `package service`)
- `ireturn` — functions should return interfaces, not concrete types
## Testing conventions

- Mocks are written manually; they implement the same interfaces as production code
- Tests use table-driven style: `[]struct{ name string; ... }` with `t.Run(tc.name, ...)`
- No real DB or SMTP in unit tests — inject mock implementations via constructor
- Test files must use `package service_test` — required by `testpackage` linter
- `internal/service/subscription_service_test.go` and `internal/service/scanner_service_test.go` are the reference examples
## Current state & next steps

- [x] Subscribe / confirm / unsubscribe flow
- [x] Scanner with polling and first-scan baseline
- [x] Double opt-in via email token
- [x] FindOrCreate repo deduplication
- [x] GitHub client with rate-limit and error handling
- [x] URLBuilder extracted to separate package
- [x] slog throughout, no log.Fatal or fmt.Print
- [x] HTTP server with timeouts and run() pattern
- [x] Unit tests for subscription service and scanner service
- [x] CI via ses-6-ops shared workflow
- [x] DIP fix — URLBuilder interface, NewGitHubRepository returns interface
- [x] Split internal/model — Subscription → internal/subscription
- [x] Split internal/service → internal/subscription + internal/release (Poller)
- [x] Move bootstrap logic to internal/app, simplify main/main.go
- [x] Rename cmd/server → main
- [x] TokenGenerator interface next to SubscriptionService, token.Generator in internal/token
- [x] Interfaces to caller side (private, in http/handlers)
- [x] Split subscription_handler.go into per-endpoint files
- [x] Move model.GitHubRepository → internal/release as release.Repository, delete internal/model