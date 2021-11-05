create or replace view competition_results as (
  with b as (
    select
      storage_provider_id,
      1 - extract(epoch from current_timestamp - received_at) / extract(epoch from interval '2 weeks') as freshness,
      case when deal_confirmed_at is null then 1 else 0 end failed,
		  case when deal_confirmed_at is null then 0 else deal_size/(1.074*pow(10,9)) end deal_size
    from bids
      join auctions on bids.auction_id = auctions.id
    where received_at > current_timestamp - interval '2 weeks'
      and won_at is not null
      and (deal_confirmed_at is not null or deal_failed_at is not null)
  )
  select
    b.storage_provider_id,
    sum(b.failed) as failed_deals,
    count(*) as auctions_won,
    sum(b.failed)::float/count(*)::float as failure_rate,
    sum(b.freshness*b.failed)/count(*) as decaying_failure_rate,
    sum(deal_size) as gibs,
    case when sum(deal_size) >= 1024 then true else false end qualified
  from b
  group by storage_provider_id
);