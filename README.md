# Relay

A self-hosted webhook relay built from scratch. Relay captures HTTP requests sent to a slug endpoint, delivers them in real time to a local server via a lightweight CLI, and lets you inspect or replay any request from a live dashboard.

<br/>

<p align="center">
  <img src="screenshots/01-dashboard.png" alt="Relay dashboard – request selected with body and delivery" width="100%"/>
</p>

---

## What it does

- **Captures any HTTP request** — POST, PUT, DELETE, GET — sent to `/hook/{slug}` and stores it with headers, query params, and body
- **Delivers in real time** over WebSocket to the relay CLI running on your machine
- **Forwards to your local server** — the CLI proxies the captured request to a target URL (e.g. `http://localhost:9001`)
- **Shows delivery status** — 200 OK, 500, timeout — and the full local server response body + headers
- **Replay any request** with one click — resends the original to your local server without needing to re-trigger from Stripe/GitHub/Slack
- **Live dashboard** — requests appear instantly via WebSocket; no polling, no refresh
- **Pipeline animation** — animated active-stage border shows which step (Captured → CLI → Forwarded to Local) is in progress during replay

---

## Screenshots

### Dashboard — request selected

<p align="center">
  <img src="screenshots/01-dashboard.png" alt="Relay – request list with body and delivery section" width="100%"/>
</p>

The left panel lists every captured request with method badge and status code. The right panel shows the request body immediately (so you know exactly what fired), then the delivery result and your local server's response below.

### Delivery + local server response

<p align="center">
  <img src="screenshots/02-delivery.png" alt="Relay – local server response with headers" width="100%"/>
</p>

After delivery the panel shows the local server response body (up to 64 KB), response headers, and the incoming request headers — everything you need to debug what your handler received.

---

## Architecture

```
  Stripe / GitHub / Slack
           │  POST /hook/{slug}
           ▼
  ┌────────────────────┐       WebSocket /ws/{slug}      ┌───────────────────┐
  │   Relay Server     │ ──────────────────────────────► │   relay-cli       │
  │   (Go / HTTP)      │ ◄────── delivery attempt ─────── │   (local machine) │
  └────────────────────┘                                  └────────┬──────────┘
           │                                                       │ HTTP forward
           │ PostgreSQL                                            ▼
  ┌────────────────────┐                               ┌───────────────────────┐
  │   requests         │                               │  Your local server    │
  │   delivery_attempts│                               │  http://localhost:9001│
  └────────────────────┘                               └───────────────────────┘
```

**Relay Server** — an HTTP mux that:
1. Accepts incoming webhooks at `/hook/{slug}` and saves them (headers, body, source IP)
2. Broadcasts a `new_request` WS event to all connected CLI sessions for that slug
3. Receives the delivery attempt result from the CLI and saves it to `delivery_attempts`
4. Exposes a REST API for the dashboard to read requests and attempts

**relay-cli** — a single binary that:
1. Connects to the relay server via WebSocket for the given slug
2. On each `new_request` event, issues the original HTTP request against the target URL
3. Captures the response (status, body up to 64 KB, headers) and sends a `delivery_attempt` event back to the server

**Dashboard** — a Next.js App Router frontend with a live WebSocket feed. Selecting a request shows body + delivery + local response. The replay button triggers a new delivery attempt with a step-by-step animated pipeline.

---

## Tech stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.25 | Goroutines for WebSocket fan-out; lightweight binary for the CLI |
| Database | PostgreSQL 16 | `DISTINCT ON` for latest delivery attempt per request; JSONB for headers |
| SQL | sqlc | Type-safe Go from SQL, no ORM |
| Migrations | golang-migrate | Plain SQL files, version-tracked |
| HTTP router | stdlib `net/http` | Method-pattern routing added in Go 1.22 — no dependency needed |
| WebSocket | `golang.org/x/net/websocket` | Standard library WebSocket, no extra framework |
| Frontend | Next.js 15 (App Router) | Server components for initial fetch; client component for WS live feed |
| UI components | shadcn/ui | Unstyled primitives, full control |
| Styling | Tailwind CSS v4 | Dark theme, CSS variables, conic-gradient animation for active pipeline stage |
| Font | Inter | Clean proportional font; JetBrains Mono for code/body blocks only |

---

## Getting started

### Prerequisites

- Go 1.22+
- Docker + Docker Compose (for Postgres)
- Node.js 20+ (for the frontend)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### 1. Start the database

```bash
docker compose up -d db
```

### 2. Run migrations

```bash
migrate -path db/migrations \
        -database "postgres://relay:relay@localhost:5433/relay?sslmode=disable" \
        up
```

### 3. Start the relay server

```bash
DATABASE_URL="postgres://relay:relay@localhost:5433/relay?sslmode=disable" \
go run ./cmd/relay
```

Server runs on port `8080` by default.

### 4. Create a slug endpoint (one-time setup)

