ALTER TABLE bids DROP COLUMN deal_confirmed_at;
DROP INDEX IF EXISTS bids_received_at;
