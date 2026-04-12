# CaseTaskNotifier

A backend service that allows users to subscribe to email notifications for new releases of GitHub repositories.
The system tracks repositories, detects new releases via the GitHub API, and sends email notifications to subscribed users.

---

## Overview

Users can:
- Subscribe to a repository (`owner/repo`)
- Confirm their subscription via email
- Unsubscribe at any time using a unique link
- Receive notifications when a new release is published

The system periodically scans GitHub repositories and notifies users about new releases.

---

## Tech Stack

- **Go** (Golang)
- **PostgreSQL**
- **Docker / Docker Compose**
- **GitHub REST API**
- **SMTP** (MailHog for local development)

---

## Architecture

The application follows a layered monolithic architecture:

| Layer | Responsibility |
|---|---|
| HTTP Layer | Request handling and routing |
| Service Layer | Business logic |
| Repository Layer | Database access |
| Scanner | Background job for release detection |
| Mailer | Email delivery abstraction |

---

## API Endpoints

### Subscribe

```
POST /api/subscribe
```

**Request:**
```json
{
  "email": "user@example.com",
  "repo": "owner/repo"
}
```

**Responses:**

| Status | Description |
|---|---|
| `200 OK` | Subscription created, confirmation email sent |
| `400 Bad Request` | Invalid input |
| `404 Not Found` | Repository does not exist on GitHub |
| `409 Conflict` | Already subscribed |

---

### Confirm Subscription

```
GET /api/confirm/{token}
```

**Responses:**

| Status | Description |
|---|---|
| `200 OK` | Subscription confirmed |
| `400 Bad Request` | Invalid token |
| `404 Not Found` | Token not found |

---

### Unsubscribe

```
GET /api/unsubscribe/{token}
```

**Responses:**

| Status | Description |
|---|---|
| `200 OK` | Successfully unsubscribed |
| `400 Bad Request` | Invalid token |
| `404 Not Found` | Token not found |

---

### Get Subscriptions

```
GET /api/subscriptions?email=user@example.com
```

**Response:**
```json
[
  {
    "email": "user@example.com",
    "repo": "cli/cli",
    "confirmed": true,
    "last_seen_tag": "v2.0.0"
  }
]
```

> Returns only active subscriptions.

---

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | HTTP server port |
| `DATABASE_URL` | PostgreSQL connection string |
| `GITHUB_TOKEN` | GitHub API token (optional, increases rate limit from 60 to 5000 req/hour) |
| `SMTP_HOST` | SMTP server host |
| `SMTP_PORT` | SMTP server port |
| `SMTP_USER` | SMTP username |
| `SMTP_PASS` | SMTP password |
| `BASE_URL` | Base URL for generating confirmation and unsubscribe links |
| `SCAN_INTERVAL` | Scanner interval (e.g. `5m`, `10s`) |

---

## Running Locally

### 1. Start dependencies

```bash
docker-compose up -d
```

This will start:
- **PostgreSQL** on port `5432`
- **MailHog** — SMTP on `1025`, Web UI at [http://localhost:8025](http://localhost:8025)

### 2. Set environment variables

```bash
export PORT=8080
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/notifier?sslmode=disable"
export BASE_URL="http://localhost:8080"
export SMTP_HOST=localhost
export SMTP_PORT=1025
export SCAN_INTERVAL=10s
```

### 3. Run the application

```bash
go run cmd/server/main.go
```

---

## Email Testing

Emails are sent to MailHog: [http://localhost:8025](http://localhost:8025)

You can inspect:
- Confirmation emails
- Release notification emails
- Unsubscribe links

---

## Release Scanner

The scanner runs periodically and performs the following steps:

1. Fetch all confirmed and active subscriptions
2. Group subscriptions by repository
3. Call GitHub API for the latest release (1 request per repo, not per subscriber)
4. Compare with stored `last_seen_tag`
5. If a new release is detected:
    - Send email notifications to all subscribers
    - Update `last_seen_tag` in the database

---

## Database

Two main tables:

**`repositories`** — stores tracked repositories and last known release tag

**`subscriptions`** — stores user subscriptions, confirmation status, and tokens

Database migrations are automatically applied on application startup.

---

## Running Tests

```bash
go test ./... -v
```

Test coverage includes:
- Validators (email, repo format)
- Subscription service logic
- Release scanner logic

---

## Docker

**Start all services:**
```bash
docker-compose up -d
```

**Stop all services:**
```bash
docker-compose down
```

---

## Author

**Daria Ukshe**