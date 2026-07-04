ALTER TABLE delivery_attempts
    DROP COLUMN IF EXISTS response_body,
    DROP COLUMN IF EXISTS response_headers;
