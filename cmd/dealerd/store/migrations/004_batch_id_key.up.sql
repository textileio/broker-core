BEGIN;

--------------------------------------------------------------
-- remove all foreign keys that reference auction_data.id
--------------------------------------------------------------

-- auction_deals
ALTER TABLE auction_deals
DROP CONSTRAINT fk_auction_data_id;

-- remote_wallet
ALTER TABLE remote_wallet
DROP CONSTRAINT fk_remote_wallet_auction_data_id;


--------------------------------------------------------------
-- auction_data
--------------------------------------------------------------

-- remove old pkey
ALTER TABLE auction_data
DROP CONSTRAINT auction_data_pkey;

-- add new pkey on batch_id
ALTER TABLE auction_data
ADD CONSTRAINT auction_data_pkey
PRIMARY KEY (batch_id);


--------------------------------------------------------------
-- auction_deals
--------------------------------------------------------------
 
-- add batch_id column
ALTER TABLE auction_deals 
ADD COLUMN batch_id TEXT;

-- populate batch_id column
UPDATE auction_deals 
SET batch_id = auction_data.batch_id
FROM auction_data
WHERE auction_deals.auction_data_id = auction_data.id;

-- add not null constraint to batch_id
ALTER TABLE auction_deals
ALTER COLUMN batch_id SET NOT NULL;

-- add foreign key to auction_data for batch_id
ALTER TABLE auction_deals
ADD CONSTRAINT fk_auction_deals_batch_id FOREIGN KEY (batch_id)
REFERENCES auction_data (batch_id) MATCH SIMPLE
ON UPDATE NO ACTION
ON DELETE NO ACTION;

-- remove old useless column
ALTER TABLE auction_deals
DROP COLUMN auction_data_id;

-- remove old pkey
ALTER TABLE auction_deals
DROP CONSTRAINT auction_deals_pkey;

-- create new pkey
ALTER TABLE auction_deals
ADD CONSTRAINT auction_deals_pkey
PRIMARY KEY (storage_provider_id, auction_id);

-- delete old pkey column
ALTER TABLE auction_deals
DROP COLUMN id;


--------------------------------------------------------------
-- remote_wallet
--------------------------------------------------------------

-- add batch_id column
ALTER TABLE remote_wallet 
ADD COLUMN batch_id TEXT;

-- populate batch_id column
UPDATE remote_wallet
SET batch_id = auction_data.batch_id
FROM auction_data
WHERE remote_wallet.auction_data_id = auction_data.id;

-- add not null constraint to batch_id
ALTER TABLE remote_wallet
ALTER COLUMN batch_id SET NOT NULL;

-- add foreign key to auction_data for batch_id
ALTER TABLE remote_wallet
ADD CONSTRAINT fk_remote_wallet_batch_id FOREIGN KEY (batch_id)
REFERENCES auction_data (batch_id) MATCH SIMPLE
ON UPDATE NO ACTION
ON DELETE NO ACTION;

-- remove old pkey
ALTER TABLE remote_wallet
DROP CONSTRAINT remote_wallet_pkey;

-- add new pkey on batch_id
ALTER TABLE remote_wallet
ADD CONSTRAINT remote_wallet_pkey
PRIMARY KEY (batch_id);

-- remove old useless column
ALTER TABLE remote_wallet
DROP COLUMN auction_data_id;


--------------------------------------------------------------
-- auction_data
--------------------------------------------------------------
 
-- remove old useless column
ALTER TABLE auction_data
DROP COLUMN id;


--------------------------------------------------------------
-- add a new status
--------------------------------------------------------------
ALTER TYPE status ADD VALUE 'finalized';

COMMIT;