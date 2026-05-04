CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    original TEXT NOT NULL,
    short TEXT NOT NULL UNIQUE,
    title TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    clicks BIGINT NOT NULL DEFAULT 0,
    creator_reference TEXT,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_urls_short ON urls(short);
CREATE INDEX IF NOT EXISTS idx_urls_original ON urls(original);

CREATE TABLE IF NOT EXISTS clicks (
    id SERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    url_short TEXT NOT NULL,
    ip TEXT NOT NULL,
    location TEXT,
    browser TEXT,
    device TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
CREATE INDEX IF NOT EXISTS idx_clicks_url_short ON clicks(url_short);

CREATE TABLE IF NOT EXISTS url_history (
    id SERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    url_short TEXT NOT NULL,
    action TEXT NOT NULL,
    old_value JSONB,
    new_value JSONB,
    modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_url_history_url_id ON url_history(url_id);
CREATE INDEX IF NOT EXISTS idx_url_history_url_short ON url_history(url_short);