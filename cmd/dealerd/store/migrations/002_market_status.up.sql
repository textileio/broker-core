CREATE TABLE IF NOT EXISTS market_deal_status (
    id bigint NOT NULL PRIMARY KEY,
    name text NOT NULL
);

INSERT INTO market_deal_status (id, name) 
   VALUES
    (0, 'Unknown'),
    (1, 'Proposal not found'),
    (2, 'Proposal rejected'),
    (3, 'Proposal accepted'),
    (4, 'Staged'),
    (5, 'Sealing'),
    (6, 'Finalizing'),
    (7, 'Active'),
    (8, 'Expired'),
    (9, 'Slashed'),
    (10, 'Rejecting'),
    (11, 'Failing'),
    (12, 'Funds reserved'),
    (13, 'Checking for deal acceptance'),
    (14, 'Validating'),
    (15, 'Accept wait'),
    (16, 'Starting data transfer'),
    (17, 'Transferring'),
    (18, 'Waiting for data'),
    (19, 'Verifying data'),
    (20, 'Reserving provider funds'),
    (21, 'Reserving client funds'),
    (22, 'Provider funding'),
    (23, 'Client funding'),
    (24, 'Publish'),
    (25, 'Publishing'),
    (26, 'Error'),
    (27, 'Provider transfer await restart'),
    (28, 'Client transfer restart'),
    (29, 'Awaiting a PreCommit message on chain');
	
