CREATE TYPE unprepared_batch_status AS ENUM ('pending','executing','done');

CREATE TABLE IF NOT EXISTS unprepared_batches (
    storage_deal_id TEXT PRIMARY KEY,
    status unprepared_batch_status NOT NULL DEFAULT 'pending',
    data_cid TEXT NOT NULL,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX unprepared_batches_status_idx ON unprepared_batches (status, ready_at);

