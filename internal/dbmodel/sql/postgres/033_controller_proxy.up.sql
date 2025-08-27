-- Add controller_proxies info table.

CREATE TABLE IF NOT EXISTS controller_proxies (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    controller_id INTEGER NOT NULL REFERENCES controllers (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    config JSONB NOT NULL,
    UNIQUE (controller_id)
);

