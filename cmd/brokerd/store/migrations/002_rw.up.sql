CREATE TABLE IF NOT EXISTS batch_remote_wallet (
    batch_id text PRIMARY KEY,
    peer_id text NOT NULL,
    auth_token text NOT NULL,
    wallet_addr text NOT NULL,
    multiaddrs text[] NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_batch_remote_wallet_batch_id FOREIGN KEY(batch_id) REFERENCES batches(id)
    );

