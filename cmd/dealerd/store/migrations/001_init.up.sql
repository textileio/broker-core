CREATE TABLE IF NOT EXISTS auction_data (
    id text PRIMARY KEY,
    storage_deal_id text NOT NULL,
    payload_cid text NOT NULL,
    piece_cid text NOT NULL,
    piece_size bigint NOT NULL,
    duration bigint NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TYPE status AS ENUM (
  'deal-making',
  'confirmation',
  'report-finalized'
);

CREATE TABLE IF NOT EXISTS auction_deals (
    id text PRIMARY KEY,
    auction_data_id text NOT NULL,
    miner_id text NOT NULL,
    price_per_gib_per_epoch bigint NOT NULL,
    start_epoch bigint NOT NULL,
    verified boolean NOT NULL,
    fast_retrieval boolean NOT NULL,
    auction_id text NOT NULL,
    bid_id text NOT NULL,
    status status NOT NULL,
    executing boolean NOT NULL,
    error_cause text NOT NULL,
    retries smallint NOT NULL,
    proposal_cid text NOT NULL,
    deal_id bigint NOT NULL,
    deal_expiration bigint NOT NULL,
    deal_market_status bigint NOT NULL,
    ready_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_auction_data_id FOREIGN KEY(auction_data_id) REFERENCES auction_data(id)
    );

CREATE INDEX IF NOT EXISTS auction_deals_miner_id ON auction_deals (miner_id);
