BEGIN;
ALTER TABLE batches ADD COLUMN IF NOT EXISTS proposal_start_offset_seconds BIGINT;
UPDATE batches set proposal_start_offset_seconds=172800;
ALTER TABLE batches ALTER COLUMN proposal_start_offset_seconds SET NOT NULL;
COMMIT;
