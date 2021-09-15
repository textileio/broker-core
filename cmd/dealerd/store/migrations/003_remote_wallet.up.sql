CREATE TABLE IF NOT EXISTS remote_wallet (
    auction_data_id text PRIMARY KEY,
    peer_id text NOT NULL,
    auth_token text NOT NULL,
    wallet_addr text NOT NULL,
    multiaddrs text[] NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_remote_wallet_auction_data_id FOREIGN KEY(auction_data_id) REFERENCES auction_data(id)
    );

