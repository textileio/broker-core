ALTER TABLE deals
ADD CONSTRAINT deals_pkey
PRIMARY KEY (storage_provider_id, auction_id);
