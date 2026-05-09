# System Design — CaseTaskNotifier

## Overview

CaseTaskNotifier is a Go-based backend service that lets users subscribe to email notifications for new releases of GitHub repositories. The system polls the GitHub API on a configurable interval and sends notifications via SMTP when a new release is detected.

---

## 1. System Requirements

### Functional Requirements

| # | Requirement |
|---|---|
| FR-1 | User can subscribe to a GitHub repository (`owner/repo` format) using their email |
| FR-2 | System validates that the repository exists on GitHub before creating a subscription |
| FR-3 | Subscription is confirmed via a double opt-in link sent to the user's email |
| FR-4 | User can unsubscribe at any time using a unique link included in every notification |
| FR-5 | User can list their active subscriptions by email |
| FR-6 | System periodically polls GitHub API and detects new releases |
| FR-7 | On new release detection, all confirmed subscribers of that repository receive an email notification |
| FR-8 | Each repository is polled once per scan cycle regardless of subscriber count |

### Non-Functional Requirements

| # | Requirement | Target |
|---|---|---|
| NFR-1 | **Availability** | Service recovers from transient GitHub API / SMTP failures without crashing |
| NFR-2 | **Fault Tolerance** | All outbound HTTP calls use `context.WithTimeout`; rate-limit errors are logged and skipped |
| NFR-3 | **Observability** | Structured logging via `slog`; Prometheus metrics exposed at `/metrics` |
| NFR-4 | **Maintainability** | Layered architecture (Handler → Service → Repository); all external dependencies behind interfaces |
| NFR-5 | **Testability** | Business logic testable in isolation via mocked interfaces |
| NFR-6 | **Data Integrity** | Unique index on `(email, repository_id)` prevents duplicate subscriptions; UUID tokens prevent enumeration |
| NFR-7 | **Configurability** | Scan interval, SMTP, DB, and GitHub token configurable via environment variables |

---

## 2. Load Assessment

### Traffic

| Metric | Estimate | Notes |
|---|---|---|
| Daily Active Subscribers | 1,000 | |
| Inbound API requests | ~3,000 / day (~0.04 RPS) | Subscribe, confirm, unsubscribe, list |
| Tracked repositories | 500 unique | |
| Scan interval | 5 min (configurable) | |
| GitHub API calls | 500 repos × 12 scans/hour = **6,000 req/hour** | See rate limit analysis below |
| Peak notification spike | ~1,000 emails | All subscribers notified on a popular repo release |

### GitHub API Rate Limit Analysis

GitHub allows **5,000 authenticated requests/hour** per token.

| Scan Interval | Repos | API calls/hour | Within limit? |
|---|---|---|---|
| 5 min | 500 | 6,000 | **No** — exceeds limit |
| 10 min | 500 | 3,000 | Yes |
| 5 min | 400 | 4,800 | Yes (marginal) |

With the default 5-minute interval the system hits the GitHub rate limit at ~416 tracked repositories. The scanner handles `ErrRateLimited` gracefully by logging and skipping the affected repo until the next cycle. For production use, a longer scan interval or per-repository request spreading would be required.

### Data Storage

| Table | Rows | Row size | Total |
|---|---|---|---|
| `repositories` | 500 | ~250 B | ~125 KB |
| `subscriptions` | 1,000 | ~350 B | ~350 KB |
| Indexes (B-tree) | — | — | ~500 KB |
| **Total DB** | | | **< 1 MB** |

The database footprint is negligible at this scale. PostgreSQL is not a storage bottleneck.

### Bandwidth

| Direction | Volume |
|---|---|
| Inbound (API requests) | < 1 KB/request → negligible |
| GitHub API outbound | 6,000 req/hour × ~2 KB = ~12 MB/hour |
| SMTP outbound (notification spike) | 1,000 emails × ~5 KB = ~5 MB |

---

## 3. Architecture

### Level 1 — System Context

Who interacts with the system and what external systems it depends on.

