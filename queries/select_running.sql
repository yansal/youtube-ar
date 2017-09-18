select
    url, started_at, retries, geoip->>'ip' as ip, geoip->>'country_name' as country
    from jobs
    where status = 'running'
    order by started_at desc;