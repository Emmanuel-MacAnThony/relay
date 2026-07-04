ALTER TABLE delivery_attempts
    ADD COLUMN response_body    BYTEA,
    ADD COLUMN response_headers JSONB NOT NULL DEFAULT '{}';
