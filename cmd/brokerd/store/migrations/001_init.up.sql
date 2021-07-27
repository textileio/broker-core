CREATE TABLE IF NOT EXISTS batches (
    id text PRIMARY KEY,
    status smallint NOT NULL,
    rep_factor int NOT NULL,
    deal_duration int NOT NULL,
    payload_cid text NOT NULL,
    piece_cid text NOT NULL,
    piece_size bigint NOT NULL,
    car_url text NOT NULL DEFAULT '',
    car_ipfs_cid text NOT NULL DEFAULT '',
    car_ipfs_addrs text NOT NULL DEFAULT '',
    disallow_rebatching boolean NOT NULL DEFAULT FALSE,
    fil_epoch_deadline bigint NOT NULL,
    error text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS storage_requests (
    id text PRIMARY KEY,
    data_cid text NOT NULL,
    batch_id text,
    status smallint NOT NULL,
    rebatch_count int NOT NULL DEFAULT 0,
    error_cause text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_batch_id FOREIGN KEY(batch_id) REFERENCES batches(id)
    );

CREATE TABLE IF NOT EXISTS deals (
    batch_id text NOT NULL,
    auction_id text NOT NULL,
    bid_id text NOT NULL,
    storage_provider_id text NOT NULL,
    deal_id bigint NOT NULL,
    deal_expiration bigint NOT NULL,
    error_cause text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_batch_id FOREIGN KEY(batch_id) REFERENCES batches(id)
    );
CREATE INDEX IF NOT EXISTS deals_storage_provider_id ON deals (storage_provider_id);

CREATE TABLE IF NOT EXISTS unpin_jobs (
    id text PRIMARY KEY,
    executing boolean DEFAULT FALSE,
    cid text NOT NULL,
    type smallint NOT NULL,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
