CREATE TYPE batch_status AS ENUM ('open','ready','done');

CREATE TABLE IF NOT EXISTS batches (
    batch_id TEXT PRIMARY KEY,
    status batch_status NOT NULL default 'open',
    total_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX batches_status_idx ON batches(status, created_at);

CREATE TABLE IF NOT EXISTS storage_requests (
    storage_request_id TEXT PRIMARY KEY,
    data_cid TEXT NOT NULL,
    batch_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_storage_deal_id FOREIGN KEY(storage_deal_id) REFERENCES storage_deals(id)
);