```bash
curl -X POST http://localhost:8080/endpoint \
  -H "Content-Type: application/json" \
  -d '{"slug": "my-webhook"}'
```

Your capture URL is now `http://localhost:8080/hook/my-webhook`.

### 5. Start the CLI on your local machine

```bash
# Build the CLI binary
go build -o relay-cli ./cmd/relay-cli

# Connect to the server and forward to your local app
./relay-cli \
  --slug my-webhook \
  --server ws://localhost:8080 \
  --target http://localhost:3000
```

The CLI connects over WebSocket and forwards every captured request to `--target` in real time.

### 6. Start the frontend

```bash
cd web
npm install
npm run dev   # http://localhost:3001
```

Navigate to `http://localhost:3001/my-webhook` to see the live feed.

---

## REST API

| Method | Path | Description |
|---|---|---|
| `POST` | `/endpoint` | Create a new slug endpoint |
| `GET` | `/endpoint` | List all endpoints |
| `*` | `/hook/{slug}` | Capture a webhook (any method) |
| `GET` | `/request/{slug}` | List captured requests for a slug |
| `GET` | `/request/{slug}/{id}` | Get a single request |
| `GET` | `/request/{slug}/attempts` | Latest delivery attempt per request |
| `POST` | `/request/{slug}/{id}/replay` | Replay a request |
| `GET` | `/ws/{slug}` | WebSocket — CLI connects here |

### WebSocket protocol

The server sends JSON events to the CLI:

```json
{ "type": "new_request", "data": { "id": "...", "method": "POST", "path": "/hook/...", "headers": {}, "body": "<base64>", ... } }
```

The CLI sends delivery results back:

```json
{ "type": "delivery_attempt", "data": { "request_id": "...", "delivered": true, "status_code": 200, "response_body": "...", "response_headers": {} } }
```

---

## Project structure

```
03-relay/
├── cmd/
│   ├── relay/              # Server main + wiring
│   └── relay-cli/          # CLI main, WS client, HTTP executor
├── db/
│   ├── migrations/         # golang-migrate SQL files (001–003)
│   └── queries/            # sqlc input SQL
├── internal/
│   ├── api/                # HTTP router, request handler, endpoint handler, WS handler
│   ├── config/             # env-based config
│   ├── db/                 # sqlc-generated code
│   ├── endpoint/           # endpoint domain + infra + usecases
│   └── request/
│       ├── domain/         # Request, DeliveryAttempt (with JSON tags)
│       ├── infra/          # sqlc repository impl
│       └── usecases/       # create, get, list, replay
├── pkg/
│   ├── logger/             # slog-based structured logger
│   └── result/             # Result[T] for explicit error handling
├── screenshots/            # UI screenshots for documentation
└── web/                    # Next.js 15 frontend
    ├── app/
    │   ├── page.tsx              # Endpoint list (server component)
    │   ├── actions.ts            # Server action: create endpoint
    │   └── [slug]/
    │       ├── page.tsx          # Slug page shell
    │       └── live-feed.tsx     # WS client, request list, detail panel, replay pipeline
    ├── components/
    │   ├── delivery-badge.tsx    # Delivered / Failed / Pending badge
    │   ├── method-badge.tsx      # POST / GET / DELETE badge
    │   └── copy-button.tsx       # One-click copy
    └── lib/
        └── api.ts                # Typed fetch helpers + domain types
```

---

## Design notes

**Why a CLI instead of a server-side tunnel?**
The CLI approach means zero open ports on your machine and zero NAT traversal. The relay server holds the WebSocket; the CLI opens it outbound. Your local server never sees internet traffic directly — all it sees is a local HTTP request from the CLI.

**Why replay matters**
When developing against Stripe or GitHub webhooks you can't force a re-delivery on demand — Stripe has rate limits, GitHub has retry windows, and neither lets you replay with a modified payload. Relay stores every request and lets you replay instantly from the dashboard, including after you've fixed a bug in your handler.

**Why `DISTINCT ON` for delivery attempts?**
Each request can have multiple delivery attempts (original + replays). The dashboard shows one status badge per request. `DISTINCT ON (request_id) ORDER BY attempted_at DESC` returns the latest attempt per request in a single query with no application-level grouping.

**Why capture response body in the CLI?**
The server never talks to your local service — only the CLI does. The server has no way to see the response unless the CLI captures and sends it back as part of the `delivery_attempt` WS event. The 64 KB cap keeps the WS messages practical while covering virtually all API response payloads.

---

## What's not built yet

- **Authentication** — slugs are public; add a shared secret or token header before exposing the server publicly
- **Multi-CLI fan-out** — currently the WS hub broadcasts to all connected sessions for a slug; only one CLI should be connected at a time
- **Payload filtering / transformation** — strip headers, rewrite body, or route to different targets based on request content
- **Retry logic in the CLI** — if the local target is down the delivery attempt is recorded as failed; the dashboard replay button covers the manual retry case

