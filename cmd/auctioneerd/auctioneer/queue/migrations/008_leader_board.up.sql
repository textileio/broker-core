create or replace view competition_results as (
  with b as (
    select
      storage_provider_id,
      1 - extract(epoch from current_timestamp - received_at) / extract(epoch from interval '2 weeks') AS freshness,
          case when deal_confirmed_at is null then 1 else 0 END failed,
      case when deal_confirmed_at IS null then 0 else deal_size/(1.074*pow(10,9)) end deal_size
      from bids
      join auctions on bids.auction_id = auctions.id
      where received_at > current_timestamp - interval '2 weeks' and won_at IS NOT NULL
        and (deal_failed_at IS NOT NULL OR received_at < current_timestamp - interval '1 days')
  )
  select
    b.storage_provider_id,
    sum(b.failed) as failed_deals,
    count(*) as auctions_won,
    sum(b.failed)::float/count(*)::float as failure_rate,
    sum(b.freshness*b.failed)/count(*) as decaying_failure_rate,
    sum(deal_size) as gibs,
    case when sum(deal_size) >= 500 then true else false end qualified
  from b
  group by storage_provider_id
  order by qualified desc, decaying_failure_rate asc, gibs desc
);