```mermaid
C4Context
    title System Context — CaseTaskNotifier

    Person(subscriber, "Subscriber", "A user who subscribes to GitHub repository release notifications via email")

    System(ctn, "CaseTaskNotifier", "Tracks GitHub repositories and delivers release notifications to confirmed subscribers")

    System_Ext(github, "GitHub API", "REST API — repository lookup and latest release detection")
    System_Ext(smtp, "SMTP Server", "Email delivery — confirmation and release notification emails")

    Rel(subscriber, ctn, "Subscribe / Confirm / Unsubscribe / List", "HTTP REST")
    Rel(ctn, github, "Check repo existence, fetch latest release", "HTTPS / REST")
    Rel(ctn, smtp, "Send transactional emails", "SMTP")
    Rel(smtp, subscriber, "Deliver emails to inbox")

    UpdateRelStyle(subscriber, ctn, $textColor="black", $lineColor="#0066cc")
    UpdateRelStyle(ctn, github, $textColor="black", $lineColor="#2da44e")
    UpdateRelStyle(ctn, smtp, $textColor="black", $lineColor="#e67300")
```

---

### Level 2 — Containers

Internal containers and their responsibilities.

```mermaid
C4Container
    title Container Diagram — CaseTaskNotifier

    Person(subscriber, "Subscriber")

    System_Boundary(app, "CaseTaskNotifier") {
        Container(api, "HTTP API", "Go / chi v5", "Exposes REST endpoints. Routes requests to handlers, returns JSON responses.")
        Container(sub_svc, "Subscription Service", "Go", "Subscription lifecycle: input validation, GitHub existence check, token generation, DB writes, email triggering.")
        Container(scanner, "Release Scanner", "Go / goroutine", "Background polling loop. Runs on SCAN_INTERVAL. Groups subscriptions by repo, checks GitHub, sends notifications.")
        Container(gh_client, "GitHub Client", "Go / net/http", "Thin HTTP client over GitHub REST API v3. Handles auth headers, rate-limit errors, and response parsing.")
        Container(mailer, "SMTP Mailer", "Go / net/smtp", "Sends confirmation and release notification emails. Abstracts SMTP transport behind an interface.")
        ContainerDb(db, "PostgreSQL", "PostgreSQL 15", "Persists repositories (with last_seen_tag) and subscriptions (with confirmation status and UUID tokens).")
    }

    System_Ext(github, "GitHub API", "api.github.com")
    System_Ext(smtp, "SMTP Server", "e.g. MailHog / SendGrid")

    Rel(subscriber, api, "POST /subscribe, GET /confirm/:token, GET /unsubscribe/:token, GET /subscriptions", "HTTP/JSON")
    Rel(api, sub_svc, "Delegates to service layer")
    Rel(sub_svc, gh_client, "RepositoryExists(owner, name)")
    Rel(sub_svc, db, "FindByFullName, Create, ExistsByEmailAndRepo, ConfirmByToken, DeactivateByToken", "SQL / pgx")
    Rel(sub_svc, mailer, "SendConfirmation(email, link)")
    Rel(scanner, db, "GetAllConfirmedActive, GetByID, UpdateLastSeenTag", "SQL / pgx")
    Rel(scanner, gh_client, "GetLatestRelease(owner, name)")
    Rel(scanner, mailer, "SendNewRelease(email, repo, tag, url, unsubLink)")
    Rel(gh_client, github, "GET /repos/:owner/:repo, GET /repos/:owner/:repo/releases/latest", "HTTPS")
    Rel(mailer, smtp, "SMTP AUTH + MAIL FROM / RCPT TO / DATA")
```

---

### Level 3 — Components

Internal components of the Go application and how they are wired together.

