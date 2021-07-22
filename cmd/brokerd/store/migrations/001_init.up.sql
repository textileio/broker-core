CREATE TABLE IF NOT EXISTS storage_deals (
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
    auction_retries int NOT NULL DEFAULT 0,
    fil_epoch_deadline bigint NOT NULL,
    error text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS broker_requests (
    id text PRIMARY KEY,
    data_cid text NOT NULL,
    storage_deal_id text,
    status smallint NOT NULL,
    rebatch_count int NOT NULL DEFAULT 0,
    error_cause text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_storage_deal_id FOREIGN KEY(storage_deal_id) REFERENCES storage_deals(id)
    );

CREATE TABLE IF NOT EXISTS miner_deals (
    storage_deal_id text NOT NULL,
    auction_id text NOT NULL,
    bid_id text NOT NULL,
    miner_id text NOT NULL,
    deal_id bigint NOT NULL,
    deal_expiration bigint NOT NULL,
    error_cause text NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_storage_deal_id FOREIGN KEY(storage_deal_id) REFERENCES storage_deals(id)
    );
CREATE INDEX IF NOT EXISTS miner_deals_miner_id ON miner_deals (miner_id);

CREATE TABLE IF NOT EXISTS unpin_jobs (
    id text PRIMARY KEY,
    executing boolean DEFAULT FALSE,
    cid text NOT NULL,
    type smallint NOT NULL,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
