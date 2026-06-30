CREATE TABLE confirmation_deliveries (
    saga_id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    confirm_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'SENT', 'CANCELED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
