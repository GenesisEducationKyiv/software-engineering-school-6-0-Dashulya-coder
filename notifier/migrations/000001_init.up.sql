CREATE TABLE sent_notifications (
    dedup_key CHAR(64) PRIMARY KEY
        CHECK (dedup_key ~ '^[0-9a-f]{64}$'),
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
