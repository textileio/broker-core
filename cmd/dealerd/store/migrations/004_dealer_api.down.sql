BEGIN;

ALTER TABLE auction_deals 
DROP COLUMN batch_id;

DELETE FROM auction_data
WHERE id NOT IN (
  SELECT DISTINCT auction_data_id
  FROM auction_deals
  WHERE status != 'finalized'
);

DELETE FROM auction_deals WHERE status = 'finalized';

-- revert to the old version of the status enum
CREATE TYPE status_new AS ENUM (
  'deal-making',
  'confirmation',
  'report-finalized'
);

ALTER TABLE auction_deals 
ALTER COLUMN status TYPE status_new 
USING (status::text::status_new);

DROP TYPE status;

ALTER TYPE status_new RENAME TO status;

COMMIT;