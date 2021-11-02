-- name: CreateStorageRequest :exec
insert into storage_requests (id, data_cid, origin) values ($1, $2, $3);

-- name: CreateBatch :exec
insert into batches (id, storage_request_ids) values ($1, $2);

-- name: CreateFinalDeal :exec
insert into final_deals (
  batch_id,
  origin,
  storage_provider_id,
  deal_id,
  deal_expiration,
  error_cause
) values ($1, $2, $3, $4, $5, $6);

-- name: ListAll :many
select 
  batch_id,
  storage_provider_id,
  deal_id,
  data_cid,
  origin
from final_deals
inner join (
  select id, unnest(storage_request_ids) storage_request_id from batches
) as batches on batch_id = batches.id
inner join storage_requests on storage_request_id = storage_requests.id;
