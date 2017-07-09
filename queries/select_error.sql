select
    id, url, started_at, error, retries, geoip->>'ip' as ip, geoip->>'country_name' as country
    from jobs
    where error is not null
    order by started_at desc
    limit 100;