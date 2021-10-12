BEGIN;

ALTER TABLE auction_deals ADD COLUMN deal_market_status BIGINT;
UPDATE auction_deals SET deal_market_status = (
	SELECT id FROM market_deal_status 
	WHERE auction_deals.market_deal_status = market_deal_status.type
);
ALTER TABLE auction_deals ALTER COLUMN deal_market_status SET NOT NULL;
ALTER TABLE auction_deals DROP CONSTRAINT fk_market_deal_status;
ALTER TABLE auction_deals DROP COLUMN market_deal_status;

ALTER TABLE market_deal_status RENAME COLUMN description TO name;
ALTER TABLE market_deal_status DROP CONSTRAINT id_unique;
ALTER TABLE market_deal_status DROP CONSTRAINT market_deal_status_pkey;
ALTER TABLE market_deal_status ADD PRIMARY KEY (id);
ALTER TABLE market_deal_status DROP COLUMN type;

COMMIT;