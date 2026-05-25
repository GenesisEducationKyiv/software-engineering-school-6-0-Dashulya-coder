# Testing

## Prerequisites

- Go 1.24+
- Docker + Docker Compose v2
- Node.js 20+ (E2E only)

---

## Unit tests

Pure Go tests with no external dependencies. Each layer is tested in isolation using hand-written mocks for the GitHub client, SMTP mailer, and database. Covers service logic, handler routing, token generation, URL building, and input validation.

```bash
go test ./internal/... -v
```

No setup required. Runs in seconds.

---

## Integration tests

Tests all four API endpoints (`POST /api/subscribe`, `GET /api/confirm/{token}`, `GET /api/unsubscribe/{token}`, `GET /api/subscriptions`) against a real PostgreSQL database. The GitHub client and mailer are stubbed; only the HTTP→service→repository→SQL path is real.

**One command (starts and stops Postgres automatically):**

```bash
make test-integration
```

This spins up `postgres:15` on port 5433, runs the tests, and tears the container down. The database is ephemeral (tmpfs).

To run against an existing database instead:

```bash
DATABASE_URL="postgres://postgres:postgres@localhost:5433/notifier_test?sslmode=disable" \
INTEGRATION=1 \
go test ./tests/integration/... -v -count=1 -timeout=120s
```

---

## E2E tests

Loads the subscription page in a real browser (WebKit/Safari) and verifies the full frontend flow: form rendering, fetch calls intercepted at the network layer via Playwright, success and error messages, field clearing on success, and button disabled state during an in-flight request. The app runs in Docker against a real database.

**First-time setup (installs browser binaries):**

```bash
npm install
npx playwright install --with-deps webkit
```

**One command (builds and starts the full stack, then tears it down):**

```bash
make test-e2e
```

This builds the app image, starts Postgres (port 5434), Mailhog, and the app (port 8080), waits for the app health check to pass, runs Playwright, and tears everything down.

To run tests against an already-running app:

```bash
E2E_BASE_URL=http://localhost:8080 npx playwright test
```

To run headed (visible browser):

```bash
E2E_BASE_URL=http://localhost:8080 npx playwright test --headed
```

Failed runs produce an HTML report in `playwright-report/`:

```bash
npx playwright show-report
```