```mermaid
C4Component
    title Component Diagram — CaseTaskNotifier (Go Application)

    Person(subscriber, "Subscriber")
    System_Ext(github, "GitHub API")
    System_Ext(smtp_srv, "SMTP Server")
    ContainerDb(db, "PostgreSQL")

    System_Boundary(app, "Go Application") {

        Boundary(delivery, "HTTP Layer") {
            Component(router, "Router", "chi.Router", "Registers routes: /api/subscribe, /api/confirm/:token, /api/unsubscribe/:token, /api/subscriptions, /metrics")
            Component(handler, "SubscriptionHandler", "net/http", "Decodes JSON requests, maps service errors to HTTP status codes, writes JSON responses")
        }

        Boundary(business, "Service Layer") {
            Component(sub_service, "SubscriptionService", "Go interface", "Orchestrates: validation → GitHub check → DB upsert → token generation → email send")
            Component(scanner_svc, "Scanner", "Go goroutine", "Polling loop: fetch subscriptions → group by repo → check GitHub → compare tag → notify → update DB")
        }

        Boundary(infra, "Infrastructure Layer") {
            Component(gh_client, "GitHubClient", "Go interface", "Wraps GitHub REST API. Sets auth headers, handles 404 / 429 / 403 rate-limit responses")
            Component(smtp_mailer, "SMTPMailer", "Go interface", "Builds and sends MIME emails via net/smtp. Implements Mailer interface")
            Component(sub_repo, "SubscriptionRepository", "Go interface", "SQL queries for subscriptions table: create, find by token, confirm, deactivate, list by email")
            Component(repo_repo, "GitHubRepository", "Go interface", "SQL queries for repositories table: find by name, create, get by ID, update last_seen_tag")
        }
    }

    Rel(subscriber, router, "HTTP requests")
    Rel(router, handler, "Dispatches matched route")
    Rel(handler, sub_service, "Subscribe / Confirm / Unsubscribe / GetByEmail")
    Rel(sub_service, gh_client, "RepositoryExists")
    Rel(sub_service, sub_repo, "Create, ExistsByEmailAndRepo, FindByToken, Confirm, Deactivate, GetByEmail")
    Rel(sub_service, repo_repo, "FindByFullName, Create")
    Rel(sub_service, smtp_mailer, "SendConfirmation")
    Rel(scanner_svc, sub_repo, "GetAllConfirmedActive")
    Rel(scanner_svc, repo_repo, "GetByID, UpdateLastSeenTag")
    Rel(scanner_svc, gh_client, "GetLatestRelease")
    Rel(scanner_svc, smtp_mailer, "SendNewRelease")
    Rel(gh_client, github, "HTTPS REST")
    Rel(smtp_mailer, smtp_srv, "SMTP")
    Rel(sub_repo, db, "SQL")
    Rel(repo_repo, db, "SQL")
```

---

## 4. Sequence Diagrams

### Subscription Flow

Full lifecycle: subscribe → confirm → (optionally) unsubscribe.

```mermaid
sequenceDiagram
    participant U as Subscriber
    participant API as HTTP API
    participant SVC as Subscription Service
    participant GH as GitHub Client
    participant DB as PostgreSQL
    participant M as Mailer
    participant SMTP as SMTP Server

    Note over U, SMTP: Subscribe

    U->>API: POST /api/subscribe {email, repo}
    API->>SVC: Subscribe(email, repo)

    SVC->>SVC: ValidateEmail(email)
    SVC->>SVC: ValidateRepo(repo)

    alt Validation fails
        SVC-->>API: ErrInvalidEmail / ErrInvalidRepo
        API-->>U: 400 Bad Request
    end

    SVC->>GH: RepositoryExists(owner, name)
    GH->>GH: GET /repos/{owner}/{name}

    alt Repository does not exist on GitHub
        GH-->>SVC: false
        SVC-->>API: ErrRepoNotFound
        API-->>U: 404 Not Found
    end

    GH-->>SVC: true

    SVC->>DB: FindByFullName(repo)
    alt Repository not yet in DB
        SVC->>DB: Create(repository)
    end
    DB-->>SVC: repository record

    SVC->>DB: ExistsByEmailAndRepo(email, repoID)
    alt Already subscribed
        SVC-->>API: ErrAlreadySubscribed
        API-->>U: 409 Conflict
    end

    SVC->>SVC: Generate confirmToken + unsubscribeToken (UUID v4)
    SVC->>DB: Create(subscription{confirmed=false, active=true})
    SVC->>M: SendConfirmation(email, confirmLink)
    M->>SMTP: SMTP AUTH + send email
    SMTP-->>U: Confirmation email delivered

    SVC-->>API: nil
    API-->>U: 200 OK

    Note over U, SMTP: Confirm Subscription

    U->>API: GET /api/confirm/{token}
    API->>SVC: Confirm(token)
    SVC->>DB: FindByConfirmToken(token)

    alt Token not found
        DB-->>SVC: nil
        SVC-->>API: error "token not found"
        API-->>U: 404 Not Found
    end

    DB-->>SVC: subscription record
    SVC->>DB: ConfirmByToken(token) — confirmed=true
    SVC-->>API: nil
    API-->>U: 200 OK

    Note over U, SMTP: Unsubscribe

    U->>API: GET /api/unsubscribe/{token}
    API->>SVC: Unsubscribe(token)
    SVC->>DB: FindByUnsubscribeToken(token)

    alt Token not found
        SVC-->>API: error "token not found"
        API-->>U: 404 Not Found
    end

    SVC->>DB: DeactivateByToken(token) — active=false
    SVC-->>API: nil
    API-->>U: 200 OK
```

