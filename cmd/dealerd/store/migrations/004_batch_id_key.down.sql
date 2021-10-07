--------------------------------------------------------------
-- can't remove finalized status enum
--------------------------------------------------------------

--------------------------------------------------------------
-- auction_data
--------------------------------------------------------------

-- create id column
ALTER TABLE auction_data ADD COLUMN id TEXT;

-- set id column
UPDATE auction_data SET id = batch_id;

-- remove old pkey
ALTER TABLE auction_data
DROP CONSTRAINT auction_data_pkey;

-- add new pkey on id
ALTER TABLE auction_data
ADD CONSTRAINT auction_data_pkey
PRIMARY KEY (id);


--------------------------------------------------------------
-- remote_wallet
--------------------------------------------------------------

-- drop foreign key to auction_data
ALTER TABLE remote_wallet
DROP CONSTRAINT fk_remote_wallet_batch_id;

-- drop primary key
ALTER TABLE remote_wallet
DROP CONSTRAINT remote_wallet_pkey;

-- create auction_data_id column
ALTER TABLE remote_wallet ADD COLUMN auction_data_id TEXT;

-- populate auction_data_id
UPDATE remote_wallet
SET auction_data_id = auction_data.id
FROM auction_data
WHERE remote_wallet.batch_id = auction_data.batch_id;

-- add foreign key to auction_data for auction_data_id
ALTER TABLE remote_wallet
ADD CONSTRAINT fk_remote_wallet_auction_data_id FOREIGN KEY (auction_data_id)
REFERENCES auction_data (id) MATCH SIMPLE
ON UPDATE NO ACTION
ON DELETE NO ACTION;

-- add pkey on auction_data_id
ALTER TABLE remote_wallet
ADD CONSTRAINT remote_wallet_pkey
PRIMARY KEY (auction_data_id);

-- drop batch_id column
ALTER TABLE DROP COLUMN batch_id;

--------------------------------------------------------------
-- auction_deals
--------------------------------------------------------------

-- drop pkey
ALTER TABLE auction_deals
DROP CONSTRAINT auction_deals_pkey;

-- drop foreign key to auction_data
ALTER TABLE auction_deals
DROP CONSTRAINT fk_auction_deals_batch_id;

-- add id column
ALTER TABLE auction_deals 
ADD COLUMN id TEXT;

-- set id
UPDATE auction_deals
SET id = CONCAT(storage_provider_id, '-', auction_id);

-- create pkey on id
ALTER TABLE auction_deals
ADD CONSTRAINT auction_deals_pkey
PRIMARY KEY (id);

-- add auction_data_id column
ALTER TABLE auction_deals
ADD COLUMN auction_data_id TEXT;

-- set auction_data_id
UPDATE auction_deals
SET auction_data_id = auction_data.id
FROM auction_data
WHERE auction_deals.batch_id = auction_data.batch_id;

-- add not null to auction_data_id
ALTER TABLE auction_deals
ALTER COLUMN auction_data_id SET NOT NULL;

-- add foreign key to auction_data for auction_data_id
ALTER TABLE auction_deals
ADD CONSTRAINT fk_auction_deals_auction_data_id FOREIGN KEY (auction_data_id)
REFERENCES auction_data (id) MATCH SIMPLE
ON UPDATE NO ACTION
ON DELETE NO ACTION;

-- drop batch_id
ALTER TABLE auction_deals
DROP COLUMN batch_id;