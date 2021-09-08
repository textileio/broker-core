ALTER TABLE bids ADD COLUMN deal_confirmed_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS bids_received_at ON bids(received_at);
