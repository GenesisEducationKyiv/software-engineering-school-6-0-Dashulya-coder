ALTER TABLE subscriptions ADD COLUMN saga_id UUID;

CREATE UNIQUE INDEX idx_subscriptions_saga_id ON subscriptions(saga_id);
