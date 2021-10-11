BEGIN;

ALTER TABLE market_deal_status ADD COLUMN type TEXT;
UPDATE market_deal_status SET type = REPLACE(UPPER(name), ' ', '_');
ALTER TABLE market_deal_status DROP CONSTRAINT market_deal_status_pkey;
ALTER TABLE market_deal_status ADD PRIMARY KEY (type);
ALTER TABLE market_deal_status ADD CONSTRAINT id_unique UNIQUE (id);
ALTER TABLE market_deal_status RENAME COLUMN name TO description;

ALTER TABLE auction_deals ADD COLUMN market_deal_status TEXT;
UPDATE auction_deals SET market_deal_status = (
	SELECT type FROM market_deal_status 
	WHERE auction_deals.deal_market_status = market_deal_status.id
);
ALTER TABLE auction_deals ALTER COLUMN market_deal_status SET NOT NULL;
ALTER TABLE auction_deals
  ADD CONSTRAINT fk_market_deal_status
  FOREIGN KEY (market_deal_status) 
  REFERENCES market_deal_status (type);
ALTER TABLE auction_deals DROP COLUMN deal_market_status;

COMMIT;