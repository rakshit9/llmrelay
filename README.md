# LLM Relay

> One endpoint. Every LLM. With caching, failover, auth, and cost controls built in.

LLM Relay is an **OpenAI-compatible gateway** that sits between your apps and the major model providers. Point your existing OpenAI client at Relay and get multi-provider routing, automatic failover, two-tier caching, per-key budgets, and Prometheus metrics — without changing your application code.

The core proxy is written in **Go** for low-latency, high-concurrency request handling; an admin **API + dashboard** manage projects, keys, and analytics.

## Features

- **Drop-in OpenAI compatibility** — `POST /v1/chat/completions`, including streaming (SSE).
- **Multi-provider routing** — OpenAI, Anthropic, Google, and Groq behind one interface, selected by named routes (`default`, `fast`, `cheap`).
- **Automatic failover** — retries with exponential backoff across a provider chain; non-retryable errors fail fast.
- **Two-tier caching**
  - *Exact* cache in Redis for identical requests.
  - *Semantic* cache in Postgres + **pgvector**: embeds the request and returns a cached response when cosine similarity ≥ 0.92.
- **API key management** — hashed keys, per-key rate limits (RPM) and USD budgets, expiry, and project grouping.
- **Cost tracking** — a model catalog with per-1K input/output pricing drives spend accounting.
- **Observability** — structured JSON logs, request IDs, Prometheus `/metrics`, and a Grafana dashboard (cache hit rate, latency, per-model request counts).
- **Secrets handled properly** — client keys stored as hashes, upstream provider keys encrypted at rest.

## Architecture

```
   your app (OpenAI SDK)
          │  POST /v1/chat/completions
          ▼
   ┌──────────────────────────────────────────────┐
   │  Go proxy                                       │
   │   auth → exact cache (Redis) → semantic cache   │
   │   (pgvector) → router (failover + backoff)      │
   │                       │                         │
   │     ┌─────────┬───────┼────────┬─────────┐      │
   │   OpenAI   Anthropic Google   Groq        │     │
   └──────────────────────────────────────────────┘
          │ metrics (/metrics)
          ▼
   Prometheus ──► Grafana

   Admin API (FastAPI)  ──►  Admin UI (Next.js)
   projects · api keys · analytics
```

| Component | Stack | Role |
|-----------|-------|------|
| `proxy/` | Go, pgx, Redis | The hot path: auth, cache, routing, streaming, metrics |
| `admin-api/` | FastAPI, SQLAlchemy | Manage projects, API keys, and analytics |
| `admin-ui/` | Next.js, TypeScript | Operator dashboard |
| `migrations/` | SQL (pgvector) | Schema: users, projects, api_keys, provider_keys, models, routes, cache_vectors |
| `deploy/` | Prometheus / Grafana | Monitoring config |

## Quickstart

```bash
# 1. Start Postgres (pgvector) + Redis + Prometheus + Grafana
make up

# 2. Apply the schema
make migrate

# 3. Run the Go proxy
export GATEWAY_API_KEY=sk-relay-dev
export OPENAI_API_KEY=...        # plus ANTHROPIC/GOOGLE/GROQ as desired
make proxy && ./bin/proxy
```

### Call it like OpenAI

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-relay-dev" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "messages": [{"role":"user","content":"Hello"}]}'
```

Responses include an `X-Cache` header (`exact` / `semantic` / miss) so you can see caching at work. Set `"stream": true` for SSE streaming.

## Endpoints (proxy)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/chat/completions` | Chat completions (auth required; streaming supported) |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/health` | Health check |

## Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `GATEWAY_API_KEY` | — (required) | Client-facing gateway key |
| `OPENAI_API_KEY` | — (required) | Upstream provider key |
| `ANTHROPIC_API_KEY` / `GOOGLE_API_KEY` / `GROQ_API_KEY` | — | Enable additional providers |
| `REDIS_ADDR` | `localhost:6379` | Exact-cache store |
| `DATABASE_URL` | local Postgres | Semantic cache + admin data |
| `PORT` | `8080` | Proxy listen port |

## Make targets

```bash
make up       # start infra (Postgres + Redis + Prometheus + Grafana)
make migrate  # run SQL migrations
make proxy    # build the Go binary
make test     # go test ./...
make lint     # go vet ./...
make down     # stop everything
```

## Tech stack

`Go` · `FastAPI` · `Next.js` / `TypeScript` · `PostgreSQL` + `pgvector` · `Redis` · `Prometheus` · `Grafana` · `Docker Compose`
