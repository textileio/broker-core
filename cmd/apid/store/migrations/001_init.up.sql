CREATE TYPE auction_status AS ENUM ('unspecified', 'queued','started','finalized');

CREATE TABLE IF NOT EXISTS auctions (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    payload_cid TEXT NOT NULL,
    deal_size BIGINT NOT NULL,
    deal_duration BIGINT NOT NULL,
    deal_replication INTEGER NOT NULL,
    deal_verified BOOLEAN NOT NULL,
    fil_epoch_deadline BIGINT NOT NULL,
    excluded_miners TEXT[],
    status auction_status NOT NULL DEFAULT 'unspecified',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration INTERVAL NOT NULL,
    attempts INTEGER NOT NULL,
    error_cause TEXT,
    broker_already_notified_closed_auction BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS bids (
    id TEXT PRIMARY KEY,
    auction_id TEXT NOT NULL,
    miner_addr TEXT NOT NULL,
    wallet_addr_sig BYTEA NOT NULL,
    bidder_id TEXT NOT NULL,
    ask_price BIGINT NOT NULL,
    verified_ask_price BIGINT,
    start_epoch BIGINT NOT NULL,
    fast_retrieval BOOLEAN NOT NULL DEFAULT FALSE,
    received_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
);

CREATE TABLE IF NOT EXISTS winning_bids (
    bid_id TEXT PRIMARY KEY,
    auction_id TEXT NOT NULL,
    bidder_id TEXT NOT NULL,
    acknowledged BOOLEAN NOT NULL,
    proposal_cid TEXT NOT NULL,
    proposal_cid_acknowledged BOOLEAN NOT NULL,
    CONSTRAINT fk_bid_id FOREIGN KEY(bid_id) REFERENCES bids(id),
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
);

CREATE TABLE IF NOT EXISTS car_url_sources (
    auction_id TEXT PRIMARY KEY,
    url_string TEXT NOT NULL,
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
);

CREATE TABLE IF NOT EXISTS car_ipfs_sources (
    auction_id TEXT PRIMARY KEY,
    cid TEXT NOT NULL,
    multiaddrs TEXT[],
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
);
