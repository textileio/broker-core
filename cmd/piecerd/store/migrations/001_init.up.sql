CREATE TABLE IF NOT EXISTS unprepared_batches_statuses (
    id SMALLINT PRIMARY KEY,
    name TEXT NOT NULL
);
INSERT INTO unprepared_batches_statuses values (1, 'pending'), (2, 'executing');


CREATE TABLE IF NOT EXISTS unprepared_batches (
    storage_deal_id TEXT PRIMARY KEY,
    status smallint NOT NULL,
    data_cid TEXT NOT NULL,
    ready_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_unprepared_batches_statuses FOREIGN KEY(status) REFERENCES unprepared_batches_statuses(id)
);
