select
    url, started_at, retries, geoip->>'ip' as ip, geoip->>'country_name' as country
    from jobs
    where error is null and uploaded_at is null
    order by started_at desc;