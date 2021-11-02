begin;

create table if not exists storage_requests (
  id text primary key not null,
  data_cid text not null,
  origin text not null
);

create table if not exists batches (
  id text primary key not null,
  storage_request_ids text[]
);

create table if not exists final_deals(
  batch_id text not null,
  origin text not null,
  storage_provider_id text not null,
  deal_id bigint not null,
  deal_expiration bigint not null,
  error_cause text,
  pending boolean default true,
  retries integer default 0,
  txn text,
  reporting_error text,
  created_at timestamp not null default current_timestamp,

  primary key(storage_provider_id, deal_id)
);

-- operations for readyToBatch and finalizedDeal

commit;