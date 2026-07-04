CREATE TABLE requests (
    id          TEXT        PRIMARY KEY,
    slug        TEXT        NOT NULL,
    method      TEXT        NOT NULL,
    path        TEXT        NOT NULL,
    headers     JSONB       NOT NULL DEFAULT '{}',
    query_params JSONB      NOT NULL DEFAULT '{}',
    body        BYTEA,
    source_ip   TEXT        NOT NULL DEFAULT '',
    received_at TIMESTAMPTZ NOT NULL,
    body_size   INT         NOT NULL DEFAULT 0
);

CREATE INDEX idx_requests_slug ON requests(slug);

CREATE TABLE delivery_attempts (
    id           TEXT        PRIMARY KEY,
    request_id   TEXT        NOT NULL REFERENCES requests(id),
    delivered    BOOLEAN     NOT NULL DEFAULT false,
    status_code  INT         NOT NULL DEFAULT 0,
    error        TEXT        NOT NULL DEFAULT '',
    is_replay    BOOLEAN     NOT NULL DEFAULT false,
    attempted_at TIMESTAMPTZ NOT NULL
);
