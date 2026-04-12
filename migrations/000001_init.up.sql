CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    full_name TEXT UNIQUE NOT NULL,
    owner TEXT NOT NULL,
    name TEXT NOT NULL,
    last_seen_tag TEXT,
    last_release_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    repository_id INT NOT NULL REFERENCES repositories(id),
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    confirm_token TEXT UNIQUE NOT NULL,
    unsubscribe_token TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_email_repo
    ON subscriptions(email, repository_id);

CREATE INDEX idx_subscriptions_repository_id
    ON subscriptions(repository_id);