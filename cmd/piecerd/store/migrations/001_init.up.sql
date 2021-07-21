CREATE TABLE IF NOT EXISTS unprepared_batches_statuses (
    id SMALLINT PRIMARY KEY,
    name TEXT NOT NULL
);
INSERT INTO unprepared_batches_statuses values (0, 'pending'), (1, 'executing'), (2, 'done');


CREATE TABLE IF NOT EXISTS unprepared_batches (
    storage_deal_id TEXT PRIMARY KEY,
    status smallint NOT NULL DEFAULT 0,
    data_cid TEXT NOT NULL,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_unprepared_batches_statuses FOREIGN KEY(status) REFERENCES unprepared_batches_statuses(id)
);

CREATE INDEX unprepared_batches_status_idx ON unprepared_batches (status, ready_at);