---

### Release Scan Flow

Background goroutine; fires on every `SCAN_INTERVAL` tick.

```mermaid
sequenceDiagram
    participant T as Ticker (SCAN_INTERVAL)
    participant S as Release Scanner
    participant DB as PostgreSQL
    participant GH as GitHub Client
    participant M as Mailer
    participant SMTP as SMTP Server
    participant U as Subscriber

    T->>S: tick

    S->>DB: GetAllConfirmedActive()
    DB-->>S: []Subscription

    alt No confirmed active subscriptions
        S->>S: log "no subscriptions" and return
    end

    S->>S: groupByRepoID(subscriptions)

    loop For each repository group
        S->>DB: GetByID(repoID)
        DB-->>S: Repository{owner, name, last_seen_tag}

        S->>GH: GetLatestRelease(owner, name)
        GH->>GH: GET /repos/{owner}/{name}/releases/latest

        alt No releases on GitHub
            GH-->>S: ErrNoReleases
            S->>S: log and skip
        end

        alt GitHub rate limit exceeded
            GH-->>S: ErrRateLimited
            S->>S: log warning and skip
        end

        GH-->>S: tagName, releaseURL

        alt last_seen_tag is nil — first scan for this repo
            S->>DB: UpdateLastSeenTag(repoID, tag, releaseURL)
            S->>S: log "baseline tag set" and continue
        end

        alt tag == last_seen_tag — no new release
            S->>S: log "no new release" and continue
        end

        Note over S, U: New release detected — tag != last_seen_tag

        loop For each subscriber of this repo
            S->>M: SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink)
            M->>SMTP: Send email
            SMTP-->>U: Release notification delivered
        end

        S->>DB: UpdateLastSeenTag(repoID, newTag, newURL)
        S->>S: log "new release processed"
    end
```

---

## 5. Database Schema

```mermaid
erDiagram
    repositories {
        SERIAL      id                  PK
        TEXT        full_name           "UNIQUE — owner/repo"
        TEXT        owner
        TEXT        name
        TEXT        last_seen_tag       "NULL until first successful scan"
        TEXT        last_release_url
        TIMESTAMP   created_at
        TIMESTAMP   updated_at
    }

    subscriptions {
        SERIAL      id                  PK
        TEXT        email
        INT         repository_id       FK
        BOOLEAN     confirmed           "false — pending; true — active"
        BOOLEAN     active              "false after unsubscribe"
        TEXT        confirm_token       "UNIQUE UUID v4"
        TEXT        unsubscribe_token   "UNIQUE UUID v4"
        TIMESTAMP   created_at
        TIMESTAMP   updated_at
    }

    repositories ||--o{ subscriptions : "tracked by"
```

**Indexes:**
- `UNIQUE (full_name)` on `repositories` — fast lookup on subscribe
- `UNIQUE (email, repository_id)` on `subscriptions` — prevents duplicate subscriptions
- `INDEX (repository_id)` on `subscriptions` — fast grouping in scanner

---

## 6. Detailed Component Design

### HTTP Layer

**Router** (`internal/http/router/router.go`)

Registers all routes using `chi.Router`. Also mounts the Prometheus `/metrics` handler.

| Method | Path | Handler |
|---|---|---|
| POST | `/api/subscribe` | `SubscriptionHandler.Subscribe` |
| GET | `/api/confirm/{token}` | `SubscriptionHandler.Confirm` |
| GET | `/api/unsubscribe/{token}` | `SubscriptionHandler.Unsubscribe` |
| GET | `/api/subscriptions?email=` | `SubscriptionHandler.GetSubscriptions` |
| GET | `/metrics` | `promhttp.Handler()` |

