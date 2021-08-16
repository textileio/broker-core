CREATE TYPE auction_status AS ENUM ('unspecified', 'queued','started','finalized');

CREATE TABLE IF NOT EXISTS auctions (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    deal_verified BOOLEAN NOT NULL,
    excluded_storage_providers TEXT[],
    status auction_status NOT NULL,
    started_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    closed_at TIMESTAMP,
    duration BIGINT NOT NULL,
    error_cause TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bids (
    auction_id TEXT NOT NULL,
    storage_provider_id TEXT NOT NULL,
    bidder_id TEXT NOT NULL,
    ask_price BIGINT NOT NULL,
    verified_ask_price BIGINT NOT NULL,
    start_epoch BIGINT NOT NULL,
    fast_retrieval BOOLEAN NOT NULL DEFAULT FALSE,
    received_at TIMESTAMP NOT NULL,
    won_at TIMESTAMP,
    acknowledged_at TIMESTAMP,
    proposal_cid_delivered_at TIMESTAMP,
    proposal_cid TEXT,
    CONSTRAINT pk_auction_id_bidder_id PRIMARY KEY(auction_id, bidder_id),
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
);
