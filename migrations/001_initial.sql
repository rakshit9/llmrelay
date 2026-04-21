-- Enable pgvector extension for semantic cache
CREATE EXTENSION IF NOT EXISTS vector;

-- Operator accounts for admin UI login
CREATE TABLE users (
    id          BIGSERIAL PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Logical groupings of API keys (e.g. "my-app", "staging")
CREATE TABLE projects (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API keys issued to clients
CREATE TABLE api_keys (
    id              BIGSERIAL PRIMARY KEY,
    project_id      BIGINT NOT NULL REFERENCES projects(id),
    key_hash        TEXT NOT NULL UNIQUE,   -- we store hash, never plaintext
    name            TEXT NOT NULL,
    rate_limit_rpm  INT NOT NULL DEFAULT 60,
    budget_usd      NUMERIC(10,4),          -- NULL = unlimited
    spent_usd       NUMERIC(10,4) NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ
);

-- Upstream provider API keys (encrypted at rest)
CREATE TABLE provider_keys (
    id          BIGSERIAL PRIMARY KEY,
    provider    TEXT NOT NULL,              -- openai, anthropic, google, groq
    key_enc     TEXT NOT NULL,             -- AES-256 encrypted
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Model catalog: what models exist and what they cost
CREATE TABLE models (
    id              BIGSERIAL PRIMARY KEY,
    provider        TEXT NOT NULL,
    model_id        TEXT NOT NULL,          -- e.g. gpt-4o, claude-3-5-sonnet
    input_cost_per_1k  NUMERIC(10,6) NOT NULL,
    output_cost_per_1k NUMERIC(10,6) NOT NULL,
    context_window  INT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(provider, model_id)
);

-- Named routing rules: alias → model selection
CREATE TABLE routes (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,       -- e.g. "default", "fast", "cheap"
    provider    TEXT NOT NULL,
    model_id    TEXT NOT NULL,
    priority    INT NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE
);

-- Metadata for every proxied request
CREATE TABLE requests (
    id              BIGSERIAL PRIMARY KEY,
    api_key_id      BIGINT REFERENCES api_keys(id),
    provider        TEXT,
    model_id        TEXT,
    prompt_tokens   INT,
    completion_tokens INT,
    cost_usd        NUMERIC(10,6),
    latency_ms      INT,
    cache_hit       BOOLEAN NOT NULL DEFAULT FALSE,
    cache_type      TEXT,                   -- 'exact', 'semantic', or NULL
    status_code     INT,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Exact cache: hash of request → stored response
CREATE TABLE cache_entries (
    id          BIGSERIAL PRIMARY KEY,
    key_hash    TEXT NOT NULL UNIQUE,
    response    TEXT NOT NULL,
    model_id    TEXT NOT NULL,
    hit_count   INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

-- Semantic cache: embedding vector → stored response
CREATE TABLE cache_vectors (
    id              BIGSERIAL PRIMARY KEY,
    embedding       vector(384),            -- all-MiniLM-L6-v2 produces 384 dims
    request_hash    TEXT NOT NULL,
    response        TEXT NOT NULL,
    model_id        TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);
CREATE INDEX ON cache_vectors USING ivfflat (embedding vector_cosine_ops);

-- Append-only ledger of spend per key
CREATE TABLE budget_events (
    id          BIGSERIAL PRIMARY KEY,
    api_key_id  BIGINT NOT NULL REFERENCES api_keys(id),
    request_id  BIGINT REFERENCES requests(id),
    amount_usd  NUMERIC(10,6) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
