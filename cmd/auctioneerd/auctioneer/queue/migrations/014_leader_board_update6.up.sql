create or replace view competition_results as (
  with a as(
    select
      storage_provider_id,
      width_bucket(
        received_at,
        (select array(
            select generate_series(
                timestamp '2021-11-08 00:00:00.000000+00', 
                timestamp '2021-11-22 00:00:00.000000+00', 
                '2 hours'
          )
        ))
      ) bucket
    from bids
    where received_at > timestamp '2021-11-08 00:00:00.000000+00'
      and received_at < timestamp '2021-11-22 00:00:00.000000+00'
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
      where received_at > timestamp '2021-11-08 00:00:00.000000+00'
        and won_at is not null
        and auctioneer.auctions.client_address != 'f3w5fx6wta4ewl2iyf7xcogmzffz2fmrngpzdpduj3xmk3dwjxc6dyq36gdf3rflkkrblh5nci5xymc5hal3qq'
        and (
          (deal_confirmed_at is not null and deal_confirmed_at < timestamp '2021-11-22 00:00:00.000000+00')
          or (deal_failed_at is not null and deal_failed_at < timestamp '2021-11-22 00:00:00.000000+00')
        )
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
