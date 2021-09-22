BEGIN;
ALTER TABLE auctions ADD COLUMN client_address TEXT;
UPDATE auctions SET client_address = 'f144zep4gitj73rrujd3jw6iprljicx6vl4wbeavi';
ALTER TABLE auctions ALTER COLUMN client_address SET NOT NULL;
COMMIT;
