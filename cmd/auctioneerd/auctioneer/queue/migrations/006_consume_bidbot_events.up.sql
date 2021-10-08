CREATE TABLE IF NOT EXISTS storage_providers (
    id text PRIMARY KEY,
    bidbot_version text NOT NULL,
    deal_start_window bigint NOT NULL,
    cid_gravity_configured boolean NOT NULL,
    cid_gravity_strict boolean NOT NULL,
    first_seen_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_unhealthy_at timestamp,
    last_unhealthy_error text
    );

CREATE TYPE bid_event_type AS ENUM ('start_fetching', 'error_fetching', 'start_importing', 'end_importing', 'finalized', 'errored');
CREATE TABLE IF NOT EXISTS bid_events (
    bid_id text NOT NULL,
    event_type bid_event_type NOT NULL,
    attempts int,
    error text,
    happened_at timestamp NOT NULL,
    received_at timestamp NOT NULL,
    CONSTRAINT fk_bid_id FOREIGN KEY(bid_id) REFERENCES bids(id)
    );
CREATE INDEX IF NOT EXISTS bid_events_bid_id ON bid_events(bid_id);
