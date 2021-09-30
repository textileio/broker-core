BEGIN;

ALTER TABLE storage_requests ADD COLUMN status_int smallint;

UPDATE storage_requests SET status_int = 0 WHERE status = 'unknown';
UPDATE storage_requests SET status_int = 1 WHERE status = 'batching';
UPDATE storage_requests SET status_int = 2 WHERE status = 'preparing';
UPDATE storage_requests SET status_int = 3 WHERE status = 'auctioning';
UPDATE storage_requests SET status_int = 4 WHERE status = 'deal_making';
UPDATE storage_requests SET status_int = 5 WHERE status = 'success';
UPDATE storage_requests SET status_int = 6 WHERE status = 'error';

ALTER TABLE storage_requests DROP COLUMN status;

ALTER TABLE storage_requests RENAME COLUMN status_int TO status;

ALTER TABLE storage_requests ALTER COLUMN status SET NOT NULL;

DROP TYPE storage_request_status;

ALTER TABLE batches ADD COLUMN status_str text;

UPDATE batches SET status_str = 'unknown' WHERE status = 'unknown';
UPDATE batches SET status_str = 'preparing' WHERE status = 'preparing';
UPDATE batches SET status_str = 'auctioning' WHERE status = 'auctioning';
UPDATE batches SET status_str = 'deal making' WHERE status = 'deal_making';
UPDATE batches SET status_str = 'success' WHERE status = 'success';
UPDATE batches SET status_str = 'error' WHERE status = 'error';

ALTER TABLE batches DROP COLUMN status;

ALTER TABLE batches RENAME COLUMN status_str TO status;

ALTER TABLE batches ALTER COLUMN status SET NOT NULL;

DROP TYPE batch_status;

COMMIT;
