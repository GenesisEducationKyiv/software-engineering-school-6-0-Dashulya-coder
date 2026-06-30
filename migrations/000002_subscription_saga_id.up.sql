ALTER TABLE subscriptions ADD COLUMN saga_id UUID;

CREATE INDEX idx_subscriptions_saga_id ON subscriptions(saga_id);
