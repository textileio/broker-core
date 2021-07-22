CREATE TYPE batch_status AS ENUM ('open','ready','executing','done');

CREATE TABLE IF NOT EXISTS batches (
    batch_id TEXT PRIMARY KEY,
    status batch_status NOT NULL default 'open',
    total_size BIGINT NOT NULL DEFAULT 0,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS batches_status_ready_at_idx ON batches(status, ready_at);

CREATE TABLE IF NOT EXISTS storage_requests (
    operation_id TEXT NOT NULL,
    storage_request_id TEXT NOT NULL,
    data_cid TEXT NOT NULL,
    batch_id TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY(operation_id, storage_request_id),
    CONSTRAINT fk_batch_id FOREIGN KEY(batch_id) REFERENCES batches(batch_id)
);


