BEGIN;

CREATE TYPE storage_request_status AS ENUM ('unknown', 'batching', 'preparing', 'auctioning', 'deal_making', 'success', 'error');

ALTER TABLE storage_requests ADD COLUMN status_enum storage_request_status;

UPDATE storage_requests SET status_enum = 'unknown' WHERE status = 0;
UPDATE storage_requests SET status_enum = 'batching' WHERE status = 1;
UPDATE storage_requests SET status_enum = 'preparing' WHERE status = 2;
UPDATE storage_requests SET status_enum = 'auctioning' WHERE status = 3;
UPDATE storage_requests SET status_enum = 'deal_making' WHERE status = 4;
UPDATE storage_requests SET status_enum = 'success' WHERE status = 5;
UPDATE storage_requests SET status_enum = 'error' WHERE status = 6;

ALTER TABLE storage_requests DROP COLUMN status;

ALTER TABLE storage_requests RENAME COLUMN status_enum TO status;

ALTER TABLE storage_requests ALTER COLUMN status SET NOT NULL;

CREATE TYPE batch_status AS ENUM ('unknown', 'preparing', 'auctioning', 'deal_making', 'success', 'error');

ALTER TABLE batches ADD COLUMN status_enum batch_status;

UPDATE batches SET status_enum = 'unknown' WHERE status = 'unknown';
UPDATE batches SET status_enum = 'preparing' WHERE status = 'preparing';
UPDATE batches SET status_enum = 'auctioning' WHERE status = 'auctioning';
UPDATE batches SET status_enum = 'deal_making' WHERE status = 'deal making';
UPDATE batches SET status_enum = 'success' WHERE status = 'success';
UPDATE batches SET status_enum = 'error' WHERE status = 'error';

ALTER TABLE batches DROP COLUMN status;

ALTER TABLE batches RENAME COLUMN status_enum TO status;

ALTER TABLE batches ALTER COLUMN status SET NOT NULL;

COMMIT;
