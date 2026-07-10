DROP INDEX idx_subscriptions_saga_id;

ALTER TABLE subscriptions DROP COLUMN saga_id;
