BEGIN;

ALTER TABLE auction_deals 
ADD COLUMN batch_id TEXT;

UPDATE auction_deals 
SET batch_id = auction_data.batch_id
FROM auction_data
WHERE auction_deals.auction_data_id = auction_data.id;

ALTER TABLE auction_deals
ALTER COLUMN batch_id SET NOT NULL;

ALTER TYPE status ADD VALUE 'finalized';

COMMIT;