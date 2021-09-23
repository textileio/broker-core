--------------------------------------------------------------
-- broker.deals
--------------------------------------------------------------
 
-- add a primary key
ALTER TABLE broker.deals
ADD CONSTRAINT deals_pkey
PRIMARY KEY (storage_provider_id, auction_id);
