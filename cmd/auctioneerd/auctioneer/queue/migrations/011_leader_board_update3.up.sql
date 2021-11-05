begin;

drop view if exists competition_results;

create view competition_results as (
  with a as(
    select
      storage_provider_id,
      width_bucket(
        received_at,
        (select array(
            select generate_series(
                now()::timestamp - interval '2 weeks', 
                now()::timestamp, 
                '2 hours'
          )
        ))
      ) bucket
    from bids
    where received_at > now()::timestamp - interval '2 weeks'
      and won_at is not null
    group by storage_provider_id, bucket
  ), 
  b as (
    select
      bids.storage_provider_id,
          case when deal_confirmed_at is null then 0 else 1 end success,
      case when deal_confirmed_at is null then 0 else auctions.deal_size/(1.074*pow(10,9)) end deal_size
      from bids
      join auctions on bids.auction_id = auctions.id
      where received_at > current_timestamp - interval '2 weeks'
      and won_at is not null
      and (deal_confirmed_at is not null or deal_failed_at is not null)
  ),
  c as (
    select
      b.storage_provider_id,
      sum(b.success) as successful_deals,
      count(*) as auctions_won,
      sum(b.success)::float/count(*)::float*100 as success_rate,
      sum(deal_size) as gibs,
      case when sum(deal_size) >= 1024 then true else false end qualified
    from b
    group by storage_provider_id
  )
  select
    c.storage_provider_id,
    successful_deals,
    auctions_won,
    success_rate,
    count(bucket) * 0.05 as bonus,
    count(bucket) * 0.05 + success_rate as total,
    gibs,
    qualified
  from c
    join a on c.storage_provider_id = a.storage_provider_id
  group by c.storage_provider_id,
    successful_deals,
    auctions_won,
    success_rate,
    gibs,
    qualified
  order by qualified desc, total desc, gibs desc
);

commit;
