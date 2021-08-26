CREATE TABLE IF NOT EXISTS auctions (
    id text PRIMARY KEY,
    batch_id text NOT NULL,
    deal_size bigint NOT NULL,
    deal_duration bigint NOT NULL,
    deal_replication int NOT NULL,
    deal_verified boolean NOT NULL,
    fil_epoch_deadline bigint NOT NULL,
    excluded_storage_providers text[] NOT NULL DEFAULT '{}',
    payload_cid text NOT NULL,
    car_url text NOT NULL DEFAULT '',
    car_ipfs_cid text NOT NULL DEFAULT '',
    car_ipfs_addrs text[] NOT NULL DEFAULT '{}',
    status text NOT NULL,
    error_cause text NOT NULL DEFAULT '',
    duration bigint NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
CREATE INDEX IF NOT EXISTS auctions_fil_epoch_deadline ON auctions(fil_epoch_deadline);
CREATE INDEX IF NOT EXISTS auctions_status ON auctions(status);


CREATE TABLE IF NOT EXISTS bids (
    id text PRIMARY KEY,
    auction_id TEXT NOT NULL,
    wallet_addr_sig bytea NOT NULL,
    storage_provider_id TEXT NOT NULL,
    bidder_id TEXT NOT NULL,
    ask_price BIGINT NOT NULL,
    verified_ask_price BIGINT NOT NULL,
    start_epoch BIGINT NOT NULL,
    fast_retrieval BOOLEAN NOT NULL DEFAULT FALSE,
    received_at TIMESTAMP NOT NULL,
    won_at TIMESTAMP,
    proposal_cid text,
    proposal_cid_delivered_at TIMESTAMP,
    proposal_cid_delivery_error text,
    CONSTRAINT fk_auction_id FOREIGN KEY(auction_id) REFERENCES auctions(id)
    );
CREATE INDEX IF NOT EXISTS bids_auction_id ON bids(auction_id);
