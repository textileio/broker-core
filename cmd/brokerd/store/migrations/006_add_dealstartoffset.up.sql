BEGIN;
ALTER TABLE batches ADD COLUMN IF NOT EXISTS proposal_start_offset_seconds BIGINT NOT NULL;
UPDATE batches set proposal_start_offset_seconds=172800;
COMMIT;