**SubscriptionHandler** (`internal/http/handlers/subscription_handler.go`)

Responsibilities:
- Decode JSON request bodies
- Extract URL/query parameters
- Map service-layer errors to HTTP status codes (`400`, `404`, `409`, `500`)
- Write JSON responses

---

### Subscription Service

**SubscriptionService** (`internal/service/subscription_service.go`)

Orchestrates the subscription lifecycle. Accepts `SubscriptionRepository`, `GitHubRepository`, `github.Client`, and `Mailer` as constructor dependencies (dependency injection via interfaces).

Key operations:

| Method | Steps |
|---|---|
| `Subscribe` | Validate → GitHub check → DB upsert repo → check duplicate → create subscription → send confirmation email |
| `Confirm` | Find by token → mark `confirmed=true` |
| `Unsubscribe` | Find by token → mark `active=false` |
| `GetSubscriptionsByEmail` | Query active subs → enrich with repo data |

---

### Release Scanner

**Scanner** (`internal/scanner/scanner.go`)

Runs as a background goroutine started at application boot. Fires immediately on start, then on every `SCAN_INTERVAL` tick.

Key design points:
- Fetches **all** confirmed and active subscriptions in one query
- Groups by `repository_id` — ensures **one GitHub API call per repo per cycle**, not one per subscriber
- Sets a baseline `last_seen_tag` on first encounter (no notification sent)
- On new release: notifies all subscribers sequentially, then updates `last_seen_tag`
- Handles `ErrNoReleases` and `ErrRateLimited` gracefully — logs and skips without crashing

---

### GitHub Client

**GitHubClient** (`internal/github/client.go`)

Thin wrapper over `net/http` against `api.github.com`. Implements the `github.Client` interface.

| Method | GitHub endpoint | Purpose |
|---|---|---|
| `RepositoryExists` | `GET /repos/{owner}/{repo}` | Validates repo on subscribe |
| `GetLatestRelease` | `GET /repos/{owner}/{repo}/releases/latest` | Fetches latest tag for scanner |

Error handling:
- `404` → `ErrNotFound` / `ErrNoReleases`
- `403` + `X-RateLimit-Remaining: 0` or `429` → `ErrRateLimited`
- Sets `Authorization: Bearer <token>` and `X-GitHub-Api-Version: 2022-11-28` headers when token is configured

---

### Mailer

**SMTPMailer** (`internal/mailer/smtp_mailer.go`)

Implements the `Mailer` interface. Connects to SMTP server using credentials from environment variables.

| Method | Triggered by | Content |
|---|---|---|
| `SendConfirmation` | `SubscriptionService.Subscribe` | Confirmation link with `confirm_token` |
| `SendNewRelease` | `Scanner` | Release tag, release URL, unsubscribe link |

Decoupled behind a `Mailer` interface — allows swapping SMTP for another provider or a mock in tests.

---

### Repository Layer

**SubscriptionRepository** (`internal/repository/subscription_repository.go`)

SQL operations on the `subscriptions` table. All methods accept `context.Context` for cancellation and timeout propagation.

**GitHubRepository** (`internal/repository/github_repository.go`)

SQL operations on the `repositories` table. Stores `last_seen_tag` and `last_release_url` — the only mutable state updated by the scanner.

---

## 7. Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Release detection | Polling (background goroutine) | No webhook infrastructure required; GitHub webhooks need a publicly accessible endpoint and app registration — too much overhead for this use case |
| Subscription confirmation | Double opt-in via UUID email token | Prevents fake subscriptions, protects third-party emails from being abused |
| Repo deduplication | Single `repositories` row per `owner/repo` | GitHub API is called once per repo per cycle regardless of subscriber count — avoids N×M API calls |
| Architecture | Layered monolith | Scale does not justify microservices; clean layer separation still allows future extraction |
| External dependencies | Behind Go interfaces | Enables unit testing with mocks; decouples business logic from infrastructure |
| Observability | `slog` + Prometheus `/metrics` | Standard library logging (no external SDK); metrics compatible with any Prometheus-based stack |
| Rate limit handling | Skip and log on `ErrRateLimited` | Graceful degradation — one failed repo does not abort the entire scan cycle |
